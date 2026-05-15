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
)

func TestInstagramWebhookIngestionStoresPayloadAndPublishesToNATS(t *testing.T) {
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

	// Create Instagram channel
	chRow, err := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "Instagram Main",
		ChannelType: "instagram",
		Config:      []byte(`{"instagram_account_id":"123456"}`),
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

	// Setup Instagram channel
	ig := channel.NewInstagram("http://example.com", "token", "123456", "app-secret", "verify-token")

	// Setup handler
	handler := NewHandler(queries, eb, ig, chRow.ID, tenant.ID)

	// Subscribe to NATS before sending webhook
	natsReceived := make(chan eventbus.Event, 1)
	_, _ = eb.Subscribe(ctx, "message.instagram.inbound", func(_ context.Context, evt eventbus.Event) error {
		natsReceived <- evt
		return nil
	})

	// Build Instagram webhook payload
	payload := []byte(`{
		"object": "instagram",
		"entry": [{
			"id": "IG_ACCOUNT_ID",
			"time": 1234567890,
			"messaging": [{
				"sender": {"id": "IG_SENDER_001"},
				"recipient": {"id": "IG_ACCOUNT_ID"},
				"timestamp": 1234567890,
				"message": {
					"mid": "IG_MSG_001",
					"text": "Hello from Instagram"
				}
			}]
		}]
	}`)

	// Compute HMAC
	mac := hmac.New(sha256.New, []byte("app-secret"))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/webhook/instagram", bytes.NewReader(payload))
	req.Header.Set("X-Hub-Signature-256", signature)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Serve
	handler.Instagram(w, req)

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
	if evt.ChannelType != "instagram" {
		t.Errorf("expected channel_type instagram, got %s", evt.ChannelType)
	}
	if evt.EventID != "IG_MSG_001" {
		t.Errorf("expected event_id IG_MSG_001, got %s", evt.EventID)
	}

	var storedPayload map[string]interface{}
	if err := json.Unmarshal(evt.RawPayload, &storedPayload); err != nil {
		t.Fatalf("failed to unmarshal raw payload: %v", err)
	}
	if storedPayload["object"] != "instagram" {
		t.Errorf("unexpected payload object: %v", storedPayload["object"])
	}

	// Assert event published to NATS with instagram subject
	select {
	case natsEvt := <-natsReceived:
		if natsEvt.Subject != "message.instagram.inbound" {
			t.Errorf("expected subject message.instagram.inbound, got %s", natsEvt.Subject)
		}
		if natsEvt.TenantID != tenant.ID {
			t.Errorf("expected tenant_id %v in NATS event, got %v", tenant.ID, natsEvt.TenantID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for NATS event")
	}
}

func TestInstagramWebhookGETVerification(t *testing.T) {
	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	ctx := context.Background()

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	chRow, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID: tenant.ID, Name: "IG", ChannelType: "instagram",
		Config: []byte(`{}`), Status: "active",
	})

	ig := channel.NewInstagram("http://example.com", "token", "account-id", "secret", "my-verify-token")
	handler := NewHandler(queries, nil, ig, chRow.ID, tenant.ID)

	req := httptest.NewRequest(http.MethodGet, "/webhook/instagram?hub.mode=subscribe&hub.verify_token=my-verify-token&hub.challenge=challenge-123", nil)
	w := httptest.NewRecorder()
	handler.Instagram(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "challenge-123" {
		t.Errorf("expected challenge-123, got %s", w.Body.String())
	}
}

func TestInstagramWebhookGETVerification_InvalidToken(t *testing.T) {
	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	ctx := context.Background()

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	chRow, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID: tenant.ID, Name: "IG", ChannelType: "instagram",
		Config: []byte(`{}`), Status: "active",
	})

	ig := channel.NewInstagram("http://example.com", "token", "account-id", "secret", "my-verify-token")
	handler := NewHandler(queries, nil, ig, chRow.ID, tenant.ID)

	req := httptest.NewRequest(http.MethodGet, "/webhook/instagram?hub.mode=subscribe&hub.verify_token=wrong-token&hub.challenge=challenge-123", nil)
	w := httptest.NewRecorder()
	handler.Instagram(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
