package core

import (
	"context"

	"github.com/redis-server/resp"
)

type SubCmd struct{}

func (SubCmd) Name() string { return "SUBSCRIBE" }

func (SubCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	var out []byte
	for _, key := range args {
		count, added := c.Subscribe(key)
		if !added {
			continue
		}
		out = append(out, resp.Encode([]any{"subscribe", key, count}, false)...)
	}
	return out
}

type UnsubCmd struct{}

func (UnsubCmd) Name() string { return "UNSUBSCRIBE" }

func (UnsubCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	targets := args
	if len(targets) == 0 {
		targets = c.SubscribedKeys()
	}

	var out []byte
	for _, key := range targets {
		count, removed := c.Unsubscribe(key)
		if !removed {
			continue
		}
		out = append(out, resp.Encode([]any{"unsubscribe", key, count}, false)...)
	}
	return out
}

type PublishCmd struct{}

func (PublishCmd) Name() string { return "PUBLISH" }

func (PublishCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 2 {
		return errWrongArgs("publish")
	}
	key, message := args[0], args[1]
	delivered := c.Publish(key, message)
	return resp.Encode(delivered, false)
}
