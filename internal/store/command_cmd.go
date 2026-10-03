package core

import (
	"context"

	"github.com/redis-server/internal/runtime"
)

type CommandCmd struct{}

func (CommandCmd) Name() string { return "COMMAND" }

func (CommandCmd) Execute(ctx context.Context, c runtime.ClientCommander, args []string) {

	if len(args) > 1 {
		c.AppendError(errWrongArgs("command"))
		return
	}

	if len(args) == 1 {
		cmd, ok := registry[args[0]]
		if !ok {
			c.AppendError(errWrongArgs("command"))
			return
		}

		c.AppendBulkString(cmd.Name())
		return
	}

	for k, _ := range registry {
		c.AppendBulkString(k)
	}
}
