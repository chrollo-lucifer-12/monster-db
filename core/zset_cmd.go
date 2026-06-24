package core

import (
	"context"
	"strconv"

	"github.com/redis-server/resp"
)

type ZaddCmd struct{}

func (ZaddCmd) Name() string { return "ZADD" }

func (ZaddCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 3 {
		return errWrongArgs("zadd")
	}
	key := args[0]
	score, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return errInvalidInt()
	}
	member := args[2]

	obj, exists := Get(key)
	var z *Zset
	if !exists {
		z = NewZset()
		Put(key, NewObj(z, OBJ_TYPE_ZSET, OBJ_ENCODING_SKIPLIST), -1)
	} else {
		if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
			return errWrongType()
		}
		z = obj.Value.(*Zset)
	}
	z.Add(member, int(score))
	return RESP_ONE
}

type ZremCmd struct{}

func (ZremCmd) Name() string { return "ZREM" }

func (ZremCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) < 2 {
		return errWrongArgs("zrem")
	}
	key := args[0]
	obj, exists := Get(key)
	if !exists {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		return errWrongType()
	}
	zs := obj.Value.(*Zset)
	count := 0
	for _, member := range args[1:] {
		count += zs.Delete(member)
	}
	return resp.Encode(count, false)
}

type ZscoreCmd struct{}

func (ZscoreCmd) Name() string { return "ZSCORE" }

func (ZscoreCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 2 {
		return errWrongArgs("zscore")
	}
	key := args[0]
	member := args[1]

	obj, exists := Get(key)
	if !exists {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		return errWrongType()
	}
	zs := obj.Value.(*Zset)
	score, exists := zs.Search(member)
	if !exists {
		return RESP_NIL
	}
	return resp.Encode(score, false)
}

type ZrangeCmd struct{}

func (ZrangeCmd) Name() string { return "ZRANGE" }

func (ZrangeCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 3 {
		return errWrongArgs("zrange")
	}
	key := args[0]
	start, err1 := strconv.ParseInt(args[1], 10, 64)
	stop, err2 := strconv.ParseInt(args[2], 10, 64)
	if err1 != nil || err2 != nil {
		return errInvalidInt()
	}

	obj, exists := Get(key)
	if !exists {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		return errWrongType()
	}
	zs := obj.Value.(*Zset)
	members := zs.Range(int(start), int(stop))
	return resp.Encode(members, false)
}
