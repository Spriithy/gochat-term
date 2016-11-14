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

	running bool

	pks chan network.Packet
}

// NewServer creates a new instance of a Server struct on the given port
//
func NewServer(name string, port int) *Server {
	s := new(Server)
	s.name = name
	s.ip = here()
	s.port = port

	s.running = false

	s.pks = make(chan network.Packet)

	return s
}

func (s *Server) head() string {
	return "[" + colors.RedString(network.GetTimeStamp()) + "][" + colors.PurpleString(s.name) + "]"
}

func (s *Server) log(a ...interface{}) {
	fmt.Print(a...)
}

func (s *Server) logln(a ...interface{}) {
	print(s.head(), " ")
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

	go func() {
		for {
			p := <-s.pks
			println("[ PACKET ]----------------------")
			fmt.Println(p.From())
			println(format("%c", p.Header()))
			println(p.TimeStamp().String())
			println(p.ID())
			println(p.Content())
			println("--------------------------------")
		}
	}()

	<-sem
}

func (s *Server) listen() {
	var (
		conn net.Conn
		data []byte
		err  error
	)

	address := format("%s:%d", s.ip, s.port)

	l, err := net.Listen("tcp", address)
	if err != nil {
		s.logln("couldn't listen on", address)
		os.Exit(1)
	}
	defer l.Close()

	s.logln("Server started on", colors.GreenString(address))
	for {
		if conn != nil {
			// close previous connection if need be
			conn.Close()
		}

		data = make([]byte, network.MaxPacketSize)
		conn, err = l.Accept()
		if err != nil {
			s.logln(err)
			continue
		}

		n, err := conn.Read(data)

		if err != nil {
			s.logln("couldn't read packet")
			s.logln(err)
			continue
		}

		go s.emmit(conn, data[:n])
	}
}

func (s *Server) emmit(conn net.Conn, data []byte) {
	println(string(data))
	p, err := network.CompilePacket(conn, []byte("\\C\\"+network.GetTimeStamp().String()+"\\yolo\\hey\\content"))
	if err != nil {
		panic(err)
	}
	s.pks <- p
}

func (s *Server) sendAll(h network.PacketHeader, content string) {

}

func (s *Server) send(c *server.Client, h network.PacketHeader, context string) {

}

func (s *Server) disconnect(id uuid.UUID, reason string) {

}

func (s *Server) disconnectClient(c *server.Client, reason string) {

}

func (s *Server) quit() {

}
