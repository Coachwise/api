package ws

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// The socket carries no content and is guarded per-user; allow any origin so
	// the web/app clients connect without CORS preflight on the upgrade.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handle upgrades the request to a websocket and registers it under the user id
// from ?user_id=. No JWT: the socket only ever delivers contentless refetch
// hints, so identifying the session by user id is enough.
func Handle(c *gin.Context) {
	userID, err := uuid.Parse(c.Query("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // upgrader already wrote the error
	}
	cl := &client{conn: conn}
	hub.add(userID, cl)
	defer func() {
		hub.remove(userID, cl)
		_ = conn.Close()
	}()

	// Keepalive: reply to pongs; the read loop also detects disconnects.
	conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(70 * time.Second))
		return nil
	})
	go pinger(cl)

	// Read loop — we don't expect inbound messages; it exists to notice closes.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

// pinger keeps the connection alive and prunes dead ones.
func pinger(c *client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return
		}
		err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
		if err != nil {
			c.closed = true
			_ = c.conn.Close()
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()
	}
}
