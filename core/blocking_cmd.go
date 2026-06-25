package core

import (
	"context"
	"strconv"
)

type BlpopCmd struct{}

func (BlpopCmd) Name() string { return "BLPOP" }

func (BlpopCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 2 {
		c.AppendReply(errWrongArgs("blpop"), false)
		return
	}
	key := args[0]
	timeout, err := strconv.Atoi(args[1])
	if err != nil {
		c.AppendReply(errInvalidInt(), false)
		return
	}
	if IsEmpty(key) {
		c.BlockOn(key, timeout)
		return
	}
	LPOPCmd{}.Execute(ctx, c, []string{key})
}
