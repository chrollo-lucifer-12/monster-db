package server

import (
	"log"
	"net"
	"strconv"

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

}
