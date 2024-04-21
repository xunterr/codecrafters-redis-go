package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

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

type ReplicaContext struct {
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
	server.AddMiddleware(func(c net.Conn, req []byte) {
		mc.Propagate(req)
	}, []RequestType{Write})

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

func (mc MasterContext) GetReplica(c net.Conn) (Replica, error) { //this is so silly :3 (i hate this)
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
	log.Printf("%v", mc.replicas)
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

func RegisterReplica(sv *Server, host string, port string, listeningPort int) *ReplicaContext {
	rc := &ReplicaContext{
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
			{Name: "onStart", Src: []string{None}, Dst: Ping},
			{Name: "onPong", Src: []string{Ping}, Dst: ReplconfLP},
			{Name: "onOK", Src: []string{ReplconfLP}, Dst: ReplconfCapa},
			{Name: "onOK", Src: []string{ReplconfCapa}, Dst: Psync},
		},
		fsm.Callbacks{
			"enter_state": func(_ context.Context, e *fsm.Event) { rc.onReplHandshakeStateChange(e) },
		},
	)

	rc.handshakeFsm = fsm

	go sv.Serve(c)

	fsm.Event(context.Background(), "onStart")
	return rc
}

func (rc ReplicaContext) onReplHandshakeStateChange(e *fsm.Event) {
	switch e.Dst {
	case Ping:
		pingMaster(rc.masterConn)
	case ReplconfLP:
		setListeningPort(rc.masterConn, rc.listeningPort)
	case ReplconfCapa:
		setCapabilities(rc.masterConn)
	case Psync:
		psync(rc.masterConn)
	}
}

func (rc ReplicaContext) OnOk() {
	err := rc.handshakeFsm.Event(context.Background(), "onOK")
	if err != nil {
		log.Println("Unexpected command from master")
	}
}

func (rc ReplicaContext) OnPong() {
	err := rc.handshakeFsm.Event(context.Background(), "onPong")
	if err != nil {
		log.Println("Unexpected command from master: PONG")
	}
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
