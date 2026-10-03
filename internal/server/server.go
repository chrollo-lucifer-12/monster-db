package server

import (
	"log/slog"
	"net"
	"os"
	"sync"
	"sync/atomic"

	_ "net/http/pprof"

	"github.com/redis-server/internal/config"
	core "github.com/redis-server/internal/store"

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

	maxClients := 100000

	slog.Info("server starting",
		"host", config.Host,
		"port", config.Port,
		"max_clients", maxClients,
	)

	loop, err := CreateEventLoop(maxClients)
	if err != nil {
		slog.Error("failed to create event loop",
			"err", err,
		)
		return err
	}
	defer unix.Close(loop.EpollFD)

	serverFD, err := unix.Socket(unix.AF_INET, unix.O_NONBLOCK|unix.SOCK_STREAM, 0)
	if err != nil {
		slog.Error("failed to create server socket",
			"err", err,
		)
		return err
	}
	defer unix.Close(serverFD)

	err = unix.SetsockoptInt(serverFD, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if err != nil {
		slog.Error("failed to set socket option",
			"option", "SO_REUSEADDR",
			"err", err,
		)
		return err
	}

	ip4 := net.ParseIP(config.Host)
	err = unix.Bind(serverFD, &unix.SockaddrInet4{
		Port: config.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	})
	if err != nil {
		slog.Error("failed to bind server socket",
			"host", config.Host,
			"port", config.Port,
			"err", err,
		)
		return err
	}

	if err = unix.Listen(serverFD, maxClients); err != nil {
		slog.Error("failed to listen",
			"err", err,
		)
		return err
	}

	err = loop.AddFileEvent(serverFD, unix.EPOLLIN, AcceptTcpHandler, nil)
	if err != nil {
		slog.Error("failed to register server socket",
			"err", err,
		)
		return err
	}

	loop.addTimeEvent(1000, serverCronHandler, nil)
	loop.addTimeEvent(100, HandleBlockedClients, nil)

	if err := loop.Main(); err != nil {
		slog.Error("event loop stopped",
			"err", err,
		)
		return err
	}

	slog.Info("event loop stopped")

	return nil
}
