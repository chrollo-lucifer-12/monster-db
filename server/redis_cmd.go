package server

import (
	"context"
	"strconv"

	"github.com/redis-server/core"
	"github.com/redis-server/resp"
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

func encodeCommand(cmd core.RedisCmd) []byte {
	buf := make([]byte, 0, 128)

	buf = resp.EncodeArrayLen(buf, 1+len(cmd.Args))

	buf = resp.EncodeString(buf, cmd.Cmd)

	for _, arg := range cmd.Args {
		buf = resp.EncodeString(buf, arg)
	}

	return buf
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

		if cmd.Cmd == "REPLICAOF" {
			port, _ := strconv.Atoi(cmd.Args[1])
			if err := AddReplica(loop, cmd.Args[0], port); err != nil {
				client.AppendError("unable to replicate master")
				clientsPendingWrite[client.Fd] = client
				continue
			}
			client.AppendSimpleString("OK")
			clientsPendingWrite[client.Fd] = client
			continue
		}

		if cmd.Cmd == "REPLICA" {
			RegisterReplica(client)
			client.AppendSimpleString("OK")
			clientsPendingWrite[client.Fd] = client
			continue
		}

		ctx := core.WithClient(context.Background(), client)

		cmdImpl, ok := core.Lookup(cmd.Cmd)
		if !ok {
			client.ReplyBuf = append(client.ReplyBuf, errUnknownCmd...)
			continue
		}

		cmdImpl.Execute(ctx, client, cmd.Args)

		if isReplicatedCmd(cmd.Cmd) && MasterFD == -1 {
			pendindCommands = append(pendindCommands, *cmd)
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
