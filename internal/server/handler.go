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

type BaseHandler struct {
	storage storage.Storage
	server  *Server
	silent  bool
}

func RouteBasic(server Server, storage storage.Storage, silent bool) {
	handler := BaseHandler{storage: storage, server: &server, silent: silent}
	server.AddHandler("ECHO", handler.handleEcho)
	server.AddHandler("SET", handler.handleSet)
	server.AddHandler("GET", handler.handleGet)
	server.AddHandler("PING", handler.handlePing)
	server.AddHandler("INFO", handler.handleInfo)
}

func (h BaseHandler) handleEcho(c net.Conn, cmd commands.Command) {
	if len(cmd.Arguments) < 1 {
		h.Send(c, newErr("Not enough arguments for the ECHO command"))
		return
	}

	for _, arg := range cmd.Arguments {
		h.Send(c, string(parser.BulkStringData(arg).Marshal()))
	}
}

func (h BaseHandler) handleSet(c net.Conn, cmd commands.Command) {
	if len(cmd.Arguments) != 2 {
		h.Send(c, newErr("SET command requirees exactly 2 arguments"))
		return
	}

	optArgs, ok := cmd.Options["PX"]
	if ok {
		if len(optArgs) != 1 {
			h.Send(c, newErr("PX parameter requires exactly 1 argument"))
			return
		}

		exp, err := strconv.Atoi(optArgs[0])
		if err != nil {
			h.Send(c, newErr("PX parameter must be integer"))
			return
		}

		err = h.storage.SetWithTimer(cmd.Arguments[0], cmd.Arguments[1], exp)
		if err != nil {
			h.Send(c, newErr(err.Error()))
			return
		}
	} else {
		err := h.storage.Set(cmd.Arguments[0], cmd.Arguments[1])
		if err != nil {
			h.Send(c, newErr(err.Error()))
			return
		}
	}

	h.Send(c, string(parser.StringData("OK").Marshal()))
}

func (h BaseHandler) handleGet(c net.Conn, cmd commands.Command) {
	if len(cmd.Arguments) != 1 {
		h.Send(c, newErr("GET command requires exactly 1 argument"))
		return
	}
	val, err := h.storage.Get(cmd.Arguments[0])
	if err != nil {
		h.Send(c, string(parser.NullBulkStringData().Marshal()))
		log.Println(err.Error())
		return
	}

	h.Send(c, string(parser.StringData(val).Marshal()))
}

func (h BaseHandler) handlePing(c net.Conn, cmd commands.Command) {
	h.Send(c, string(parser.StringData("PONG").Marshal()))
}

func (h BaseHandler) handleInfo(c net.Conn, cmd commands.Command) {
	var info map[string]any
	err := mapstructure.Decode(GetReplInfo(), &info)
	if err != nil {
		h.Send(c, newErr(err.Error()))
		return
	}

	var b strings.Builder
	for k, v := range info {
		b.WriteString(fmt.Sprintf("%s:%v\r\n", k, v))
	}

	str := b.String()[:len(b.String())-2]
	resStr := string(parser.BulkStringData(str).Marshal())

	h.Send(c, resStr)
}

func (h BaseHandler) Send(c net.Conn, data string) {
	if !h.silent {
		io.WriteString(c, data)
	}
}

type MasterHandler struct {
	server   Server
	mc       *MasterContext
	sessions map[string]Replica
}

func RouteMaster(server Server, mc *MasterContext) {
	handler := MasterHandler{server, mc, map[string]Replica{}}
	server.AddHandler("REPLCONF", handler.handleReplconf)
	server.AddHandler("PSYNC", handler.handlePsync)
}

func (h MasterHandler) handleReplconf(c net.Conn, cmd commands.Command) {
	repl, err := h.mc.GetReplica(c)
	if err != nil {
		repl = Replica{
			Conn: c,
			IsUp: false,
		}
		h.mc.SetReplica(repl)
	}

	if lp, ok := cmd.Options["LISTENING-PORT"]; ok {
		host := strings.Split(c.RemoteAddr().String(), ":")[0]
		addr := fmt.Sprintf("%s:%s", host, lp[0])
		repl.ServerAddr = addr
	} else if capa, ok := cmd.Options["CAPA"]; ok {
		repl.Capas = capa
	}

	h.mc.SetReplica(repl)
	io.WriteString(c, string(parser.StringData("OK").Marshal()))
}

func (h MasterHandler) handlePsync(c net.Conn, cmd commands.Command) {
	serverInfo := GetReplInfo()
	replica, err := h.mc.GetReplica(c)
	if err != nil {
		io.WriteString(c, newErr(err.Error()))
		return
	}

	fullresync := fmt.Sprintf("FULLRESYNC %s %d", serverInfo.ReplId, serverInfo.ReplOffset)
	io.WriteString(c, string(parser.StringData(fullresync).Marshal()))

	rdb, err := base64.StdEncoding.DecodeString("UkVESVMwMDEx+glyZWRpcy12ZXIFNy4yLjD6CnJlZGlzLWJpdHPAQPoFY3RpbWXCbQi8ZfoIdXNlZC1tZW3CsMQQAPoIYW9mLWJhc2XAAP/wbjv+wP9aog==")
	if err != nil {
		io.WriteString(c, newErr("Can't decode base64 RDB file"))
	}

	io.WriteString(c, fmt.Sprintf("$%d\r\n%s", len(rdb), string(rdb)))
	replica.IsUp = true
	h.mc.SetReplica(replica)
}

func newErr(str string) string {
	errString := fmt.Sprintf("ERR %s", str)
	return string(parser.ErrorData(errString).Marshal())
}
