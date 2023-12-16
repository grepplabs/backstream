package ws

import (
	"sync"
)

type Pool struct {
	mu sync.Mutex
	// registered clients.
	clients map[*Conn]string
}

func NewPool() *Pool {
	return &Pool{
		clients: make(map[*Conn]string),
	}
}

func (m *Pool) GetConn() *Conn {
	m.mu.Lock()
	defer m.mu.Unlock()

	for client := range m.clients {
		return client
	}
	return nil
}

func (m *Pool) GetConnByID(id string) *Conn {
	m.mu.Lock()
	defer m.mu.Unlock()

	for client, clientId := range m.clients {
		if clientId == id {
			return client
		}
	}
	return nil
}

func (m *Pool) GetConnsByID(id string) (result []*Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for client, clientId := range m.clients {
		if clientId == id {
			result = append(result, client)
		}
	}
	return result
}

func (m *Pool) Size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.clients)
}

func (m *Pool) register(conn *Conn) {
	m.mu.Lock()
	m.clients[conn] = conn.clientID
	m.mu.Unlock()
}

func (m *Pool) unregister(conn *Conn) {
	m.mu.Lock()
	delete(m.clients, conn)
	m.mu.Unlock()
}
