package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/server"
	"github.com/codecrafters-io/redis-starter-go/internal/storage"
)

var (
	PORT        = 6379
	MASTER_ADDR = ""
)

func init() {
	flag.IntVar(&PORT, "port", PORT, "Port number")
	flag.StringVar(&MASTER_ADDR, "replicaof", MASTER_ADDR, "Master server address and port: \"<host port>\"")
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
	connHandler := server.NewConnectionHandler(cmdParser)
	sv := server.NewServer(connHandler)
	storage := storage.NewStorage()
	server.RouteBasic(sv, storage)

	if MASTER_ADDR != "" {
		masterAddr := strings.Split(MASTER_ADDR, " ")
		if len(masterAddr) != 2 {
			log.Fatalln("<MASTER_ADDR> parameter should contain address and port")
			return
		}
		replicaCtx := server.RegisterReplica(sv, masterAddr[0], masterAddr[1], PORT)
		server.RouteReplica(sv, replicaCtx)
	} else {
		mc := server.SetAsMaster(sv)
		server.RouteMaster(sv, mc)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	sv.Listen(ctx, fmt.Sprintf(":%d", PORT))
}
