package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/storage"
	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
	"github.com/mitchellh/mapstructure"
)

type BaseHandler struct {
	storage *storage.Storage
	server  *Server
}
type MasterHandler struct {
	server   *Server
	mc       *MasterContext
	sessions map[string]Replica
}

type ReplicaHandler struct {
	replicaCtx *ReplicaContext
}

func RouteBasic(server *Server, storage *storage.Storage) {
	handler := BaseHandler{storage: storage, server: server}
	server.AddHandler("ECHO", handler.handleEcho)
	server.AddHandler("SET", handler.handleSet)
	server.AddHandler("GET", handler.handleGet)
	server.AddHandler("PING", handler.handlePing)
	server.AddHandler("INFO", handler.handleInfo)
}

func (h BaseHandler) handleEcho(req Request, rw ResponseWriter) {
	if len(req.Command.Arguments) < 1 {
		rw.Write(parser.ErrorData("ERR: Not enough arguments for the ECHO command").Marshal())
		return
	}

	for _, arg := range req.Command.Arguments {
		rw.Write(parser.BulkStringData(arg).Marshal())
	}
}

func (h BaseHandler) handleSet(req Request, rw ResponseWriter) {
	if len(req.Command.Arguments) != 2 {
		rw.Write(parser.ErrorData("ERR: SET command requirees exactly 2 arguments").Marshal())
		return
	}

	optArgs, ok := req.Command.Options["PX"]
	if ok {
		if len(optArgs) != 1 {
			rw.Write(parser.ErrorData("ERR: PX parameter requires exactly 1 argument").Marshal())
			return
		}

		exp, err := strconv.Atoi(optArgs[0])
		if err != nil {
			rw.Write(parser.ErrorData("ERR: PX parameter must be integer").Marshal())
			return
		}

		err = h.storage.SetWithTimer(req.Command.Arguments[0], req.Command.Arguments[1], exp)
		if err != nil {
			rw.Write(parser.ErrorData(err.Error()).Marshal())
			return
		}
	} else {
		err := h.storage.Set(req.Command.Arguments[0], req.Command.Arguments[1])
		if err != nil {
			rw.Write(parser.ErrorData(err.Error()).Marshal())
			return
		}
	}

	rw.Write(parser.StringData("OK").Marshal())
}

func (h BaseHandler) handleGet(req Request, rw ResponseWriter) {
	if len(req.Command.Arguments) != 1 {
		rw.Write(parser.ErrorData("GET command requires exactly 1 argument").Marshal())
		return
	}
	val, err := h.storage.Get(req.Command.Arguments[0])
	if err != nil {
		rw.Write(parser.NullBulkStringData().Marshal())
		log.Println(err.Error())
		return
	}

	rw.Write(parser.StringData(val).Marshal())
}

func (h BaseHandler) handlePing(req Request, rw ResponseWriter) {
	rw.Write(parser.StringData("PONG").Marshal())
}

