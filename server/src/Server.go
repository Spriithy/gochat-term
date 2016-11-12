package server

import (
	"github.com/Spriithy/go-colors"
	"net"
	"os"
	"fmt"
	"github.com/Spriithy/go-uuid"
	"strings"
	"time"
	"bufio"
	"strconv"
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
)

type Server struct {
	// Server Infos
	name      string

	// Network infos
	port      int
	addr      string

	running   bool

	clients   *ClientMap
	responses *ResponseMap
}

func NewServer(name string) *Server {
	return &Server{name, 0, "", false, NewClientMap(), NewResponseMap()}
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
		s.pingAll()
		sem <- 0
	}()

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
				if len(cmd) == 1 {
					println(SERVER_HEADER(s), colors.RED + "Missing username in `kick` command.", colors.NONE)
					continue
				}

				name := cmd[1]
				found := false
				for c := range s.clients.Iter() {
					if c.name == name {
						s.disconnectClient(c, true)
						found = true
						break
					}
				}

				if !found {
					println(SERVER_HEADER(s), colors.RED + "Unknown username", "`" + name + "`", colors.NONE)
				}
			case "list":
				if len(s.clients.Iter()) == 0 {
					println(SERVER_HEADER(s), "No clients are connected.")
					continue
				}

				for c := range s.clients.Iter() {
					println("\t*", colors.LIGHT_BLUE + c.name, colors.NONE)
				}
			case "clear": clear()
			case "say":
				s.sendAll(MSG_HEADER + s.name + ": " + input[4:])
			default:
				println(SERVER_HEADER(s), colors.RED + "Unknown command /" + cmd[0], colors.NONE)
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

		_, err = conn.Read(data)
		if err != nil {
			println(SERVER_HEADER(s), colors.RED + "error when reading packet")
			println(strings.Repeat(" ", len(SERVER_HEADER(s))), err.Error())
			println(strings.Repeat(" ", len(SERVER_HEADER(s))), "Ignoring it.", colors.NONE)
			continue
		}

		go s.process(conn, data)
	}
}

func (s *Server) process(conn net.Conn, data []byte) {
	content := string(data)

	switch {
	case strings.HasPrefix(content, CONNECT_HEADER):
		id := uuid.NextUUID()
		name := strings.Split(content, CONNECT_HEADER)[1]
		name = name[:len(name) - 2]
		addr := strings.Split(conn.RemoteAddr().String(), ":")
		port, _ := strconv.Atoi(addr[1])
		s.clients.Set(id, ServerClient(name, addr[0], port))
		println(SERVER_HEADER(s), "User", colors.LIGHT_CYAN + name + colors.NONE + "@" + colors.LIGHT_GREEN + addr[0] + ":" + addr[1] + colors.NONE, "has connected!")
	}
}

func (s *Server) pingAll() {
	for s.running {
		s.sendAll(PING_HEADER + s.name)
		time.Sleep(time.Second * 2)
		for c := range s.clients.Iter() {
			if _, ok := s.responses.Get(c.id); !ok {
				if c.attempt >= maxAttempts {
					s.disconnect(c.id, false)
				} else {
					c.attempt++
				}
			} else {
				s.responses.Remove(c.id)
				c.attempt = 0
				break
			}
		}
	}
}

func (s *Server) sendAll(data string) {
	if data[:3] == MSG_HEADER {
		println("[" + colors.LIGHT_RED + "message" + colors.NONE + "]", data[3:])
	}

	for c := range s.clients.Iter() {
		s.send(c, data)
	}
}

func (s *Server) send(c *SClient, data string) {
	ca := formatAddress(c.addr, c.port)
	conn, err := net.Dial("tcp", ca)
	if err != nil {
		println(SERVER_HEADER(s), "Couldn't reach client :", colors.LIGHT_CYAN + c.name + colors.NONE + "@" + colors.LIGHT_GREEN + ca + colors.NONE)
		println(strings.Repeat(" ", len(SERVER_HEADER(s)) - 1), colors.RED, err.Error(), colors.NONE, + c.attempt)
		return
	}
	_, err = conn.Write([]byte(data))

	if err != nil {
		println(SERVER_HEADER(s), "couldn't send data to client :", )
		println(strings.Repeat(" ", len(SERVER_HEADER(s)) - 1), colors.RED, err.Error(), colors.NONE)
	}
}

func (s *Server) disconnect(id uuid.UUID, status bool) {
	for k := range s.clients.Iter() {
		if id.Match(k.id) {
			s.disconnectClient(k, status)
			return
		}
	}
}

func (s *Server) disconnectClient(c *SClient, status bool) {
	ca := formatAddress(c.addr, c.port)
	s.clients.Remove(c.id)
	s.responses.Remove(c.id)
	if status {
		println("Client", colors.LIGHT_CYAN + c.name + colors.NONE + "@" + colors.LIGHT_GREEN + ca + colors.NONE + " disconnected.")
	} else {
		println("Client", colors.LIGHT_CYAN + c.name + colors.NONE + "@" + colors.LIGHT_GREEN + ca + colors.NONE + " timed out.")
	}
}

func (s *Server) quit() {
	for k := range s.clients.Iter() {
		s.disconnectClient(k, true)
	}
	s.running = false
	println(SERVER_HEADER(s), "shut down.")
	os.Exit(1)
}