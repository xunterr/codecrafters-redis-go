package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

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
	var wg sync.WaitGroup
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
	server.RouteBasic(sv, *storage)

	wg.Add(1)
	go func() {
		sv.Listen(fmt.Sprintf(":%d", PORT))
		wg.Done()
	}()

	if MASTER_HOST != "" && len(flag.Args()) > 0 {
		server.RegisterReplica(MASTER_HOST, flag.Arg(0), PORT)
	} else {
		mc := server.SetAsMaster(&sv)
		server.RouteMaster(sv, mc)
	}
	wg.Wait()
}
