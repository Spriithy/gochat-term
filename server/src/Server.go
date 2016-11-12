package server

import (
	"net"
	"os"
	"fmt"
	"strings"
	"bufio"
	"strconv"

	"github.com/Spriithy/go-uuid"
	"github.com/Spriithy/go-colors"
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
	//PING_HEADER = "/P/"
)

type Server struct {
	// Server Infos
	name    string

	// Network infos
	port    int
	addr    string

	running bool

	clients *ClientMap
}

func NewServer(name string) *Server {
	return &Server{name, 0, "", false, NewClientMap()}
}

func timestamp() string {
	return fmt.Sprintf("%02d:%02d", time.Now().Hour(), time.Now().Second())
}

func local() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		println(colors.RED + "Error recording net interfaces :", err.Error(), colors.NONE)
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

func formatAddress(addr string, port int) string {
	return fmt.Sprintf("%s:%d", addr, port)
}

func (s *Server) hasClientNamed(name string) (bool, *SClient) {
	for c := range s.clients.Iter() {
		if c.name == name {
			return true, c
		}
	}
	return false, nil
}

func (s *Server) Start(port int) {
	clear()

	s.addr = local()
	s.port = port
	s.running = true

	s.run()
}

func help() {
	println(colors.LIGHT_GREEN + "---[ Available commands ]---", colors.NONE)
	println(colors.RED + "  help", colors.NONE, "\t\tprints a list of commands")
	println(colors.RED + "  say ", colors.NONE, "\t\tbroadcasts a message")
	println(colors.RED + "  clear", colors.NONE, "\t\tclears the console")
	println(colors.RED + "  list", colors.NONE, "\t\tlists the currently connected clients")
	println(colors.RED + "  kick", colors.LIGHT_CYAN, "[username]", colors.NONE, "\tkicks a user")
	println(colors.RED + "  quit", colors.NONE, "\t\tshuts down the server")
}

func (s *Server) run() {
	sem := make(chan int) // Semaphore pattern

	go func() {
		s.listen()
		sem <- 1
	}()

	var (
		input string
		cmd []string
		err error
	)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for s.running {
			input, err = reader.ReadString('\n')

			if err != nil {
				println(SERVER_HEADER(s), colors.RED + "Error when reading your input")
				println(strings.Repeat(" ", len(SERVER_HEADER(s))), err.Error())
				println(strings.Repeat(" ", len(SERVER_HEADER(s))), "Ignoring it.", colors.NONE)
				continue
			}

			input = input[:len(input) - 1]
			cmd = strings.Split(input, " ")
			switch cmd[0] {
			case "help": help()
			case "quit": s.quit()
			case "kick":
				if len(cmd) == 1 || len(cmd[1]) == 0 {
					println(SERVER_HEADER(s), colors.RED + "Missing username in `kick` command.", colors.NONE)
					continue
				}

				name := cmd[1]

				if len(name) < 3 {
					println(SERVER_HEADER(s), colors.RED + "Name must be at least 3 characters long." + colors.NONE)
					continue
				}

				found, c := s.hasClientNamed(name)

				if !found {
					println(SERVER_HEADER(s), colors.RED + "Unknown username", "`" + name + "`", colors.NONE)
				} else {
					s.disconnectClient(c, "kick")
				}
			case "list", "ls":
				if s.clients.Size() == 0 {
					println(SERVER_HEADER(s), "No clients are connected yet.")
					continue
				}

				println(SERVER_HEADER(s), "Connected clients :")
				for c := range s.clients.Iter() {
					println("\t*", colors.LIGHT_BLUE + c.name, colors.NONE)
				}
			case "clear": clear()
			case "say":
				s.sendAll(MSG_HEADER + "/" + s.name + "/" + input[4:])
			default:
				println(SERVER_HEADER(s), colors.RED + "Unknown command `" + cmd[0] + "`", colors.NONE)
			}
		}
		sem <- 2
	}()
	<-sem
}

func (s *Server) listen() {
	address := formatAddress(s.addr, s.port)

	l, err := net.Listen("tcp", address)
	if err != nil {
		println(SERVER_HEADER(s), colors.RED + "Error when listening")
		println(strings.Repeat(" ", len(SERVER_HEADER(s))), err.Error(), colors.NONE)
		os.Exit(1)
	}
	defer l.Close()

	println(SERVER_HEADER(s), "is now running on", colors.LIGHT_GREEN + address + colors.NONE)
	var conn net.Conn
	for s.running {
		data := make([]byte, 1024)
		conn, err = l.Accept()
		if err != nil {
			continue
		}

		n, err := conn.Read(data)
		if err != nil {
			println(SERVER_HEADER(s), colors.RED + "error when reading packet")
			println(strings.Repeat(" ", len(SERVER_HEADER(s))), err.Error())
			println(strings.Repeat(" ", len(SERVER_HEADER(s))), "Ignoring it.", colors.NONE)
			continue
		}

		go s.process(conn, data[:n])
	}
}

