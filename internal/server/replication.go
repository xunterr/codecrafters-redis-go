package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/pkg/client"
	"github.com/looplab/fsm"
)

type ReplInfo struct {
	Role       ServerRole `mapstructure:"role"`
	ReplId     string     `mapstructure:"master_replid"`
	ReplOffset int        `mapstructure:"master_repl_offset"`
}

var replInfo ReplInfo

type ServerRole string

type Replica struct {
	Conn       net.Conn
	ServerAddr string
	Capas      []string
	IsUp       bool
}

const (
	None         = "none"
	Ping         = "ping"
	ReplconfLP   = "replconfLP"
	ReplconfCapa = "replconfCapa"
	Psync        = "psync"
	Done         = "done"
)

const (
	OnStart = "onStart"
	OnPong  = "onPong"
	OnOk    = "onOk"
	OnFsync = "onFsync"
)

type ReplicaContext struct {
	server        *Server
	listeningPort int
	masterConn    net.Conn
	handshakeFsm  *fsm.FSM
}

type MasterContext struct {
	replicas map[string]Replica
}

const (
	Master ServerRole = "master"
	Slave  ServerRole = "slave"
)

func SetAsMaster(server *Server) *MasterContext {
	replInfo = ReplInfo{
		Role:       Master,
		ReplId:     "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb",
		ReplOffset: 0,
	}

	mc := MasterContext{
		make(map[string]Replica),
	}
	go mc.HealthCheck()

	server.SetCallChain(
		NewNode(func(current *Node, request Request, rw ResponseWriter) error {
			if request.Command.Type == commands.Write {
				mc.Propagate(request.Raw)
			}
			current.Next(request, rw)
			return nil
		}).
			SetNext(server.CallHandlers).
			First(),
	)
	return &mc
}

func (mc *MasterContext) HealthCheck() {
	t := time.NewTicker(time.Second * 30)
	for range t.C {
		for k := range mc.replicas {
			if !mc.replicas[k].IsUp {
				repl := mc.replicas[k]
				c, err := net.Dial("tcp", repl.Conn.RemoteAddr().String())
				if err == nil {
					log.Printf("[HEALTHCHECK] Replica %s is UP again", k)
					repl.Conn = c
					repl.IsUp = true
					mc.replicas[k] = repl
				} else {
					log.Println(err.Error())
					continue
				}
			}
		}
	}
}

func (mc *MasterContext) MarkAsDown(addr string, msg string) {
	log.Printf("[HEALTHCHECK] Replica %s is down: %s", addr, msg)
	repl := mc.replicas[addr]
	repl.IsUp = false
	mc.replicas[addr] = repl
}

func (mc MasterContext) GetReplica(c net.Conn) (Replica, error) {
	repl, ok := mc.replicas[c.RemoteAddr().String()]
	if !ok {
		return Replica{}, errors.New("No such replica")
	}
	return repl, nil
}

func (mc *MasterContext) SetReplica(replica Replica) {
	log.Println("Configuring replica..")
	mc.replicas[replica.Conn.RemoteAddr().String()] = replica
}

func (mc *MasterContext) Propagate(req []byte) {
	log.Printf("Propagating to %d replicas", len(mc.replicas))
	for i, r := range mc.replicas {
		if r.IsUp {
			log.Printf("Propagating to %s", r.Conn.RemoteAddr())
			_, err := r.Conn.Write(req)
			if err != nil {
				mc.MarkAsDown(i, "Error writing to the connection")
				continue
			}
		}
	}
}

func GetReplInfo() ReplInfo {
	return replInfo
}

func UpdateReplInfo(replId string, replOffset int) {
	replInfo.ReplId = replId
	replInfo.ReplOffset = replOffset
}

func RegisterReplica(sv *Server, host string, port string, listeningPort int) *ReplicaContext {
	rc := &ReplicaContext{
		server:        sv,
		listeningPort: listeningPort,
	}

	replInfo = ReplInfo{
		Role: Slave,
	}

	c, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		log.Fatalf("Error connecting to the master: %s", err.Error())
		return nil
	}
	rc.masterConn = c

	fsm := fsm.NewFSM(
		None,
		fsm.Events{
			{Name: OnStart, Src: []string{None}, Dst: Ping},
			{Name: OnPong, Src: []string{Ping}, Dst: ReplconfLP},
			{Name: OnOk, Src: []string{ReplconfLP}, Dst: ReplconfCapa},
			{Name: OnOk, Src: []string{ReplconfCapa}, Dst: Psync},
			{Name: OnFsync, Src: []string{Psync}, Dst: Done},
		},
		fsm.Callbacks{
			Ping:         func(_ context.Context, e *fsm.Event) { pingMaster(rc.masterConn) },
			ReplconfLP:   func(_ context.Context, e *fsm.Event) { setListeningPort(rc.masterConn, listeningPort) },
			ReplconfCapa: func(_ context.Context, e *fsm.Event) { setCapabilities(rc.masterConn) },
			Psync:        func(_ context.Context, e *fsm.Event) { psync(rc.masterConn) },
			Done: func(ctx context.Context, e *fsm.Event) {
				sv.SetCallChain(
					NewNode(sv.CallHandlers).
						SetNext(func(current *Node, request Request, rw ResponseWriter) error {
							replInfo.ReplOffset += len(request.Raw)
							current.Next(request, rw)
							return nil
						}).
						First(),
				)
			},
		})
	rc.handshakeFsm = fsm

	sv.SetRwProvider(func(c net.Conn) ResponseWriter {
		if c == rc.masterConn {
			return SilentResponseWriter{}
		} else {
			return NewBasicResponseWriter(c)
		}
	})
	go func(sv *Server, c net.Conn) {
		sv.Serve(c)
	}(sv, c)

	fsm.Event(context.Background(), OnStart)
	return rc
}

func (rc ReplicaContext) onHandshakeEnd(server *Server) {
}

func (rc ReplicaContext) Event(name string) error {
	return rc.handshakeFsm.Event(context.Background(), name)
}

func pingMaster(c net.Conn) {
	log.Println("Ping master")
	client.Send(c, []string{"ping"})
}

func setListeningPort(c net.Conn, lp int) {
	log.Println("Set listening port")
	client.Send(c, []string{"REPLCONF", "listening-port", strconv.FormatInt(int64(lp), 10)})
}

func setCapabilities(c net.Conn) {
	log.Println("Set capabilities")
	client.Send(c, []string{"REPLCONF", "capa", "psync2"})
}

func psync(c net.Conn) {
	log.Println("Psync")
	client.Send(c, []string{"PSYNC", "?", "-1"})
}
