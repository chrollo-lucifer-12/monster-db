package main

import (
	"flag"
	"log"
	"os"

	"github.com/redis-server/config"
	"github.com/redis-server/core"
	"github.com/redis-server/server"
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

	core.Init()
	//	core.RESTOREAOF()
	setupFlags()
	log.Println("starting the server...")

	err := server.RunAsyncServer2()

	if err != nil {
		log.Panic(err)
	}
}
