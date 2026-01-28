package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"ride-hail-system/internal/logger"

	"github.com/gorilla/websocket"
)

type Hub struct {
	log   logger.Logger
	mu    sync.RWMutex
	conns map[string]*websocket.Conn
}

func NewHub(l logger.Logger) *Hub {
	return &Hub{log: l, conns: map[string]*websocket.Conn{}}
}

func (h *Hub) Set(id string, c *websocket.Conn) {
	h.mu.Lock()
	h.conns[id] = c
	h.mu.Unlock()
}

func (h *Hub) Del(id string) {
	h.mu.Lock()
	delete(h.conns, id)
	h.mu.Unlock()
}

func (h *Hub) Send(id string, msg any) error {
	h.mu.RLock()
	c := h.conns[id]
	h.mu.RUnlock()
	if c == nil {
		return nil
	}
	_ = c.SetWriteDeadline(time.Now().Add(2 * time.Second))
	return c.WriteJSON(msg)
}

func (h *Hub) Broadcast(msg any) {
	b, _ := json.Marshal(msg)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.conns {
		_ = c.SetWriteDeadline(time.Now().Add(2 * time.Second))
		_ = c.WriteMessage(websocket.TextMessage, b)
	}
}
