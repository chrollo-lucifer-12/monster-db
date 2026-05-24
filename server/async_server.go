package server

import (
	"log"
	"net"
	"syscall"
	"time"

	"github.com/redis-server/config"
	"github.com/redis-server/core"
)

var con_clients int = 0
var cronFrequency time.Duration = 1 * time.Second
var lastCronExecTime time.Time = time.Now()
var rdbFrequency time.Duration = 5 * time.Second
var lastRdbExecTime time.Time = time.Now()

func RunAsyncServer() error {
	log.Println("starting server on ", config.Host, config.Port)

	max_clients := 20_000

	var events []syscall.EpollEvent = make([]syscall.EpollEvent, max_clients)

	serverFD, err := syscall.Socket(syscall.AF_INET, syscall.O_NONBLOCK|syscall.SOCK_STREAM, 0)
	if err != nil {
		return err
	}

	defer syscall.Close(serverFD)

	if err = syscall.SetNonblock(serverFD, true); err != nil {
		return err
	}

	ip4 := net.ParseIP(config.Host)
	if err = syscall.Bind(serverFD, &syscall.SockaddrInet4{
		Port: config.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	}); err != nil {
		return err
	}

	if err = syscall.Listen(serverFD, max_clients); err != nil {
		return err
	}

	epollFD, err := syscall.EpollCreate1(0)
	if err != nil {
		log.Fatal(err)
	}

	defer syscall.Close(epollFD)

	var socketServerEvent syscall.EpollEvent = syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:     int32(serverFD),
	}

	if err = syscall.EpollCtl(epollFD, syscall.EPOLL_CTL_ADD, serverFD, &socketServerEvent); err != nil {
		return err
	}

	for {

		if time.Now().After(lastRdbExecTime.Add(rdbFrequency)) {
			core.TriggerRDB()
			lastRdbExecTime = time.Now()
		}

		if time.Now().After(lastCronExecTime.Add(cronFrequency)) {
			core.DeleteExpiredKey()
			lastCronExecTime = time.Now()
		}

		nevents, e := syscall.EpollWait(epollFD, events[:], -1)
		if e != nil {
			continue
		}

		for i := 0; i < nevents; i++ {
			if int(events[i].Fd) == serverFD {
				fd, _, err := syscall.Accept(serverFD)
				if err != nil {
					log.Fatal(err)
					return err
				}

				con_clients++
				syscall.SetNonblock(fd, true)

				var socketServerEvent syscall.EpollEvent = syscall.EpollEvent{
					Events: syscall.EPOLLIN,
					Fd:     int32(fd),
				}

				if err := syscall.EpollCtl(epollFD, syscall.EPOLL_CTL_ADD, fd, &socketServerEvent); err != nil {
					log.Fatal(err)
				}
			} else {
				comm := core.FDComm{Fd: int(events[i].Fd)}
				cmds, err := readCommands(comm)

				if err != nil {
					syscall.Close(int(events[i].Fd))
					con_clients -= 1
					continue
				}
				respond(cmds, comm)
			}
		}
	}
}
