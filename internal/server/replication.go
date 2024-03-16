package server

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
)

func RegisterReplica(host string, port string, listeningPort int) {
	c, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		log.Fatalf("Error connecting master: %s", err.Error())
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
	send(c, []string{"ping"})
	res, err := read(c)
	if err != nil {
		log.Fatalf("Error reading server response: %s", err.Error())
	}
	if len(res) == 0 || res[0] != "PONG" {
		log.Fatalf("Unexpected server response: %v", res)
	}
}

func setListeningPort(c net.Conn, lp int) {
	log.Println("Set listening port")
	send(c, []string{"REPLCONF", "listening-port", strconv.FormatInt(int64(lp), 10)})
	_, err := read(c)
	if err != nil {
		log.Fatalf("Error reading server response: %s", err.Error())
	}
}

func setCapabilities(c net.Conn) {
	log.Println("Set capabilities")
	send(c, []string{"REPLCONF", "capa", "psync2"})
	_, err := read(c)
	if err != nil {
		log.Fatalf("Error reading server response: %s", err.Error())
	}
}

func psync(c net.Conn) {
	log.Println("Psync")
	send(c, []string{"PSYNC", "?", "-1"})
	_, err := read(c)
	if err != nil {
		log.Fatalf("Error reading server response: %s", err.Error())
	}
}

func send(c net.Conn, cmd []string) {
	var msg []parser.Data
	for _, e := range cmd {
		msg = append(msg, parser.BulkStringData(e))
	}

	_, err := c.Write(parser.ArrayData(msg).Marshal())
	if err != nil {
		log.Fatal(err.Error())
	}
}

func read(c net.Conn) ([]string, error) {
	buff := make([]byte, 1024)
	ln, err := c.Read(buff)
	if err != nil {
		return nil, err
	}

	parsed, err := parser.NewParser(string(buff[:ln])).Parse()
	if err != nil {
		return nil, err
	}

	res := parsed.Flat()
	return res, err
}
