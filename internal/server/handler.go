package server

import (
	"io"
	"net"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/storage"
	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
)

type Handler struct {
	storage storage.Storage
}

func Route(server Server, storage storage.Storage) {
	handler := Handler{storage: storage}
	server.AddHandler("ECHO", handler.handleEcho)
	server.AddHandler("SET", handler.handleSet)
	server.AddHandler("GET", handler.handleGet)
	server.AddHandler("PING", handler.handlePing)
}

func (h Handler) handleEcho(c net.Conn, cmd commands.Command) {
	if len(cmd.Arguments) < 1 {
		io.WriteString(c, error("Not enough arguments for the ECHO command"))
		return
	}

	for _, arg := range cmd.Arguments {
		io.WriteString(c, string(parser.BulkStringData(arg).Marshal()))
	}
}

func (h Handler) handleSet(c net.Conn, cmd commands.Command) {
	if len(cmd.Arguments) != 2 {
		io.WriteString(c, error("GET command requirees exactly 2 arguments"))
		return
	}

	optArgs, ok := cmd.Options["PX"]
	if ok {
		if len(optArgs) != 1 {
			io.WriteString(c, error("PX parameter requires exactly 1 argument"))
			return
		}

		exp, err := strconv.Atoi(optArgs[0])
		if err != nil {
			io.WriteString(c, error("PX parameter must be integer"))
			return
		}

		err = h.storage.SetWithTimer(optArgs[0], optArgs[1], exp)
		if err != nil {
			io.WriteString(c, error(err.Error()))
			return
		}
	} else {
		err := h.storage.Set(cmd.Arguments[0], cmd.Arguments[1])
		if err != nil {
			io.WriteString(c, error(err.Error()))
			return
		}
	}

	io.WriteString(c, string(parser.StringData("OK").Marshal()))
}

func (h Handler) handleGet(c net.Conn, cmd commands.Command) {
	if len(cmd.Arguments) != 1 {
		io.WriteString(c, error("GET command requires exactly 1 argument"))
		return
	}
	val, err := h.storage.Get(cmd.Arguments[0])
	if err != nil {
		io.WriteString(c, error(err.Error()))
		return
	}

	io.WriteString(c, string(parser.StringData(val).Marshal()))
}

func (h Handler) handlePing(c net.Conn, cmd commands.Command) {
	io.WriteString(c, string(parser.StringData("PONG").Marshal()))
}

func error(str string) string {
	return string(parser.ErrorData(str).Marshal())
}
