package core

import (
	"context"
	"strconv"
)

type BlpopCmd struct{}

func (BlpopCmd) Name() string { return "BLPOP" }

func (BlpopCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 2 {
		return errWrongArgs("blpop")
	}
	key := args[0]
	timeout, err := strconv.Atoi(args[1])
	if err != nil {
		return errInvalidInt()
	}

	if IsEmpty(key) {
		c.BlockOn(key, timeout)
		return nil
	}

	return LPOPCmd{}.Execute(ctx, c, []string{key})
}
