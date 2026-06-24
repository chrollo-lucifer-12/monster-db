package core

import (
	"context"
	"errors"
	"strconv"

	"github.com/redis-server/resp"
)

type SaddCmd struct{}

func (SaddCmd) Name() string { return "SADD" }

func (SaddCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) < 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'sadd' command"), false)
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
	return resp.Encode(count, false)
}

type ScardCmd struct{}

func (ScardCmd) Name() string { return "SCARD" }

func (ScardCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'scard' command"), false)
	}
	key := args[0]
	obj, exists := Get(key)
	if !exists {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		return RESP_NIL
	}
	is := obj.Value.(*Intset)
	return resp.Encode(is.length, false)
}

type SismemberCmd struct{}

func (SismemberCmd) Name() string { return "SISMEMBER" }

func (SismemberCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'sismember' command"), false)
	}
	key := args[0]
	obj, exsist := Get(key)
	if !exsist {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		return RESP_NIL
	}
	value, err := strconv.ParseInt(args[1], 10, 16)
	if err != nil {
		return RESP_ZERO
	}
	is := obj.Value.(*Intset)
	idx := is.search(int16(value))
	if idx != -1 && is.get(idx) == int16(value) {
		return RESP_ONE
	}
	return RESP_ZERO
}

type SmembersCmd struct{}

func (SmembersCmd) Name() string { return "SMEMBERS" }

func (SmembersCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'smembers' command"), false)
	}
	key := args[0]
	obj, exists := Get(key)
	if !exists {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		return RESP_NIL
	}
	is := obj.Value.(*Intset)
	elements := make([]any, 0, is.length)
	for i := 0; i < int(is.length); i++ {
		elements = append(elements, is.get(i))
	}
	return resp.Encode(elements, false)
}

type SremCmd struct{}

func (SremCmd) Name() string { return "SREM" }

func (SremCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'srem' command"), false)
	}
	key := args[0]
	obj, exists := Get(key)
	if !exists {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		return RESP_NIL
	}
	value, err := strconv.ParseInt(args[1], 10, 16)
	if err != nil {
		return resp.Encode(errors.New("ERR value is not an integer"), false)
	}
	is := obj.Value.(*Intset)
	return resp.Encode(is.del(int16(value)), false)
}
