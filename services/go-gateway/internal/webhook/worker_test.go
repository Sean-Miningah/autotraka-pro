package webhook

import (
	"context"
	"testing"
	"time"

	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/eventbus"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestWorkerPublishesUnprocessedEvents(t *testing.T) {
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

	// Subscribe to NATS to confirm event is published
	received := make(chan eventbus.Event, 1)
	_, _ = eb.Subscribe(ctx, "message.whatsapp.inbound", func(_ context.Context, evt eventbus.Event) error {
		received <- evt
		return nil
	})

	// Insert unprocessed webhook event
	_, err := queries.CreateWebhookEvent(ctx, sqlcgen.CreateWebhookEventParams{
		TenantID:    tenant.ID,
		ChannelID:   pgtype.UUID{Bytes: chRow.ID, Valid: true},
		ChannelType: "whatsapp",
		EventID:     "WORKER_MSG_001",
		RawPayload:  []byte(`{"test":"worker"}`),
	})
	if err != nil {
		t.Fatalf("failed to create webhook event: %v", err)
	}

	// Start worker
	wa := channel.NewWhatsApp("http://example.com", "token", "phone-id", "secret", "verify")
	w := NewWorker(queries, eb, wa)
	go w.Run(ctx, 100*time.Millisecond)
	defer w.Stop()

	// Wait for event to be processed
	select {
	case evt := <-received:
		if evt.Subject != "message.whatsapp.inbound" {
			t.Errorf("expected subject message.whatsapp.inbound, got %s", evt.Subject)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker to publish event")
	}

	// Assert event is marked processed (poll with timeout)
	var unprocessed []sqlcgen.WebhookEvent
	for i := 0; i < 50; i++ {
		unprocessed, _ = queries.ListUnprocessedWebhookEvents(ctx, 100)
		if len(unprocessed) == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if len(unprocessed) != 0 {
		t.Errorf("expected 0 unprocessed events, got %d", len(unprocessed))
	}
}
