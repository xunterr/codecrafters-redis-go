package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
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
	Offset     int
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
	connHandler *ConnectionHandler
	replicas    map[string]Replica
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
		replicas:    make(map[string]Replica),
		connHandler: server.connHandler,
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
			SetNext(func(current *Node, request Request, rw ResponseWriter) error {
				_, isReplica := mc.replicas[request.Conn.RemoteAddr().String()]
				if !isReplica {
					return ReplOffsetMW(current, request, rw)
				}
				return nil
			}).
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

func (mc MasterContext) UpdateReplicasOffset(ctx context.Context) map[string]int {
	var mu sync.Mutex
	var wg sync.WaitGroup
	offsets := make(map[string]int)
	for _, e := range mc.replicas {
		wg.Add(1)
		go func(e Replica) {
			defer wg.Done()
			offset, err := mc.FetchReplicaOffset(ctx, e)
			if err != nil {
				log.Printf("Failed to fetch offset for the replica (%s): %s", e.Conn.RemoteAddr().String(), err.Error())
				return
			}

			mu.Lock()
			offsets[e.Conn.RemoteAddr().String()] = offset
			mu.Unlock()
		}(e)
	}

	println("waiting")
	wg.Wait()
	println("waited")
	return offsets
}

func (mc MasterContext) FetchReplicaOffset(ctx context.Context, replica Replica) (int, error) {
	client.Send(replica.Conn, []string{"REPLCONF", "GETACK", "*"})
	messages := mc.connHandler.GetMessages(replica.Conn.RemoteAddr().String())
	println("here")
	select {
	case <-ctx.Done():
		return -1, errors.New("Hit a timeout while reaching replica. Check replica's health.")
	case msg, ok := <-messages:
		println("here2")
		if !ok {
			return -1, errors.New("Messages channel is closed. Check replica's health.")
		}

		offsetStr, ok := msg.Command.Options["ACK"]
		if msg.Command.Name != "REPLCONF" || !ok {
			return -1, errors.New(fmt.Sprintf("Unexpected response from the replica %s -- %s", replica.Conn.RemoteAddr().String(), string(msg.Raw)))
		}

		offset, err := strconv.Atoi(offsetStr[0])
		if err != nil {
			return -1, errors.New(fmt.Sprintf("Unexpected response from the replica %s -- %s", replica.Conn.RemoteAddr().String(), string(msg.Raw)))
		}
		return offset, nil
	}
}

func (mc MasterContext) GetReplicas() (res []Replica) {
	for _, v := range mc.replicas {
		res = append(res, v)
	}
	return
}

func (mc *MasterContext) SetReplica(replica Replica) {
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

func ReplOffsetMW(current *Node, request Request, rw ResponseWriter) error {
	if request.Command.Type != commands.Write {
		return nil
	}
	replInfo.ReplOffset += len(request.Raw)
	current.Next(request, rw)
	return nil
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
						SetNext(ReplOffsetMW).
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

	client, ctx := sv.AddClient(context.Background(), c)
	go sv.Serve(ctx, client)

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
