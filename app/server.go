package main

import (
	"fmt"
	"io"
	"log"

	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/storage"
)

type Server struct {
	storage *storage.Storage
}

func NewServer(storage *storage.Storage) Server {
	return Server{storage}
}

func main() {
}

func (s Server) Listen(addr string) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Failed to bind to %s\n", addr)
		os.Exit(1)
	}

	for {
		c, err := l.Accept()
		fmt.Println("Accepted")
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go func(c net.Conn) {
			handle(c)
			c.Close()
		}(c)
	}

}

func handle(c net.Conn) {
	for {
		var buf []byte = make([]byte, 1024)
		size, err := c.Read(buf)

		if err != nil {
			io.WriteString(c, "Error reading request!")
			return
		}

		buf = buf[:size]

		//tokens := parser.NewScanner(string(buf)).ScanTokens()
		//parsed := parser.NewParser(tokens).Parse()
	}

}
