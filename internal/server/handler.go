package server

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

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
	server.AddHandler("REPLCONF", handler.handleReplconf)
	server.AddHandler("PSYNC", handler.handlePsync)
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

	var b strings.Builder
	for k, v := range info {
		b.WriteString(fmt.Sprintf("%s:%v\r\n", k, v))
	}

	str := b.String()[:len(b.String())-2]
	resStr := string(parser.BulkStringData(str).Marshal())

	io.WriteString(c, resStr)
}

func (h Handler) handleReplconf(c net.Conn, cmd commands.Command) {
	io.WriteString(c, string(parser.StringData("OK").Marshal()))
}

func (h Handler) handlePsync(c net.Conn, cmd commands.Command) {
	serverInfo := h.server.GetServerInfo()

	fullresync := fmt.Sprintf("FULLRESYNC %s %d", serverInfo.ReplId, serverInfo.ReplOffset)
	io.WriteString(c, string(parser.StringData(fullresync).Marshal()))

	rdb, err := base64.StdEncoding.DecodeString("UkVESVMwMDEx+glyZWRpcy12ZXIFNy4yLjD6CnJlZGlzLWJpdHPAQPoFY3RpbWXCbQi8ZfoIdXNlZC1tZW3CsMQQAPoIYW9mLWJhc2XAAP/wbjv+wP9aog==")
	if err != nil {
		io.WriteString(c, sendErr("Can't decode base64 RDB file"))
	}

	io.WriteString(c, fmt.Sprintf("$%d\r\n%s", len(rdb), string(rdb)))
}

func sendErr(str string) string {
	errString := fmt.Sprintf("ERR %s", str)
	return string(parser.ErrorData(errString).Marshal())
}
