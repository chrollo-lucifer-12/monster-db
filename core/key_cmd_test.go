package core

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/redis-server/config"
	"github.com/redis-server/resp"
)

type benchClient struct {
	buf []byte
}

func (c *benchClient) AppendBytesReply(val []byte) {
	c.buf = resp.EncodeStringBytes(c.buf, val)

}

func (c *benchClient) AppendArrayLen(len int) {
	c.buf = resp.EncodeArrayLen(c.buf, len)
}

func (c *benchClient) AppendIntArray(v []int64) {
	c.buf = resp.EncodeArrayLen(c.buf, len(v))

	for _, item := range v {
		c.buf = resp.EncodeInt(c.buf, item)
	}
}

func (c *benchClient) AppendSimpleString(val string) {
	c.buf = append(c.buf, '+')
	c.buf = append(c.buf, val...)
	c.buf = append(c.buf, '\r', '\n')
}

func (c *benchClient) AppendBulkString(val string) {
	c.buf = append(c.buf, '$')
	c.buf = strconv.AppendInt(c.buf, int64(len(val)), 10)
	c.buf = append(c.buf, '\r', '\n')
	c.buf = append(c.buf, val...)
	c.buf = append(c.buf, '\r', '\n')
}

func (c *benchClient) AppendStrArray(v []string) {
	c.buf = resp.EncodeArrayLen(c.buf, len(v))

	for _, item := range v {
		c.buf = resp.EncodeString(c.buf, item)
	}
}

func (c *benchClient) AppendStringArrayArray(v [][]string) {
	c.buf = resp.EncodeArrayLen(c.buf, len(v))

	for _, arr := range v {
		if arr == nil {
			c.buf = append(c.buf, '$', '-', '1', '\r', '\n')
			continue
		}

		c.AppendStrArray(arr)
	}
}

func (c *benchClient) AppendFloat(val float64) {
	c.buf = strconv.AppendFloat(c.buf, val, 'g', -1, 64)
}

func (c *benchClient) AppendNull() {
	c.buf = append(c.buf, "$-1\r\n"...)
}

func (c *benchClient) AppendError(err string) {
	c.buf = append(c.buf, '-')
	c.buf = append(c.buf, err...)
	c.buf = append(c.buf, '\r', '\n')
}

func (c *benchClient) AppendIntReply(val int64) {
	c.buf = resp.EncodeInt(c.buf, val)
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

func BenchmarkSetCmdExecute(b *testing.B) {
	store = make(map[string]Obj, b.N)
	config.KeyLimit = math.MaxInt

	keys := make([]string, b.N)

	for i := 0; i < b.N; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
	}

	cmd := SetCmd{}
	ctx := context.Background()
	client := &benchClient{
		buf: make([]byte, 0, 4096),
	}
	args := []string{"", "1"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]
		args[0] = keys[i]

		cmd.Execute(ctx, client, args)
	}
}

func BenchmarkGetCmdExecuteHot(b *testing.B) {
	store = make(map[string]Obj)
	Put("key-0", NewObj("myvalue", OBJ_TYPE_STRING, OBJ_ENCODING_EMBSTR), -1)

	cmd := GetCmd{}
	ctx := context.Background()
	benchClient := &benchClient{
		buf: make([]byte, 0, 4096),
	}
	args := []string{"key-0"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cmd.Execute(ctx, benchClient, args)
		benchClient.buf = benchClient.buf[:0]
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
	benchClient := &benchClient{buf: make([]byte, 0, 4096)}
	args := []string{"key-0"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchClient.buf = benchClient.buf[:0]
		args[0] = fmt.Sprintf("key-%d", i)
		cmd.Execute(ctx, benchClient, args)
	}
}
