package server

import (
	"fmt"
	"net"
	"time"

	"errors"

	"github.com/Spriithy/go-uuid"
	"github.com/Spriithy/gochat-term/network"
)

// fmt.Sprintf alias for code readability
var format = fmt.Sprintf

// MaxSendAttempts is the maximum amount of tries the Server will do
// to send data to a Client. If the client couldn't be reached within
// this amount of tries, it is timed-out.
const MaxSendAttempts = 5

// ErrClientTimeout notifies the caller that a client couldn't be reached within
// MaxSendAttempts tries.
var ErrClientTimeout = errors.New("client timed out")

// ErrClientUnreachable notifies the caller that a client cannot be reached
var ErrClientUnreachable = errors.New("client is unreachable")

// ErrClientUnavailable notifies the caller that the server cannot send data to the client
var ErrClientUnavailable = errors.New("client is unavailable")

// Client is the representation of the actual Server's client
//
type Client struct {
	id   uuid.UUID
	name string
	ip   string
	port int

	attempts int
}

// NewServerClient creates a new instance of a Client using its ConnectionPacket
func NewServerClient(p *network.ConnectionPacket) *Client {
	ip, port := p.From()
	return &Client{
		id:       p.UserID(),
		name:     p.UserName(),
		ip:       ip,
		port:     port,
		attempts: 0}
}

// Send attempts to sending data to the Client
// If it fails in the first place, it tries up to
func (c *Client) Send(errors chan error, data []byte) {
	go func() {
		addr := format("%s:%d", c.ip, c.port)
		conn, err := net.Dial("tcp", addr)
		if conn != nil {
			defer conn.Close()
		}

	outside:
		for {
			if c.attempts >= MaxSendAttempts {
				errors <- ErrClientTimeout
				return
			}

			if err != nil {
				errors <- ErrClientUnreachable
				c.attempts++
				conn, err = net.Dial("tcp", addr)
				if err != nil {
					errors <- err
				}
				time.Sleep(time.Second)
				continue outside
			}

			if c.attempts > 0 {
				// if first attempt failed, then conn would have never been closed
				defer conn.Close()
			}

			_, err = conn.Write(data)

			if err != nil {
				errors <- ErrClientUnavailable
				c.attempts++
				time.Sleep(time.Second)
				continue outside
			}
			return
		}
	}()
	c.attempts = 0
}
