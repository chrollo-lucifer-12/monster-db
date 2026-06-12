package server

import (
	"log"
	"strconv"
	"time"

	"github.com/redis-server/core"
	"github.com/redis-server/resp"
)

func UnwatchAllKeys(client *Client) {
	for key, clients := range watchedKeys {
		watchedKeys[key] = removeClient(clients, client)

		if len(watchedKeys[key]) == 0 {
			delete(watchedKeys, key)
		}
	}

	client.flag &= ^CLIENT_CAS
}

func removeClient(slice []*Client, target *Client) []*Client {
	newSlice := make([]*Client, 0, len(slice))

	for _, c := range slice {
		if c != target {
			newSlice = append(newSlice, c)
		}
	}

	return newSlice
}

func respond(cmds core.RedisCmds, client *Client, loop *EventLoop) {

	for _, cmd := range cmds {

		if client.flag&MULTI_MODE != 0 && cmd.Cmd != "EXEC" && cmd.Cmd != "DISCARD" {
			client.multistate.cmds = append(client.multistate.cmds, cmd)
			client.ReplyBuf = append(client.ReplyBuf, []byte("+QUEUED\r\n")...)
			continue
		}

		if client.flag&CLIENT_BLOCKED != 0 {
			return
		}

		if client.flag&CLIENT_SUB != 0 {
			client.ReplyBuf = append(client.ReplyBuf, []byte("-ERR only (P)SUBSCRIBE / (P)UNSUBSCRIBE / PING / QUIT allowed in this context\r\n")...)
			continue
		}

		switch cmd.Cmd {

		case "REPLICAOF":
			HandleReplicaOfCommand(client, cmd.Args)

		case "SUBSCRIBE":

			for _, key := range cmd.Args {

				if _, exists := client.subscriptions[key]; exists {
					continue
				}

				subscribers[key] = append(subscribers[key], client)

				client.subscriptions[key] = struct{}{}
				client.flag |= CLIENT_SUB

				client.ReplyBuf = append(client.ReplyBuf, resp.Encode([]any{"subscribe", key, len(client.subscriptions)}, false)...)
			}

			continue

		case "UNSUBSCRIBE":

			var targets []string
			if len(cmd.Args) > 0 {
				targets = cmd.Args
			} else {
				for key := range client.subscriptions {
					targets = append(targets, key)
				}
			}

			for _, key := range targets {
				if _, exists := client.subscriptions[key]; !exists {
					continue
				}

				delete(client.subscriptions, key)
				subscribers[key] = removeClient(subscribers[key], client)

				if len(subscribers[key]) == 0 {
					delete(subscribers, key)
				}

				client.ReplyBuf = append(client.ReplyBuf, resp.Encode([]any{"unsubscribe", key, len(client.subscriptions)}, false)...)
			}

			if len(client.subscriptions) == 0 {
				client.flag &= ^CLIENT_SUB
			}

			continue

		case "PUBLISH":
			key := cmd.Args[0]
			message := cmd.Args[1]

			c := 0

			for _, sub_client := range subscribers[key] {
				sub_client.ReplyBuf = append(sub_client.ReplyBuf, resp.Encode([]string{"message", key, message}, false)...)
				clientsPendingWrite[sub_client.Fd] = sub_client
				c++
			}

			client.ReplyBuf = append(client.ReplyBuf, resp.Encode(c, false)...)

			continue

		case "BLPOP":

			if core.IsEmpty(cmd.Args[0]) {
				waitingKeys[cmd.Args[0]] = append(waitingKeys[cmd.Args[0]], client)
				timeout, _ := strconv.Atoi(cmd.Args[1])
				log.Printf(
					"SETTING BLOCKED: client=%p fd=%d flags_before=%08b",
					client,
					client.Fd,
					client.flag,
				)
				client.flag |= CLIENT_BLOCKED
				if timeout > 0 {
					client.when = time.Now().Add(time.Duration(timeout) * time.Millisecond)
				} else {
					client.when = time.Time{}
				}
			} else {
				client.ReplyBuf = append(client.ReplyBuf, core.Eval(&core.RedisCmd{
					Cmd:  "LPOP",
					Args: []string{cmd.Args[0]},
				})...)
			}

			return

		case "WATCH":
			if client.flag&MULTI_MODE != 0 {
				client.ReplyBuf = append(client.ReplyBuf, resp.Encode("WATCH inside MULTI not allowed", false)...)
				return
			}

			for _, key := range cmd.Args {
				watchedKeys[key] = append(watchedKeys[key], client)
			}

			client.ReplyBuf = append(client.ReplyBuf, core.RESP_OK...)

			continue

		case "MULTI":

			client.flag |= MULTI_MODE
			client.multistate.cmds = nil
			client.ReplyBuf = append(client.ReplyBuf, []byte("+OK\r\n")...)

			continue

		case "EXEC":

			if client.flag&MULTI_MODE == 0 {
				client.ReplyBuf = append(client.ReplyBuf,
					[]byte("-ERR EXEC without MULTI\r\n")...)
				break
			}

			if client.flag&CLIENT_CAS != 0 {
				UnwatchAllKeys(client)
				client.multistate.cmds = nil
				client.flag &= ^MULTI_MODE

				client.ReplyBuf = append(client.ReplyBuf, core.RESP_NIL...)
				break
			}

			if len(client.multistate.cmds) == 0 {
				client.flag &= ^MULTI_MODE
				client.ReplyBuf = append(client.ReplyBuf, []byte("*0\r\n")...)
				break
			}

			results := make([][]byte, 0, len(client.multistate.cmds))

			for _, qcmd := range client.multistate.cmds {
				results = append(results, core.Eval(qcmd))
			}

			client.multistate.cmds = nil
			client.flag &= ^MULTI_MODE

			UnwatchAllKeys(client)

			client.ReplyBuf = append(client.ReplyBuf, resp.EncodeExecArray(results)...)

			continue

		case "DISCARD":
			if client.flag&MULTI_MODE == 0 {
				client.ReplyBuf = append(client.ReplyBuf,
					[]byte("-ERR DISCARD without MULTI\r\n")...)
				break
			}

			client.multistate.cmds = nil
			client.flag &= ^MULTI_MODE

			UnwatchAllKeys(client)

			client.ReplyBuf = append(client.ReplyBuf, []byte("+OK\r\n")...)

			continue

		default:
			client.ReplyBuf = append(client.ReplyBuf, core.Eval(cmd)...)
		}
	}
}
