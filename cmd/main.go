package main

import (
	"github.com/codecrafters-io/redis-starter-go/internal/server"
	"github.com/codecrafters-io/redis-starter-go/internal/storage"
)

func main() {
	sv := server.NewServer()
	storage := storage.NewStorage()
	server.Route(sv, *storage)
	sv.Listen(":6969")
}
