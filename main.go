package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"

	"github.com/redis-server/config"

	"golang.org/x/sys/unix"
)

func setupFlags() {
	flag.StringVar(&config.Host, "host", "127.0.0.1", "host")
	flag.IntVar(&config.Port, "port", 6379, "port")
	flag.IntVar(&config.KeyLimit, "key_limit", 3, "key_limit")
	flag.Parse()
}

func main() {

	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	var sigs chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(sigs, unix.SIGTERM, unix.SIGINT)

	var wg sync.WaitGroup
	wg.Add(2)

	internal.Init()
	internal.Restoreaof()
	internal.MarkReady = server.MarkReady
	internal.SignalModifiedKey = server.TouchWatchedKeys
	setupFlags()
	log.Println("starting the server...")

	go server.RunAsyncServer(&wg)
	go server.WaitForSignal(&wg, sigs)

	wg.Wait()
}
