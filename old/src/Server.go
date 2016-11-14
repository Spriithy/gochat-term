package old_server

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"time"

	"github.com/Spriithy/go-colors"
	"github.com/Spriithy/go-uuid"
)

var (
	clear = func() {
		print("\033[H\033[2J")
	}

	TS = func() string {
		return "[" + colors.LIGHT_RED + timestamp() + colors.NONE + "]"
	}

	maxAttempts = 5

	SERVER_HEADER = func(s *Server) string {
		return "[" + colors.LIGHT_BLUE + s.name + colors.NONE + "]"
	}

	MSG_HEADER        = "/M/"
	CONNECT_HEADER    = "/C/"
	DISCONNECT_HEADER = "/D/"
	//PING_HEADER = "/P/"
)

type Server struct {
	// Server Infos
	name string

	// Network infos
	port int
	addr string

	running bool

	clients *ClientMap

	gui *ServerUI
}

func (s *Server) Print(a ...interface{}) {
	s.gui.Print(a...)
}

func (s *Server) Println(a ...interface{}) {
	s.gui.Println(a...)
}

func NewServer(name string) *Server {
	return &Server{name, 0, "", false, NewClientMap(), NewServerUI()}
}

func timestamp() string {
	return fmt.Sprintf("%02d:%02d", time.Now().Hour(), time.Now().Second())
}

func local() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		println(colors.RED+"Error recording net interfaces :", err.Error(), colors.NONE)
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
	ch := make(chan bool)

	s.addr = local()
	s.port = port
	s.running = true

	go func() {
		s.gui.Start()
		ch <- true
	}()

	time.Sleep(time.Millisecond)

	go func() {
		s.run()
		ch <- true
	}()

	<-ch
}

func (s *Server) help() {
	s.Println(colors.LIGHT_GREEN+"---[ Available commands ]---", colors.NONE)
	s.Println(colors.RED+"  help", colors.NONE, "\t\tprints a list of commands")
	s.Println(colors.RED+"  say ", colors.NONE, "\t\tbroadcasts a message")
	s.Println(colors.RED+"  clear", colors.NONE, "\t\tclears the console")
	s.Println(colors.RED+"  list", colors.NONE, "\t\tlists the currently connected clients")
	s.Println(colors.RED+"  kick", colors.LIGHT_CYAN, "[username]", colors.NONE, "\tkicks a user")
	s.Println(colors.RED+"  quit", colors.NONE, "\t\tshuts down the server")
}

