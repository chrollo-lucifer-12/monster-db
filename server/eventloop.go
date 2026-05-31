package server

import (
	"log"
	"time"

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

func (loop *EventLoop) processTimeEvents() {
	now := time.Now()
	var validTimeEvents []*TimeEvent

	for _, te := range loop.TimeEvents {
		if now.After(te.When) {
			retval := te.Proc(loop, te.ID, te.ClientData)
			if retval != -1 {
				te.When = time.Now().Add(time.Duration(retval) * time.Millisecond)
				validTimeEvents = append(validTimeEvents, te)
			}
		} else {
			validTimeEvents = append(validTimeEvents, te)
		}
	}
	loop.TimeEvents = validTimeEvents
}

func (el *EventLoop) AddFileEvent(fd int, mask uint32, proc FileProc, clientData interface{}) error {
	fe, exists := el.Events[fd]
	op := unix.EPOLL_CTL_ADD

	if exists {
		op = unix.EPOLL_CTL_MOD
	} else {
		fe = &FileEvent{}
		el.Events[fd] = fe
	}

	fe.Mask |= mask

	if mask&unix.EPOLLIN != 0 {
		fe.ReadProc = proc
	}
	if mask&unix.EPOLLOUT != 0 {
		fe.WriteProc = proc
	}

	fe.ClientData = clientData

	ev := unix.EpollEvent{
		Events: fe.Mask,
		Fd:     int32(fd),
	}

	if err := unix.EpollCtl(el.EpollFD, op, fd, &ev); err != nil {
		return err
	}

	return nil
}

func (el *EventLoop) DeleteFileEvent(fd int, mask uint32) {
	fe, exists := el.Events[fd]
	if !exists {
		return
	}

	fe.Mask &= ^mask

	if fe.Mask == 0 {
		unix.EpollCtl(el.EpollFD, unix.EPOLL_CTL_DEL, fd, nil)
		delete(el.Events, fd)
	} else {
		ev := unix.EpollEvent{
			Events: fe.Mask,
			Fd:     int32(fd),
		}
		unix.EpollCtl(el.EpollFD, unix.EPOLL_CTL_MOD, fd, &ev)
	}
}

func beforeSleep(loop *EventLoop) {
	for fd, client := range clientsPendingWrite {

		n, err := unix.Write(fd, client.ReplyBuf)

		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				loop.AddFileEvent(fd, unix.EPOLLOUT, SendReplyToClient, client)
				continue
			}
			log.Printf("Greedy write error on FD %d: %v\n", fd, err)
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

func (el *EventLoop) Main() {
	el.Stop = false

	for !el.Stop {

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
	}
}
