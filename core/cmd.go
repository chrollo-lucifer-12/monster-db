package core

import "context"

type ctxKey int

const clientCtxKey ctxKey = iota

type RedisCmd struct {
	Cmd  string
	Args []string
}

type RedisCmds []*RedisCmd

type ClientCommander interface {
	SetFlag(flag uint8)
	ClearFlag(flag uint8)
	HasFlag(flag uint8) bool

	AppendReply(value any, isSimple bool)

	AppendBytesReply(val []byte)

	AppendIntReply(val int64)

	ResetMultiState()
	QueueCommand(cmd *RedisCmd)
	MultiCommands() RedisCmds
	AbortMulti()
	IsMultiAborted() bool

	Key() []byte
	SetKey(k []byte)

	Subscribe(key string) (count int, added bool)
	Unsubscribe(key string) (count int, removed bool)
	SubscribedKeys() []string
	Publish(key, message string) (delivered int)

	WatchKeys(keys []string)
	UnwatchAllKeys()

	BlockOn(key string, timeoutMs int)
}

type Command interface {
	Execute(ctx context.Context, c ClientCommander, args []string)
	Name() string
}

func WithClient(ctx context.Context, c ClientCommander) context.Context {
	return context.WithValue(ctx, clientCtxKey, c)
}

func ClientFromContext(ctx context.Context) (ClientCommander, bool) {
	c, ok := ctx.Value(clientCtxKey).(ClientCommander)
	return c, ok
}
