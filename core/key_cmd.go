package core

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis-server/resp"
)

type PingCmd struct{}
type GetCmd struct{}
type SetCmd struct{}
type DelCmd struct{}
type TtlCmd struct{}
type ExpCmd struct{}
type IncrCmd struct{}

func (PingCmd) Name() string { return "PING" }
func (GetCmd) Name() string  { return "GET" }
func (SetCmd) Name() string  { return "SET" }
func (DelCmd) Name() string  { return "DEL" }
func (TtlCmd) Name() string  { return "TTL" }
func (ExpCmd) Name() string  { return "EXPIRE" }
func (IncrCmd) Name() string { return "INCR" }

func (PingCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
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

func (IncrCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'incr' command"), false)
	}

	var key string = args[0]
	obj := Get(key)

	if obj == nil {
		obj = NewObj("0", -1, OBJ_TYPE_STRING, OBJ_ENCODING_INT)
		Put(key, obj)
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_STRING); err != nil {
		return resp.Encode(err, false)
	}

	if err := assertEncoding(obj.TypeEncoding, OBJ_ENCODING_INT); err != nil {
		return resp.Encode(err, false)
	}

	i, _ := strconv.ParseInt(obj.Value.(string), 10, 64)
	i++
	obj.Value = strconv.FormatInt(i, 10)

	return resp.Encode(i, false)
}

func (ExpCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
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

	setExpiry(obj, exDurationSec*1000)

	return RESP_ONE
}

func (TtlCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'get' command"), false)
	}

	var key string = args[0]

	obj := Get(key)

	if obj == nil {
		return RESP_MINUS_TWO
	}

	exp, isExpirySet := getExpiry(obj)

	if !isExpirySet {
		return RESP_MINUS_ONE
	}

	if uint64(time.Now().UnixMilli()) > exp {
		return RESP_MINUS_TWO
	}

	durationMs := exp - uint64(time.Now().UnixMilli())

	return resp.Encode((durationMs / 1000), false)
}

func (DelCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	var countDeleted int = 0

	for _, key := range args {
		if ok := Del(key); ok {
			countDeleted++
		}
	}

	return resp.Encode(countDeleted, false)

}

func (SetCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) <= 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'set' command"), false)
	}

	var key, value string
	var exDurationsMs int64 = -1

	key, value = args[0], args[1]
	oType, oEnc := deduceTypeEncoding(value)

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
	Put(key, NewObj(value, exDurationsMs, oType, oEnc))
	return RESP_OK
}

func (GetCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'get' command"), false)
	}

	var key string = args[0]

	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	if hasExpired(obj) {
		return RESP_NIL
	}

	return resp.Encode(obj.Value, false)
}
