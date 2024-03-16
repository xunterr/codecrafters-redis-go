package server

import (
	"fmt"
	"io"
	"log"

	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
	"github.com/mitchellh/mapstructure"
)

type HandlerFunc func(c net.Conn, cmd commands.Command)

type Server struct {
	handlers  map[string]HandlerFunc
	cmdParser commands.CommandParser
	severInfo ServerInfo
}

type ServerRole string

const (
	Master ServerRole = "master"
	Slave  ServerRole = "slave"
)

type ServerInfo struct {
	Role       ServerRole `mapstructure:"role"`
	ReplId     string     `mapstructure:"master_replid"`
	ReplOffset int        `mapstructure:"master_repl_offset"`
}

func (si ServerInfo) ToMap() (res map[string]string, err error) {
	err = mapstructure.Decode(si, &res)
	return
}

func NewServer(cmdParser commands.CommandParser, role ServerRole) Server {
	return Server{
		handlers:  map[string]HandlerFunc{},
		cmdParser: cmdParser,
		severInfo: ServerInfo{
			Role:       role,
			ReplId:     "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb",
			ReplOffset: 0,
		},
	}
}

func (s Server) GetServerInfo() ServerInfo {
	return s.severInfo
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

		parsed, err := parser.NewParser(string(buf[:ln])).Parse()
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
