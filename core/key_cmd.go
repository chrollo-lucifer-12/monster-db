package core

import (
	"context"
	"strconv"
	"time"
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

func (PingCmd) Execute(ctx context.Context, c ClientCommander, args []string) {

	if len(args) >= 2 {
		c.AppendReply(errWrongArgs("ping"), false)
		return
	}

	if len(args) == 0 {
		c.AppendReply("PONG", true)
	} else {
		c.AppendReply(args[0], false)
	}

}

func (IncrCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 1 {
		c.AppendReply(errWrongArgs("incr"), false)
		return
	}

	var key string = args[0]
	obj, exists := Get(key)

	if !exists {
		obj = NewStringObj("0", OBJ_TYPE_STRING, OBJ_ENCODING_INT)
		Put(key, obj, -1)
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_STRING); err != nil {
		c.AppendReply(err, false)
		return
	}

	if err := assertEncoding(obj.TypeEncoding, OBJ_ENCODING_INT); err != nil {
		c.AppendReply(err, false)
		return
	}

	i, _ := strconv.ParseInt(obj.StrVal, 10, 64)
	i++
	obj.StrVal = strconv.FormatInt(i, 10)
	store[key] = obj
	c.AppendReply(i, false)
}

func (ExpCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) <= 1 {
		c.AppendReply(errWrongArgs("expire"), false)
		return
	}

	var key string = args[0]
	exDurationSec, err := strconv.ParseInt(args[1], 10, 64)

	if err != nil {
		c.AppendReply(errInvalidInt(), false)
		return
	}

	_, exists := Get(key)

	if !exists {
		c.AppendReply(RESP_ZERO, true)
		return
	}

	setExpiry(key, exDurationSec*1000)

	c.AppendReply(RESP_ONE, true)
}

func (TtlCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 1 {
		c.AppendReply(errWrongArgs("ttl"), false)
		return
	}

	var key string = args[0]

	_, exists := Get(key)

	if !exists {
		c.AppendReply(RESP_MINUS_TWO, true)
		return
	}

	exp, isExpirySet := getExpiry(key)

	if !isExpirySet {
		c.AppendReply(RESP_MINUS_ONE, true)
		return
	}

	if uint64(time.Now().UnixMilli()) > exp {
		c.AppendReply(RESP_MINUS_TWO, true)
		return
	}

	durationMs := exp - uint64(time.Now().UnixMilli())

	c.AppendReply(int64(durationMs/1000), false)
}

func (DelCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	var countDeleted int = 0

	for _, key := range args {
		if ok := Del(key); ok {
			countDeleted++
		}
	}
	c.AppendReply(countDeleted, false)

}

func (SetCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) <= 1 {
		c.AppendReply(errWrongArgs("set"), false)
		return
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
				c.AppendReply("ERR syntax error", false)
				return
			}

			exDurationSec, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				c.AppendReply(errInvalidInt(), false)
				return
			}

			exDurationsMs = exDurationSec * 1000

		default:
			c.AppendReply("ERR syntax error", false)
			return
		}
	}
	Put(key, NewStringObj(value, oType, oEnc), exDurationsMs)
	c.AppendReply("OK", true)
}

func (GetCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 1 {
		c.AppendReply(errWrongArgs("get"), false)
		return
	}

	var key string = args[0]

	obj, exists := Get(key)

	if !exists {
		c.AppendReply(nil, false)
		return
	}

	if hasExpired(key) {
		c.AppendReply(nil, false)
		return
	}

	c.AppendReply(obj.StrVal, false)
}
