package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/eventbus"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
	"github.com/testcontainers/testcontainers-go"
	tcnats "github.com/testcontainers/testcontainers-go/modules/nats"
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

func TestWebhookIngestionStoresPayloadAndReturns200(t *testing.T) {
	ctx := context.Background()

	// Setup DB
	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	// Create tenant
	tenant, err := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test Corp", Mode: "human_first"})
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	// Create channel
	chRow, err := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Main",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123456"}`),
		Status:      "active",
	})
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	// Setup NATS
	natsC, natsCleanup := startNATS(t)
	defer natsCleanup()
	natsURL, err := natsC.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get nats url: %v", err)
	}
	eb, err := eventbus.New(natsURL, nil)
	if err != nil {
		t.Fatalf("failed to create eventbus: %v", err)
	}
	defer eb.Close()

	// Setup WhatsApp channel
	wa := channel.NewWhatsApp("http://example.com", "token", "123456", "app-secret", "verify-token")

	// Setup handler
	handler := NewHandler(queries, eb, wa, chRow.ID, tenant.ID)

	// Subscribe to NATS before sending webhook
	natsReceived := make(chan eventbus.Event, 1)
	_, _ = eb.Subscribe(ctx, "message.whatsapp.inbound", func(_ context.Context, evt eventbus.Event) error {
		natsReceived <- evt
		return nil
	})

	// Build webhook payload
	payload := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "BUSINESS_ID",
			"changes": [{
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {"display_phone_number": "PHONE", "phone_number_id": "123456"},
					"contacts": [{"profile": {"name": "John"}, "wa_id": "12345"}],
					"messages": [{
						"from": "12345",
						"id": "MSG_001",
						"timestamp": "1234567890",
						"text": {"body": "Hello there"},
						"type": "text"
					}]
				},
				"field": "messages"
			}]
		}]
	}`)

	// Compute HMAC
	mac := hmac.New(sha256.New, []byte("app-secret"))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", bytes.NewReader(payload))
	req.Header.Set("X-Hub-Signature-256", signature)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Serve
	handler.WhatsApp(w, req)

	// Assert 200
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Assert event stored in DB
	events, err := queries.ListUnprocessedWebhookEvents(ctx, 100)
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 unprocessed event, got %d", len(events))
	}

	evt := events[0]
	if evt.TenantID != tenant.ID {
		t.Errorf("expected tenant_id %v, got %v", tenant.ID, evt.TenantID)
	}
	if evt.ChannelType != "whatsapp" {
		t.Errorf("expected channel_type whatsapp, got %s", evt.ChannelType)
	}
	if evt.EventID != "MSG_001" {
		t.Errorf("expected event_id MSG_001, got %s", evt.EventID)
	}

	var storedPayload map[string]interface{}
	if err := json.Unmarshal(evt.RawPayload, &storedPayload); err != nil {
		t.Fatalf("failed to unmarshal raw payload: %v", err)
	}
	if storedPayload["object"] != "whatsapp_business_account" {
		t.Errorf("unexpected payload object: %v", storedPayload["object"])
	}

	// Assert event published to NATS
	select {
	case natsEvt := <-natsReceived:
		if natsEvt.Subject != "message.whatsapp.inbound" {
			t.Errorf("expected subject message.whatsapp.inbound, got %s", natsEvt.Subject)
		}
		if natsEvt.TenantID != tenant.ID {
			t.Errorf("expected tenant_id %v in NATS event, got %v", tenant.ID, natsEvt.TenantID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for NATS event")
	}
}

func TestWebhookIngestionDeduplicatesByEventID(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	chRow, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID: tenant.ID, Name: "WA", ChannelType: "whatsapp",
		Config: []byte(`{}`), Status: "active",
	})

	natsC, natsCleanup := startNATS(t)
	defer natsCleanup()
	natsURL, _ := natsC.ConnectionString(ctx)
	eb, _ := eventbus.New(natsURL, nil)
	defer eb.Close()

	wa := channel.NewWhatsApp("http://example.com", "token", "phone-id", "secret", "verify")
	handler := NewHandler(queries, eb, wa, chRow.ID, tenant.ID)

	payload := []byte(`{"object":"whatsapp_business_account","entry":[{"id":"1","changes":[{"value":{"messages":[{"from":"1","id":"DUP_001","timestamp":"1","text":{"body":"hi"},"type":"text"}]},"field":"messages"}]}]}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(payload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// First request
	req := httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", bytes.NewReader(payload))
	req.Header.Set("X-Hub-Signature-256", sig)
	w := httptest.NewRecorder()
	handler.WhatsApp(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w.Code)
	}

	// Second request (duplicate)
	req = httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", bytes.NewReader(payload))
	req.Header.Set("X-Hub-Signature-256", sig)
	w = httptest.NewRecorder()
	handler.WhatsApp(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("second request: expected 200, got %d", w.Code)
	}

	// Should still be only 1 event in DB
	events, _ := queries.ListUnprocessedWebhookEvents(ctx, 100)
	if len(events) != 1 {
		t.Errorf("expected 1 event after dedup, got %d", len(events))
	}
}

func TestWebhookGETVerification(t *testing.T) {
	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	ctx := context.Background()

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	chRow, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID: tenant.ID, Name: "WA", ChannelType: "whatsapp",
		Config: []byte(`{}`), Status: "active",
	})

	wa := channel.NewWhatsApp("http://example.com", "token", "phone-id", "secret", "my-verify-token")
	handler := NewHandler(queries, nil, wa, chRow.ID, tenant.ID)

	req := httptest.NewRequest(http.MethodGet, "/webhook/whatsapp?hub.mode=subscribe&hub.verify_token=my-verify-token&hub.challenge=challenge-123", nil)
	w := httptest.NewRecorder()
	handler.WhatsApp(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "challenge-123" {
		t.Errorf("expected challenge-123, got %s", w.Body.String())
	}
}

func TestWebhookGETVerification_InvalidToken(t *testing.T) {
	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	ctx := context.Background()

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	chRow, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID: tenant.ID, Name: "WA", ChannelType: "whatsapp",
		Config: []byte(`{}`), Status: "active",
	})

	wa := channel.NewWhatsApp("http://example.com", "token", "phone-id", "secret", "my-verify-token")
	handler := NewHandler(queries, nil, wa, chRow.ID, tenant.ID)

	req := httptest.NewRequest(http.MethodGet, "/webhook/whatsapp?hub.mode=subscribe&hub.verify_token=wrong-token&hub.challenge=challenge-123", nil)
	w := httptest.NewRecorder()
	handler.WhatsApp(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
