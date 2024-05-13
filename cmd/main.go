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
	PORT        = 6379
	MASTER_HOST = ""
)

func init() {
	flag.IntVar(&PORT, "port", PORT, "Port number")
	flag.StringVar(&MASTER_HOST, "replicaof", MASTER_HOST, "Master server address and port: <host port> (should be the last argument)")
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

	sv := server.NewServer(cmdParser)
	storage := storage.NewStorage()
	server.RouteBasic(sv, storage)

	if MASTER_HOST != "" && len(flag.Args()) > 0 {
		replicaCtx := server.RegisterReplica(sv, MASTER_HOST, flag.Arg(0), PORT)
		server.RouteReplica(sv, replicaCtx)
	} else {
		mc := server.SetAsMaster(sv)
		server.RouteMaster(sv, mc)
	}
	sv.Listen(fmt.Sprintf(":%d", PORT))
}
