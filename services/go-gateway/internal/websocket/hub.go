package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/autotraka/go-gateway/internal/eventbus"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// WSEvent is the lightweight envelope pushed over WebSocket.
type WSEvent struct {
	Type           string          `json:"type"`
	ConversationID uuid.UUID       `json:"conversation_id"`
	Payload        json.RawMessage `json:"payload"`
}

// Client is a single WebSocket connection.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	tenantID uuid.UUID
}

// readPump reads messages from the WebSocket (mostly to detect disconnects).
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Default().Error("websocket unexpected close", "error", err)
			}
			break
		}
	}
}

// writePump writes messages from the hub to the WebSocket.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Hub maintains the set of active clients and routes NATS events.
type Hub struct {
	clients    map[uuid.UUID]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan WSEvent
	eventbus   *eventbus.Client
	mu         sync.RWMutex
}

// NewHub creates a new hub.
func NewHub(eb *eventbus.Client) *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan WSEvent, 256),
		eventbus:   eb,
	}
}

// Run starts the hub's goroutines for client management and event routing.
func (h *Hub) Run() {
	go h.runClients()
	if h.eventbus != nil {
		go h.runNATSConsumer()
	}
}

func (h *Hub) runClients() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.tenantID] == nil {
				h.clients[client.tenantID] = make(map[*Client]bool)
			}
			h.clients[client.tenantID][client] = true
			h.mu.Unlock()
			slog.Default().Info("websocket client registered", "tenant_id", client.tenantID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.tenantID][client]; ok {
				delete(h.clients[client.tenantID], client)
				close(client.send)
				if len(h.clients[client.tenantID]) == 0 {
					delete(h.clients, client.tenantID)
				}
			}
			h.mu.Unlock()
			slog.Default().Info("websocket client unregistered", "tenant_id", client.tenantID)

		case event := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[event.ConversationID]
			if clients == nil {
				// Try to route by tenant ID from payload
				clients = h.routeByTenant(event)
			}
			for client := range clients {
				select {
				case client.send <- mustJSON(event):
				default:
					close(client.send)
					delete(h.clients[client.tenantID], client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) routeByTenant(event WSEvent) map[*Client]bool {
	// Parse tenant_id from payload
	var payload struct {
		TenantID uuid.UUID `json:"tenant_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return nil
	}
	return h.clients[payload.TenantID]
}

func (h *Hub) runNATSConsumer() {
	subjects := []string{"message.inbound", "message.status_updated", "conversation.updated", "conversation.escalated"}
	streams := map[string]string{
		"message.inbound":         "MESSAGES",
		"message.status_updated":  "MESSAGES",
		"conversation.updated":    "CONVERSATIONS",
		"conversation.escalated":  "CONVERSATIONS",
	}

	for _, subject := range subjects {
		subject := subject // capture for closure
		_, err := h.eventbus.Subscribe(context.Background(), subject, func(_ context.Context, evt eventbus.Event) error {
			wsEvent := h.transformEvent(subject, evt)
			if wsEvent != nil {
				h.broadcast <- *wsEvent
			}
			return nil
		})
		if err != nil {
			slog.Default().Error("failed to subscribe to nats", "subject", subject, "stream", streams[subject], "error", err)
		}
	}
}

func (h *Hub) transformEvent(subject string, evt eventbus.Event) *WSEvent {
	var payload map[string]interface{}
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return nil
	}

	var eventType string
	switch subject {
	case "message.inbound":
		eventType = "new_message"
	case "message.status_updated":
		eventType = "message_status"
	case "conversation.updated":
		eventType = "conversation_updated"
	case "conversation.escalated":
		eventType = "escalation"
	default:
		return nil
	}

	convIDRaw, ok := payload["conversation_id"]
	if !ok {
		return nil
	}
	convIDStr, ok := convIDRaw.(string)
	if !ok {
		return nil
	}
	convID, err := uuid.Parse(convIDStr)
	if err != nil {
		return nil
	}

	payloadBytes, _ := json.Marshal(payload)
	return &WSEvent{
		Type:           eventType,
		ConversationID: convID,
		Payload:        payloadBytes,
	}
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
