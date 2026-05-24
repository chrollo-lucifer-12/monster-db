package server

import (
	"fmt"
	"io"
	"net"
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

func readCommands(client io.ReadWriter) (core.RedisCmds, error) {

	var accumulatedData []byte
	chunk := make([]byte, 512)

	for {
		n, err := client.Read(chunk)
		if n > 0 {
			accumulatedData = append(accumulatedData, chunk[:n]...)
		}

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			if err == io.EOF {
				return nil, err
			}
			return nil, err
		}

		if n < len(chunk) {
			break
		}
	}

	if len(accumulatedData) == 0 {
		return nil, nil
	}

	values, err := resp.Decode(accumulatedData)

	if err != nil {
		return nil, err
	}

	var cmds []*core.RedisCmd = make([]*core.RedisCmd, 0)

	for _, value := range values {
		tokens, err := toArrayString(value.([]interface{}))
		if err != nil {
			return nil, err
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
