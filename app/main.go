package main

import (
	"flag"
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/internal/server"
	"github.com/codecrafters-io/redis-starter-go/internal/storage"
)

var (
	PORT = 6379
)

func init() {
	flag.IntVar(&PORT, "port", PORT, "Port number")
}

func main() {
	flag.Parse()

	sv := server.NewServer()
	storage := storage.NewStorage()
	server.Route(sv, *storage)
	sv.Listen(fmt.Sprintf(":%d", PORT))
}
