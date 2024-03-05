package main

import (
	"fmt"
	"io"
	"net"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/server"
	"github.com/codecrafters-io/redis-starter-go/internal/storage"
)

func main() {
	server := server.NewServer(storage.NewStorage())
	server.AddHandler("ECHO", func(c net.Conn, cmd commands.Command) {
		if len(cmd.Arguments) < 1 {
			io.WriteString(c, "Error: Not enough arguments for the ECHO command\n")
			return
		}
		for _, arg := range cmd.Arguments {
			io.WriteString(c, fmt.Sprintln(arg))
		}
	})
	server.Listen(":6969")
}
