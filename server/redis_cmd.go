package server

import (
	"context"

	"github.com/redis-server/core"
)

var errSubscribeOnly = []byte("-ERR only (P)SUBSCRIBE / (P)UNSUBSCRIBE / PING / QUIT allowed in this context\r\n")
var queued = []byte("+QUEUED\r\n")
var errUnknownCmd = []byte("-ERR unknown command\r\n")

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
				client.ReplyBuf = append(client.ReplyBuf, []byte("-ERR unknown command '"+cmd.Cmd+"'\r\n")...)
				continue
			}
			client.QueueCommand(cmd)
			client.ReplyBuf = append(client.ReplyBuf, queued...)
			continue
		}

		if client.HasFlag(CLIENT_BLOCKED) {
			return
		}

		if client.HasFlag(CLIENT_SUB) &&
			cmd.Cmd != "SUBSCRIBE" && cmd.Cmd != "UNSUBSCRIBE" &&
			cmd.Cmd != "PSUBSCRIBE" && cmd.Cmd != "PUNSUBSCRIBE" &&
			cmd.Cmd != "PING" && cmd.Cmd != "QUIT" {
			client.ReplyBuf = append(client.ReplyBuf, errSubscribeOnly...)
			continue
		}

		// switch cmd.Cmd {
		// case "REPLCONF":
		// 	HandleMasterReplConfCommand(client, cmd.Args)
		// 	continue
		// case "PSYNC":
		// 	HandleMasterPsyncCommand(loop, client, cmd.Args)
		// 	continue
		// case "REPLICAOF":
		// 	HandleReplicaOfCommand(client, cmd.Args)
		// 	continue
		// }

		ctx := core.WithClient(context.Background(), client)

		cmdImpl, ok := core.Lookup(cmd.Cmd)
		if !ok {
			client.ReplyBuf = append(client.ReplyBuf, errUnknownCmd...)
			continue
		}

		// if isReplicatedCmd(cmd.Cmd) {
		// 	args := append([]string{cmd.Cmd}, cmd.Args...)
		// 	//	HandleInformReplicas(resp.Encode(args, false))
		// }

		cmdImpl.Execute(ctx, client, cmd.Args)

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
