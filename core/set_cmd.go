package core

import (
	"context"
	"strconv"
)

var (
	errWrongArgsSadd      = "ERR wrong number of arguments for 'sadd' command"
	errWrongArgsScard     = "ERR wrong number of arguments for 'scard' command"
	errWrongArgsSismember = "ERR wrong number of arguments for 'sismember' command"
	errWrongArgsSmembers  = "ERR wrong number of arguments for 'smembers' command"
	errWrongArgsSrem      = "ERR wrong number of arguments for 'srem' command"
	errNotInteger         = "ERR value is not an integer"
)

type SaddCmd struct{}

func (SaddCmd) Name() string { return "SADD" }

func (SaddCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) < 2 {
		c.AppendReply(errWrongArgsSadd, false)
		return
	}
	key := args[0]
	obj, exists := Get(key)

	var is *Intset
	if !exists {
		is = NewIntset()
		Put(key, NewObj(is, uint8(OBJ_TYPE_SET), OBJ_ENCODING_INSET), -1)
	} else {
		is = obj.Value.(*Intset)
	}

	count := 0
	for _, v := range args[1:] {
		n, err := strconv.ParseInt(v, 10, 16)
		if err == nil {
			if is.set(int16(n)) {
				count++
			}
		}
	}
	c.AppendReply(count, false)
}

type ScardCmd struct{}

func (ScardCmd) Name() string { return "SCARD" }

func (ScardCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 1 {
		c.AppendReply(errWrongArgsScard, false)
		return
	}
	key := args[0]
	obj, exists := Get(key)
	if !exists {
		c.AppendReply(nil, false)
		return
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		c.AppendReply(nil, false)
	}
	is := obj.Value.(*Intset)
	c.AppendReply(is.length, false)
}

type SismemberCmd struct{}

func (SismemberCmd) Name() string { return "SISMEMBER" }

func (SismemberCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 2 {
		c.AppendReply(errWrongArgsSismember, false)
	}
	key := args[0]
	obj, exsist := Get(key)
	if !exsist {
		c.AppendReply(nil, false)
		return
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		c.AppendReply(nil, false)
		return
	}
	value, err := strconv.ParseInt(args[1], 10, 16)
	if err != nil {
		c.AppendReply(RESP_ZERO, true)
		return
	}
	is := obj.Value.(*Intset)
	idx := is.search(int16(value))
	if idx != -1 && is.get(idx) == int16(value) {
		c.AppendReply(RESP_ONE, true)
		return
	}
	c.AppendReply(RESP_ZERO, true)
}

type SmembersCmd struct{}

func (SmembersCmd) Name() string { return "SMEMBERS" }

func (SmembersCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 1 {
		c.AppendReply(errWrongArgsSmembers, false)
		return
	}
	key := args[0]
	obj, exists := Get(key)
	if !exists {
		c.AppendReply(nil, false)
		return
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		c.AppendReply(nil, false)
		return
	}
	is := obj.Value.(*Intset)
	elements := make([]any, 0, is.length)
	for i := 0; i < int(is.length); i++ {
		elements = append(elements, is.get(i))
	}
	c.AppendReply(elements, false)
}

type SremCmd struct{}

func (SremCmd) Name() string { return "SREM" }

func (SremCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 2 {
		c.AppendReply(errWrongArgsSrem, false)
		return
	}
	key := args[0]
	obj, exists := Get(key)
	if !exists {
		c.AppendReply(nil, false)
		return
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		c.AppendReply(nil, false)
		return
	}
	value, err := strconv.ParseInt(args[1], 10, 16)
	if err != nil {
		c.AppendReply(errNotInteger, false)
		return
	}
	is := obj.Value.(*Intset)
	c.AppendReply(is.del(int16(value)), false)
}
