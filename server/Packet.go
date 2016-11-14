package server

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"unicode"

	"github.com/Spriithy/go-uuid"
)

// MaxPacketSize is the editable maximum size for packet content process
// Default value = 1024
var MaxPacketSize = 1 << 10

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// PacketHeader describes the possible Packet headers
//
type PacketHeader byte

const (
	// InvalidHeader is the PacketHeader 0 value, expressing that the received Packet
	// has an unknown signature, or has no meaning for the Server
	InvalidHeader = PacketHeader(iota)

	// ConnectHeader is the header's representation of a connection Packet
	ConnectHeader = 'C'

	// DisconnectHeader is the header's representation of a Disconnec Packet
	DisconnectHeader = 'D'

	// MessageHeader is the header's representation of a MessagePacket
	MessageHeader = 'M'

	// WhisperHeader is the header's representation of a WhisperPacket
	WhisperHeader = 'W'
)

// A TimeStamp is the internal representation of the time the Packet was emitted
//
type TimeStamp struct {
	Hours, Minutes, Seconds int
}

// GetTimeStamp returns the TimeStamp of the current system time
func GetTimeStamp() *TimeStamp {
	t := time.Now()
	return &TimeStamp{t.Hour(), t.Minute(), t.Second()}
}

// ParseTimeStamp parses the input string and tries to read it as a TimeStamp
// Returns an error if str has not the expected format, panics if an error is encountered
func ParseTimeStamp(str string) (*TimeStamp, error) {
	t := strings.Split(str, ":")
	if len(t) != 3 {
		return nil, errors.New("invalid TimeStamp format to parse")
	}

	h, err := strconv.Atoi(t[0])
	check(err)
	m, err := strconv.Atoi(t[1])
	check(err)
	s, err := strconv.Atoi(t[2])
	check(err)

	return &TimeStamp{h, m, s}, nil
}

func (t *TimeStamp) String() string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hours, t.Minutes, t.Seconds)
}

// Packet is the interface of any valid Packet that the server can receive
//
type Packet interface {
	// ID is the only possible way to authentify each Packet uniqueness
	ID() uuid.UUID

	// From returns the Address and Port from which the Packet has been sent
	From() (string, int)

	// Header returns the PacketHeader of the Packet
	Header() PacketHeader

	// TimeStamp returns a pointer to the TimeStamp at which the Packet
	// has been emitted
	TimeStamp() *TimeStamp

	// Content returns the string content of the Packet
	Content() string
}

// DataPacket is the simplest kind of Packet a server can handle
// It implements the Packet interface
//
// A default DataPacket looks like this :
//      \H\HH:MM:SS\...
// Where    H is the PacketHeader
//          HH:MM:SS is the TimeStamp
type dataPacket struct {
	addr string
	port int

	header  PacketHeader
	time    *TimeStamp
	content string
	id      uuid.UUID
}

// NewDataPacket tries to compile down a connection into a DataPacket
// The function only panics if the TimeStamp format is not respected
func newDataPacket(conn net.Conn) (*dataPacket, error) {
	var (
		err error
		b   []byte
		p   *dataPacket
	)

	p = new(dataPacket)
	b = make([]byte, MaxPacketSize)
	p.id = uuid.NextUUID()

	a := strings.Split(conn.RemoteAddr().String(), ":") // recover Packet address infos
	p.addr = a[0]
	p.port, err = strconv.Atoi(a[1])
	if err != nil {
		return nil, errors.New("couldn't read DataPacket port")
	}

	n, err := conn.Read(b)
	if err != nil {
		return nil, err
	}

	h := b[1] // bruteforce extract the PacketHeader from the sources
	switch h {
	case ConnectHeader, DisconnectHeader, MessageHeader, WhisperHeader:
		p.header = PacketHeader(h)
	default:
		return nil, errors.New("unknown PacketHeader `" + string(h) + "`")
	}

	t, err := ParseTimeStamp(string(b[3:12]))
	if err != nil {
		return nil, err
	}
	p.time = t

	// 12 = fixed metadata
	p.content = string(b[12:n])

	return p, nil
}

// ID returns the DataPacket's Unique ID
func (p *dataPacket) ID() uuid.UUID {
	return p.id
}

// From returns basic informations about the DataPacket source
func (p *dataPacket) From() (string, int) {
	return p.addr, p.port
}

// Header returns the DataPacket's header type
func (p *dataPacket) Header() PacketHeader {
	return p.header
}

// TimeStamp returns the DataPacket's source's TimeStamp at emmit time
func (p *dataPacket) TimeStamp() *TimeStamp {
	return p.time
}

// Content provides the DataPacket content, i.e. everything after the TimeStamp
func (p *dataPacket) Content() string {
	return p.content
}

// ConnectionPacket is the simplest kind of Packet a server can handle
// It implements the Packet interface
//
type ConnectionPacket struct {
	*dataPacket
	userID   uuid.UUID
	userName string
}

// NewConnectionPacket tries to compile down a connection into a ConnectionPacket
// If the username
//
func NewConnectionPacket(conn net.Conn) (*ConnectionPacket, error) {
	var (
		err error
		p   *ConnectionPacket
	)

	p = new(ConnectionPacket)

	p.dataPacket, err = newDataPacket(conn)
	if err != nil {
		return nil, err
	}

	switch p.header {
	case ConnectHeader, DisconnectHeader:
	default:
		return nil, errors.New("cannot read non-connection related Packet as ConnectionPacket")
	}

	split := strings.Split(p.content, "\\")

	id, err := uuid.ParseUUID(split[0])
	if err != nil {
		return nil, err
	}
	p.userID = id

	name := split[1]
	if len(name) < 3 || len(name) > 16 {
		// We must refuse connection to server if username is not as expected
		return nil, errors.New("username is either too long or too short (min:3,max:16)")
	}
	for _, c := range name {
		// Check for invalid characters ...
		switch {
		case unicode.IsSpace(c):
			return nil, errors.New("username contains space(s)")
		case unicode.IsPunct(c):
			return nil, errors.New("username contains punctuation")
		case unicode.IsSymbol(c):
			return nil, errors.New("username contains symbol(s)")
		}
	}
	p.userName = name

	return p, nil
}

// Kind is a simple alias for p.Header()
// This method exists for better code readability
func (p *ConnectionPacket) Kind() PacketHeader {
	return p.header
}

// UserID returns the user's Packet owner ID
func (p *ConnectionPacket) UserID() uuid.UUID {
	return p.userID
}

// UserName returns the user's Packet owner username
func (p *ConnectionPacket) UserName() string {
	return p.userName
}
