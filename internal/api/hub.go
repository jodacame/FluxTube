package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// event is a message broadcast to connected clients.
type event struct {
	Type   string     `json:"type"` // "added" | "removed" | "stopped" | "state"
	ID     string     `json:"id,omitempty"`
	Videos []videoDTO `json:"videos,omitempty"`
}

// hub fans out events to all connected WebSocket clients.
type hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}
	bcast   chan event
}

type client struct {
	conn *websocket.Conn
	send chan event
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func newHub() *hub {
	return &hub{
		clients: map[*client]struct{}{},
		bcast:   make(chan event, 16),
	}
}

func (h *hub) run() {
	for e := range h.bcast {
		h.mu.Lock()
		for c := range h.clients {
			select {
			case c.send <- e:
			default: // drop for slow clients
			}
		}
		h.mu.Unlock()
	}
}

func (h *hub) broadcast(e event) {
	select {
	case h.bcast <- e:
	default:
	}
}

func (h *hub) add(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *hub) remove(c *client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}

func (h *hub) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{conn: conn, send: make(chan event, 16)}
	h.add(c)

	go func() {
		defer func() {
			h.remove(c)
			_ = conn.Close()
		}()
		for e := range c.send {
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(e); err != nil {
				return
			}
		}
	}()

	// Drain reads (and detect close) until the client disconnects.
	go func() {
		defer h.remove(c)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

// broadcastLoop periodically pushes a state snapshot so dashboards stay live.
func (s *Server) broadcastLoop() {
	t := time.NewTicker(3 * time.Second)
	defer t.Stop()
	for range t.C {
		entries := s.store.ListEntries()
		videos := make([]videoDTO, 0, len(entries))
		for _, e := range entries {
			videos = append(videos, s.toDTO(e))
		}
		s.hub.broadcast(event{Type: "state", Videos: videos})
	}
}
