package server

import (
	"net"
	"os"

	"fmt"

	"github.com/Spriithy/go-colors"
	uuid "github.com/Spriithy/go-uuid"
)

// fmt.Sprintf alias for code readability
var format = fmt.Sprintf

func here() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		println(colors.RedString("Error recording net interfaces :", err.Error()))
		os.Exit(1)
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "127.0.0.1"
}

// Server is the struct defining a gochat-term Server
//
type Server struct {
	name string
	ip   string
	port int

	packets chan Packet
	errors  chan error

	running bool
}

// NewServer creates a new instance of a Server struct on the given port
//
func NewServer(name string, port int) *Server {
	s := new(Server)
	s.name = name
	s.ip = here()
	s.port = port

	s.packets = make(chan Packet)
	s.errors = make(chan error)

	s.running = false

	return s
}

func (s *Server) log(a ...interface{}) {
	fmt.Print(a...)
}

func (s *Server) logln(a ...interface{}) {
	fmt.Println(a...)
}

// Start is the server's main loop and starts its own goroutine
func (s *Server) Start() {
	s.running = true

	sem := make(chan byte)

	go func() {
		// Start listening
		s.listen()
		sem <- 0
	}()

	<-sem
}

func (s *Server) listen() {
	address := format("%s:%d", s.ip, s.port)

	l, err := net.Listen("tcp", address)
	if err != nil {
		s.logln(colors.RedString("Error when listening :\n", err.Error()))
		os.Exit(1)
	}
	defer l.Close()

	s.logln("Server is now running on", colors.GreenString(address))
	var conn net.Conn
	for s.running {
		conn, err = l.Accept()
		if err != nil {
			continue
		}

		p, err := CompilePacket(conn)
		if err != nil {
			s.logln(colors.RedString("Couldn't decode incoming Packet, ignoring it."))
		}
		s.packets <- p
	}
}

func (s *Server) process() {
	var (
		p  Packet
		cp ConnectionPacket
		mp MessagePacket
	)

	p = <-s.packets
	switch p.(type) {
	case ConnectionPacket:
		cp = p.(ConnectionPacket)

		switch cp.Header() {
		case ConnectHeader:
			// TODO Add client
		case DisconnectHeader:
			// TODO Remove client
		default:
			// should never be reached
			return
		}
	case MessagePacket:
		mp = p.(MessagePacket)
		s.sendAll(MessageHeader, mp.Message())
	}
}

func (s *Server) sendAll(h PacketHeader, content string) {

}

func (s *Server) send(c *Client, h PacketHeader, context string) {

}

func (s *Server) disconnect(id uuid.UUID, reason string) {

}

func (s *Server) disconnectClient(c *Client, reason string) {

}

func (s *Server) quit() {

}
