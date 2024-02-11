package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"strconv"
	"strings"

	"net"
	"os"
)

var(
  storage map[string]string = make(map[string]string)
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
    case "set":
      if len(cmd) <= i+2 {
        writeMessage(c, "Not enough arguments")
        continue
      }
      
      if len(cmd) > i+4 && strings.ToLower(cmd[i+3]) == "px"{
        expire, err := strconv.Atoi(cmd[i+4])
        if err != nil{
          writeMessage(c, "Wrong time format")
          continue
        }
        err = setWithTimer(cmd[i+1], cmd[i+2], expire)
      } else {
        err := set(cmd[i+1], cmd[i+2])
        if err != nil{
          writeMessage(c, err.Error())
          continue
        }
      }
      writeMessage(c, "OK")

    case "get":
      if len(cmd) <= i+1 {
        writeMessage(c, "Not enough arguments")
        continue
      }
      v, err := get(cmd[i+1]) 
      if err != nil{
        c.Write([]byte(""))
        continue
      }
      writeMessage(c, v)
    }
  }
}

func writeMessage(c net.Conn, msg string){
  io.WriteString(c, fmt.Sprintf("+%s\r\n", msg))
}

func setWithTimer(key string, value string, expire int) error {
  err := set(key, value)
  if err != nil{
    return err
  }
  
  go func(expire int, key string){
    expiryMs := time.Millisecond * time.Duration(expire)
    timer := time.After(expiryMs)
    log.Printf("Expiry (ms): %d", expire)
    <-timer
    delete(storage, key)
  }(expire, key)
  return nil
}

func set(key string, value string) error{
  if _, ok := storage[key]; ok{
    return errors.New("Key already exists") 
  }
  storage[key] = value
  return nil
}

func get(key string) (string, error){ 
  value, ok := storage[key]  
  if !ok{
    return "", errors.New("No such key") 
  }
  return value, nil
}
