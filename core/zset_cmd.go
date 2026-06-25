package core

import (
	"context"
	"strconv"
)

type ZaddCmd struct{}

func (ZaddCmd) Name() string { return "ZADD" }

func (ZaddCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 3 {
		c.AppendReply(errWrongArgs("zadd"), false)
		return
	}
	key := args[0]
	score, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		c.AppendReply(errInvalidInt(), false)
		return
	}
	member := args[2]
	obj, exists := Get(key)
	var z *Zset
	if !exists {
		z = NewZset()
		Put(key, NewObj(z, OBJ_TYPE_ZSET, OBJ_ENCODING_SKIPLIST), -1)
	} else {
		if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
			c.AppendReply(errWrongType(), false)
			return
		}
		z = obj.Value.(*Zset)
	}
	z.Add(member, int(score))
	c.AppendReply(RESP_ONE, false)
}

type ZremCmd struct{}

func (ZremCmd) Name() string { return "ZREM" }

func (ZremCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) < 2 {
		c.AppendReply(errWrongArgs("zrem"), false)
		return
	}
	key := args[0]
	obj, exists := Get(key)
	if !exists {
		c.AppendReply(nil, false)
		return
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		c.AppendReply(errWrongType(), false)
		return
	}
	zs := obj.Value.(*Zset)
	count := 0
	for _, member := range args[1:] {
		count += zs.Delete(member)
	}
	c.AppendReply(count, false)
}

type ZscoreCmd struct{}

func (ZscoreCmd) Name() string { return "ZSCORE" }

func (ZscoreCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 2 {
		c.AppendReply(errWrongArgs("zscore"), false)
		return
	}
	key := args[0]
	member := args[1]
	obj, exists := Get(key)
	if !exists {
		c.AppendReply(nil, false)
		return
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		c.AppendReply(errWrongType(), false)
		return
	}
	zs := obj.Value.(*Zset)
	score, exists := zs.Search(member)
	if !exists {
		c.AppendReply(nil, false)
		return
	}
	c.AppendReply(score, false)
}

type ZrangeCmd struct{}

func (ZrangeCmd) Name() string { return "ZRANGE" }

func (ZrangeCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 3 {
		c.AppendReply(errWrongArgs("zrange"), false)
		return
	}
	key := args[0]
	start, err1 := strconv.ParseInt(args[1], 10, 64)
	stop, err2 := strconv.ParseInt(args[2], 10, 64)
	if err1 != nil || err2 != nil {
		c.AppendReply(errInvalidInt(), false)
		return
	}
	obj, exists := Get(key)
	if !exists {
		c.AppendReply(nil, false)
		return
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		c.AppendReply(errWrongType(), false)
		return
	}
	zs := obj.Value.(*Zset)
	members := zs.Range(int(start), int(stop))
	c.AppendReply(members, false)
}
