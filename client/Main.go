package main

import (
	"net"
	"strings"
	"github.com/Spriithy/go-colors"
)

var (
	MSG_HEADER = "/M/"
	CONNECT_HEADER = "/C/"
	DISCONNECT_HEADER = "/D/"
	PING_HEADER = "/P/"
)

var ID string

func main() {
	conn, err := net.Dial("tcp", "192.168.0.10:8081")
	if err != nil {
		panic(err)
	}

	l, err := net.Listen("tcp", conn.LocalAddr().String())
	if err != nil {
		panic(err)
	}
	defer l.Close()
	conn.Write([]byte(CONNECT_HEADER + "badboy64"))
	data := make([]byte, 1024)
	for {
		conn, err = l.Accept()
		if err != nil {
			panic(err)
		}

		data = make([]byte, 1024)
		conn.Read(data)
		go process(conn, data)
	}
}

func process(conn net.Conn, data []byte) {
	content := string(data)
	println("RECEIVED: " + content)
	switch {
	case strings.HasPrefix(content, CONNECT_HEADER):
		ID = content[len(CONNECT_HEADER):]
		println("My ID is :", ID)
	}
}

func send(addr, data string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		println("[" + colors.LIGHT_RED + "error" + colors.NONE + "]", "Couldn't reach server at", colors.LIGHT_GREEN + addr + colors.NONE)
		println(strings.Repeat(" ", 7 - 1), colors.RED, err.Error(), colors.NONE)
		return
	}
	_, err = conn.Write([]byte(data))

	if err != nil {
		println("[" + colors.LIGHT_RED + "error" + colors.NONE + "]", colors.RED + "couldn't send data to server :")
		println(strings.Repeat(" ", 7), err.Error(), colors.NONE)
	}
}
