package server

import (
	"log"
	"strconv"
)

type ReplState int

const (
	REPL_STATE_NONE ReplState = iota
	REPL_STATE_CONNECT
	REPL_STATE_CONNECTING
	REPL_STATE_RECEIVING_PING
	REPL_STATE_RECEIVING_REPLCONF
	REPL_STATE_RECEIVING_PSYNC
	REPL_STATE_ONLINE
)

var (
	ReplicaState ReplState = REPL_STATE_NONE
	MasterHost   string
	MasterPort   int
	MasterClient *Client
	MasterReplID string = "?"
	CachedOffset int64  = -1
)

func HandleReplicaOfCommand(c *Client, args []string) {
	if len(args) < 2 {
		c.ReplyBuf = append(c.ReplyBuf, []byte("-ERR syntax error\r\n")...)
		return
	}

	port, err := strconv.Atoi(args[1])
	if err != nil {
		c.ReplyBuf = append(c.ReplyBuf, []byte("-ERR Invalid master port\r\n")...)
		return
	}

	MasterHost = args[0]
	MasterPort = port
	ReplicaState = REPL_STATE_CONNECT

	log.Printf("Scheduling replication shift to trail Master -> %s:%d\n", MasterHost, MasterPort)
	c.ReplyBuf = append(c.ReplyBuf, []byte("+OK\r\n")...)
}
