package core

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/redis-server/config"
)

func BenchmarkLPUSHCmdExecute(b *testing.B) {
	store = make(map[string]Obj, b.N)
	config.KeyLimit = math.MaxInt

	cmd := LPUSHCmd{}
	ctx := context.Background()
	client := &benchClient{buf: make([]byte, 0, 4096)}
	args := []string{"mylist", "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]
		args[1] = fmt.Sprintf("val-%d", i)
		cmd.Execute(ctx, client, args)
	}
}

func BenchmarkRPUSHCmdExecute(b *testing.B) {
	store = make(map[string]Obj, b.N)
	config.KeyLimit = math.MaxInt

	cmd := RPUSHCmd{}
	ctx := context.Background()
	client := &benchClient{buf: make([]byte, 0, 4096)}
	args := []string{"mylist", "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]
		args[1] = fmt.Sprintf("val-%d", i)
		cmd.Execute(ctx, client, args)
	}
}

func BenchmarkLRANGECmdExecute(b *testing.B) {
	store = make(map[string]Obj, 1)
	config.KeyLimit = math.MaxInt

	pushCmd := RPUSHCmd{}
	ctx := context.Background()
	client := &benchClient{buf: make([]byte, 0, 4096)}
	for i := 0; i < 1000; i++ {
		pushCmd.Execute(ctx, client, []string{"mylist", fmt.Sprintf("val-%d", i)})
	}

	cmd := LRANGECmd{}
	args := []string{"mylist", "0", "99"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]
		cmd.Execute(ctx, client, args)
	}
}

func BenchmarkLPOPCmdExecute(b *testing.B) {
	store = make(map[string]Obj, b.N)
	config.KeyLimit = math.MaxInt

	pushCmd := RPUSHCmd{}
	ctx := context.Background()
	client := &benchClient{buf: make([]byte, 0, 4096)}
	for i := 0; i < b.N; i++ {
		pushCmd.Execute(ctx, client, []string{"mylist", fmt.Sprintf("val-%d", i)})
	}

	cmd := LPOPCmd{}
	args := []string{"mylist"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]
		cmd.Execute(ctx, client, args)
	}
}

func BenchmarkLLENCmdExecute(b *testing.B) {
	store = make(map[string]Obj, 1)
	config.KeyLimit = math.MaxInt

	// pre-populate with 1000 elements
	pushCmd := RPUSHCmd{}
	ctx := context.Background()
	client := &benchClient{buf: make([]byte, 0, 4096)}
	for i := 0; i < 1000; i++ {
		pushCmd.Execute(ctx, client, []string{"mylist", fmt.Sprintf("val-%d", i)})
	}

	cmd := LLENCmd{}
	args := []string{"mylist"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]
		cmd.Execute(ctx, client, args)
	}
}
