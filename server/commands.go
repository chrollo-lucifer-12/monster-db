package server

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/redis-server/core"
	"github.com/redis-server/resp"
)

func toArrayString(ai []interface{}) ([]string, error) {
	as := make([]string, len(ai))
	for i := range ai {
		as[i] = ai[i].(string)
	}
	return as, nil
}

func readCommands(data []byte) (core.RedisCmds, int, error) {
	if len(data) == 0 {
		return nil, 0, nil
	}

	values, bytesConsumed, err := resp.Decode(data)
	if err != nil {
		return nil, 0, err
	}

	cmds := make([]*core.RedisCmd, 0, len(values))

	for _, value := range values {

		arrayVals, ok := value.([]interface{})
		if !ok {
			continue
		}

		tokens, err := toArrayString(arrayVals)
		if err != nil {
			return nil, 0, err
		}

		if len(tokens) == 0 {
			continue
		}

		cmds = append(cmds, &core.RedisCmd{
			Cmd:  strings.ToUpper(tokens[0]),
			Args: tokens[1:],
		})
	}

	return cmds, bytesConsumed, nil
}

func respondWithError(client io.ReadWriter, err error) {
	client.Write([]byte(fmt.Sprintf("-%s\r\n", err)))
}

func respond(cmds core.RedisCmds, client *Client, loop *EventLoop) {
	for _, cmd := range cmds {
		if client.flag == 1 && cmd.Cmd != "EXEC" && cmd.Cmd != "DISCARD" && cmd.Cmd != "BLPOP" {
			client.multistate.cmds = append(client.multistate.cmds, cmd)
			client.ReplyBuf = append(client.ReplyBuf, []byte("+QUEUED\r\n")...)
			continue
		}

		switch cmd.Cmd {

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

			if client.flag == 1 {
				client.ReplyBuf = append(client.ReplyBuf, []byte("-ERR EXEC without MULTI\r\n")...)
				break
			}

			client.flag = 1
			client.multistate.cmds = nil
			client.ReplyBuf = append(client.ReplyBuf, []byte("+OK\r\n")...)

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

		case "DISCARD":
			if client.flag != 1 {
				client.ReplyBuf = append(client.ReplyBuf,
					[]byte("-ERR DISCARD without MULTI\r\n")...)
				break
			}

			client.multistate.cmds = nil
			client.flag = 0
			client.ReplyBuf = append(client.ReplyBuf, []byte("+OK\r\n")...)

		default:
			client.ReplyBuf = append(client.ReplyBuf, core.Eval(cmd)...)
		}
	}
}
