package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/redis-server/alloc"
	"github.com/redis-server/config"
	"github.com/redis-server/core"
	"github.com/redis-server/server"
	"golang.org/x/sys/unix"
)

func setupFlags() {
	flag.StringVar(&config.Host, "host", "127.0.0.1", "host")
	flag.IntVar(&config.Port, "port", 6379, "port")
	flag.IntVar(&config.KeyLimit, "key_limit", 3, "key_limit")
	flag.Parse()
}

func main() {

	if len(os.Args) > 1 && os.Args[1] == "--rdb-dump" {
		core.SaveRDB()
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "--rewrite-aof" {
		core.DumpAllAOF()
		return
	}

	var sigs chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(sigs, unix.SIGTERM, unix.SIGINT)

	var wg sync.WaitGroup
	wg.Add(2)

	alloc.InitGlobalAllocator(config.Maxmem)
	core.Init()
	//	core.RESTOREAOF()
	setupFlags()
	log.Println("starting the server...")

	go server.RunAsyncServer(&wg)
	go server.WaitForSignal(&wg, sigs)

	wg.Wait()
}
