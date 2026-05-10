package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/autotraka/go-gateway/internal/auth"
	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/contact"
	"github.com/autotraka/go-gateway/internal/conversation"
	"github.com/autotraka/go-gateway/internal/eventbus"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestIntegrationAPIToWebSocket(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	natsC, natsCleanup := startNATS(t)
	defer natsCleanup()

	natsURL, _ := natsC.ConnectionString(ctx)
	eb, _ := eventbus.New(natsURL, nil)
	defer eb.Close()

	// Setup services
	contactSvc := contact.NewService(queries)
	convSvc := conversation.NewService(queries, contactSvc, eb)

	// Create tenant, channel
	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID: tenant.ID, Name: "WA", ChannelType: "whatsapp",
		Config: []byte(`{}`), Status: "active",
	})

	// Setup WebSocket hub and handler
	hub := NewHub(eb)
	hub.Run()

	secret := []byte("integration-test-secret")
	wsHandler := NewHandler(hub, secret)

	// Middleware to inject tenant context for tests
	tenantID := tenant.ID
	injectTenant := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = auth.WithTenantID(ctx, tenantID)
			ctx = auth.WithMemberID(ctx, uuid.New())
			ctx = auth.WithRole(ctx, "admin")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Setup HTTP router with conversation API
	r := chi.NewRouter()
	r.Use(injectTenant)
	convHandler := conversation.NewHandler(convSvc)
	convHandler.RegisterRoutes(r)
	r.Get("/ws", wsHandler.ServeHTTP)

	server := httptest.NewServer(r)
	defer server.Close()

	// Generate JWT token
	token := generateTestToken(tenant.ID, secret)

	// Connect WebSocket
	wsURL, _ := url.Parse(server.URL)
	wsURL.Scheme = "ws"
	wsURL.Path = "/ws"
	q := wsURL.Query()
	q.Set("token", token)
	wsURL.RawQuery = q.Encode()

	ws, resp, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		t.Fatalf("ws dial failed: %v (status: %d)", err, resp.StatusCode)
	}
	defer ws.Close()

	// Wait for NATS consumers
	time.Sleep(500 * time.Millisecond)

	// Trigger inbound message via service (simulating webhook processing)
	evt := channel.WebhookEvent{
		EventID:   "EVT_WS_001",
		From:      "15551234567",
		MessageID: "MSG_WS_001",
		Type:      "text",
		Content:   []byte(`{"body":"Hello via WS"}`),
		Timestamp: time.Now().Unix(),
	}
	conv, _, err := convSvc.ProcessInboundMessage(ctx, tenant.ID, ch.ID, evt)
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	// Read new_message event on WebSocket
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read ws message failed: %v", err)
	}

	var received WSEvent
	if err := json.Unmarshal(msg, &received); err != nil {
		t.Fatalf("failed to unmarshal ws event: %v", err)
	}
	if received.Type != "new_message" {
		t.Errorf("expected type new_message, got %s", received.Type)
	}
	if received.ConversationID != conv.ID {
		t.Errorf("expected conversation_id %v, got %v", conv.ID, received.ConversationID)
	}

	// Send outbound message via API
	outboundBody := []byte(`{"content":{"text":"Agent reply"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/"+conv.ID.String()+"/messages", bytes.NewReader(outboundBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// The outbound message publishes message.outbound, which doesn't have a WS event type
	// But the conversation update might trigger conversation_updated
	// Let's escalate the conversation to test escalation event
	patchBody := []byte(`{"status":"escalated"}`)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/conversations/"+conv.ID.String(), bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Read events after escalation (may get both conversation_updated and escalation)
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	foundEscalation := false
	for i := 0; i < 3; i++ {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			break
		}
		var evt WSEvent
		if err := json.Unmarshal(msg, &evt); err != nil {
			continue
		}
		if evt.Type == "escalation" {
			foundEscalation = true
			break
		}
	}
	if !foundEscalation {
		t.Error("expected to receive escalation event")
	}
}
