package server

import (
	"log"
	"net"
	"runtime"
	"time"

	"github.com/redis-server/config"
	"github.com/redis-server/core"
	"golang.org/x/sys/unix"
)

const EPOLLET uint32 = 1 << 31

var con_clients int = 0
var cronFrequency time.Duration = 10 * time.Second
var lastCronExecTime time.Time = time.Now()
var rdbFrequency time.Duration = 900 * time.Second
var lastRdbExecTime time.Time = time.Now()

func RunAsyncServer() error {

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.Println("starting server on ", config.Host, config.Port)

	max_clients := 20_000

	var events []unix.EpollEvent = make([]unix.EpollEvent, max_clients)

	serverFD, err := unix.Socket(unix.AF_INET, unix.O_NONBLOCK|unix.SOCK_STREAM, 0)
	if err != nil {
		return err
	}

	defer unix.Close(serverFD)

	if err = unix.SetNonblock(serverFD, true); err != nil {
		return err
	}

	ip4 := net.ParseIP(config.Host)
	if err = unix.Bind(serverFD, &unix.SockaddrInet4{
		Port: config.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	}); err != nil {
		return err
	}

	if err = unix.Listen(serverFD, max_clients); err != nil {
		return err
	}

	epollFD, err := unix.EpollCreate1(0)
	if err != nil {
		log.Fatal(err)
	}

	defer unix.Close(epollFD)

	var socketServerEvent unix.EpollEvent = unix.EpollEvent{
		Events: uint32(unix.EPOLLIN) | EPOLLET,
		Fd:     int32(serverFD),
	}

	if err = unix.EpollCtl(epollFD, unix.EPOLL_CTL_ADD, serverFD, &socketServerEvent); err != nil {
		return err
	}

	for {
		now := time.Now()
		nextRDB := lastRdbExecTime.Add(rdbFrequency)
		nextCron := lastCronExecTime.Add(cronFrequency)

		next := nextRDB

		if nextCron.Before(next) {
			next = nextCron
		}

		if now.After(lastRdbExecTime.Add(rdbFrequency)) {
			core.TriggerRDB()
			lastRdbExecTime = now
		}

		if now.After(lastCronExecTime.Add(cronFrequency)) {
			core.DeleteExpiredKey()
			lastCronExecTime = now
		}

		timeout := time.Until(next)
		if timeout < 0 {
			timeout = 0
		}

		nevents, e := unix.EpollWait(epollFD, events[:], int(timeout.Milliseconds()))
		if e != nil {
			continue
		}

		for i := 0; i < nevents; i++ {
			currentFD := int(events[i].Fd)

			if currentFD == serverFD {
				if err := handleNetwork(currentFD, epollFD); err != nil {
					continue
				}
			}

			handleMessages(epollFD, currentFD)
		}
	}
}
