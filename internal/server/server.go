package server

import (
	"fmt"
	"io"
	"log"

	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/storage"
	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
)

type HandlerFunc func(c net.Conn, cmd commands.Command)

type Server struct {
	storage  *storage.Storage
	handlers map[string]HandlerFunc
}

func NewServer(storage *storage.Storage) Server {
	return Server{
		storage:  storage,
		handlers: map[string]HandlerFunc{},
	}
}

func (s Server) Listen(addr string) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Failed to bind to %s\n", addr)
		os.Exit(1)
	}

	for {
		c, err := l.Accept()
		log.Printf("Accepted: %s", c.RemoteAddr().String())
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go func(c net.Conn) {
			s.handle(c)
			c.Close()
		}(c)
	}

}

func (s *Server) AddHandler(name string, handler HandlerFunc) {
	s.handlers[name] = handler
}

func (s Server) handle(c net.Conn) {
	for {
		var buf []byte = make([]byte, 1024)
		size, err := c.Read(buf)

		if err != nil {
			io.WriteString(c, "Error reading request!\n")
			continue
		}

		buf = buf[:size]

		parsed, err := parser.NewParser(string(buf)).Parse()
		if err != nil {
			io.WriteString(c, fmt.Sprintf("Error parsing RESP: %s\n", err.Error()))
			continue
		}

		command, err := commands.GetCommand(parsed.Flat())
		if err != nil {
			io.WriteString(c, fmt.Sprintf("Error parsing command: %s\n", err.Error()))
			continue
		}

		handler, ok := s.handlers[command.Name]
		if ok {
			go handler(c, command)
		}
	}
}
