package main

import (
	"fmt"
	"io"

	// Uncomment this block to pass the first stage
	"net"
	"os"
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")
	
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
  for{
    c, err := l.Accept()
    
    if err != nil {
      fmt.Println("Error accepting connection: ", err.Error())
      os.Exit(1)
    }
    go func(c net.Conn){
      handle(c)
      c.Close()
    }(c)
  }
}

func handle(c net.Conn){
  for {
    _, err := io.ReadAll(c)
    if err != nil{
      io.WriteString(c, "Error reading request!")
      return
    }
    io.WriteString(c, "+PONG\r\n")
  }
   
}

func process(c net.Conn, cmd string){
  switch cmd{
  case "ping":
    io.WriteString(c, "+PONG\r\n")
  }
}
