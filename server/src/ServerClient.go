package server

import (
	"github.com/Spriithy/go-uuid"
)

type SClient struct {
	name    string
	addr    string
	port    int
	attempt int

	id      uuid.UUID
}

func ServerClient(id uuid.UUID, name, addr string, port int) *SClient {
	return &SClient{name, addr, port, 0, id}
}

func (c *SClient) ID() uuid.UUID {
	return c.id
}