func (s *Server) run() {
	sem := make(chan int) // Semaphore pattern

	go func() {
		s.listen()
		sem <- 1
	}()

	var (
		input string
		cmd   []string
	)
	go func() {
		for s.running {
			input = s.gui.FlushInput()

			if len(input) == 0 {
				continue
			}

			input = input[:len(input)-1]
			cmd = strings.Split(input, " ")
			switch cmd[0] {
			case "help":
				s.help()
			case "quit":
				s.quit()
			case "kick":
				if len(cmd) == 1 || len(cmd[1]) == 0 {
					s.Println(TS(), colors.RED+"Missing username in `kick` command.", colors.NONE)
					continue
				}

				name := cmd[1]

				if len(name) < 3 {
					s.Println(TS(), colors.RED+"Name must be at least 3 characters long."+colors.NONE)
					continue
				}

				found, c := s.hasClientNamed(name)

				if !found {
					s.Println(TS(), colors.RED+"Unknown username", "`"+name+"`", colors.NONE)
				} else {
					s.disconnectClient(c, "kick")
				}
			case "list", "ls":
				if s.clients.Size() == 0 {
					s.Println(TS(), "No clients are connected yet.")
					continue
				}

				info := false
				if len(cmd) > 1 {
					if cmd[1] == "-i" {
						info = true
					} else {
						s.Println(TS(), "Unknown parameter to", "`"+cmd[0]+"`", ":", "`"+cmd[1]+"`")
						continue
					}
				}

				s.Println(TS(), s.clients.Size(), " Connected clients:")
				for c := range s.clients.Iter() {
					print("\t* ", colors.LIGHT_CYAN+c.name+colors.NONE)
					if info {
						s.Println("@"+colors.LIGHT_GREEN+c.addr+":"+strconv.Itoa(c.port)+colors.RED, fmt.Sprintf("%40s", c.id), colors.NONE)
					} else {
						s.Println()
					}
				}
			case "clear":
				s.Println("Not yet implemented")
			case "say":
				s.sendAll(MSG_HEADER + s.name + "/" + timestamp() + "/" + input[4:])
			default:
				s.Println(SERVER_HEADER(s), colors.RED+"Unknown command `"+cmd[0]+"`", colors.NONE)
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
		s.Println(SERVER_HEADER(s), colors.RED+"Error when listening")
		s.Println(strings.Repeat(" ", len(SERVER_HEADER(s))), err.Error(), colors.NONE)
		os.Exit(1)
	}
	defer l.Close()

	s.Println(SERVER_HEADER(s), "is now running on", colors.LIGHT_GREEN+address+colors.NONE)
	var conn net.Conn
	for s.running {
		data := make([]byte, 1024)
		conn, err = l.Accept()
		if err != nil {
			continue
		}

		n, err := conn.Read(data)
		if err != nil {
			s.Println(SERVER_HEADER(s), colors.RED+"error when reading packet")
			s.Println(strings.Repeat(" ", len(s.name)+1), colors.RED, err.Error())
			s.Println(strings.Repeat(" ", len(s.name)+2), "Ignoring it.", colors.NONE)
			continue
		}

		go s.process(conn, data[:n])
	}
}

func (s *Server) process(conn net.Conn, data []byte) {
	content := string(data)
	s.Println(content)

	switch {
	case strings.HasPrefix(content, CONNECT_HEADER):
		id := uuid.NextUUID()
		name := strings.Split(content, CONNECT_HEADER)[1]
		addr := strings.Split(conn.RemoteAddr().String(), ":")
		port, _ := strconv.Atoi(addr[1])
		s.Println(SERVER_HEADER(s), "User", colors.LIGHT_CYAN+name+colors.NONE+"@"+colors.LIGHT_GREEN+addr[0]+":"+addr[1]+colors.NONE, "has joined!")
		s.clients.Set(id, ServerClient(id, name, addr[0], port))
		c, _ := s.clients.Get(id)
		s.send(c, CONNECT_HEADER+string(id))
	case strings.HasPrefix(content, DISCONNECT_HEADER):
		id := strings.Split(content[3:], "/")[0][:35]
		s.disconnect(uuid.UUID(id), "leave")
	}
}

func (s *Server) sendAll(data string) {
	header := data[:3]

	if header == MSG_HEADER {
		parts := strings.Split(data[3:], "/")
		sender := parts[0]
		t := parts[1]
		message := data[len(sender)+10:]
		s.Println("["+colors.LIGHT_RED+t+colors.NONE+"] <"+colors.LIGHT_BLUE+sender+colors.NONE+">", message)
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
				s.Println(SERVER_HEADER(s), "Couldn't reach client :", colors.LIGHT_CYAN+c.name+colors.NONE+"@"+colors.LIGHT_GREEN+ca+colors.NONE, "(attempt:", c.attempt, ")")
				s.Println(strings.Repeat(" ", len(s.name)+1), colors.RED+err.Error(), colors.NONE)
				conn, err = net.Dial("tcp", ca)
				c.attempt++
				time.Sleep(time.Second)
				continue outside
			}
			_, err = conn.Write([]byte(data))

			if err != nil {
				s.Println(SERVER_HEADER(s), "couldn't send data to client", colors.LIGHT_BLUE+c.name+colors.NONE, "(attempt:", c.attempt, ")")
				s.Println(strings.Repeat(" ", len(s.name)+1), colors.RED, err.Error(), colors.NONE)
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
	var c *SClient
	for k := range s.clients.Iter() {
		if id.Match(k.id) {
			c = k
			break
		}
	}
	s.disconnectClient(c, reason)
}

func (s *Server) disconnectClient(c *SClient, reason string) {
	if _, ok := s.clients.Get(c.id); !ok {
		return
	}

	ca := formatAddress(c.addr, c.port)
	s.clients.Remove(c.id)
	s.sendAll(DISCONNECT_HEADER + c.name + "/" + timestamp() + "/" + reason)

	s.Print(TS(), " Client ", colors.LIGHT_CYAN+c.name+colors.NONE+"@"+colors.LIGHT_GREEN+ca+colors.NONE, " ")
	s.send(c, DISCONNECT_HEADER+c.name+"/"+timestamp()+"/"+reason)
	switch reason {
	case "kick":
		s.Println("has been kicked from the server.")
	case "timeout":
		s.Println("has timed out.")
	case "shutdown":
		s.Println("has disconnected. Server shut down.")
	case "leave":
		s.Println("has left the channel.")
	default:
		s.Println("has left the channel.")
	}
}

func (s *Server) quit() {
	for k := range s.clients.Iter() {
		s.disconnectClient(k, "shutdown")
	}
	s.running = false
	s.Println(SERVER_HEADER(s), "shut down.")
	os.Exit(1)
}
