package server

import (
	"log"
	"net"
	"runtime"

	"github.com/redis-server/config"
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

func RunAsyncServer2() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.Println("Starting server on ", config.Host, config.Port)

	maxClients := 100000
	loop, err := CreateEventLoop(maxClients)
	if err != nil {
		return err
	}
	defer unix.Close(loop.EpollFD)

	serverFD, err := unix.Socket(unix.AF_INET, unix.O_NONBLOCK|unix.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(serverFD)

	unix.SetsockoptInt(serverFD, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)

	ip4 := net.ParseIP(config.Host)
	err = unix.Bind(serverFD, &unix.SockaddrInet4{
		Port: config.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	})
	if err != nil {
		return err
	}

	if err = unix.Listen(serverFD, maxClients); err != nil {
		return err
	}

	err = loop.AddFileEvent(serverFD, unix.EPOLLIN, AcceptTcpHandler, nil)
	if err != nil {
		return err
	}

	loop.Main()

	return nil
}
