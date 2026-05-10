package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/autotraka/go-gateway/internal/eventbus"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	tcnats "github.com/testcontainers/testcontainers-go/modules/nats"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func startNATS(tb testing.TB) (*tcnats.NATSContainer, func()) {
	tb.Helper()

	ctx := context.Background()
	c, err := tcnats.Run(ctx,
		"nats:2-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Server is ready"),
		),
	)
	if err != nil {
		tb.Fatalf("failed to start nats container: %v", err)
	}

	cleanup := func() {
		_ = c.Terminate(ctx)
	}
	return c, cleanup
}

func generateTestToken(tenantID uuid.UUID, secret []byte) string {
	claims := jwt.MapClaims{
		"tenant_id": tenantID.String(),
		"member_id": uuid.New().String(),
		"role":      "admin",
		"exp":       time.Now().Add(15 * time.Minute).Unix(),
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString(secret)
	return s
}

func TestHubRegistersAndUnregistersClient(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		tenantID: tenantID,
	}

	hub.register <- client
	time.Sleep(100 * time.Millisecond)

	hub.mu.RLock()
	if len(hub.clients[tenantID]) != 1 {
		t.Fatalf("expected 1 client for tenant, got %d", len(hub.clients[tenantID]))
	}
	hub.mu.RUnlock()

	hub.unregister <- client
	time.Sleep(100 * time.Millisecond)

	hub.mu.RLock()
	if _, ok := hub.clients[tenantID]; ok {
		t.Fatal("expected client to be unregistered")
	}
	hub.mu.RUnlock()
}

func TestHubBroadcastsToTenantClients(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()

	tenantA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tenantB := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	clientA := &Client{hub: hub, send: make(chan []byte, 256), tenantID: tenantA}
	clientB := &Client{hub: hub, send: make(chan []byte, 256), tenantID: tenantB}

	hub.register <- clientA
	hub.register <- clientB
	time.Sleep(100 * time.Millisecond)

	event := WSEvent{
		Type:           "new_message",
		ConversationID: uuid.New(),
		Payload:        []byte(`{"tenant_id":"11111111-1111-1111-1111-111111111111"}`),
	}
	hub.broadcast <- event

	select {
	case msg := <-clientA.send:
		var received WSEvent
		if err := json.Unmarshal(msg, &received); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if received.Type != "new_message" {
			t.Errorf("expected new_message, got %s", received.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event on client A")
	}

	select {
	case <-clientB.send:
		t.Fatal("client B should not receive event for tenant A")
	case <-time.After(500 * time.Millisecond):
		// expected
	}
}

func TestHandlerRejectsMissingToken(t *testing.T) {
	hub := NewHub(nil)
	handler := NewHandler(hub, []byte("secret"))

	server := httptest.NewServer(handler)
	defer server.Close()

	u, _ := url.Parse(server.URL)
	u.Scheme = "ws"

	_, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err == nil {
		t.Fatal("expected dial to fail without token")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHandlerUpgradesWithValidToken(t *testing.T) {
	ctx := context.Background()

	natsC, natsCleanup := startNATS(t)
	defer natsCleanup()

	natsURL, _ := natsC.ConnectionString(ctx)
	eb, _ := eventbus.New(natsURL, nil)
	defer eb.Close()

	hub := NewHub(eb)
	hub.Run()

	secret := []byte("test-secret")
	handler := NewHandler(hub, secret)

	server := httptest.NewServer(handler)
	defer server.Close()

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	token := generateTestToken(tenantID, secret)

	u, _ := url.Parse(server.URL)
	u.Scheme = "ws"
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()

	ws, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial failed: %v (status: %d)", err, resp.StatusCode)
	}
	defer ws.Close()

	// Wait for NATS consumers to be ready
	time.Sleep(500 * time.Millisecond)

	// Publish a NATS event and verify it arrives on WS
	convID := uuid.New()
	eb.Publish(ctx, "message.inbound", map[string]interface{}{
		"message_id":      uuid.New().String(),
		"conversation_id": convID.String(),
		"tenant_id":       tenantID.String(),
	})

	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read message failed: %v", err)
	}

	var received WSEvent
	if err := json.Unmarshal(msg, &received); err != nil {
		t.Fatalf("failed to unmarshal ws message: %v", err)
	}
	if received.Type != "new_message" {
		t.Errorf("expected type new_message, got %s", received.Type)
	}
}
