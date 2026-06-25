package core

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/redis-server/config"
	"github.com/redis-server/resp"
)

type benchClient struct {
	buf []byte
}

func (c *benchClient) AppendReply(value any, isSimple bool) {
	c.buf = resp.Encode(c.buf, value, isSimple)
}
func (c *benchClient) SetFlag(flag uint8)                 {}
func (c *benchClient) ClearFlag(flag uint8)               {}
func (c *benchClient) HasFlag(flag uint8) bool            { return false }
func (c *benchClient) ResetMultiState()                   {}
func (c *benchClient) QueueCommand(cmd *RedisCmd)         {}
func (c *benchClient) MultiCommands() RedisCmds           { return nil }
func (c *benchClient) AbortMulti()                        {}
func (c *benchClient) IsMultiAborted() bool               { return false }
func (c *benchClient) Key() []byte                        { return nil }
func (c *benchClient) SetKey(k []byte)                    {}
func (c *benchClient) Subscribe(key string) (int, bool)   { return 0, false }
func (c *benchClient) Unsubscribe(key string) (int, bool) { return 0, false }
func (c *benchClient) SubscribedKeys() []string           { return nil }
func (c *benchClient) Publish(key, message string) int    { return 0 }
func (c *benchClient) WatchKeys(keys []string)            {}
func (c *benchClient) UnwatchAllKeys()                    {}
func (c *benchClient) BlockOn(key string, timeoutMs int)  {}
func (c *benchClient) AppendBytesReply(val []byte)        {}
func (c *benchClient) AppendIntReply(val int64)           {}

// try sharding

func BenchmarkSetCmdExecute(b *testing.B) {

	store = make(map[string]Obj, b.N)
	config.KeyLimit = math.MaxInt

	cmd := SetCmd{}
	ctx := context.Background()
	client := &benchClient{buf: make([]byte, 0, 4096)}
	args := []string{"mykey", "1"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]
		args[0] = fmt.Sprintf("key-%d", i)
		cmd.Execute(ctx, client, args)
	}
}

func BenchmarkGetCmdExecute(b *testing.B) {

	store = make(map[string]Obj, b.N)
	config.KeyLimit = math.MaxInt

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		Put(key, NewObj("myvalue", OBJ_TYPE_STRING, OBJ_ENCODING_EMBSTR), -1)
	}

	cmd := GetCmd{}
	ctx := context.Background()
	client := &benchClient{buf: make([]byte, 0, 4096)}
	args := []string{"key-0"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args[0] = fmt.Sprintf("key-%d", i)
		cmd.Execute(ctx, client, args)
	}
}

func BenchmarkIncrCmdExecute(b *testing.B) {
	store = make(map[string]Obj, b.N)
	config.KeyLimit = math.MaxInt

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		Put(key, NewStringObj("0", OBJ_TYPE_STRING, OBJ_ENCODING_INT), -1)
	}

	cmd := IncrCmd{}
	ctx := context.Background()
	client := &benchClient{buf: make([]byte, 0, 4096)}
	args := []string{"key-0"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]
		args[0] = fmt.Sprintf("key-%d", i)
		cmd.Execute(ctx, client, args)
	}
}
