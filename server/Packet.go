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

func checkUsername(name string) (bool, error) {
	if len(name) < 3 || len(name) > 16 {
		return false, errors.New("src username is either too long or too short (min:3,max:16)")
	}
	for _, c := range name {
		// Check for invalid characters ...
		switch {
		case unicode.IsSpace(c):
			return false, errors.New("src username contains space(s)")
		case unicode.IsPunct(c):
			return false, errors.New("src username contains punctuation")
		case unicode.IsSymbol(c):
			return false, errors.New("src username contains symbol(s)")
		}
	}
	return true, nil
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
	if err != nil {
		return nil, err
	} else if h > 23 || h < 0 {
		return nil, errors.New("out of bounds hour in ParseTimeStamp")
	}

	m, err := strconv.Atoi(t[1])
	if err != nil {
		return nil, err
	} else if m > 59 || m < 0 {
		return nil, errors.New("out of bounds minute in ParseTimeStamp")
	}

	s, err := strconv.Atoi(t[2])
	if err != nil {
		return nil, err
	} else if s > 59 || s < 0 {
		return nil, errors.New("out of bounds second in ParseTimeStamp")
	}

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
	case ConnectHeader, DisconnectHeader, MessageHeader:
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
		//
	default:
		return nil, errors.New("cannot read non-connection related Packet as new ConnectionPacket")
	}

	split := strings.Split(p.content, "\\")

	id, err := uuid.ParseUUID(split[0])
	if err != nil {
		return nil, err
	}
	p.userID = id

	name := split[1]
	if ok, err := checkUsername(name); !ok {
		// check for username validity
		return nil, err
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

// A MessagePacket is a simple structure to define a Message sent on the server
// It has a emmitter (src) that is identified by its ID (srcID) and a destinator
// that is either a username or a empty. If empty, the message is sent to the whole
// chat room.
//
// A valid MessagePacket follows the pattern :
//  \H\HH:MM:SS\<uuid>\src\dst\message
type MessagePacket struct {
	*dataPacket
	srcID uuid.UUID
	src   string
	dst   string
	msg   string
}

// NewMessagePacket tries to compile a net.Conn packet input to a MessagePacket
// that is compatible with the server.
func NewMessagePacket(conn net.Conn) (*MessagePacket, error) {
	var (
		err error
		p   *MessagePacket
	)

	p = new(MessagePacket)
	p.dataPacket, err = newDataPacket(conn)
	if err != nil {
		return nil, err
	}

	switch p.header {
	case MessageHeader:
		//
	default:
		return nil, errors.New("cannot read non-message packet as new MessagePacket")
	}

	split := strings.Split(p.content, "\\")

	id, err := uuid.ParseUUID(split[0])
	if err != nil {
		return nil, err
	}
	p.srcID = id

	src := split[1]

	if ok, err := checkUsername(src); !ok {
		// check for username validity
		return nil, err
	}
	p.src = src

	dst := split[1]
	switch len(dst) {
	case 0:
		// send to all
	default:
		if ok, err := checkUsername(dst); !ok {
			// check for username validity
			return nil, err
		}
		p.dst = dst
	}

	for i, str := range split[1:] {
		p.msg += str // reconstruct possible fragmented message
		if i > 1 {
			p.msg += "\\"
		}
	}

	return p, nil
}

// SourceID returns the emmiter's ID
func (p *MessagePacket) SourceID() uuid.UUID {
	return p.srcID
}

// Source returns the source's name
func (p *MessagePacket) Source() string {
	return p.src
}

// Destination returns the MessagePacket's destinator(s)
func (p *MessagePacket) Destination() string {
	return p.dst
}

// Content : Overwrite dataPacket.Conten() to return only the message's content
func (p *MessagePacket) Content() string {
	return p.msg
}
