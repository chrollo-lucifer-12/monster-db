package server

import (
	"log"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

const (
	PubSubHardLimit = 32 * 1024 * 1024
)

type EventLoop struct {
	EpollFD         int
	Events          map[int]*FileEvent
	Fired           []unix.EpollEvent
	TimeEvents      []*TimeEvent
	NextTimeEventID int64
}

func CreateEventLoop(maxClients int) (*EventLoop, error) {
	epollFD, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &EventLoop{EpollFD: epollFD,
		Events: make(map[int]*FileEvent),
		Fired:  make([]unix.EpollEvent, maxClients),
	}, nil
}

func beforeSleep(loop *EventLoop) {

	if ReplicaState == REPL_STATE_CONNECT {
		InitMasterConnection(loop)
	}

	processKeys(loop)

	for fd, client := range clientsPendingWrite {

		if isSlowClient(client) {
			cleanupClient(client, loop)
			continue
		}

		n, err := unix.Write(fd, client.ReplyBuf)

		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				loop.AddFileEvent(fd, unix.EPOLLOUT, SendReplyToClient, client)
				continue
			}

			freeClient(loop, client)
			delete(clientsPendingWrite, fd)
			continue
		}

		client.ReplyBuf = client.ReplyBuf[n:]

		if len(client.ReplyBuf) > 0 {
			loop.AddFileEvent(fd, unix.EPOLLOUT, SendReplyToClient, client)
		} else {
			loop.DeleteFileEvent(fd, unix.EPOLLOUT)
		}

		delete(clientsPendingWrite, fd)
	}
}

func (el *EventLoop) Main() error {

	for atomic.LoadInt32(&eStatus) != EnngineStatus_SHUTTING_DOWN {

		timeoutMs := 100

		if len(el.TimeEvents) > 0 {
			now := time.Now()
			shortest := el.TimeEvents[0].When

			for _, te := range el.TimeEvents {
				if te.When.Before(shortest) {
					shortest = te.When
				}
			}

			diff := int(shortest.Sub(now).Milliseconds())
			if diff <= 0 {
				timeoutMs = 0
			} else {
				timeoutMs = diff
			}
		}

		beforeSleep(el)
		nevents, err := unix.EpollWait(el.EpollFD, el.Fired, timeoutMs)

		if err != nil {
			if err == unix.EINTR {
				continue
			}
			log.Println("EpollWait error: ", err)
			break
		}

		if !atomic.CompareAndSwapInt32(&eStatus, EngineStatus_WAITING, EngineStatus_BUSY) {
			switch eStatus {
			case EnngineStatus_SHUTTING_DOWN:
				return nil
			}
		}

		for i := 0; i < nevents; i++ {
			fd := int(el.Fired[i].Fd)
			mask := el.Fired[i].Events

			fe, exists := el.Events[fd]
			if !exists {
				continue
			}

			if mask&unix.EPOLLIN != 0 && fe.ReadProc != nil {
				fe.ReadProc(el, fd, fe.ClientData)
			}

			if mask&unix.EPOLLOUT != 0 && fe.WriteProc != nil {
				fe.WriteProc(el, fd, fe.ClientData)
			}
		}

		el.processTimeEvents()

		atomic.StoreInt32(&eStatus, EngineStatus_WAITING)
	}

	return nil
}
