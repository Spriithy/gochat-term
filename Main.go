package main

import "github.com/Spriithy/gochat-term/server"

func main() {
	serv := server.NewServer("ChatRoom", 8081)
	serv.Start()
}
