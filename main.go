package main

import (
	"flag"
	"log"

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
	core.Init()
	setupFlags()
	log.Println("starting the server...")
	server.RunAsyncServer()
}
