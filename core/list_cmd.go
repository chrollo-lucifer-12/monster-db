package core

import (
	"context"
	"errors"
	"strconv"

	"github.com/redis-server/resp"
)

type LLENCmd struct{}
type LPUSHCmd struct{}
type RPUSHCmd struct{}
type LRANGECmd struct{}
type LPOPCmd struct{}

func (LLENCmd) Name() string   { return "LLEN" }
func (LPUSHCmd) Name() string  { return "LPUSH" }
func (RPUSHCmd) Name() string  { return "RPUSH" }
func (LRANGECmd) Name() string { return "LRANGE" }
func (LPOPCmd) Name() string   { return "LPOP" }

func (LLENCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'llen' command"), false)
	}

	var key string = args[0]
	obj, exists := Get(key)

	if !exists {
		return RESP_ZERO
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
		return RESP_ZERO
	}

	ql := obj.Value.(*Quicklist)

	return resp.Encode(ql.len, false)
}

func (LPUSHCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) < 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'lpush' command"), false)
	}

	var key string = args[0]
	obj, exists := Get(key)

	var ql *Quicklist

	if !exists {
		ql = NewQuicklist()
		obj = NewObj(ql, OBJ_TYPE_LIST, OBJ_ENCODING_LISTPACK)
		Put(key, obj, -1)
	} else {
		if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
			return resp.Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
		}

		ql = obj.Value.(*Quicklist)
	}

	oldLen := ql.len

	for i := 1; i < len(args); i++ {
		val, err := strconv.Atoi(args[i])
		if err != nil {
			ql.addToHead(args[i])
		} else {
			ql.addToHead(val)
		}
	}

	if oldLen == 0 {
		MarkReady(key)
	}

	return resp.Encode(ql.len, false)
}

func (RPUSHCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) < 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'lpush' command"), false)
	}

	var key string = args[0]
	obj, exists := Get(key)

	var ql *Quicklist

	if !exists {
		ql = NewQuicklist()
		obj = NewObj(ql, OBJ_TYPE_LIST, OBJ_ENCODING_LISTPACK)
		Put(key, obj, -1)
	} else {
		if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
			return resp.Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
		}

		ql = obj.Value.(*Quicklist)
	}

	oldLen := ql.len

	for i := 1; i < len(args); i++ {
		val, err := strconv.Atoi(args[i])
		if err != nil {
			ql.addToTail(args[i])
		} else {
			ql.addToTail(val)
		}
	}

	if oldLen == 0 {
		MarkReady(key)
	}

	return resp.Encode(ql.len, false)
}

func (LRANGECmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 3 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'lrange' command"), false)
	}

	var key string = args[0]
	start, _ := strconv.Atoi(args[1])
	stop, _ := strconv.Atoi(args[2])

	obj, exists := Get(key)

	if !exists {
		return RESP_NIL
	}

	ql := obj.Value.(*Quicklist)

	res := ql.GetElements(start, stop)

	return resp.Encode(res, false)
}

func (LPOPCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) < 1 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'lpop' command"), false)
	}

	var key string = args[0]
	count := 1

	if len(args) == 2 {
		count, _ = strconv.Atoi(args[1])
	}

	obj, exists := Get(key)

	if !exists {
		return RESP_NIL
	}

	ql := obj.Value.(*Quicklist)

	res := ql.RemoveElements(count)

	return resp.Encode(res, false)
}
