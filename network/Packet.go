package network

import (
	"errors"
	"fmt"
	"time"

	"net"
	"unicode"

	"github.com/Spriithy/go-uuid"
	"github.com/Spriithy/springwater-db/serial"
)

// Packet format over the gochat-term protocol
//
// - There are several valid Packet types :
//  	-> C|D for ConnectionPacket : connection status of clients, disconnections etc
//  	-> M|W for MessagePacket	: Messages and Whispers sent through the Server
//  	-> S   for ServerPacket 	: Server info (ie. restart, commands results)
//
// - ID is a general purpose UUID formatted as (xxxxxxxx-xxxx-xxxx-xxxxxxxxxxxxxxxx)
// in hexadecimal digits (see uuid.UUID at gtihub.com/Spriithy/go-uuid)
//
// - CODE is one of :
//  	-> 0x0 : success
//  	-> 0x1 : permission error
//  	-> 0x2 : server quit, restart
//
// - User
//  	HEAD MMSS CONTENT\r\n
//      Content format for :
//  		-> Channel message	: ID MESSAGE
//  		-> Whisper message	: ID to MESSAGE
//  		-> Connection    	:
// - Server
//  	HEAD CODE MMSS CONTENT\r\n
//

var format = fmt.Sprintf

// MaxPacketSize is the editable maximum size for packet content process
// Default value = 1024
var MaxPacketSize = 1 << 10

const crlf = "\r\n"

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

const (
	invalidHead = byte(iota)
	joinHead    = 'J'
	leaveHead   = 'L'
	messageHead = 'M'
	whisperHead = 'W'
	serverHead  = 'S'
)

const (
	minCode = '0' - 1
	// SuccessCode is the code returned by the server if it didn't encounter any issue
	SuccessCode = '0'

	// PermissionErrorCode is used by the server to tell a client the action requested
	// requires higher permissions
	PermissionErrorCode = '1'

	// ShutdownCode is the code used by the server to notify about its shutdown process
	ShutdownCode = '2'

	maxCode = '9' + 1
)

// Packet is the interface of any valid Packet that the server can receive
//
type Packet interface {
	// Header returns the byte header of the Packet
	Header() byte

	// TimeStamp returns a pointer to the TimeStamp at which the Packet
	// has been emitted
	TimeStamp() *TimeStamp

	// Content returns the string content of the Packet
	Content() string

	// Transfer is used to send the Packet over the network to a given adress
	Transfer(string, int) error

	String() string
}

type serverPacket struct {
	timeOffset    int
	contentOffset int
	crlfOffset    int
	data          serial.Data
}

var metaSize = 5 + 2*serial.GetSize(serial.UInt8)

// ServerPacket creates a server-emmitable packet ready to be sent of the network
func ServerPacket(code byte, content string) (Packet, error) {
	if code <= minCode || code >= maxCode {
		return nil, errors.New("server code out of bounds")
	}

	if len(content)+metaSize > MaxPacketSize {
		return nil, errors.New("content too long")
	}

	t := time.Now()
	data := make(serial.Data, len(content)+metaSize)
	ptr := data.WriteBytes(0, []byte{serverHead, ' ', code, ' '})

	to := ptr
	ptr = data.WriteUInt8(ptr, uint8(t.Minute()))
	ptr = data.WriteUInt8(ptr, uint8(t.Second()))
	ptr = data.WriteByte(ptr, ' ')

	co := ptr
	ptr = data.WriteBytes(ptr, []byte(content))

	cr := ptr
	ptr = data.WriteBytes(ptr, []byte(crlf))

	return &serverPacket{to, co, cr, data}, nil
}

func (p *serverPacket) Header() byte {
	return p.data[0]
}

func (p *serverPacket) TimeStamp() *TimeStamp {
	m := p.data.ReadUInt8(p.timeOffset)
	s := p.data.ReadUInt8(p.timeOffset + 1)
	h := time.Now().Hour()
	return &TimeStamp{Hours: h, Minutes: int(m), Seconds: int(s)}
}

func (p *serverPacket) Content() string {
	return string(p.data[p.contentOffset:p.crlfOffset])
}

func (p *serverPacket) Transfer(addr string, port int) error {
	var err error

	conn, err := net.Dial("tcp", format("%s:%d", addr, port))
	if err != nil {
		return err
	}
	defer conn.Close()

	n, err := conn.Write([]byte(p.data))
	if err != nil {
		return err
	}

	if n != len(p.data) { // edgy case really
		return errors.New("message was sent uncomplete")
	}

	return nil
}

func (p *serverPacket) String() string {
	return string(p.data)
}

type userPacket struct {
	timeOffset    int
	contentOffset int
	crlfOffset    int
	kind          byte
	owner         uuid.UUID
	data          serial.Data
}

var userMetaSize = 7 + 2*serial.GetSize(serial.UInt8)

// UserPacket is used to wrap the data of a UserPacket over the network
func UserPacket(kind byte, owner uuid.UUID, content string) (Packet, error) {
	if owner == uuid.UUID("") {
		owner = uuid.NextUUID()
	}

	data := make(serial.Data, MaxPacketSize)

	t := time.Now()
	ptr := data.WriteByte(0, kind)
	ptr = data.WriteByte(ptr, ' ')

	to := ptr
	ptr = data.WriteUInt8(ptr, uint8(t.Minute()))
	ptr = data.WriteUInt8(ptr, uint8(t.Second()))
	ptr = data.WriteByte(ptr, ' ')

	co := ptr
	ptr = data.WriteBytes(ptr, []byte(content))

	cr := ptr
	ptr = data.WriteBytes(ptr, []byte(crlf))

	switch kind {
	case messageHead, whisperHead, joinHead, leaveHead:
		return &userPacket{to, co, cr, kind, owner, data}, nil
	default:
		return nil, errors.New("invalid UserPacket kind")
	}
}

func (p *userPacket) Header() byte {
	return p.data[0]
}

func (p *userPacket) TimeStamp() *TimeStamp {
	m := p.data.ReadUInt8(p.timeOffset)
	s := p.data.ReadUInt8(p.timeOffset + 1)
	h := time.Now().Hour()
	return &TimeStamp{Hours: h, Minutes: int(m), Seconds: int(s)}
}

func (p *userPacket) Content() string {
	return string(p.data[p.contentOffset:p.crlfOffset])
}

func (p *userPacket) Transfer(addr string, port int) error {
	var err error

	conn, err := net.Dial("tcp", format("%s:%d", addr, port))
	if err != nil {
		return err
	}
	defer conn.Close()

	n, err := conn.Write([]byte(p.data))
	if err != nil {
		return err
	}

	if n != len(p.data) { // edgy case really
		return errors.New("message was sent uncomplete")
	}

	return nil
}

func (p *userPacket) String() string {
	return string(p.data)
}
