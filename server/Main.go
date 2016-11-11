package main

import "github.com/Spriithy/gochat-term/server/src"

func main() {
	s := server.NewServer("ChatRoom")
	s.Start(8081)
}
