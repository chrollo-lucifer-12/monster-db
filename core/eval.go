package core

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/redis-server/resp"
)

var RESP_NIL []byte = []byte("$-1\r\n")
var RESP_OK []byte = []byte("+OK\r\n")
var RESP_ZERO []byte = []byte(":0\r\n")
var RESP_ONE []byte = []byte(":1\r\n")
var RESP_MINUS_ONE []byte = []byte(":-1\r\n")
var RESP_MINUS_TWO []byte = []byte(":-2\r\n")

func evalPING(args []string) []byte {
	var b []byte

	if len(args) >= 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'ping' command"), false)
	}

	if len(args) == 0 {
		b = resp.Encode("PONG", true)
	} else {
		b = resp.Encode(args[0], false)
	}

	return b
}

func evalSET(args []string) []byte {
	if len(args) <= 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'set' command"), false)
	}

	var key, value string
	var exDurationsMs int64 = -1

	key, value = args[0], args[1]

	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "EX", "ex":
			i++
			if i == len(args) {
				return resp.Encode(errors.New("(error) ERR syntax error"), false)
			}

			exDurationSec, err := strconv.ParseInt(args[3], 10, 64)
			if err != nil {
				return resp.Encode(err, false)
			}

			exDurationsMs = exDurationSec * 1000

		default:
			return resp.Encode(errors.New("(error) ERR syntax error"), false)
		}
	}
	Put(key, NewObj(value, exDurationsMs))
	return RESP_OK
}

func evalGET(args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'get' command"), false)
	}

	var key string = args[0]

	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	if obj.ExpiresAt != -1 && obj.ExpiresAt <= time.Now().UnixMilli() {
		return RESP_NIL
	}

	return resp.Encode(obj.Value, false)
}

func evalTTL(args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'get' command"), false)
	}

	var key string = args[0]

	obj := Get(key)

	if obj == nil {
		return RESP_MINUS_TWO
	}

	if obj.ExpiresAt == -1 {
		return RESP_MINUS_ONE
	}

	durationMs := obj.ExpiresAt - time.Now().UnixMilli()

	if durationMs < 0 {
		return RESP_MINUS_TWO
	}

	return resp.Encode((durationMs / 1000), false)
}

func evalDEL(args []string) []byte {
	var countDeleted int = 0

	for _, key := range args {
		if ok := Del(key); ok {
			countDeleted++
		}
	}

	return resp.Encode(countDeleted, false)

}

func evalEXPIRE(args []string) []byte {
	if len(args) <= 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'espire' command"), false)
	}

	var key string = args[0]
	exDurationSec, err := strconv.ParseInt(args[1], 10, 64)

	if err != nil {
		return resp.Encode(errors.New("(error) ERR value is not an integer or out of range"), false)
	}

	obj := Get(key)

	if obj == nil {
		return RESP_ZERO
	}

	obj.ExpiresAt = time.Now().UnixMilli() + exDurationSec*1000

	return RESP_ONE
}

// make new process
func evalBGREWRITE(args []string) []byte {
	DumpAllAOF()
	return RESP_OK
}

func EvalAndInput(cmds RedisCmds, client io.ReadWriter) {
	var response []byte
	buf := bytes.NewBuffer(response)

	for _, cmd := range cmds {
		switch cmd.Cmd {
		case "PING":
			buf.Write(evalPING(cmd.Args))
		case "SET":
			buf.Write(evalSET(cmd.Args))
		case "GET":
			buf.Write(evalGET(cmd.Args))
		case "TTL":
			buf.Write(evalTTL(cmd.Args))
		case "DEL":
			buf.Write(evalDEL(cmd.Args))
		case "EXPIRE":
			buf.Write(evalEXPIRE(cmd.Args))
		case "BGREWRITEAOF":
			buf.Write(evalBGREWRITE(cmd.Args))
		default:
			buf.Write(evalPING(cmd.Args))
		}
	}

	client.Write(buf.Bytes())
}
