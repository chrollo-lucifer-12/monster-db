package core

import (
	"context"
	"strconv"

	"github.com/redis-server/internal/runtime"
)

type BlpopCmd struct{}

func (BlpopCmd) Name() string { return "BLPOP" }

func (BlpopCmd) Execute(ctx context.Context, c runtime.ClientCommander, args []string) {
	if len(args) != 2 {
		c.AppendError(errWrongArgs("blpop"))
		return
	}
	key := args[0]
	timeout, err := strconv.Atoi(args[1])
	if err != nil {
		c.AppendError(errInvalidInt())
		return
	}
	if IsEmpty(key) {
		c.BlockOn(key, timeout)
		return
	}
	LPOPCmd{}.Execute(ctx, c, []string{key})
}
