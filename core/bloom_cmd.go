package core

import (
	"context"
	"errors"
	"strconv"

	"github.com/redis-server/resp"
)

type BFRESERVECmd struct{}
type BFADDCmd struct{}
type BFEXISTSCmd struct{}

func (BFRESERVECmd) Name() string { return "BF.RESERVE" }
func (BFADDCmd) Name() string     { return "BF.ADD" }
func (BFEXISTSCmd) Name() string  { return "BF.EXISTS" }

func (BFEXISTSCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'bfadd' command"), false)
	}

	key := args[0]
	pat := args[1]

	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_BLOOM_FILTERS)); err != nil {
		return resp.Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	bl := obj.Value.(*BloomFilter)

	if bl.Exists(pat) {
		return RESP_ONE
	}

	return RESP_ZERO
}

func (BFADDCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'bfadd' command"), false)
	}

	key := args[0]
	pat := args[1]

	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	bl := obj.Value.(*BloomFilter)

	bl.Add(pat)

	return RESP_ONE
}

func (BFRESERVECmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) < 3 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'bfreserve' command"), false)
	}

	key := args[0]
	errorRate, _ := strconv.ParseFloat(args[1], 32)
	capacity, _ := strconv.Atoi(args[2])

	bl := NewBloomFilter(capacity, errorRate)

	obj := NewObj(bl, -1, uint8(OBJ_TYPE_BLOOM_FILTERS), OBJ_ENCODING_BOOL_ARR)

	Put(key, obj)

	return RESP_OK
}
