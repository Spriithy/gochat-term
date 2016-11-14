package server

// Server is the struct defining a gochat-term Server
//
type Server struct {
	name string
	ip   string
	port int

	packets chan Packet
}
