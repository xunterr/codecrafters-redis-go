package server

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/pkg/client"
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
	mc.replicas[replica.Conn.RemoteAddr().String()] = replica
}

func (mc *MasterContext) Propagate(req []byte) {
	for i, r := range mc.replicas {
		if r.IsUp {
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

func RegisterReplica(sv *Server, host string, port string, listeningPort int) {
	replInfo = ReplInfo{
		Role: Slave,
	}

	c, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		log.Fatalf("Error connecting to the master: %s", err.Error())
		return
	}
	pingMaster(c)
	setListeningPort(c, listeningPort)
	setCapabilities(c)
	psync(c)
	go sv.Handle(c)
}

func pingMaster(c net.Conn) {
	log.Println("Ping master")
	client.Send(c, []string{"ping"})
	res, err := client.Read(c)
	if err != nil {
		log.Fatalf("Error reading server response: %s", err.Error())
	}
	if !client.Expect(res[0], "PONG") {
		log.Fatalf("Unexpected server response: %v", res)
	}
}

func setListeningPort(c net.Conn, lp int) {
	log.Println("Set listening port")
	client.Send(c, []string{"REPLCONF", "listening-port", strconv.FormatInt(int64(lp), 10)})
	res, err := client.Read(c)
	if err != nil {
		log.Fatalf("Error reading server response: %s", err.Error())
	}
	if !client.Expect(res[0], "OK") {
		log.Fatalf("Unexpected server response: %v", res)
	}
}

func setCapabilities(c net.Conn) {
	log.Println("Set capabilities")
	client.Send(c, []string{"REPLCONF", "capa", "psync2"})
	res, err := client.Read(c)
	if err != nil {
		log.Fatalf("Error reading server response: %s", err.Error())
	}
	if !client.Expect(res[0], "OK") {
		log.Fatalf("Unexpected server response: %v", res)
	}

}

func psync(c net.Conn) {
	log.Println("Psync")
	client.Send(c, []string{"PSYNC", "?", "-1"})
	_, err := client.Read(c)
	if err != nil {
		return
	}
}
