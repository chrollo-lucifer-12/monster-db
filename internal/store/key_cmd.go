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
		c.AppendError(errWrongArgs("ping"))
		//	c.AppendReply(errWrongArgs("ping"), false)
		return
	}

	if len(args) == 0 {
		c.AppendSimpleString("PONG")
		//	c.AppendReply("PONG", true)
	} else {
		c.AppendBulkString(args[0])
		//	c.AppendReply(args[0], false)
	}

}

func (IncrCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 1 {
		c.AppendError(errWrongArgs("incr"))
		//	c.AppendReply(errWrongArgs("incr"), false)
		return
	}

	var key string = args[0]
	obj, exists := Get(key)

	if !exists {
		obj = NewStringObj("0", OBJ_TYPE_STRING, OBJ_ENCODING_INT)
		Put(key, obj, -1)
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_STRING); err != nil {
		c.AppendError(err.Error())
		//c.AppendReply(err, false)
		return
	}

	if err := assertEncoding(obj.TypeEncoding, OBJ_ENCODING_INT); err != nil {
		c.AppendError(err.Error())
		return
	}

	i, _ := strconv.ParseInt(obj.StrVal, 10, 64)
	i++
	obj.StrVal = strconv.FormatInt(i, 10)
	store[key] = obj
	c.AppendIntReply(i)
	//	c.AppendReply(i, false)
}

func (ExpCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) <= 1 {
		c.AppendError(errWrongArgs("expire"))
		//c.AppendReply(errWrongArgs("expire"), false)
		return
	}

	var key string = args[0]
	exDurationSec, err := strconv.ParseInt(args[1], 10, 64)

	if err != nil {
		c.AppendError(errInvalidInt())
		//	c.AppendReply(errInvalidInt(), false)
		return
	}

	_, exists := Get(key)

	if !exists {
		c.AppendBytesReply(RESP_ZERO)
		return
	}

	setExpiry(key, exDurationSec*1000)

	c.AppendBytesReply(RESP_ONE)
}

func (TtlCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 1 {
		c.AppendError(errWrongArgs("ttl"))
		//c.AppendReply(errWrongArgs("ttl"), false)
		return
	}

	var key string = args[0]

	_, exists := Get(key)

	if !exists {
		c.AppendBytesReply(RESP_MINUS_TWO)
		return
	}

	exp, isExpirySet := getExpiry(key)

	if !isExpirySet {
		c.AppendBytesReply(RESP_MINUS_ONE)
		return
	}

	if uint64(time.Now().UnixMilli()) > exp {
		c.AppendBytesReply(RESP_MINUS_TWO)
		return
	}

	durationMs := exp - uint64(time.Now().UnixMilli())

	c.AppendIntReply(int64(durationMs / 1000))
}

func (DelCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	var countDeleted int = 0

	for _, key := range args {
		if ok := Del(key); ok {
			countDeleted++
		}
	}
	c.AppendIntReply(int64(countDeleted))

}

func (SetCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) <= 1 {
		c.AppendError(errWrongArgs("set"))
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
				c.AppendError("ERR syntax error")
				return
			}

			exDurationSec, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				c.AppendError(errInvalidInt())
				return
			}

			exDurationsMs = exDurationSec * 1000

		default:
			c.AppendError("ERR syntax error")
			return
		}
	}
	Put(key, NewStringObj(value, oType, oEnc), exDurationsMs)
	c.AppendSimpleString("OK")
}

func (GetCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 1 {
		c.AppendError("ERR wrong number of arguments for 'get' command")
		return
	}

	key := args[0]
	obj, exists := Get(key)

	if !exists || hasExpired(key) {
		c.AppendNull()
		return
	}

	c.AppendBulkString(obj.StrVal)
}
