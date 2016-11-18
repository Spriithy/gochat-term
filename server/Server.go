package server

import (
	"net"
	"os"

	"fmt"

	"github.com/Spriithy/go-colors"
	"github.com/Spriithy/go-uuid"
	"github.com/Spriithy/gochat-term/network"
	"github.com/Spriithy/gochat-term/server/client"
)

var bold = colors.Bold
var none = colors.None

// fmt.Sprintf alias for code readability
var format = fmt.Sprintf

// Here Returns the local address
func Here() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		println(colors.Red(bold, "Error recording net interfaces :", err.Error()))
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

// Server is a basic Server wrapper interface
//
type Server interface {
	Log(...interface{})
	Logln(...interface{})
	Error(...interface{})
	Start()
	Quit()
}

type serv struct {
	name string
	ip   string
	port int

	running bool

	pks chan network.Packet
}

// NewServer creates a new instance of a Server struct on the given port
//
func NewServer(name string, port int) Server {
	s := new(serv)
	s.name = name
	s.ip = Here()
	s.port = port

	s.running = false

	s.pks = make(chan network.Packet)

	return s
}

func (s *serv) head() string {
	return "[" + colors.Red(bold, network.GetTimeStamp()) + "][" + colors.Purple(none, s.name) + "]"
}

func (s *serv) Log(a ...interface{}) {
	fmt.Print(a...)
}

func (s *serv) Logln(a ...interface{}) {
	print(s.head(), " ")
	fmt.Println(a...)
}

func (s *serv) Error(a ...interface{}) {
	print(s.head(), " ")
	fmt.Println(colors.Red(bold, a...))
}

// Start is the serv's main loop and starts its own goroutine
func (s *serv) Start() {
	s.running = true

	sem := make(chan byte)

	go func() {
		// Start listening
		s.listen()
		sem <- 0
	}()

	go func() {
		for {
			p := <-s.pks
			// Dispatch work
			switch p.(type) {
			case *network.MessagePacket:
				_ = p.(*network.MessagePacket)
			case *network.ConnectionPacket:
				_ = p.(*network.ConnectionPacket)
			default:
				continue
			}
		}
	}()

	<-sem
}

func (s *serv) listen() {
	var (
		conn net.Conn
		data []byte
		err  error
	)

	address := format("%s:%d", s.ip, s.port)

	l, err := net.Listen("tcp", address)
	if err != nil {
		s.Logln("couldn't listen on", address)
		os.Exit(1)
	}
	defer l.Close()

	s.Logln("Server started on", colors.Green(bold, address))
	for {
		if conn != nil {
			// close previous connection if need be
			conn.Close()
		}

		data = make([]byte, network.MaxPacketSize)
		conn, err = l.Accept()
		if err != nil {
			s.Logln(err)
			continue
		}

		n, err := conn.Read(data)

		if err != nil {
			s.Logln("couldn't read packet")
			s.Logln(err)
			continue
		}

		s.emmit(conn, data[:n])
	}
}

func (s *serv) emmit(conn net.Conn, data []byte) {
	println(string(data))
	p, err := network.Compile(conn, []byte("\\C\\"+network.GetTimeStamp().String()+"\\"+string(uuid.NextUUID())+"\\yolo\\hey\\put in your path \\usr\\bin\\"))
	if err != nil {
		panic(err)
	}
	s.pks <- p
}

func (s *serv) sendAll(h network.PacketHeader, content string) {}

func (s *serv) send(c *server.Client, h network.PacketHeader, context string) {}

func (s *serv) disconnect(id uuid.UUID, reason string) {

}

func (s *serv) disconnectClient(c *server.Client, reason string) {

}

func (s *serv) Quit() {
	s.running = false
}
