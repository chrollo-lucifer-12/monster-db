package server

import (
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"

	_ "net/http/pprof"

	"github.com/redis-server/config"
	"github.com/redis-server/core"

	"golang.org/x/sys/unix"
)

const EngineStatus_WAITING int32 = 1 << 1
const EngineStatus_BUSY int32 = 1 << 2
const EnngineStatus_SHUTTING_DOWN int32 = 1 << 3

func MarkReady(key string) {
	if clients, exists := waitingKeys[key]; exists && len(clients) > 0 {
		readyKeys[key] = struct{}{}
	}
}

func TouchWatchedKeys(key string) {
	if len(watchedKeys[key]) == 0 {
		return
	}

	for _, client := range watchedKeys[key] {
		client.flag |= CLIENT_CAS
	}
}

func WaitForSignal(wg *sync.WaitGroup, sigs chan os.Signal) {
	defer wg.Done()
	<-sigs

	for atomic.LoadInt32(&eStatus) == EngineStatus_BUSY {
	}

	atomic.StoreInt32(&eStatus, EnngineStatus_SHUTTING_DOWN)

	core.Shutdown()
	os.Exit(0)
}

func RunAsyncServer(wg *sync.WaitGroup) error {

	defer wg.Done()
	defer func() {
		atomic.StoreInt32(&eStatus, EnngineStatus_SHUTTING_DOWN)
	}()

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

	//	loop.AddTimeEvent(1000, ReplicationHeartbeatCron, nil)
	loop.AddTimeEvent(1000, ServerCronHandler, nil)
	loop.AddTimeEvent(100, HandleBlockedClients, nil)

	return loop.Main()

}
