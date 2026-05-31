package server

import (
	"log"
	"time"

	"github.com/redis-server/alloc"
	"golang.org/x/sys/unix"
)

type Client struct {
	Fd       int
	QueryBuf []byte
	ReplyBuf []byte
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

type EventLoop struct {
	EpollFD         int
	Events          map[int]*FileEvent
	Fired           []unix.EpollEvent
	TimeEvents      []*TimeEvent
	NextTimeEventID int64
	Stop            bool
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

func CreateEventLoop(maxClients int) (*EventLoop, error) {
	epollFD, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &EventLoop{EpollFD: epollFD,
		Events: make(map[int]*FileEvent),
		Fired:  make([]unix.EpollEvent, maxClients),
		Stop:   false}, nil
}
