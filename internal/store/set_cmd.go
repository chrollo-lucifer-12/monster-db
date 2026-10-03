package core

import (
	"context"
	"strconv"

	"github.com/redis-server/internal/ds"
	"github.com/redis-server/internal/runtime"
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

func (SaddCmd) Execute(ctx context.Context, c runtime.ClientCommander, args []string) {
	if len(args) < 2 {
		c.AppendError(errWrongArgsSadd)
		return
	}
	key := args[0]
	obj, exists := Get(key)

	var is *ds.Intset
	if !exists {
		is = ds.NewIntset()
		Put(key, NewObj(is, uint8(OBJ_TYPE_SET), OBJ_ENCODING_INSET), -1)
	} else {
		is = obj.Value.(*ds.Intset)
	}

	count := 0
	for _, v := range args[1:] {
		n, err := strconv.ParseInt(v, 10, 16)
		if err == nil {
			if is.Set(int16(n)) {
				count++
			}
		}
	}
	c.AppendIntReply(int64(count))
}

type ScardCmd struct{}

func (ScardCmd) Name() string { return "SCARD" }

func (ScardCmd) Execute(ctx context.Context, c runtime.ClientCommander, args []string) {
	if len(args) != 1 {
		c.AppendError(errWrongArgsScard)
		return
	}
	key := args[0]
	obj, exists := Get(key)
	if !exists {
		c.AppendNull()
		return
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		c.AppendNull()
	}
	is := obj.Value.(*ds.Intset)
	c.AppendIntReply(int64(is.Size()))
}

type SismemberCmd struct{}

func (SismemberCmd) Name() string { return "SISMEMBER" }

func (SismemberCmd) Execute(ctx context.Context, c runtime.ClientCommander, args []string) {
	if len(args) != 2 {
		c.AppendError(errWrongArgsSismember)
	}
	key := args[0]
	obj, exsist := Get(key)
	if !exsist {
		c.AppendNull()
		return
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		c.AppendNull()
		return
	}
	value, err := strconv.ParseInt(args[1], 10, 16)
	if err != nil {
		c.AppendBytesReply(RESP_ZERO)
		return
	}
	is := obj.Value.(*ds.Intset)
	idx := is.Search(int16(value))
	if idx != -1 && is.Get(idx) == int16(value) {
		c.AppendBytesReply(RESP_ONE)
		return
	}
	c.AppendBytesReply(RESP_ZERO)
}

type SmembersCmd struct{}

func (SmembersCmd) Name() string { return "SMEMBERS" }

func (SmembersCmd) Execute(ctx context.Context, c runtime.ClientCommander, args []string) {
	if len(args) != 1 {
		c.AppendError(errWrongArgsSmembers)
		return
	}
	key := args[0]
	obj, exists := Get(key)
	if !exists {
		c.AppendNull()
		return
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		c.AppendNull()
		return
	}
	is := obj.Value.(*ds.Intset)
	elements := make([]int64, 0, is.Size())
	for i := 0; i < int(is.Size()); i++ {
		elements = append(elements, int64(is.Get(i)))
	}

	c.AppendIntArray(elements)
}

type SremCmd struct{}

func (SremCmd) Name() string { return "SREM" }

func (SremCmd) Execute(ctx context.Context, c runtime.ClientCommander, args []string) {
	if len(args) != 2 {
		c.AppendError(errWrongArgsSrem)
		return
	}
	key := args[0]
	obj, exists := Get(key)
	if !exists {
		c.AppendNull()
		return
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		c.AppendNull()
		return
	}
	value, err := strconv.ParseInt(args[1], 10, 16)
	if err != nil {
		c.AppendError(errNotInteger)
		return
	}
	is := obj.Value.(*ds.Intset)
	c.AppendIntReply(int64(is.Del(int16(value))))
}
