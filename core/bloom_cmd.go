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
		c.AppendError(errWrongArgs("bf.exists"))
		return
	}
	obj, exists := Get(args[0])
	if !exists {
		c.AppendNull()
		return
	}
	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_BLOOM_FILTERS)); err != nil {
		c.AppendError(errWrongType())
		return
	}
	if obj.Value.(*BloomFilter).Exists(args[1]) {
		c.AppendIntReply(1)
		return
	}
	c.AppendIntReply(0)
}

func (BFADDCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 2 {
		c.AppendError(errWrongArgs("bf.add"))
		return
	}
	obj, exists := Get(args[0])
	if !exists {
		c.AppendNull()
		return
	}
	obj.Value.(*BloomFilter).Add(args[1])
	c.AppendIntReply(1)
}

func (BFRESERVECmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) < 3 {
		c.AppendError(errWrongArgs("bf.reserve"))
		return
	}
	errorRate, _ := strconv.ParseFloat(args[1], 32)
	capacity, _ := strconv.Atoi(args[2])
	bl := NewBloomFilter(capacity, errorRate)
	Put(args[0], NewObj(bl, uint8(OBJ_TYPE_BLOOM_FILTERS), OBJ_ENCODING_BOOL_ARR), -1)
	c.AppendSimpleString("OK")
}
