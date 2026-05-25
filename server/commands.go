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

func readCommands(data []byte) (core.RedisCmds, error) {
	if len(data) == 0 {
		return nil, nil
	}

	values, err := resp.Decode(data)
	if err != nil {
		return nil, err
	}

	cmds := make([]*core.RedisCmd, 0, len(values))

	for _, value := range values {

		tokens, err := toArrayString(value.([]interface{}))
		if err != nil {
			return nil, err
		}

		if len(tokens) == 0 {
			continue
		}

		cmds = append(cmds, &core.RedisCmd{
			Cmd:  strings.ToUpper(tokens[0]),
			Args: tokens[1:],
		})
	}

	return cmds, nil
}

func respondWithError(client io.ReadWriter, err error) {
	client.Write([]byte(fmt.Sprintf("-%s\r\n", err)))
}

func respond(cmds core.RedisCmds, client io.ReadWriter) {
	core.EvalAndInput(cmds, client)
}
