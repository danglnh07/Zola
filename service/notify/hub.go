package notify

import (
	"sync"

	"github.com/danglnh07/zola/db"
)

// Subscriber of the notification service
type Subscriber chan db.Notification

// Hub, store mutex and the subscribers map
type Hub struct {
	mu          sync.RWMutex
	subscribers map[Subscriber]struct{}
}

// Constructor method for the hub
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[Subscriber]struct{}),
	}
}

// Method to subscriber into the hub
func (h *Hub) Subscribe() Subscriber {
	ch := make(Subscriber, 10) // buffered channel
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Method to unsubscribe from the hub
func (h *Hub) Unsubscribe(ch Subscriber) {
	h.mu.Lock()
	delete(h.subscribers, ch)
	close(ch)
	h.mu.Unlock()
}

// Method to publish notification to the hub
func (h *Hub) Publish(noti db.Notification) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subscribers {
		select {
		case ch <- noti:
		default: // avoid blocking if buffer full
		}
	}
}
