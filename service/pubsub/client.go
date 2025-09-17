package pubsub

import (
	"github.com/gorilla/websocket"
)

// Client struct, which holds the account ID and their web socket connection
type Client struct {
	AccountID uint
	conn      *websocket.Conn
}

// Constructor method for Client struct
func NewClient(accountID uint, conn *websocket.Conn) *Client {
	return &Client{
		AccountID: accountID,
		conn:      conn,
	}
}

// Method to write a message back to client using WebSocket connection
func (client *Client) WriteMessage(message any) error {
	return client.conn.WriteJSON(message)
}
