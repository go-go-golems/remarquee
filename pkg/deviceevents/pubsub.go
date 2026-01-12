package deviceevents

import (
	"sync"
	"time"
)

// PubSub publishes input events to subscribers.
type PubSub struct {
	subscribers map[chan InputEventFromSource]bool
	mu          sync.Mutex
}

// NewPubSub creates a new event pub/sub bus.
func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[chan InputEventFromSource]bool),
	}
}

// Publish sends an event to all subscribers with a short timeout.
func (ps *PubSub) Publish(event InputEventFromSource) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	ps.mu.Lock()
	defer ps.mu.Unlock()

	for ch := range ps.subscribers {
		select {
		case ch <- event:
		case <-ticker.C:
		}
	}
}

// Subscribe registers a new subscriber channel.
func (ps *PubSub) Subscribe() chan InputEventFromSource {
	ch := make(chan InputEventFromSource)
	ps.mu.Lock()
	ps.subscribers[ch] = true
	ps.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (ps *PubSub) Unsubscribe(ch chan InputEventFromSource) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if _, ok := ps.subscribers[ch]; ok {
		delete(ps.subscribers, ch)
		close(ch)
	}
}
