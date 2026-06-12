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

		case "REPLCONF":
			HandleMasterReplConfCommand(client, cmd.Args)

		case "PSYNC":
			HandleMasterPsyncCommand(loop, client, cmd.Args)

		case "REPLICAOF":
			HandleReplicaOfCommand(client, cmd.Args)

		case "SUBSCRIBE":

			HandleSubscribe(client, cmd.Args)
			continue

		case "UNSUBSCRIBE":

			HandleUnsubscribe(client, cmd.Args)
			continue

		case "PUBLISH":

			HandlePublish(client, cmd.Args)
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

			err := HandleWatch(client, cmd.Args)
			if err != nil {
				break
			}
			continue

		case "MULTI":

			HandleMulti(client, cmd.Args)
			continue

		case "EXEC":
			err := HandleExec(client, cmd.Args)
			if err != nil {
				break
			}
			continue

		case "DISCARD":

			err := HandleDiscard(client, cmd.Args)
			if err != nil {
				break
			}
			continue

		default:
			if cmd.Cmd == "SET" || cmd.Cmd == "DEL" || cmd.Cmd == "RPUSH" || cmd.Cmd == "LPUSH" {
				args := append([]string{cmd.Cmd}, cmd.Args...)
				HandleInformReplicas(resp.Encode(args, false))
			}

			client.ReplyBuf = append(client.ReplyBuf, core.Eval(cmd)...)
		}
	}
}
