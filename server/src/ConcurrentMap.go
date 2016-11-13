package server

import (
	"sync"
	"github.com/Spriithy/go-uuid"
)

type ClientMap struct {
	sync.RWMutex
	items map[uuid.UUID]*SClient
}

func NewClientMap() *ClientMap {
	return &ClientMap{sync.RWMutex{}, make(map[uuid.UUID]*SClient)}
}

func (m *ClientMap) Size() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.items)
}

func (m *ClientMap) Get(id uuid.UUID) (*SClient, bool) {
	m.RLock()
	defer m.RUnlock()
	value, ok := m.items[id]
	return value, ok
}

func (m *ClientMap) Set(id uuid.UUID, c *SClient) {
	m.Lock()
	defer m.Unlock()
	m.items[id] = c
}

func (m *ClientMap) Remove(id uuid.UUID) {
	m.Lock()
	defer m.Unlock()
	delete(m.items, id)
}

func (m *ClientMap) Iter() <-chan *SClient {
	c := make(chan *SClient)

	go func() {
		m.RLock()
		for _, v := range m.items {
			c <- v
		}
		m.RUnlock()
		close(c)
	}()

	return c
}
