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

var (
	lastCron time.Time = time.Now()

	lastRDB      time.Time = time.Now()
	rdbFrequency           = 900 * time.Second

	cronFrequency = 1000 * time.Millisecond
)

func serverCron() {
	now := time.Now()

	if now.Sub(lastRDB) > rdbFrequency {
		core.TriggerRDB()
		lastRDB = now
	}

	core.DeleteExpiredKey()
}

func RunAsyncServer() error {

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.Println("starting server on ", config.Host, config.Port)

	max_clients := 1_00_000

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
		Events: unix.EPOLLIN,
		Fd:     int32(serverFD),
	}

	if err = unix.EpollCtl(epollFD, unix.EPOLL_CTL_ADD, serverFD, &socketServerEvent); err != nil {
		return err
	}

	for {

		if time.Since(lastCron) >= cronFrequency {
			serverCron()
			lastCron = time.Now()
		}

		timeout := int(time.Until(lastCron.Add(cronFrequency)).Milliseconds())

		if timeout < 0 {
			timeout = 0
		}
		if timeout > 100 {
			timeout = 100
		}

		nevents, e := unix.EpollWait(epollFD, events[:], timeout)
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

						if err == unix.EMFILE || err == unix.ENFILE {
							log.Println("FD limit reached, cannot accept more clients")
							break
						}

						log.Println("accept error:", err)
						return nil
					}

					con_clients++
					unix.SetNonblock(fd, true)

					var socketServerEvent unix.EpollEvent = unix.EpollEvent{
						Events: unix.EPOLLIN,
						Fd:     int32(fd),
					}

					if err := unix.EpollCtl(epollFD, unix.EPOLL_CTL_ADD, fd, &socketServerEvent); err != nil {
						unix.Close(fd)
						con_clients -= 1
						log.Println("epoll add error:", err)
						continue
					}
				}

			} else {
				comm := core.FDComm{Fd: int(currentFD)}
				cmds, err := readCommands(comm)

				if err != nil {
					log.Println(err.Error())
					unix.EpollCtl(epollFD, unix.EPOLL_CTL_DEL, int(currentFD), nil)
					unix.Close(int(currentFD))
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
