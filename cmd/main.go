package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
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
		StartAsReplica(sv)
	} else {
		StartAsMaster(sv, connHandler)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	sv.Listen(ctx, fmt.Sprintf(":%d", PORT))
}

func StartAsReplica(sv *server.Server) {
	masterAddr := strings.Split(MASTER_ADDR, " ")
	if len(masterAddr) != 2 {
		log.Fatalln("<MASTER_ADDR> parameter should contain address and port")
		return
	}

	c, err := net.Dial("tcp", fmt.Sprintf("%s:%s", masterAddr[0], masterAddr[1]))
	if err != nil {
		log.Fatalln(err.Error())
		return
	}

	replicaCtx, err := server.NewReplica(sv, c, PORT)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}

	sv.SetRwProvider(replicaCtx.ReplicaRwProvider)
	server.RouteReplica(sv, replicaCtx)
	client, ctx := sv.AddClient(context.Background(), c)
	go sv.Serve(ctx, client)

	replicaCtx.InitHandshake()
}

func StartAsMaster(sv *server.Server, connHandler *server.ConnectionHandler) {
	mc := server.NewMaster(connHandler)
	sv.SetCallChain(mc.MasterCallChain(sv))
	server.RouteMaster(sv, mc)
}
