package server

import (
	"github.com/Spriithy/go-colors"
	"net"
	"os"
	"fmt"
	"github.com/Spriithy/go-uuid"
	"strings"
	"time"
)

var (
	clear = func() {
		print("\033[H\033[2J")
	}

	maxAttempts = 5

	SERVER_HEADER = func(s *Server) string {
		return "[" + colors.LIGHT_BLUE + s.name + colors.NONE + "]"
	}

	MSG_HEADER = "/M/"
	CONNECT_HEADER = "/C/"
	DISCONNECT_HEADER = "/D/"
	PING_HEADER = "/P/"
	END_OF_DATA = "\r\n"
)

type Server struct {
	// Server Infos
	name      string

	// Network infos
	port      int
	addr      string

	clients   map[uuid.UUID]*SClient
	responses map[uuid.UUID]int

	running   bool
}

func local() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		println(colors.RED + "Error recording net interfaces :", err.Error(), colors.NONE)
		os.Exit(1)
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			println(colors.RED + "Error reading net interface address :", err.Error(), colors.NONE)
			os.Exit(1)
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			return ip.String()
		}
	}
	return "127.0.0.1"
}

func formatAddress(addr string, port int) string {
	return fmt.Sprintf("%s:%d", addr, port)
}

func (s *Server) Start(port int) {
	clear()

	s.addr = local()
	s.port = port
	address := formatAddress(s.addr, s.port)

	l, err := net.Listen("tcp", address)
	if err != nil {
		println(SERVER_HEADER(s), colors.RED + "Error when listening :", err.Error(), colors.NONE)
		os.Exit(1)
	}
	defer l.Close()

	println(SERVER_HEADER(s), "is now running on", colors.LIGHT_GREEN + address + colors.NONE)
	s.running = true
	s.run()
}

func (s *Server) run() {
	go s.pingAll()
}

func (s *Server) pingAll() {
	for s.running {
		s.sendAll(PING_HEADER + s.name)
		time.Sleep(time.Second * 2)
		for _, c := range s.clients {
			if _, ok := s.responses[c.id]; ok {
				if c.attempt >= maxAttempts {
					s.disconnect(c.id, false)
				} else {
					c.attempt++
				}
			} else {
				delete(s.responses, c.id)
				c.attempt = 0
			}
		}
	}
}

func (s *Server) sendAll(data string) {
	if data[:3] == MSG_HEADER {
		message := strings.Split(data[3:], END_OF_DATA)[0]
		println("[" + colors.LIGHT_RED + "message" + colors.NONE + "]", message)
	}

	for _, c := range s.clients {
		s.send(c, data)
	}
}

func (s *Server) send(c *SClient, data string) {
	ca := formatAddress(c.addr, c.port)
	conn, err := net.Dial("tcp", ca)
	if err != nil {
		println(SERVER_HEADER(s), "couldn't connect to client :", colors.LIGHT_BLUE + c.name + colors.NONE + "@" + colors.LIGHT_GREEN + ca + colors.NONE)
		println(strings.Repeat(" ", len(SERVER_HEADER(s)) - 1), colors.RED, err.Error(), colors.NONE)
	}
	_, err = conn.Write([]byte(data + END_OF_DATA))

	if err != nil {
		println(SERVER_HEADER(s), "couldn't send data to client :", )
		println(strings.Repeat(" ", len(SERVER_HEADER(s)) - 1), colors.RED, err.Error(), colors.NONE)
	}
}

func (s *Server) disconnect(id uuid.UUID, status bool) {
	for _, k := range s.clients {
		if id.Match(k.id) {
			s.disconnectClient(k, status)
			return
		}
	}
}

func (s *Server) disconnectClient(c *SClient, status bool) {
	ca := formatAddress(c.addr, c.port)
	s.clients[c.id] = nil
	if status {
		println("Client", colors.LIGHT_BLUE + c.name + colors.NONE + "@" + colors.LIGHT_GREEN + ca + colors.NONE + " disconnected.")
	} else {
		println("Client", colors.LIGHT_BLUE + c.name + colors.NONE + "@" + colors.LIGHT_GREEN + ca + colors.NONE + " timed out.")
	}
}

func (s *Server) quit() {
	for _, k := range s.clients {
		s.disconnectClient(k, true)
	}
	s.running = false
	println(SERVER_HEADER(s), " shut down.")
	os.Exit(1)
}