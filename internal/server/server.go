package server

import (
	"fmt"
	"io"
	"log"
	"strings"

	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
)

type HandlerFunc func(c net.Conn, cmd commands.Command)

type Server struct {
	handlers map[string]HandlerFunc
}

func NewServer() Server {
	return Server{
		handlers: map[string]HandlerFunc{},
	}
}

func (s Server) Listen(addr string) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Failed to bind to %s", addr)
		os.Exit(1)
	}

	for {
		c, err := l.Accept()
		log.Printf("Accepted: %s", c.RemoteAddr().String())
		if err != nil {
			log.Printf("Error accepting connection: %s", err.Error())
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
			msg := parser.ErrorData("Error reading request!").Marshal()
			io.WriteString(c, string(msg))
			continue
		}

		buf = buf[:size]

		parsed, err := parser.NewParser(string(buf)).Parse()
		if err != nil {
			msg := fmt.Sprintf("Error parsing RESP: %s", err.Error())
			io.WriteString(c, string(parser.ErrorData(msg).Marshal()))
			continue
		}

		command, err := commands.GetCommand(parsed.Flat())
		if err != nil {
			msg := fmt.Sprintf("Error parsing command: %s", err.Error())
			io.WriteString(c, string(parser.ErrorData(msg).Marshal()))
			continue
		}

		handler, ok := s.handlers[strings.ToUpper(command.Name)]
		if ok {
			go handler(c, command)
		}
	}
}
