package server

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

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
		client.ReplyBuf = append(client.ReplyBuf, []byte("PING\r\n")...)

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

			client.ReplyBuf = append(client.ReplyBuf, []byte("REPLCONF listening-port 6380\r\n")...)
			client.ReplyBuf = append(client.ReplyBuf, []byte("REPLCONF capa psync2\r\n")...)

			ReplicaState = REPL_STATE_RECEIVING_REPLCONF
			clientsPendingWrite[fd] = client
		}

	case REPL_STATE_RECEIVING_REPLCONF:
		if strings.Contains(string(client.QueryBuf), "+OK") {
			client.QueryBuf = client.QueryBuf[:0]

			psynCmd := fmt.Sprintf("PSYNC %s %d\r\n", MasterReplID, CachedOffset)
			client.ReplyBuf = append(client.ReplyBuf, []byte(psynCmd)...)

			ReplicaState = REPL_STATE_RECEIVING_PSYNC
			clientsPendingWrite[fd] = client
		}

	case REPL_STATE_RECEIVING_PSYNC:
		response := string(client.QueryBuf)
		if strings.HasPrefix(response, "+FULLRESYNC") {
			parts := strings.Split(strings.TrimSpace(response), " ")
			MasterReplID = parts[1]
			CachedOffset, _ = strconv.ParseInt(parts[2], 10, 64)

			log.Printf("Handshake validated! Synchronizing Master ID: %s starting at Offset: %d\n", MasterReplID, CachedOffset)
			client.QueryBuf = client.QueryBuf[:0]

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

	fmt.Printf("[Replica Log Ingest] Parsed %d bytes from Master stream\n", n)
	client.QueryBuf = client.QueryBuf[:0]
}
