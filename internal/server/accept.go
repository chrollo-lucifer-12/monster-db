package server

import (
	"log/slog"

	"golang.org/x/sys/unix"
)

func AcceptTcpHandler(el *EventLoop, serverFD int, clientData interface{}) {

	for {
		fd, _, err := unix.Accept(serverFD)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {

				break
			}

			if err == unix.EMFILE || err == unix.ENFILE {

				break
			}

			slog.Error("Accept Error", "err", err)
			return
		}

		if err := unix.SetNonblock(fd, true); err != nil {
			slog.Error("SerNonBlock", "err", err)
			unix.Close(fd)
			continue
		}

		if err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_NODELAY, 1); err != nil {
			slog.Error("Set TCP_NODELAY", "err", err)
		}

		client := NewClient(fd)

		err = el.AddFileEvent(fd, unix.EPOLLIN, readQueryFromClient, client)
		if err != nil {
			slog.Error("Failed to add client to epoll", "err", err)
			unix.Close(fd)
			continue
		}

		con_clients++
	}
}