func (s *Server) process(conn net.Conn, data []byte) {
	content := string(data)
	println(content)

	switch {
	case strings.HasPrefix(content, CONNECT_HEADER):
		id := uuid.NextUUID()
		name := strings.Split(content, CONNECT_HEADER)[1]
		addr := strings.Split(conn.RemoteAddr().String(), ":")
		port, _ := strconv.Atoi(addr[1])
		s.clients.Set(id, ServerClient(id, name, addr[0], port))
		println(SERVER_HEADER(s), "User", colors.LIGHT_CYAN + name + colors.NONE + "@" + colors.LIGHT_GREEN + addr[0] + ":" + addr[1] + colors.NONE, "has connected!")
		c, _ := s.clients.Get(id)
		s.send(c, CONNECT_HEADER + string(id))
	case strings.HasPrefix(content, DISCONNECT_HEADER):
		id := strings.Split(content, DISCONNECT_HEADER)[1]
		s.disconnect(uuid.UUID(id), "leave")
	}
}

func (s *Server) sendAll(data string) {
	header := data[:3]
	parts := strings.Split(data[3:], "/")
	sender := parts[1]
	message := data[5 + len(sender):]
	t := timestamp()

	if header == MSG_HEADER {
		println("[" + colors.LIGHT_RED + t + colors.NONE + "] <" + colors.LIGHT_BLUE + sender + colors.NONE + ">", message)
	}

	for c := range s.clients.Iter() {
		s.send(c, data)
	}
}

func (s *Server) send(c *SClient, data string) {
	go func() {
		ca := formatAddress(c.addr, c.port)
		conn, err := net.Dial("tcp", ca)
		if conn != nil {
			defer conn.Close()
		}
		outside:
		for {
			if c.attempt >= maxAttempts {
				s.disconnectClient(c, "timeout")
				return
			}

			if err != nil {
				println(SERVER_HEADER(s), "Couldn't reach client :", colors.LIGHT_CYAN + c.name + colors.NONE + "@" + colors.LIGHT_GREEN + ca + colors.NONE, "(attempt:", c.attempt, ")")
				println(strings.Repeat(" ", len(SERVER_HEADER(s))), colors.RED, err.Error(), colors.NONE)
				conn, err = net.Dial("tcp", ca)
				c.attempt++
				time.Sleep(time.Second)
				continue outside
			}
			_, err = conn.Write([]byte(data))

			if err != nil {
				println(SERVER_HEADER(s), "couldn't send data to client", colors.LIGHT_BLUE + c.name + colors.NONE, "(attempt:", c.attempt, ")")
				println(strings.Repeat(" ", len(SERVER_HEADER(s))), colors.RED, err.Error(), colors.NONE)
				c.attempt++
				time.Sleep(time.Second)
				continue outside
			}
			return
		}
	}()
	c.attempt = 0
}

func (s *Server) disconnect(id uuid.UUID, reason string) {
	for k := range s.clients.Iter() {
		if id.Match(k.id) {
			s.disconnectClient(k, reason)
			return
		}
	}
}

func (s *Server) disconnectClient(c *SClient, reason string) {
	if _, ok := s.clients.Get(c.id); !ok {
		return
	}

	ca := formatAddress(c.addr, c.port)
	s.clients.Remove(c.id)
	s.sendAll(DISCONNECT_HEADER + c.name + "/" + timestamp() + "/" + reason)

	print("Client ", colors.LIGHT_CYAN + c.name + colors.NONE + "@" + colors.LIGHT_GREEN + ca + colors.NONE, " ")
	switch reason {
	case "kick":
		s.send(c, DISCONNECT_HEADER + c.name + "/" + timestamp() + "/" + reason)
		println("has been kicked from the server.")
	case "timeout":
		println("has timed out.")
	case "shutdown":
		println("has disconnected. Server shut down.")
	case "leave":
		println("has left the channel.")
	default:
		println("has disconnected.")
	}

}

func (s *Server) quit() {
	for k := range s.clients.Iter() {
		s.disconnectClient(k, "shutdown")
	}
	s.running = false
	println(SERVER_HEADER(s), "shut down.")
	os.Exit(1)
}