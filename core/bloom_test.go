package core

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/redis-server/config"
)

func BenchmarkBFRESERVECmdExecute(b *testing.B) {
	store = make(map[string]Obj, b.N)
	config.KeyLimit = math.MaxInt

	cmd := BFRESERVECmd{}
	ctx := context.Background()
	client := &benchClient{buf: make([]byte, 0, 4096)}
	args := []string{"bf-key", "0.01", "1000"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]
		args[0] = fmt.Sprintf("bf-%d", i)
		cmd.Execute(ctx, client, args)
	}
}

func BenchmarkBFADDCmdExecute(b *testing.B) {
	store = make(map[string]Obj, b.N)
	config.KeyLimit = math.MaxInt

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bf-%d", i)
		bl := NewBloomFilter(1000, 0.01)
		Put(key, NewObj(bl, uint8(OBJ_TYPE_BLOOM_FILTERS), OBJ_ENCODING_BOOL_ARR), -1)
	}

	cmd := BFADDCmd{}
	ctx := context.Background()
	client := &benchClient{buf: make([]byte, 0, 4096)}
	args := []string{"bf-0", "member"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]
		args[0] = fmt.Sprintf("bf-%d", i)
		args[1] = fmt.Sprintf("member-%d", i)
		cmd.Execute(ctx, client, args)
	}
}

func BenchmarkBFEXISTSCmdExecute(b *testing.B) {
	store = make(map[string]Obj, b.N)
	config.KeyLimit = math.MaxInt

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bf-%d", i)
		bl := NewBloomFilter(1000, 0.01)
		bl.Add(fmt.Sprintf("member-%d", i))
		Put(key, NewObj(bl, uint8(OBJ_TYPE_BLOOM_FILTERS), OBJ_ENCODING_BOOL_ARR), -1)
	}

	cmd := BFEXISTSCmd{}
	ctx := context.Background()
	client := &benchClient{buf: make([]byte, 0, 4096)}
	args := []string{"bf-0", "member-0"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]
		args[0] = fmt.Sprintf("bf-%d", i)
		args[1] = fmt.Sprintf("member-%d", i)
		cmd.Execute(ctx, client, args)
	}
}
