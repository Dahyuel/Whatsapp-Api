package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// SSEEvent is a server-sent event payload pushed to agents.
type SSEEvent struct {
	Type    string      `json:"type"` // "new_message" | "new_chat"
	AgentID string      `json:"-"`    // routing key, not serialised
	Data    interface{} `json:"data"`
}

// SSEHub manages per-agent SSE connections.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[string][]chan SSEEvent // agentID → channels
}

// NewSSEHub creates an initialised hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[string][]chan SSEEvent),
	}
}

// Subscribe registers a new SSE channel for an agent.
func (h *SSEHub) Subscribe(agentID string) chan SSEEvent {
	ch := make(chan SSEEvent, 32)
	h.mu.Lock()
	h.clients[agentID] = append(h.clients[agentID], ch)
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes a channel.
func (h *SSEHub) Unsubscribe(agentID string, ch chan SSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.clients[agentID]
	for i, c := range list {
		if c == ch {
			h.clients[agentID] = append(list[:i], list[i+1:]...)
			close(ch)
			return
		}
	}
}

// publish sends a typed SSEEvent to all connections belonging to agentID.
func (h *SSEHub) publish(agentID string, ev SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.clients[agentID] {
		select {
		case ch <- ev:
		default: // drop if buffer full
		}
	}
}

// Publish satisfies session.SSEPublisher: sends an event to agentID's connections.
func (h *SSEHub) Publish(agentID string, eventType string, data interface{}) {
	h.publish(agentID, SSEEvent{Type: eventType, AgentID: agentID, Data: data})
}

// publishAll sends a typed SSEEvent to every connected agent.
func (h *SSEHub) publishAll(ev SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, chans := range h.clients {
		for _, ch := range chans {
			select {
			case ch <- ev:
			default:
			}
		}
	}
}

// PublishAll satisfies session.SSEPublisher: sends an event to every connected agent.
func (h *SSEHub) PublishAll(eventType string, data interface{}) {
	h.publishAll(SSEEvent{Type: eventType, Data: data})
}

// RegisterRealtimeRoutes registers GET /agent/events (SSE stream).
func RegisterRealtimeRoutes(r gin.IRouter, hub *SSEHub) {
	r.GET("/agent/events", func(c *gin.Context) {
		agentID, _ := c.Get("user_id")
		aid := fmt.Sprintf("%v", agentID)

		ch := hub.Subscribe(aid)
		defer hub.Unsubscribe(aid, ch)

		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Accel-Buffering", "no")
		c.Writer.WriteHeader(http.StatusOK)

		// Send a heartbeat immediately so the client knows it's connected
		fmt.Fprintf(c.Writer, "event: connected\ndata: {}\n\n")
		c.Writer.Flush()

		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					return
				}
				b, _ := json.Marshal(ev)
				fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", ev.Type, string(b))
				c.Writer.Flush()
			case <-c.Request.Context().Done():
				return
			}
		}
	})
}
