package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/storage"
	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
	"github.com/mitchellh/mapstructure"
)

type Handler struct {
	storage storage.Storage
	server  *Server
}

func Route(server Server, storage storage.Storage) {
	handler := Handler{storage: storage, server: &server}
	server.AddHandler("ECHO", handler.handleEcho)
	server.AddHandler("SET", handler.handleSet)
	server.AddHandler("GET", handler.handleGet)
	server.AddHandler("PING", handler.handlePing)
	server.AddHandler("INFO", handler.handleInfo)
}

func (h Handler) handleEcho(c net.Conn, cmd commands.Command) {
	if len(cmd.Arguments) < 1 {
		io.WriteString(c, sendErr("Not enough arguments for the ECHO command"))
		return
	}

	for _, arg := range cmd.Arguments {
		io.WriteString(c, string(parser.BulkStringData(arg).Marshal()))
	}
}

func (h Handler) handleSet(c net.Conn, cmd commands.Command) {
	if len(cmd.Arguments) != 2 {
		io.WriteString(c, sendErr("SET command requirees exactly 2 arguments"))
		return
	}

	optArgs, ok := cmd.Options["PX"]
	if ok {
		if len(optArgs) != 1 {
			io.WriteString(c, sendErr("PX parameter requires exactly 1 argument"))
			return
		}

		exp, err := strconv.Atoi(optArgs[0])
		if err != nil {
			io.WriteString(c, sendErr("PX parameter must be integer"))
			return
		}

		err = h.storage.SetWithTimer(cmd.Arguments[0], cmd.Arguments[1], exp)
		if err != nil {
			io.WriteString(c, sendErr(err.Error()))
			return
		}
	} else {
		err := h.storage.Set(cmd.Arguments[0], cmd.Arguments[1])
		if err != nil {
			io.WriteString(c, sendErr(err.Error()))
			return
		}
	}

	io.WriteString(c, string(parser.StringData("OK").Marshal()))
}

func (h Handler) handleGet(c net.Conn, cmd commands.Command) {
	if len(cmd.Arguments) != 1 {
		io.WriteString(c, sendErr("GET command requires exactly 1 argument"))
		return
	}
	val, err := h.storage.Get(cmd.Arguments[0])
	if err != nil {
		io.WriteString(c, string(parser.NullBulkStringData().Marshal()))
		log.Println(err.Error())
		return
	}

	io.WriteString(c, string(parser.StringData(val).Marshal()))
}

func (h Handler) handlePing(c net.Conn, cmd commands.Command) {
	io.WriteString(c, string(parser.StringData("PONG").Marshal()))
}

func (h Handler) handleInfo(c net.Conn, cmd commands.Command) {
	var info map[string]any
	err := mapstructure.Decode(h.server.GetServerInfo(), &info)
	if err != nil {
		io.WriteString(c, sendErr(err.Error()))
		return
	}

	var infoData []parser.Data
	for k, v := range info {
		infoString := fmt.Sprintf("%s:%v", k, v)
		infoData = append(infoData, parser.BulkStringData(infoString))
	}
	io.WriteString(c, string(parser.ArrayData(infoData).Marshal()))
}

func sendErr(str string) string {
	errString := fmt.Sprintf("ERR %s", str)
	return string(parser.ErrorData(errString).Marshal())
}
