package server

import (
	"time"

	"github.com/redis-server/core"
	"golang.org/x/sys/unix"
)

func (el *EventLoop) AddTimeEvent(ms int64, proc TimeProc, clientData interface{}) int64 {
	id := el.NextTimeEventID
	el.NextTimeEventID++

	te := &TimeEvent{
		ID:         id,
		When:       time.Now().Add(time.Duration(ms) * time.Millisecond),
		Proc:       proc,
		ClientData: clientData,
	}

	el.TimeEvents = append(el.TimeEvents, te)
	return id
}

func processKeys(loop *EventLoop) {
	for key := range readyKeys {
		waitingClients := waitingKeys[key]

		if len(waitingClients) == 0 {
			delete(readyKeys, key)
			continue
		}

		res := core.Eval(&core.RedisCmd{
			Cmd:  "LPOP",
			Args: []string{key},
		})

		client := waitingClients[0]
		waitingKeys[key] = waitingClients[1:]
		client.flag &= ^CLIENT_BLOCKED
		client.when = time.Time{}

		client.ReplyBuf = append(client.ReplyBuf, res...)
		clientsPendingWrite[client.Fd] = client

		delete(readyKeys, key)
	}

}

func beforeSleep(loop *EventLoop) {

	processKeys(loop)

	for fd, client := range clientsPendingWrite {

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
