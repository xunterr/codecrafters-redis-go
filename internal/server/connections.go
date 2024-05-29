package server

import (
	"context"
	"io"
	"log"
	"net"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
)

type Message struct {
	Raw     []byte
	Command *commands.Command
}

type ConnectionHandler struct {
	cmdParser commands.CommandParser
	conns     map[string]chan Message
}

func NewConnectionHandler(cmdParser commands.CommandParser) *ConnectionHandler {
	return &ConnectionHandler{
		cmdParser: cmdParser,
		conns:     make(map[string]chan Message),
	}
}

func (ch ConnectionHandler) InitNewConn(c net.Conn) chan Message {
	messages := make(chan Message, 16)
	ch.conns[c.RemoteAddr().String()] = messages
	return messages
}

func (ch ConnectionHandler) GetMessages(remote string) chan Message {
	messages, _ := ch.conns[remote]
	return messages
}

func (ch ConnectionHandler) DeleteConn(remote string) {
	delete(ch.conns, remote)
}

func (ch ConnectionHandler) Handle(ctx context.Context, c net.Conn, messages chan Message) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			ln, err := c.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading request: %s", err.Error())
					msg := parser.ErrorData("ERR: Error reading request!").Marshal()
					io.WriteString(c, string(msg))
				}
				break
			}

			ch.forMessage(string(buf[:ln]), func(m Message, err error) {
				if err != nil {
					log.Println(err.Error())
					io.WriteString(c, string(parser.ErrorData(err.Error()).Marshal()))
				} else {
					messages <- m
				}
			})
		}
	}
}

func (ch ConnectionHandler) forMessage(from string, do func(Message, error)) {
	p := parser.NewParser(from)
	for !p.IsAtEnd() {
		parsed, err := p.Parse()
		if err != nil {
			log.Println(err.Error())
		}
		if parsed == nil {
			do(Message{}, err)
			return
		}

		command, err := ch.cmdParser.ParseCommand(parsed.Flat())
		if err != nil {
			do(Message{}, err)
			continue
		}

		do(Message{
			Raw:     []byte(parsed.Marshal()),
			Command: &command,
		}, nil)
	}
}
