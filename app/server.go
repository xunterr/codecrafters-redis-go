package main

import (
	"fmt"
	"io"
	
	"strconv"
	"strings"

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
    size, err := c.Read(buf) 

    if err != nil{
      io.WriteString(c, "Error reading request!")
      return
    }

    buf = buf[:size]
    resp := parseResp(buf)
    handleCmd(c, resp)
  }
   
}

func parseResp(req []byte) []string{
  strReq := strings.TrimSpace(string(req))
  args := strings.Split(strReq, "\r\n")
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
    case "$":
      str := args[i+1]
      ln, _ := strconv.Atoi(data)
      values = append(values, str[:ln]) 
      i++
    default:
      values = append(values, args[i])
    }
  }
  return values
}

func handleCmd(c net.Conn, cmd []string){
  for i := 0; i<len(cmd); i++{
    switch strings.ToLower(strings.TrimSpace(cmd[i])){
    case "ping":
      io.WriteString(c, "+PONG\r\n")
    case "echo":
      io.WriteString(c, fmt.Sprintf("+%s\r\n", cmd[i+1]))
    }
  }
}
