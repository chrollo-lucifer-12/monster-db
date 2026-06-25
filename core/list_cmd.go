package core

import (
	"context"
	"strconv"
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

func (LLENCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 1 {
		c.AppendReply(errWrongArgs("llen"), false)
	}

	var key string = args[0]
	obj, exists := Get(key)

	if !exists {
		c.AppendReply(RESP_ZERO, true)
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
		c.AppendReply(RESP_ZERO, true)
	}

	c.AppendReply(obj.Value.(*Quicklist).len, false)
}

func (LPUSHCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) < 2 {
		c.AppendReply(errWrongArgs("lpush"), false)
		return
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
			c.AppendReply(errWrongType(), false)
			return
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

	c.AppendReply(ql.len, false)
}

func (RPUSHCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) < 2 {
		c.AppendReply(errWrongArgs("rpush"), false)
		return
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
			c.AppendReply(errWrongType(), false)
			return
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
	c.AppendReply(ql.len, false)
}

func (LRANGECmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 3 {
		c.AppendReply(errWrongArgs("lrange"), false)
		return
	}

	var key string = args[0]
	start, _ := strconv.Atoi(args[1])
	stop, _ := strconv.Atoi(args[2])

	obj, exists := Get(key)

	if !exists {
		c.AppendReply(nil, false)
		return
	}

	c.AppendReply(obj.Value.(*Quicklist).GetElements(start, stop), false)
}

func (LPOPCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) < 1 {
		c.AppendReply(errWrongArgs("lpop"), false)
		return
	}

	var key string = args[0]
	count := 1

	if len(args) == 2 {
		count, _ = strconv.Atoi(args[1])
	}

	obj, exists := Get(key)

	if !exists {
		c.AppendReply(nil, false)
		return
	}
	c.AppendReply(obj.Value.(*Quicklist).RemoveElements(count), false)
}
