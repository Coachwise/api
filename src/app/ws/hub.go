// Package ws is the realtime refetch-signal socket. It carries NO content — just
// tiny {"refetch":[...]} pokes telling a user's live clients which sections to
// reload. Signals arrive from the event bus (NATS) and fan out to that user's
// open sockets on this instance.
package ws

import (
	"encoding/json"
	"sync"

	"coachwise/src/events"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// hub tracks each user's open sockets. Concurrency-safe.
type hubT struct {
	mu    sync.RWMutex
	conns map[uuid.UUID]map[*client]struct{}
}

var hub = &hubT{conns: make(map[uuid.UUID]map[*client]struct{})}

func (h *hubT) add(userID uuid.UUID, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[userID] == nil {
		h.conns[userID] = make(map[*client]struct{})
	}
	h.conns[userID][c] = struct{}{}
}

func (h *hubT) remove(userID uuid.UUID, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.conns[userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.conns, userID)
		}
	}
}

// sendRefetch pushes a {"refetch":[topics]} message to all of a user's sockets.
func (h *hubT) sendRefetch(userID uuid.UUID, topics []string) {
	h.mu.RLock()
	set := h.conns[userID]
	targets := make([]*client, 0, len(set))
	for c := range set {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	if len(targets) == 0 {
		return
	}
	payload, err := json.Marshal(map[string]any{"refetch": topics})
	if err != nil {
		return
	}
	for _, c := range targets {
		c.send(payload)
	}
}

// Start subscribes the hub to the event bus's refetch signals. Call once after
// the bus is connected.
func Start() {
	events.SubscribeSignals(func(sig events.RefetchSignal) {
		hub.sendRefetch(sig.UserID, sig.Topics)
	})
}

// client wraps a socket with a serialized writer (gorilla requires one writer).
type client struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	closed bool
}

func (c *client) send(payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		c.closed = true
		_ = c.conn.Close()
	}
}
