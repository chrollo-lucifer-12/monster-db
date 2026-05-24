package server

import (
	"log"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/redis-server/config"
	"github.com/redis-server/core"
	"golang.org/x/sys/unix"
)

var con_clients int = 0
var cronFrequency time.Duration = 10 * time.Second
var lastCronExecTime time.Time = time.Now()
var rdbFrequency time.Duration = 900 * time.Second
var lastRdbExecTime time.Time = time.Now()

func SpliceFileToSocket(fileFD int, clientFD int, size int64) (int64, error) {

	var pipeFDs [2]int
	err := unix.Pipe(pipeFDs[:])

	if err != nil {
		return 0, err
	}

	defer unix.Close(pipeFDs[0])
	defer unix.Close(pipeFDs[1])

	var totalSpliced int64 = 0

	spliceFlags := unix.SPLICE_F_MOVE | unix.SPLICE_F_NONBLOCK

	for totalSpliced < size {
		remaining := size - totalSpliced

		nBytesIntoPipe, err := unix.Splice(fileFD, nil, pipeFDs[1], nil, int(remaining), spliceFlags)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				continue
			}
			return totalSpliced, err
		}
		if nBytesIntoPipe == 0 {
			break
		}

		var pipeBytesWritten int64 = 0
		for pipeBytesWritten < nBytesIntoPipe {
			nBytesOutToSocket, err := unix.Splice(
				pipeFDs[0],
				nil,
				clientFD,
				nil,
				int(nBytesIntoPipe-pipeBytesWritten),
				spliceFlags,
			)
			if err != nil {
				if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
					continue
				}
				return totalSpliced, err
			}
			pipeBytesWritten += nBytesOutToSocket
		}
		totalSpliced += nBytesIntoPipe
	}

	return totalSpliced, nil
}

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

	const EPOLLET uint32 = 1 << 31

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
			} else {
				comm := core.FDComm{Fd: int(events[i].Fd)}
				cmds, err := readCommands(comm)

				if err != nil {
					log.Println(err.Error())
					unix.EpollCtl(epollFD, unix.EPOLL_CTL_DEL, int(events[i].Fd), nil)
					unix.Close(int(events[i].Fd))
					con_clients -= 1
					continue
				}

				if len(cmds) > 0 && cmds[0].Cmd == "GET_LARGE" {
					file, err := os.Open("/var/redis/data/massive_payload.dat")
					if err != nil {
						log.Printf("Failed to open data block: %v", err)
						respond(cmds, comm)
						continue
					}

					fileInfo, _ := file.Stat()
					fileFD := int(file.Fd())

					_, err = SpliceFileToSocket(fileFD, currentFD, fileInfo.Size())
					file.Close()

					if err != nil {
						log.Printf("Kernel splice transmission error: %v", err)
					}
					continue
				}

				respond(cmds, comm)
			}
		}
	}
}
