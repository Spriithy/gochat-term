package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"

	"github.com/Spriithy/go-colors"
	"github.com/Spriithy/gochat-term/network"
)

var (
	clear = func() {
		print("\033[H\033[2J")
	}

	MSG_HEADER        = "\\M\\"
	CONNECT_HEADER    = "\\C\\"
	DISCONNECT_HEADER = "\\D\\"
)

var ID string

func regSplit(text string, delimeter string) []string {
	reg := regexp.MustCompile(delimeter)
	indexes := reg.FindAllStringIndex(text, -1)
	laststart := 0
	result := make([]string, len(indexes)+1)
	for i, element := range indexes {
		result[i] = text[laststart:element[0]]
		laststart = element[1]
	}
	result[len(indexes)] = text[laststart:]
	return result
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter username: ")
	text, _ := reader.ReadString('\n')
	name := regSplit(text[:len(text)-1], "[ \t\r\n]+")[0]
	clear()

	conn, err := net.Dial("tcp", "127.0.0.1:8081")
	if err != nil {
		panic(err)
	}

	l, err := net.Listen("tcp", conn.LocalAddr().String())
	if err != nil {
		panic(err)
	}
	defer l.Close()
	conn.Write([]byte(CONNECT_HEADER + name))
	data := make([]byte, 1024)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		conn, err := net.Dial("tcp", "127.0.0.1:8081")
		if err != nil {
			panic(err)
		}
		for {
			for range c {
				_, err := conn.Write([]byte(DISCONNECT_HEADER + ID + "/leave"))
				if err != nil {
					panic(err)
				}
				println()
				os.Exit(1)
			}
		}
	}()

	for {
		conn, err = l.Accept()
		if err != nil {
			panic(err)
		}

		data = make([]byte, network.MaxPacketSize)
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
		println("["+colors.RED+"error"+colors.NONE+"]", "Couldn't reach server at", colors.GREEN+addr+colors.NONE)
		println(strings.Repeat(" ", 7-1), colors.RED, err.Error(), colors.NONE)
		return
	}
	_, err = conn.Write([]byte("\\C\\" + network.GetTimeStamp().String() + data))

	if err != nil {
		println("["+colors.RED+"error"+colors.NONE+"]", colors.RED+"couldn't send data to server :")
		println(strings.Repeat(" ", 7), err.Error(), colors.NONE)
	}
}
