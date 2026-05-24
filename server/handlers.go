package server

import (
	"log"
	"os"

	"github.com/redis-server/core"
	"golang.org/x/sys/unix"
)

func handleNetwork(serverFD int, epollFD int) error {
	for {
		fd, _, err := unix.Accept(serverFD)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				break
			}
			log.Fatal(err)
			return err
		}

		con_clients++
		unix.SetNonblock(fd, true)

		var socketServerEvent unix.EpollEvent = unix.EpollEvent{
			Events: unix.EPOLLIN | EPOLLET,
			Fd:     int32(fd),
		}

		if err := unix.EpollCtl(epollFD, unix.EPOLL_CTL_ADD, fd, &socketServerEvent); err != nil {
			unix.Close(fd)
			con_clients -= 1
			log.Fatal(err)
		}
	}

	return nil
}

func handleMessages(epollFD int, fd int)  {
	comm := core.FDComm{Fd: int(fd)}
	cmds, err := readCommands(comm)

	if err != nil {
		log.Println(err.Error())
		unix.EpollCtl(epollFD, unix.EPOLL_CTL_DEL, int(fd), nil)
		unix.Close(int(fd))
		con_clients -= 1
		return
	}

	if len(cmds) > 0 && cmds[0].Cmd == "GET_LARGE" {
		file, err := os.Open("/var/redis/data/massive_payload.dat")
		if err != nil {
			log.Printf("Failed to open data block: %v", err)
			respond(cmds, comm)
			return
		}

		fileInfo, _ := file.Stat()
		fileFD := int(file.Fd())

		_, err = SpliceFileToSocket(fileFD, fd, fileInfo.Size())
		file.Close()

		if err != nil {
			log.Printf("Kernel splice transmission error: %v", err)
		}
		return
	}

	respond(cmds, comm)
}
