package server

import (
	"strconv"
	"time"

	"github.com/redis-server/core"
	"github.com/redis-server/resp"
)

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
		if client.flag&MULTI_MODE != 0 {
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

		case "SUBSCRIBE":
			client.flag |= CLIENT_SUB

			for _, key := range cmd.Args {
				subscribers[key] = append(subscribers[key], client)
				subscriberCount[key]++

				client.ReplyBuf = append(client.ReplyBuf, resp.Encode([]any{"subscribe", key, subscriberCount[key]}, false)...)
			}

			continue

		case "UNSUBSCRIBE":

			if len(cmd.Args) > 0 {
				for _, key := range cmd.Args {
					subscribers[key] = removeClient(subscribers[key], client)
					subscriberCount[key]--

					if len(subscribers[key]) == 0 {
						delete(subscribers, key)
						delete(subscriberCount, key)
					}
				}
			} else {
				for key := range subscribers {
					subscribers[key] = removeClient(subscribers[key], client)
					subscriberCount[key]--

					if len(subscribers[key]) == 0 {
						delete(subscribers, key)
						delete(subscriberCount, key)
					}
				}
			}

			client.flag &= ^CLIENT_SUB
			continue

		case "BLPOP":
			client.flag |= CLIENT_BLOCKED
			if core.IsEmpty(cmd.Args[0]) {
				waitingKeys[cmd.Args[0]] = append(waitingKeys[cmd.Args[0]], client)
				timeout, _ := strconv.Atoi(cmd.Args[1])
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

		case "MULTI":

			client.flag |= MULTI_MODE
			client.multistate.cmds = nil
			client.ReplyBuf = append(client.ReplyBuf, []byte("+OK\r\n")...)

			continue

		case "EXEC":

			if client.flag != 1 {
				client.ReplyBuf = append(client.ReplyBuf,
					[]byte("-ERR EXEC without MULTI\r\n")...)
				break
			}

			if len(client.multistate.cmds) == 0 {
				client.flag = 0
				client.ReplyBuf = append(client.ReplyBuf, []byte("*0\r\n")...)
				break
			}

			results := make([][]byte, 0, len(client.multistate.cmds))

			for _, qcmd := range client.multistate.cmds {
				results = append(results, core.Eval(qcmd))
			}

			client.multistate.cmds = nil
			client.flag = 0

			client.ReplyBuf = append(client.ReplyBuf, resp.EncodeExecArray(results)...)

			continue

		case "DISCARD":
			if client.flag != 1 {
				client.ReplyBuf = append(client.ReplyBuf,
					[]byte("-ERR DISCARD without MULTI\r\n")...)
				break
			}

			client.multistate.cmds = nil
			client.flag = 0
			client.ReplyBuf = append(client.ReplyBuf, []byte("+OK\r\n")...)

			continue

		default:
			client.ReplyBuf = append(client.ReplyBuf, core.Eval(cmd)...)
		}
	}
}
