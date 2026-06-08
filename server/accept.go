package server

import (
	"log"

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

			log.Println("Accept Error: ", err)
			return
		}

		if err := unix.SetNonblock(fd, true); err != nil {
			log.Println("SerNonBlock error :", err)
			unix.Close(fd)
			continue
		}

		if err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_NODELAY, 1); err != nil {
			log.Println("Set TCP_NODELAY error :", err)
		}

		client := NewClient(fd)

		err = el.AddFileEvent(fd, unix.EPOLLIN, ReadQueryFromClient, client)
		if err != nil {
			log.Println("Failed to add client to epoll: ", err)
			unix.Close(fd)
			continue
		}

		con_clients++
	}
}
