package server

import (
	"context"

	"github.com/redis-server/core"
	"github.com/redis-server/resp"
)

func removeClient(slice []*Client, target *Client) []*Client {
	newSlice := make([]*Client, 0, len(slice))

	for _, c := range slice {
		if c != target {
			newSlice = append(newSlice, c)
		}
	}

	return newSlice
}

func respond(cmds core.RedisCmds, client *Client, loop *EventLoop) {
	for _, cmd := range cmds {
		if client.HasFlag(core.MULTI_MODE) && cmd.Cmd != "EXEC" && cmd.Cmd != "DISCARD" {
			if _, ok := core.Lookup(cmd.Cmd); !ok {
				client.AbortMulti()
				client.AppendReply([]byte("-ERR unknown command '" + cmd.Cmd + "'\r\n"))
				continue
			}
			client.QueueCommand(cmd)
			client.AppendReply([]byte("+QUEUED\r\n"))
			continue
		}

		if client.HasFlag(CLIENT_BLOCKED) {
			return
		}

		if client.HasFlag(CLIENT_SUB) &&
			cmd.Cmd != "SUBSCRIBE" && cmd.Cmd != "UNSUBSCRIBE" &&
			cmd.Cmd != "PSUBSCRIBE" && cmd.Cmd != "PUNSUBSCRIBE" &&
			cmd.Cmd != "PING" && cmd.Cmd != "QUIT" {
			client.AppendReply([]byte("-ERR only (P)SUBSCRIBE / (P)UNSUBSCRIBE / PING / QUIT allowed in this context\r\n"))
			continue
		}

		switch cmd.Cmd {
		case "REPLCONF":
			HandleMasterReplConfCommand(client, cmd.Args)
			continue
		case "PSYNC":
			HandleMasterPsyncCommand(loop, client, cmd.Args)
			continue
		case "REPLICAOF":
			HandleReplicaOfCommand(client, cmd.Args)
			continue
		}

		ctx := core.WithClient(context.Background(), client)

		cmdImpl, ok := core.Lookup(cmd.Cmd)
		if !ok {
			client.AppendReply([]byte("-ERR unknown command\r\n"))
			continue
		}

		if isReplicatedCmd(cmd.Cmd) {
			args := append([]string{cmd.Cmd}, cmd.Args...)
			HandleInformReplicas(resp.Encode(args, false))
		}

		reply := cmdImpl.Execute(ctx, client, cmd.Args)

		if reply == nil && client.HasFlag(CLIENT_BLOCKED) {
			return
		}
		if reply != nil {
			client.AppendReply(reply)
		}
	}
}

func isReplicatedCmd(name string) bool {
	switch name {
	case "SET", "DEL", "RPUSH", "LPUSH", "INCR":
		return true
	default:
		return false
	}
}
