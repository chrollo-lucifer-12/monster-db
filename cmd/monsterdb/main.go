package main

import (
	"flag"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"

	"github.com/redis-server/internal/config"
	"github.com/redis-server/internal/logger"
	"github.com/redis-server/internal/server"
	core "github.com/redis-server/internal/store"

	"golang.org/x/sys/unix"
)

func setupFlags() {
	flag.StringVar(&config.Host, "host", "127.0.0.1", "host")
	flag.IntVar(&config.Port, "port", 6379, "port")
	flag.IntVar(&config.KeyLimit, "key_limit", 3, "key_limit")
	flag.Parse()
}

func main() {

	logger.Init()

	var sigs chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(sigs, unix.SIGTERM, unix.SIGINT)

	var wg sync.WaitGroup
	wg.Add(2)

	core.Init()
	core.Restoreaof()
	core.MarkReady = server.MarkReady
	core.SignalModifiedKey = server.TouchWatchedKeys
	setupFlags()

	go server.RunAsyncServer(&wg)
	go server.WaitForSignal(&wg, sigs)

	wg.Wait()
}
