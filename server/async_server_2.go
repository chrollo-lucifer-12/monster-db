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
				log.Println("FD limit reached, cannot accept more clients")
				break
			}

			log.Println("Accept Error: ", err)
			return
		}

		con_clients++

		if err := unix.SetNonblock(fd, true); err != nil {
			log.Println("SerNonBlock error :", err)
			unix.Close(fd)
			con_clients--
			continue
		}

		err = el.AddFileEvent(fd, unix.EPOLLIN, ReadQueryFromClient, nil)
		if err != nil {
			log.Println("Failed to add client to epoll: ", err)
			unix.Close(fd)
			con_clients--
			continue
		}
	}
}

func ReadQueryFromClient(loop *EventLoop, fd int, clientData interface{}) {
	log.Printf("Data available to read on client FD: %d\n", fd)
}
