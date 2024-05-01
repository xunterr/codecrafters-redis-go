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

type HandlerFunc func(req Request, rw ResponseWriter)
type NodeFunc func(next *Node, request Request, rw ResponseWriter) error

type Node struct {
	Handle NodeFunc
	isEnd  bool
	next   *Node
	prev   *Node
}

type Server struct {
	handlers   map[string]HandlerFunc
	callChain  *Node
	cmdParser  commands.CommandParser
	rwProvider func(c net.Conn) ResponseWriter
}

type Request struct {
	Conn    net.Conn
	Raw     []byte
	Command *commands.Command
}

type ResponseWriter interface {
	Write(data parser.Data) error
}

type BasicResponseWriter struct {
	conn net.Conn
}

func (rw BasicResponseWriter) Write(data parser.Data) error {
	_, err := io.WriteString(rw.conn, string(data.Marshal()))
	return err
}

type SilentResponseWriter struct {
}

func (rw SilentResponseWriter) Write(data parser.Data) error {
	return nil
}

func NewNode(nodeFunc NodeFunc) *Node {
	return &Node{
		Handle: nodeFunc,
		next:   &Node{isEnd: true},
	}
}

func (n *Node) SetNext(nodeFunc NodeFunc) *Node {
	n.next = NewNode(nodeFunc)
	n.next.prev = n
	return n.next

}

func (n *Node) First() *Node {
	current := n
	for current.prev != nil {
		current = current.prev
	}
	return current
}

func (n *Node) Last() *Node {
	current := n
	for current.next != nil {
		current = current.next
	}
	return current
}

func (n *Node) GetArray() (arr []*Node) {
	node := n.First()
	for node != nil {
		arr = append(arr, node)
		node = node.next
	}
	return
}

func (n Node) Call(req Request, rw ResponseWriter) error {
	if n.isEnd {
		return nil
	}
	return n.Handle(n.next, req, rw)
}

func NewServer(cmdParser commands.CommandParser) Server {
	sv := Server{
		handlers:  map[string]HandlerFunc{},
		cmdParser: cmdParser,
		rwProvider: func(c net.Conn) ResponseWriter {
			return BasicResponseWriter{conn: c}
		},
	}

	sv.SetCallChain(NewNode(sv.CallHandlers))
	return sv
}

func (s *Server) SetCallChain(first *Node) {
	s.callChain = first
}

func (s *Server) Listen(addr string) {
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

func (s *Server) SetRwProvider(rwProvider func(c net.Conn) ResponseWriter) {
	s.rwProvider = rwProvider
}

func (s Server) CallHandlers(next *Node, req Request, rw ResponseWriter) error {
	handler, ok := s.handlers[req.Command.Name]
	if ok {
		handler(req, rw)
		next.Call(req, rw)
	}
	return nil
}

func (s *Server) Serve(c net.Conn) {
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
		if parsed == nil {
			msg := fmt.Sprintf("Error parsing RESP: %s", err.Error())
			log.Println(msg)
			io.WriteString(c, string(parser.ErrorData(msg).Marshal()))
			return
		}

		command, err := s.cmdParser.ParseCommand(parsed.Flat())
		if err != nil {
			msg := fmt.Sprintf("Error parsing command: %s", err.Error())
			io.WriteString(c, string(parser.ErrorData(msg).Marshal()))
			continue
		}

		req := Request{
			Conn:    c,
			Raw:     parsed.Marshal(),
			Command: &command,
		}
		rw := s.rwProvider(c)
		s.callChain.Call(req, rw)
	}
}
