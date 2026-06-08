package server

import "golang.org/x/sys/unix"

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
