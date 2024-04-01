package server

import (
	"fmt"
	"io"
	"log"

	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
)

type HandlerFunc func(c net.Conn, cmd commands.Command)
type MiddlewareFunc func(c net.Conn, req []byte)

type Server struct {
	handlers    map[string]HandlerFunc
	middlewares []MiddlewareFunc
	cmdParser   commands.CommandParser
}

func NewServer(cmdParser commands.CommandParser) Server {
	return Server{
		handlers:    map[string]HandlerFunc{},
		middlewares: []MiddlewareFunc{}, // executes after request received and before it gets routed
		cmdParser:   cmdParser,
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
		if err != nil {
			log.Printf("Error accepting connection: %s", err.Error())
			os.Exit(1)
		}
		log.Printf("Accepted: %s", c.RemoteAddr().String())

		go func(c net.Conn) {
			s.handle(c)
			c.Close()
		}(c)
	}

}

func (s *Server) AddHandler(name string, handler HandlerFunc) {
	s.handlers[name] = handler
}

func (s *Server) AddMiddleware(mw MiddlewareFunc) {
	s.middlewares = append(s.middlewares, mw)
}

func (s Server) CallMiddlewares(c net.Conn, req []byte) { //add some datastruct to represent current state
	for _, mw := range s.middlewares {
		mw(c, req)
	}
}

func (s Server) handle(c net.Conn) {
	buf := make([]byte, 4096)
	for {
		ln, err := c.Read(buf)

		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading request: %s", err.Error())
				msg := parser.ErrorData("Error reading request!").Marshal()
				io.WriteString(c, string(msg))
			}
			break
		}
		s.CallMiddlewares(c, buf[:ln])
		p := parser.NewParser(string(buf[:ln]))
		for !p.IsAtEnd() {
			parsed, err := p.Parse()
			if err != nil {
				msg := fmt.Sprintf("Error parsing RESP: %s", err.Error())
				log.Println(msg)
				io.WriteString(c, string(parser.ErrorData(msg).Marshal()))
				continue
			}

			command, err := s.cmdParser.GetCommand(parsed.Flat())
			if err != nil {
				msg := fmt.Sprintf("Error parsing command: %s", err.Error())
				io.WriteString(c, string(parser.ErrorData(msg).Marshal()))
				continue
			}

			handler, ok := s.handlers[command.Name]
			if ok {
				go handler(c, command)
			}
		}
	}
}
