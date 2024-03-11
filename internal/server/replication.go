package server

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
)

func PingMaster(host string, port string) {
	c, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))

	if err != nil {
		log.Fatalf("Error pinging master: %s", err.Error())
		return
	}

	msg := parser.ArrayData([]parser.Data{parser.BulkStringData("ping")})
	io.WriteString(c, string(msg.Marshal()))
}
