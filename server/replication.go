package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/redis-server/core"
	"github.com/redis-server/resp"
	"golang.org/x/sys/unix"
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
		c.ReplyBuf = append(c.ReplyBuf, resp.Encode("ERR syntax error", false)...)
		return
	}

	port, err := strconv.Atoi(args[1])
	if err != nil {
		c.ReplyBuf = append(c.ReplyBuf, resp.Encode("ERR Invalid master port", false)...)
		return
	}

	MasterHost = args[0]
	MasterPort = port
	ReplicaState = REPL_STATE_CONNECT

	log.Printf("Scheduling replication shift to trail Master -> %s:%d\n", MasterHost, MasterPort)
	c.ReplyBuf = append(c.ReplyBuf, []byte("+OK\r\n")...)
}

func InitMasterConnection(loop *EventLoop) {

	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM|unix.SOCK_NONBLOCK, 0)

	if err != nil {
		ReplicaState = REPL_STATE_CONNECT
		return
	}

	ips, err := net.LookupIP(MasterHost)
	if err != nil {
		unix.Close(fd)
		ReplicaState = REPL_STATE_CONNECT
		return
	}

	var ip4 [4]byte
	copy(ip4[:], ips[0].To4())

	sockAddr := &unix.SockaddrInet4{
		Port: MasterPort,
		Addr: ip4,
	}

	err = unix.Connect(fd, sockAddr)
	if err != nil && err != unix.EINPROGRESS {
		unix.Close(fd)
		ReplicaState = REPL_STATE_CONNECT
		return
	}

	MasterClient = NewClient(fd)

	ReplicaState = REPL_STATE_CONNECTING
	loop.AddFileEvent(fd, unix.EPOLLOUT, HandleMasterHandshake, MasterClient)

}

func HandleMasterHandshake(loop *EventLoop, fd int, clientData any) {
	client := clientData.(*Client)

	if ReplicaState == REPL_STATE_CONNECTING {
		client.ReplyBuf = append(client.ReplyBuf, resp.Encode([]string{"PING"}, false)...)

		loop.DeleteFileEvent(fd, unix.EPOLLOUT)
		loop.AddFileEvent(fd, unix.EPOLLIN, HandleMasterReadableHandshake, client)

		ReplicaState = REPL_STATE_RECEIVING_PING
		clientsPendingWrite[fd] = client
	}
}

func HandleMasterReadableHandshake(loop *EventLoop, fd int, clientData any) {
	client := clientData.(*Client)

	var buf [1024]byte
	n, err := unix.Read(fd, buf[:])

	if err != nil || n <= 0 {
		cleanupClient(client, loop)
		ReplicaState = REPL_STATE_CONNECT
		return
	}

	client.QueryBuf = append(client.QueryBuf, buf[:n]...)

	switch ReplicaState {
	case REPL_STATE_RECEIVING_PING:
		if strings.Contains(string(client.QueryBuf), "+PONG") {
			client.QueryBuf = client.QueryBuf[:0]

			client.ReplyBuf = append(client.ReplyBuf, resp.Encode([]string{"REPLCONF", "listening-port", "6380"}, false)...)
			client.ReplyBuf = append(client.ReplyBuf, resp.Encode([]string{"REPLCONF", "capa", "psync2"}, false)...)

			ReplicaState = REPL_STATE_RECEIVING_REPLCONF
			clientsPendingWrite[fd] = client
		}

	case REPL_STATE_RECEIVING_REPLCONF:
		if strings.Contains(string(client.QueryBuf), "+OK") {
			client.QueryBuf = client.QueryBuf[:0]

			client.ReplyBuf = append(
				client.ReplyBuf,
				resp.Encode([]string{
					"PSYNC",
					MasterReplID,
					strconv.FormatInt(CachedOffset, 10),
				}, false)...,
			)

			ReplicaState = REPL_STATE_RECEIVING_PSYNC
			clientsPendingWrite[fd] = client
		}

	case REPL_STATE_RECEIVING_PSYNC:
		response := string(client.QueryBuf)
		if strings.HasPrefix(response, "+FULLRESYNC") {

			lineEnd := strings.Index(response, "\r\n")
			header := response[:lineEnd]
			parts := strings.Split(header, " ")

			if len(parts) != 3 {
				return
			}

			MasterReplID = parts[1]

			offset, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return
			}

			CachedOffset = offset

			client.QueryBuf = client.QueryBuf[lineEnd+2:]

			ReplicaState = REPL_STATE_ONLINE

			loop.AddFileEvent(fd, unix.EPOLLIN, ReadLiveReplicationStream, client)
		}

	}
}

func ReadLiveReplicationStream(loop *EventLoop, fd int, clientData any) {
	client := clientData.(*Client)

	var buf [4096]byte
	n, err := unix.Read(fd, buf[:])
	if err != nil || n <= 0 {
		log.Println("Connection to master broken. Re-entering connection mode.")
		cleanupClient(client, loop)
		ReplicaState = REPL_STATE_CONNECT
		return
	}

	client.QueryBuf = append(client.QueryBuf, buf[:n]...)
	CachedOffset += int64(n)

	cmds, remaining, err := readCommands(client.QueryBuf)
	if err != nil {
		return
	}

	if remaining > 0 {
		copy(client.QueryBuf, client.QueryBuf[remaining:])
		client.QueryBuf = client.QueryBuf[:len(client.QueryBuf)-remaining]
	}

	ctx := core.WithClient(context.Background(), client)

	for _, cmd := range cmds {
		cmdImpl, _ := core.Lookup(cmd.Cmd)
		cmdImpl.Execute(ctx, client, cmd.Args)
	}

	fmt.Printf("[Replica Log Ingest] Parsed %d bytes from Master stream\n", n)
	client.QueryBuf = client.QueryBuf[:remaining]
}

func ReplicationHeartbeatCron(loop *EventLoop, id int64, data interface{}) int {

	if ReplicaState == REPL_STATE_ONLINE && MasterClient != nil {

		MasterClient.ReplyBuf = append(MasterClient.ReplyBuf, resp.Encode([]string{
			"REPLCONF",
			"ACK",
			strconv.FormatInt(CachedOffset, 10),
		}, false)...)

		clientsPendingWrite[MasterClient.Fd] = MasterClient
	}
	return 1000
}
