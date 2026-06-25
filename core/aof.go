package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/redis-server/config"
	"github.com/redis-server/resp"
)

type nullClient struct{}

func (nullClient) SetFlag(flag uint8)      {}
func (nullClient) ClearFlag(flag uint8)    {}
func (nullClient) HasFlag(flag uint8) bool { return false }

func (nullClient) AppendReply(value any, isSimple bool) {}

func (nullClient) ResetMultiState()           {}
func (nullClient) QueueCommand(cmd *RedisCmd) {}
func (nullClient) MultiCommands() RedisCmds   { return nil }
func (nullClient) AbortMulti()                {}
func (nullClient) IsMultiAborted() bool       { return false }

func (nullClient) Key() []byte     { return nil }
func (nullClient) SetKey(k []byte) {}

func (nullClient) Subscribe(key string) (int, bool)   { panic("core: SUBSCRIBE during AOF replay") }
func (nullClient) Unsubscribe(key string) (int, bool) { panic("core: UNSUBSCRIBE during AOF replay") }
func (nullClient) SubscribedKeys() []string           { return nil }
func (nullClient) Publish(key, message string) int    { panic("core: PUBLISH during AOF replay") }

func (nullClient) WatchKeys(keys []string) {}
func (nullClient) UnwatchAllKeys()         {}

func (nullClient) AppendBytesReply(val []byte) {}
func (nullClient) AppendIntReply(val int64)    {}

func (nullClient) BlockOn(key string, timeoutMs int) {
	panic("core: BLPOP blocked during AOF replay")
}

var NullClient ClientCommander = nullClient{}

func writeCommand(fp *os.File, tokens []string) {
	fp.Write(resp.Encode(nil, tokens, false))
}

func dumpkey(fp *os.File, k string, obj Obj) {
	writeCommand(fp, []string{
		"SET",
		k,
		obj.Value.(string),
	})

	exp, isExpirySet := getExpiry(k)

	if isExpirySet {
		writeCommand(fp, []string{
			"EXPIRE",
			k,
			strconv.FormatInt(int64(exp), 10),
		})
	}
}

func DumpAllAOF() {
	tempFile := config.AOFFILE + ".tmp"
	fp, _ := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

	for k, obj := range store {
		dumpkey(fp, k, obj)
	}

	fp.Sync()
	fp.Close()

	os.Rename(tempFile, config.AOFFILE)
}

func Restoreaof() {
	fp, err := os.OpenFile(config.AOFFILE, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	defer fp.Close()

	data, err := io.ReadAll(fp)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	decoded, _, err := resp.Decode(data)
	if err != nil {
		fmt.Println("decode error:", err)
		return
	}

	ctx := WithClient(context.Background(), NullClient)

	for _, raw := range decoded {
		arr, ok := raw.([]interface{})
		if !ok || len(arr) == 0 {
			continue
		}

		cmdName, ok := arr[0].(string)
		if !ok {
			continue
		}

		args := make([]string, 0, len(arr)-1)

		for _, v := range arr[1:] {
			if s, ok := v.(string); ok {
				args = append(args, s)
			}
		}

		cmdImpl, _ := Lookup(cmdName)

		cmdImpl.Execute(ctx, NullClient, args)
	}

	fmt.Println("AOF replay done")
}
