package server

import (
	"log"
	"time"

	"github.com/redis-server/alloc"
	"github.com/redis-server/core"
	"golang.org/x/sys/unix"
)

const (
	MULTI_MODE     uint8 = 1 << 0
	CLIENT_BLOCKED uint8 = 1 << 1
	CLIENT_SUB     uint8 = 1 << 2
	CLIENT_CAS     uint8 = 1 << 3
)

type Multistate struct {
	cmds    core.RedisCmds
	aborted bool
}

type Client struct {
	Fd         int
	QueryBuf   []byte
	ReplyBuf   []byte
	flag       uint8
	multistate Multistate
	key        []byte
	when       time.Time

	subscriptions map[string]struct{}
}

func freeClient(loop *EventLoop, client *Client) {
	loop.DeleteFileEvent(client.Fd, unix.EPOLLIN|unix.EPOLLOUT)
	unix.Close(client.Fd)
	con_clients--

	alloc.Free(client.QueryBuf)
	alloc.Free(client.ReplyBuf)
}

func (c *Client) BlockClient(key string) {
	c.flag |= CLIENT_BLOCKED
	waitingKeys[key] = append(waitingKeys[key], c)
}

func isSlowClient(client *Client) bool {
	bufSize := len(client.ReplyBuf)

	if bufSize >= PubSubHardLimit {
		return true
	}

	return false
}

func cleanupClient(client *Client, loop *EventLoop) {
	freeClient(loop, client)

	delete(clientsPendingWrite, client.Fd)

	for key := range client.subscriptions {
		subscribers[key] = removeClient(subscribers[key], client)
		if len(subscribers[key]) == 0 {
			delete(subscribers, key)
		}
	}

	client.ReplyBuf = nil
	client.subscriptions = nil
}

func NewClient(fd int) *Client {

	qBuf, err := alloc.Alloc(4096)
	if err != nil {
		log.Println("OOM")
		return nil
	}

	rBuf, err := alloc.Alloc(4096)
	if err != nil {
		log.Println("OOM")
		alloc.Free(qBuf)
		return nil
	}

	return &Client{
		Fd:            fd,
		QueryBuf:      qBuf[:0],
		ReplyBuf:      rBuf[:0],
		subscriptions: make(map[string]struct{}),
	}
}
