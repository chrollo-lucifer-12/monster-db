package core

import (
	"context"
	"strconv"
)

type BFRESERVECmd struct{}
type BFADDCmd struct{}
type BFEXISTSCmd struct{}

func (BFRESERVECmd) Name() string { return "BF.RESERVE" }
func (BFADDCmd) Name() string     { return "BF.ADD" }
func (BFEXISTSCmd) Name() string  { return "BF.EXISTS" }

func (BFEXISTSCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 2 {
		c.AppendReply(errWrongArgs("bf.exists"), false)
		return
	}
	obj, exists := Get(args[0])
	if !exists {
		c.AppendReply(nil, false)
		return
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_BLOOM_FILTERS)); err != nil {
		c.AppendReply(errWrongType(), false)
		return
	}
	if obj.Value.(*BloomFilter).Exists(args[1]) {
		c.AppendReply(1, false)
		return
	}
	c.AppendReply(0, false)
}

func (BFADDCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 2 {
		c.AppendReply(errWrongArgs("bf.add"), false)
		return
	}
	obj, exists := Get(args[0])
	if !exists {
		c.AppendReply(nil, false)
		return
	}
	obj.Value.(*BloomFilter).Add(args[1])
	c.AppendReply(1, false)
}

func (BFRESERVECmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) < 3 {
		c.AppendReply(errWrongArgs("bf.reserve"), false)
		return
	}
	errorRate, _ := strconv.ParseFloat(args[1], 32)
	capacity, _ := strconv.Atoi(args[2])
	bl := NewBloomFilter(capacity, errorRate)
	Put(args[0], NewObj(bl, uint8(OBJ_TYPE_BLOOM_FILTERS), OBJ_ENCODING_BOOL_ARR), -1)
	c.AppendReply("OK", true)
}
