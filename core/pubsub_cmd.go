package core

import (
	"context"
)

type SubCmd struct{}

func (SubCmd) Name() string { return "SUBSCRIBE" }

func (SubCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	for _, key := range args {
		count, added := c.Subscribe(key)
		if !added {
			continue
		}
		c.AppendArrayLen(3)
		c.AppendBulkString("subscribe")
		c.AppendBulkString(key)
		c.AppendIntReply(int64(count))
	}
}

type UnsubCmd struct{}

func (UnsubCmd) Name() string { return "UNSUBSCRIBE" }

func (UnsubCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	targets := args
	if len(targets) == 0 {
		targets = c.SubscribedKeys()
	}

	for _, key := range targets {
		count, removed := c.Unsubscribe(key)
		if !removed {
			continue
		}
		c.AppendArrayLen(3)
		c.AppendBulkString("unsubscribe")
		c.AppendBulkString(key)
		c.AppendIntReply(int64(count))
	}
	return
}

type PublishCmd struct{}

func (PublishCmd) Name() string { return "PUBLISH" }

func (PublishCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 2 {
		c.AppendError(errWrongArgs("publish"))
		return
	}
	key, message := args[0], args[1]
	delivered := c.Publish(key, message)
	c.AppendIntReply(int64(delivered))
}
