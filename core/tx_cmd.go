package core

import (
	"context"
)

const (
	MULTI_MODE     uint8 = 1 << 0
	CLIENT_BLOCKED uint8 = 1 << 1
	CLIENT_SUB     uint8 = 1 << 2
	CLIENT_CAS     uint8 = 1 << 3
)

var (
	errExecWithoutMulti    = "-ERR EXEC without MULTI\r\n"
	errDiscardWithoutMulti = "-ERR DISCARD without MULTI\r\n"
	errWatchInsideMulti    = "ERR WATCH inside MULTI not allowed"
	respEmptyArray         = []byte("*0\r\n")
	respOKSimple           = []byte("+OK\r\n")
)

type MultiCmd struct{}

func (MultiCmd) Name() string { return "MULTI" }

func (MultiCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	c.SetFlag(MULTI_MODE)
	c.ResetMultiState()
	c.AppendReply(respOKSimple, true)
}

type ExecCmd struct{}

func (ExecCmd) Name() string { return "EXEC" }

func (ExecCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if !c.HasFlag(MULTI_MODE) {
		c.AppendReply(errExecWithoutMulti, false)
		return
	}

	if c.HasFlag(CLIENT_CAS) || c.IsMultiAborted() {
		c.UnwatchAllKeys()
		c.ResetMultiState()
		c.ClearFlag(MULTI_MODE)
		c.ClearFlag(CLIENT_CAS)
		c.AppendReply(nil, false)
		return
	}

	queued := c.MultiCommands()
	if len(queued) == 0 {
		c.ClearFlag(MULTI_MODE)
		c.AppendReply(respEmptyArray, true)
		return
	}
	c.AppendReply(len(queued), true)

	for _, qcmd := range queued {
		cmd, ok := Lookup(qcmd.Cmd)
		if !ok {
			c.AppendReply("-ERR unknown command '"+qcmd.Cmd+"'", false)
			continue
		}
		cmd.Execute(ctx, c, qcmd.Args)
	}

	c.ResetMultiState()
	c.ClearFlag(MULTI_MODE)
	c.UnwatchAllKeys()

}

type DiscardCmd struct{}

func (DiscardCmd) Name() string { return "DISCARD" }

func (DiscardCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if !c.HasFlag(MULTI_MODE) {
		c.AppendReply(errDiscardWithoutMulti, false)
		return
	}
	c.ResetMultiState()
	c.ClearFlag(MULTI_MODE)
	c.UnwatchAllKeys()
	c.AppendReply(respOKSimple, true)
}

type WatchCmd struct{}

func (WatchCmd) Name() string { return "WATCH" }

func (WatchCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if c.HasFlag(MULTI_MODE) {
		c.AppendReply(errWatchInsideMulti, false)
		return
	}
	c.WatchKeys(args)
	c.AppendReply(RESP_OK, true)
}
