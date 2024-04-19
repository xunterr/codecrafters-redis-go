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

type middleware struct {
	middlewareFunc MiddlewareFunc
	requestTypes   map[RequestType]struct{}
}

type Server struct {
	handlers    map[string]HandlerFunc
	middlewares []middleware
	cmdParser   commands.CommandParser
}

type RequestType int

const (
	Any RequestType = iota
	Write
	Read
	Info
)

func GetRequestType(cmd commands.Command) RequestType {
	switch cmd.Type {
	case commands.ReadCommand:
		return Read
	case commands.InfoCommand:
		return Info
	case commands.WriteCommand:
		return Write
	default:
		return Any
	}
}

func NewServer(cmdParser commands.CommandParser) Server {
	return Server{
		handlers:    map[string]HandlerFunc{},
		middlewares: []middleware{}, // executes after request received and before it gets routed
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
			s.Serve(c)
			c.Close()
		}(c)
	}

}

func (s *Server) AddHandler(name string, handler HandlerFunc) {
	s.handlers[name] = handler
}

func (s *Server) AddMiddleware(mf MiddlewareFunc, rqTypes []RequestType) {
	rqTypesMap := make(map[RequestType]struct{})
	for _, e := range rqTypes {
		rqTypesMap[e] = struct{}{}
	}
	s.middlewares = append(s.middlewares, middleware{mf, rqTypesMap})
}

func (s Server) CallMiddlewares(c net.Conn, req []byte, rqType RequestType) { //add some datastruct to represent current state
	for _, mw := range s.middlewares {
		_, isAnyType := mw.requestTypes[rqType]
		if _, ok := mw.requestTypes[rqType]; ok || isAnyType {
			mw.middlewareFunc(c, req)
		}
	}
}

func (s Server) Serve(c net.Conn) {
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
		log.Printf("[%s]: %q", c.RemoteAddr().String(), string(buf[:ln]))
		go s.route(c, string(buf[:ln]))
	}
}

func (s *Server) route(c net.Conn, input string) {
	p := parser.NewParser(input)
	for !p.IsAtEnd() {
		parsed, err := p.Parse()
		log.Printf("%v", parsed)
		if err != nil {
			log.Println(err.Error())
		}
		if parsed == nil {
			msg := fmt.Sprintf("Error parsing RESP: %s", err.Error())
			log.Println(msg)
			io.WriteString(c, string(parser.ErrorData(msg).Marshal()))
			continue
		}

		command, err := s.cmdParser.ParseCommand(parsed.Flat())
		if err != nil {
			msg := fmt.Sprintf("Error parsing command: %s", err.Error())
			io.WriteString(c, string(parser.ErrorData(msg).Marshal()))
			continue
		}

		s.CallMiddlewares(c, parsed.Marshal(), GetRequestType(command))

		handler, ok := s.handlers[command.Name]
		if ok {
			handler(c, command)
		}
	}
}
