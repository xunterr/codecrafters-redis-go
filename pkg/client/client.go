package client

import (
	"log"
	"net"

	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
)

func Expect(res []string, str string) bool {
	return len(res) != 0 && res[0] == str
}

func Send(c net.Conn, cmd []string) error {
	var msg []parser.Data
	for _, e := range cmd {
		msg = append(msg, parser.BulkStringData(e))
	}

	_, err := c.Write(parser.ArrayData(msg).Marshal())
	if err != nil {
		return err
	}
	return nil
}

func Read(c net.Conn) ([]string, error) {
	var res []string
	buff := make([]byte, 1024)
	ln, err := c.Read(buff)
	log.Printf("Received data: %s", string(buff[:ln]))
	if err != nil {
		return nil, err
	}

	p := parser.NewParser(string(buff[:ln]))
	for !p.IsAtEnd() {
		parsed, err := p.Parse()
		if err != nil {
			return nil, err
		}
		res = append(res, parsed.Flat()...)
	}
	return res, err
}
