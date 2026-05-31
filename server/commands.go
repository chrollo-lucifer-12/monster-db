package server

import (
	"fmt"
	"io"
	"strings"

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

func respond(cmds core.RedisCmds, client *Client) {
	for _, cmd := range cmds {
		if client.flag == 1 && cmd.Cmd != "EXEC" && cmd.Cmd != "DISCARD" {
			client.multistate.cmds = append(client.multistate.cmds, cmd)
			client.ReplyBuf = append(client.ReplyBuf, []byte("+QUEUED\r\n")...)
			continue
		}

		switch cmd.Cmd {

		case "MULTI":
			client.flag = 1
			client.multistate.cmds = nil
			client.ReplyBuf = append(client.ReplyBuf, []byte("+OK\r\n")...)

		case "EXEC":
			for _, qcmd := range client.multistate.cmds {
				client.ReplyBuf = append(client.ReplyBuf, core.Eval(qcmd)...)
			}
			client.multistate.cmds = nil
			client.flag = 0

		case "DISCARD":
			client.multistate.cmds = nil
			client.flag = 0
			client.ReplyBuf = append(client.ReplyBuf, []byte("+OK\r\n")...)

		default:
			client.ReplyBuf = append(client.ReplyBuf, core.Eval(cmd)...)
		}
	}
}