func (h BaseHandler) handleInfo(req Request, rw ResponseWriter) {
	var info map[string]any
	err := mapstructure.Decode(GetReplInfo(), &info)
	if err != nil {
		rw.Write(parser.ErrorData(err.Error()).Marshal())
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

func RouteMaster(server *Server, mc *MasterContext) {
	handler := MasterHandler{server, mc, map[string]Replica{}}
	server.AddHandler("REPLCONF", handler.handleReplconf)
	server.AddHandler("PSYNC", handler.handlePsync)
	server.AddHandler("WAIT", handler.handleWait)
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
		h.mc.SetReplica(repl)
		rw.Write(parser.StringData("OK").Marshal())
	} else if capa, ok := req.Command.Options["CAPA"]; ok {
		repl.Capas = capa
		h.mc.SetReplica(repl)
		rw.Write(parser.StringData("OK").Marshal())
	} else if offsetStr, ok := req.Command.Options["ACK"]; ok {
		offset, err := strconv.Atoi(offsetStr[0])
		if err != nil {
			return
		}
		repl.Offset = offset
		h.mc.SetReplica(repl)
		return
	}
}

func (h MasterHandler) handlePsync(req Request, rw ResponseWriter) {
	serverInfo := GetReplInfo()
	replica, err := h.mc.GetReplica(req.Conn)
	if err != nil {
		rw.Write(parser.ErrorData(err.Error()).Marshal())
		return
	}

	fullresync := fmt.Sprintf("FULLRESYNC %s %d", serverInfo.ReplId, serverInfo.ReplOffset)
	rw.Write(parser.StringData(fullresync).Marshal())

	rdb, err := base64.StdEncoding.DecodeString("UkVESVMwMDEx+glyZWRpcy12ZXIFNy4yLjD6CnJlZGlzLWJpdHPAQPoFY3RpbWXCbQi8ZfoIdXNlZC1tZW3CsMQQAPoIYW9mLWJhc2XAAP/wbjv+wP9aog==")
	if err != nil {
		rw.Write(parser.ErrorData("ERR: Can't decode RDB file").Marshal())
		return
	}

	rw.Write([]byte(fmt.Sprintf("$%d\r\n%s", len(rdb), string(rdb))))
	replica.IsUp = true
	h.mc.SetReplica(replica)

	h.server.StopHandling(replica.Conn) //exit handling loop, handshake is ended - no more commands expected. WIP
}

func (h MasterHandler) handleWait(req Request, rw ResponseWriter) {

	if len(req.Command.Arguments) != 2 {
		rw.Write(parser.ErrorData("ERR: Wrong command signature - requires only 2 arguments").Marshal())
		return
	}

	replNum, err := strconv.Atoi(req.Command.Arguments[0])
	if err != nil {
		rw.Write(parser.ErrorData("ERR: Wrong command signature - argument #2 should be an integer").Marshal())
		return
	}

	duration, err := strconv.Atoi(req.Command.Arguments[1])
	if err != nil {
		rw.Write(parser.ErrorData("ERR: Wrong command signature - argument #2 should be an integer").Marshal())
		return
	}

	if GetReplInfo().ReplOffset == 0 {
		log.Println("No previous write commands, skipping")
		rw.Write(parser.IntegerData(len(h.mc.GetReplicas())).Marshal())
		return
	}
	replicasDone := 0
	ping := time.NewTicker(100 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(duration)*time.Millisecond)
	defer cancel()

	for range ping.C {
		select {
		case <-ctx.Done():
			println("hereeesfdojsld")
			rw.Write(parser.IntegerData(replicasDone).Marshal())
			return
		default:
			offsets := h.mc.UpdateReplicasOffset(ctx)
			offsetsMatch := 0

			for _, offset := range offsets {
				if offset >= replInfo.ReplOffset {
					offsetsMatch++
				}
			}
			replicasDone = offsetsMatch
			if offsetsMatch == replNum {
				rw.Write(parser.IntegerData(offsetsMatch).Marshal())
				return
			}
		}

	}
	rw.Write(parser.NullBulkStringData().Marshal())
}

func RouteReplica(sv *Server, replicaContext *ReplicaContext) {
	replicaHandler := ReplicaHandler{
		replicaCtx: replicaContext,
	}
	sv.AddHandler("OK", replicaHandler.HandleOK)
	sv.AddHandler("PONG", replicaHandler.HandlePong)
	sv.AddHandler("REPLCONF", replicaHandler.HandleReplconf)
	sv.AddHandler("FULLRESYNC", replicaHandler.HandleFsync)
}

func (h ReplicaHandler) HandlePong(req Request, rw ResponseWriter) {
	if req.Conn != h.replicaCtx.masterConn {
		rw.Write(parser.ErrorData("ERR: Unexpected command").Marshal())
		return
	}
	h.replicaCtx.Event(OnPong)
}

func (h ReplicaHandler) HandleOK(req Request, rw ResponseWriter) {
	if req.Conn != h.replicaCtx.masterConn {
		rw.Write(parser.ErrorData("ERR: Unexpected command").Marshal())
		return
	}

	h.replicaCtx.Event(OnOk)
}

func (h ReplicaHandler) HandleReplconf(req Request, rw ResponseWriter) {
	if _, ok := req.Command.Options["GETACK"]; ok {
		io.WriteString(req.Conn, string(parser.ArrayData( //this is special case, as said in the docs, so we are bypassing rw
			[]parser.Data{
				parser.BulkStringData("REPLCONF"),
				parser.BulkStringData("ACK"),
				parser.BulkStringData(strconv.Itoa(GetReplInfo().ReplOffset)),
			},
		).Marshal()))
	}
}

func (h ReplicaHandler) HandleFsync(req Request, rw ResponseWriter) {
	UpdateReplInfo("", 0)
	h.replicaCtx.Event(OnFsync)
}
