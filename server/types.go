package server

import (
	"log"
	"time"

	"github.com/redis-server/alloc"
	"github.com/redis-server/core"
)

var MULTI_MODE uint8 = 00000001
var CLIENT_BLOCKED uint8 = 00000010

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
}

type ReplyBufferWrapper struct {
	client *Client
}

type TimeProc func(loop *EventLoop, id int64, clientData interface{}) int

type TimeEvent struct {
	ID         int64
	When       time.Time
	Proc       TimeProc
	ClientData interface{}
}

type FileProc func(loop *EventLoop, fd int, clientData interface{})

type FileEvent struct {
	Mask       uint32
	ReadProc   FileProc
	WriteProc  FileProc
	ClientData interface{}
}

func (c *Client) BlockClient(key string) {
	c.flag |= CLIENT_BLOCKED
	waitingKeys[key] = append(waitingKeys[key], c)
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
		Fd:       fd,
		QueryBuf: qBuf[:0],
		ReplyBuf: rBuf[:0],
	}
}
