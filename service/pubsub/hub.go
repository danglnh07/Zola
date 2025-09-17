package pubsub

import (
	"sync"
)

// Hub struct, used to track the presence of online users
type Hub struct {
	mutex   *sync.RWMutex
	Clients map[uint]*Client
}

// Constructor method of Hub
func NewHub() *Hub {
	return &Hub{
		mutex:   &sync.RWMutex{},
		Clients: make(map[uint]*Client),
	}
}

// Method to subscribe (join) into the chat server
func (hub *Hub) Subscribe(client *Client) {
	// Lock to prevent race condition
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	// Add client into the Clients map
	hub.Clients[client.AccountID] = client
}

// Method to unsubscribe the client out of the chat server
// This will also clean up any resource to prevent leak
func (hub *Hub) Unsubscribe(client *Client) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	// Remove the client out of Clients map
	delete(hub.Clients, client.AccountID)

	// Close the WebSocket connection
	client.conn.Close()
}
