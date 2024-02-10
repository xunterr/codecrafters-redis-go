package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	// Uncomment this block to pass the first stage
	"net"
	"os"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
  
  for{
    c, err := l.Accept()
    fmt.Println("Accepted")
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
    var buf []byte = make([]byte, 1024)
    _, err := c.Read(buf) 

    if err != nil{
      io.WriteString(c, "Error reading request!")
      return
    }
    
    resp := parseResp(buf)
    handleCmd(c, resp)
  }
   
}

func parseResp(req []byte) []string{
  args := strings.Split(string(req), "\r\n")
  length := 1
  values := make([]string, length)

  for i := 0; i<len(args); i++{
    tokens := strings.Split(args[i], "")
    format := tokens[0]
    data := strings.Join(tokens[1:], "")

    switch format{
    case "*":
      length, _ = strconv.Atoi(data)
      values = make([]string, length)
      i--
    case "$":
      str := args[i+1]
      ln, _ := strconv.Atoi(data)
      values[i] = str[:ln]
    }
  }
  return values
}

func handleCmd(c net.Conn, cmd []string){
  for i := 0; i<len(cmd); i++{
    switch strings.ToLower(cmd[i]){
    case "ping":
      io.WriteString(c, "+PONG\r\n")
    case "echo":
      io.WriteString(c, fmt.Sprintf("+%s\r\n"))
    }
  }
}
