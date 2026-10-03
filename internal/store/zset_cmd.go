package core

import (
	"context"
	"strconv"
)

type ZaddCmd struct{}

func (ZaddCmd) Name() string { return "ZADD" }

func (ZaddCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 3 {
		c.AppendError(errWrongArgs("zadd"))
		return
	}
	key := args[0]
	score, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		c.AppendError(errInvalidInt())
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
			c.AppendError(errWrongType())
			return
		}
		z = obj.Value.(*Zset)
	}
	z.Add(member, int(score))
	c.AppendBytesReply(RESP_ONE)
}

type ZremCmd struct{}

func (ZremCmd) Name() string { return "ZREM" }

func (ZremCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) < 2 {
		c.AppendError(errWrongArgs("zrem"))
		return
	}
	key := args[0]
	obj, exists := Get(key)
	if !exists {
		c.AppendNull()
		return
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		c.AppendError(errWrongType())
		return
	}
	zs := obj.Value.(*Zset)
	count := 0
	for _, member := range args[1:] {
		count += zs.Delete(member)
	}
	c.AppendIntReply(int64(count))
}

type ZscoreCmd struct{}

func (ZscoreCmd) Name() string { return "ZSCORE" }

func (ZscoreCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 2 {
		c.AppendError(errWrongArgs("zscore"))
		return
	}
	key := args[0]
	member := args[1]
	obj, exists := Get(key)
	if !exists {
		c.AppendNull()
		return
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		c.AppendError(errWrongType())
		return
	}
	zs := obj.Value.(*Zset)
	score, exists := zs.Search(member)
	if !exists {
		c.AppendNull()
		return
	}
	c.AppendIntReply(int64(score))
}

type ZrangeCmd struct{}

func (ZrangeCmd) Name() string { return "ZRANGE" }

func (ZrangeCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 3 {
		c.AppendError(errWrongArgs("zrange"))
		return
	}
	key := args[0]
	start, err1 := strconv.ParseInt(args[1], 10, 64)
	stop, err2 := strconv.ParseInt(args[2], 10, 64)
	if err1 != nil || err2 != nil {
		c.AppendError(errInvalidInt())
		return
	}
	obj, exists := Get(key)
	if !exists {
		c.AppendNull()
		return
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		c.AppendError(errWrongType())
		return
	}
	zs := obj.Value.(*Zset)
	members := zs.Range(int(start), int(stop))
	c.AppendStrArray(members)
}
