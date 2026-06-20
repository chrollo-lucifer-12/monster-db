package core

import (
	"context"

	"github.com/redis-server/resp"
)

const (
	MULTI_MODE     uint8 = 1 << 0
	CLIENT_BLOCKED uint8 = 1 << 1
	CLIENT_SUB     uint8 = 1 << 2
	CLIENT_CAS     uint8 = 1 << 3
)

type MultiCmd struct{}

func (MultiCmd) Name() string { return "MULTI" }

func (MultiCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	c.SetFlag(MULTI_MODE)
	c.ResetMultiState()
	return []byte("+OK\r\n")
}

type ExecCmd struct{}

func (ExecCmd) Name() string { return "EXEC" }

func (ExecCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if !c.HasFlag(MULTI_MODE) {
		return []byte("-ERR EXEC without MULTI\r\n")
	}

	if c.HasFlag(CLIENT_CAS) || c.IsMultiAborted() {
		c.UnwatchAllKeys()
		c.ResetMultiState()
		c.ClearFlag(MULTI_MODE)
		c.ClearFlag(CLIENT_CAS)
		return RESP_NIL
	}

	queued := c.MultiCommands()
	if len(queued) == 0 {
		c.ClearFlag(MULTI_MODE)
		return []byte("*0\r\n")
	}

	results := make([][]byte, 0, len(queued))
	for _, qcmd := range queued {
		cmd, ok := Lookup(qcmd.Cmd)
		if !ok {
			results = append(results, []byte("-ERR unknown command '"+qcmd.Cmd+"'\r\n"))
			continue
		}
		results = append(results, cmd.Execute(ctx, c, qcmd.Args))
	}

	c.ResetMultiState()
	c.ClearFlag(MULTI_MODE)
	c.UnwatchAllKeys()

	return resp.EncodeExecArray(results)
}

type DiscardCmd struct{}

func (DiscardCmd) Name() string { return "DISCARD" }

func (DiscardCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if !c.HasFlag(MULTI_MODE) {
		return []byte("-ERR DISCARD without MULTI\r\n")
	}
	c.ResetMultiState()
	c.ClearFlag(MULTI_MODE)
	c.UnwatchAllKeys()
	return []byte("+OK\r\n")
}

type WatchCmd struct{}

func (WatchCmd) Name() string { return "WATCH" }

func (WatchCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if c.HasFlag(MULTI_MODE) {
		return resp.Encode("WATCH inside MULTI not allowed", false)
	}
	c.WatchKeys(args)
	return RESP_OK
}
