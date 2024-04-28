package server

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/storage"
	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
	"github.com/mitchellh/mapstructure"
)

type BaseHandler struct {
	storage storage.Storage
	server  *Server
}

type MasterHandler struct {
	server   Server
	mc       *MasterContext
	sessions map[string]Replica
}

type ReplicaHandler struct {
	replicaCtx *ReplicaContext
}

func RouteBasic(server Server, storage storage.Storage) {
	handler := BaseHandler{storage: storage, server: &server}
	server.AddHandler("ECHO", handler.handleEcho)
	server.AddHandler("SET", handler.handleSet)
	server.AddHandler("GET", handler.handleGet)
	server.AddHandler("PING", handler.handlePing)
	server.AddHandler("INFO", handler.handleInfo)
}

func (h BaseHandler) handleEcho(req Request, rw ResponseWriter) {
	if len(req.Command.Arguments) < 1 {
		rw.Write(parser.ErrorData("ERR: Not enough arguments for the ECHO command"))
		return
	}

	for _, arg := range req.Command.Arguments {
		rw.Write(parser.BulkStringData(arg))
	}
}

func (h BaseHandler) handleSet(req Request, rw ResponseWriter) {
	if len(req.Command.Arguments) != 2 {
		rw.Write(parser.ErrorData("ERR: SET command requirees exactly 2 arguments"))
		return
	}

	optArgs, ok := req.Command.Options["PX"]
	if ok {
		if len(optArgs) != 1 {
			rw.Write(parser.ErrorData("ERR: PX parameter requires exactly 1 argument"))
			return
		}

		exp, err := strconv.Atoi(optArgs[0])
		if err != nil {
			rw.Write(parser.ErrorData("ERR: PX parameter must be integer"))
			return
		}

		err = h.storage.SetWithTimer(req.Command.Arguments[0], req.Command.Arguments[1], exp)
		if err != nil {
			rw.Write(parser.ErrorData(err.Error()))
			return
		}
	} else {
		err := h.storage.Set(req.Command.Arguments[0], req.Command.Arguments[1])
		if err != nil {
			rw.Write(parser.ErrorData(err.Error()))
			return
		}
	}

	rw.Write(parser.StringData("OK"))
}

func (h BaseHandler) handleGet(req Request, rw ResponseWriter) {
	if len(req.Command.Arguments) != 1 {
		rw.Write(parser.ErrorData("GET command requires exactly 1 argument"))
		return
	}
	val, err := h.storage.Get(req.Command.Arguments[0])
	if err != nil {
		rw.Write(parser.NullBulkStringData())
		log.Println(err.Error())
		return
	}

	rw.Write(parser.StringData(val))
}

func (h BaseHandler) handlePing(req Request, rw ResponseWriter) {
	rw.Write(parser.StringData("PONG"))
}

func (h BaseHandler) handleInfo(req Request, rw ResponseWriter) {
	var info map[string]any
	err := mapstructure.Decode(GetReplInfo(), &info)
	if err != nil {
		rw.Write(parser.ErrorData(err.Error()))
		return
	}

	var b strings.Builder
	for k, v := range info {
		b.WriteString(fmt.Sprintf("%s:%v\r\n", k, v))
	}

	str := b.String()[:len(b.String())-2]
	resStr := string(parser.BulkStringData(str).Marshal())

	io.WriteString(req.Conn, resStr)
}

func RouteMaster(server Server, mc *MasterContext) {
	handler := MasterHandler{server, mc, map[string]Replica{}}
	server.AddHandler("REPLCONF", handler.handleReplconf)
	server.AddHandler("PSYNC", handler.handlePsync)
}

func (h MasterHandler) handleReplconf(req Request, rw ResponseWriter) {
	repl, err := h.mc.GetReplica(req.Conn)
	if err != nil {
		repl = Replica{
			Conn: req.Conn,
			IsUp: false,
		}
		h.mc.SetReplica(repl)
	}

	if lp, ok := req.Command.Options["LISTENING-PORT"]; ok {
		host := strings.Split(req.Conn.RemoteAddr().String(), ":")[0]
		addr := fmt.Sprintf("%s:%s", host, lp[0])
		repl.ServerAddr = addr
	} else if capa, ok := req.Command.Options["CAPA"]; ok {
		repl.Capas = capa
	}

	h.mc.SetReplica(repl)
	rw.Write(parser.StringData("OK"))
}

func (h MasterHandler) handlePsync(req Request, rw ResponseWriter) {
	serverInfo := GetReplInfo()
	replica, err := h.mc.GetReplica(req.Conn)
	if err != nil {
		rw.Write(parser.ErrorData(err.Error()))
		return
	}

	fullresync := fmt.Sprintf("FULLRESYNC %s %d", serverInfo.ReplId, serverInfo.ReplOffset)
	rw.Write(parser.StringData(fullresync))

	rdb, err := base64.StdEncoding.DecodeString("UkVESVMwMDEx+glyZWRpcy12ZXIFNy4yLjD6CnJlZGlzLWJpdHPAQPoFY3RpbWXCbQi8ZfoIdXNlZC1tZW3CsMQQAPoIYW9mLWJhc2XAAP/wbjv+wP9aog==")
	if err != nil {
		rw.Write(parser.ErrorData("ERR: Can't decode RDB file"))
		return
	}

	io.WriteString(req.Conn, fmt.Sprintf("$%d\r\n%s", len(rdb), string(rdb)))
	replica.IsUp = true
	h.mc.SetReplica(replica)
}

func RouteReplica(sv Server, replicaContext *ReplicaContext) {
	replicaHandler := ReplicaHandler{
		replicaCtx: replicaContext,
	}
	sv.AddHandler("OK", replicaHandler.HandleOK)
	sv.AddHandler("PONG", replicaHandler.HandlePong)
	sv.AddHandler("REPLCONF", replicaHandler.HandleReplconf)
}

func (h ReplicaHandler) HandlePong(req Request, rw ResponseWriter) {
	if req.Conn != h.replicaCtx.masterConn {
		rw.Write(parser.ErrorData("ERR: Unexpected command"))
		return
	}
	h.replicaCtx.OnPong()
}

func (h ReplicaHandler) HandleOK(req Request, rw ResponseWriter) {
	if req.Conn != h.replicaCtx.masterConn {
		rw.Write(parser.ErrorData("ERR: Unexpected command"))
		return
	}

	h.replicaCtx.OnOk()
}

func (h ReplicaHandler) HandleReplconf(req Request, rw ResponseWriter) {
	if _, ok := req.Command.Options["GETACK"]; ok {
		io.WriteString(req.Conn, string(parser.ArrayData( //this is special case, as said in the docs, so we are bypassing rw
			[]parser.Data{
				parser.BulkStringData("REPLCONF"),
				parser.BulkStringData("ACK"),
				parser.BulkStringData("0"),
			},
		).Marshal()))
	}
}
