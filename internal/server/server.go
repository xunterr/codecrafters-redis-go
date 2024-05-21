package server

import (
	"context"
	"io"
	"log"
	"sync"

	"net"
)

type HandlerFunc func(req Request, rw ResponseWriter)
type NodeFunc func(current *Node, request Request, rw ResponseWriter) error

type Node struct {
	handle NodeFunc
	isEnd  bool
	next   *Node
	prev   *Node
}

type Server struct {
	handlers    map[string]HandlerFunc
	callChain   *Node
	connHandler *ConnectionHandler
	rwProvider  func(c net.Conn) ResponseWriter
	mu          sync.RWMutex
	clients     map[string]Client
	quit        chan struct{}
}

type Client struct {
	conn         net.Conn
	messages     chan Message
	stopHandling context.CancelFunc
}

type Request struct {
	Conn net.Conn
	Message
}

type ResponseWriter interface {
	Write(data []byte)
	Release() error
}

type BasicResponseWriter struct {
	conn net.Conn
	buff []byte
}

func NewBasicResponseWriter(c net.Conn) *BasicResponseWriter {
	return &BasicResponseWriter{conn: c}
}

func (rw *BasicResponseWriter) Write(data []byte) {
	rw.buff = append(rw.buff, data...)
}

func (rw BasicResponseWriter) Release() error {
	_, err := io.WriteString(rw.conn, string(rw.buff))
	return err
}

type SilentResponseWriter struct {
}

func (rw SilentResponseWriter) Write(data []byte) {
}

func (rw SilentResponseWriter) Release() error {
	return nil
}

func NewNode(nodeFunc NodeFunc) *Node {
	return &Node{
		handle: nodeFunc,
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

func (n *Node) Call(req Request, rw ResponseWriter) error {
	return n.handle(n, req, rw)
}

func (n *Node) Next(req Request, rw ResponseWriter) error {
	if n.next == nil {
		return nil
	}

	return n.next.Call(req, rw)
}

func NewServer(connHandler *ConnectionHandler) *Server {
	sv := Server{
		handlers:    map[string]HandlerFunc{},
		connHandler: connHandler,
		rwProvider: func(c net.Conn) ResponseWriter {
			return NewBasicResponseWriter(c)
		},
		clients: make(map[string]Client),
		quit:    make(chan struct{}),
	}

	sv.SetCallChain(NewNode(sv.CallHandlers))
	return &sv
}

func (s *Server) SetCallChain(first *Node) {
	s.callChain = first
}

func (s *Server) AddClient(ctx context.Context, c net.Conn) (Client, context.Context) {
	clientCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	client := Client{
		conn:         c,
		stopHandling: cancel,
		messages:     s.connHandler.InitNewConn(c),
	}
	s.clients[c.RemoteAddr().String()] = client
	s.mu.Unlock()
	return client, clientCtx
}

func (s *Server) StopHandling(c net.Conn) {
	s.mu.Lock()
	client := s.clients[c.RemoteAddr().String()]
	client.stopHandling()
	delete(s.clients, c.RemoteAddr().String())
	s.mu.Unlock()
}

func (s *Server) Listen(ctx context.Context, addr string) {
	var lc net.ListenConfig
	l, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		log.Printf("Failed to bind to %s", addr)
		return
	}

	go func() {
		<-ctx.Done()
		close(s.quit)
		log.Println("Shutting service down...")
		l.Close()
	}()

	for {
		c, err := l.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Printf("Error accepting connection: %s", err.Error())
				continue
			}
		}
		log.Printf("Accepted: %s", c.RemoteAddr().String())

		go func(c net.Conn) {
			client, clientCtx := s.AddClient(ctx, c)
			s.Serve(clientCtx, client)
		}(c)
	}
}

func (s *Server) AddHandler(name string, handler HandlerFunc) {
	s.handlers[name] = handler
}

func (s *Server) SetRwProvider(rwProvider func(c net.Conn) ResponseWriter) {
	s.rwProvider = rwProvider
}

func (s *Server) CallHandlers(current *Node, req Request, rw ResponseWriter) error {
	handler, ok := s.handlers[req.Command.Name]
	if ok {
		handler(req, rw)
		current.Next(req, rw)
	}
	return nil
}

func (s *Server) Serve(ctx context.Context, client Client) {
	go s.connHandler.Handle(context.Background(), client.conn, client.messages)
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-client.messages:
			log.Printf("[%s]: %q", client.conn.RemoteAddr().String(), msg.Raw)
			req := Request{
				Conn:    client.conn,
				Message: msg,
			}
			rw := s.rwProvider(client.conn)
			s.callChain.Call(req, rw)
			rw.Release()
		}
	}
}
