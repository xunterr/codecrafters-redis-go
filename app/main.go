package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/server"
	"github.com/codecrafters-io/redis-starter-go/internal/storage"
)

var (
	PORT      = 6379
	REPLICAOF = ""
)

func init() {
	flag.IntVar(&PORT, "port", PORT, "Port number")
	flag.StringVar(&REPLICAOF, "replicaof", REPLICAOF, "Master server address and port: <address port>")
}

func main() {
	flag.Parse()

	path, err := filepath.Abs("cmds.json")
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	table, err := commands.LoadJSON(path)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	cmdParser := commands.NewCommandParser(table)

	sInfo := server.ServerInfo{
		Role: server.Master,
	}
	if REPLICAOF != "" {
		sInfo.Role = server.Slave
	}
	sv := server.NewServer(cmdParser, sInfo)

	storage := storage.NewStorage()
	server.Route(sv, *storage)
	sv.Listen(fmt.Sprintf(":%d", PORT))
}
