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

func (m *ClientMap) Get(id uuid.UUID) (*SClient, bool) {
	m.RLock()
	defer m.RUnlock()
	value, ok := m.items[id]
	return value, ok
}

func (m *ClientMap) Set(id uuid.UUID, c *SClient) {
	m.RLock()
	defer m.RUnlock()
	m.items[id] = c
}

func (m *ClientMap) Remove(id uuid.UUID) {
	m.RLock()
	defer m.RUnlock()
	delete(m.items, id)
}

func (m *ClientMap) Iter() <-chan *SClient {
	c := make(chan *SClient)

	go func() {
		m.RLock()
		defer m.RUnlock()

		for _, v := range m.items {
			c <- v
		}
		close(c)
	}()

	return c
}

type ResponseMap struct {
	sync.RWMutex
	items map[uuid.UUID]int
}

func NewResponseMap() *ResponseMap {
	return &ResponseMap{sync.RWMutex{}, make(map[uuid.UUID]int)}
}

func (m *ResponseMap) Get(id uuid.UUID) (int, bool) {
	m.RLock()
	defer m.RUnlock()
	value, ok := m.items[id]
	return value, ok
}

func (m *ResponseMap) Set(id uuid.UUID, i int) {
	m.RLock()
	defer m.RUnlock()
	m.items[id] = i
}

func (m *ResponseMap) Remove(id uuid.UUID) {
	m.RLock()
	defer m.RUnlock()
	delete(m.items, id)
}

func (m *ResponseMap) Iter() <-chan int {
	c := make(chan int)

	go func() {
		m.RLock()
		defer m.RUnlock()

		for _, v := range m.items {
			c <- v
		}
		close(c)
	}()

	return c
}