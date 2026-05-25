package server

import "golang.org/x/sys/unix"

type FileProc func(loop *EventLoop, fd int, clientData interface{})

type FileEvent struct {
	Mask       uint32
	ReadProc   FileProc
	WriteProc  FileProc
	ClientData interface{}
}

type EventLoop struct {
	EpollFD int
	Events  map[int]*FileEvent
	Fired   []unix.EpollEvent
	Stop    bool
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
