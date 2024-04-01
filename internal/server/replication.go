package server

import (
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
	Conn  net.Conn
	Addr  string
	Capas []string
	IsUp  bool
}

type MasterContext struct {
	replicas []Replica
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

	mc := MasterContext{}
	go mc.HealthCheck()
	server.AddMiddleware(func(c net.Conn, req []byte) {
		mc.Propagate(req)
	})

	return &mc
}

func (mc *MasterContext) HealthCheck() {
	t := time.NewTicker(time.Second * 30)
	for range t.C {
		for i := 0; i < len(mc.replicas); i++ {
			if !mc.replicas[i].IsUp {
				c, err := net.Dial("tcp", mc.replicas[i].Addr)
				if err == nil {
					log.Printf("[HEALTHCHECK] Replica %s is UP again", mc.replicas[i].Addr)
					mc.replicas[i].Conn = c
					mc.replicas[i].IsUp = true
				} else {
					log.Println(err.Error())
					continue
				}
			}
		}
	}
}

func (mc *MasterContext) MarkAsDown(idx int, msg string) {
	log.Printf("[HEALTHCHECK] Replica %s is down: %s", mc.replicas[idx].Addr, msg)
	mc.replicas[idx].IsUp = false
}

func (mc *MasterContext) AddReplica(addr string, capas []string) {
	c, err := net.Dial("tcp", addr)
	mc.replicas = append(mc.replicas, Replica{
		Conn:  c,
		Addr:  addr,
		Capas: capas,
		IsUp:  err == nil,
	})
}

func (mc *MasterContext) Propagate(msg []byte) {
	for i, r := range mc.replicas {
		if r.IsUp {
			_, err := r.Conn.Write(msg)
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

func RegisterReplica(host string, port string, listeningPort int) {
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
	c.Close()
}

func pingMaster(c net.Conn) {
	log.Println("Ping master")
	client.Send(c, []string{"ping"})
	res, err := client.Read(c)
	if err != nil {
		log.Fatalf("Error reading server response: %s", err.Error())
	}
	if !client.Expect(res, "PONG") {
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
	if !client.Expect(res, "OK") {
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
	if !client.Expect(res, "OK") {
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
