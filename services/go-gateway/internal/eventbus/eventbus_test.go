package eventbus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestPublishAndSubscribeReceivesEventWithCorrectEnvelope(t *testing.T) {
	ctx := context.Background()

	c, cleanup := startNATS(t)
	defer cleanup()

	url, err := c.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	client, err := New(url, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	received := make(chan Event, 1)

	_, err = client.Subscribe(ctx, "message.whatsapp.inbound", func(_ context.Context, evt Event) error {
		received <- evt
		return nil
	})
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	payload := map[string]string{"text": "hello"}
	if err := client.Publish(ctx, "message.whatsapp.inbound", payload); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	select {
	case evt := <-received:
		if evt.Subject != "message.whatsapp.inbound" {
			t.Errorf("expected subject message.whatsapp.inbound, got %s", evt.Subject)
		}
		if evt.CorrelationID == "" {
			t.Error("expected correlation_id to be set")
		}
		if evt.Timestamp.IsZero() {
			t.Error("expected timestamp to be set")
		}
		if evt.TenantID != uuid.Nil {
			t.Errorf("expected tenant_id to be nil when not in context, got %v", evt.TenantID)
		}

		var got map[string]string
		if err := json.Unmarshal(evt.Payload, &got); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		if got["text"] != "hello" {
			t.Errorf("expected payload text hello, got %s", got["text"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestCorrelationIDPropagatesThroughEnvelope(t *testing.T) {
	ctx := context.Background()

	c, cleanup := startNATS(t)
	defer cleanup()

	url, err := c.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	client, err := New(url, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	received := make(chan Event, 1)

	_, err = client.Subscribe(ctx, "conversation.created", func(_ context.Context, evt Event) error {
		received <- evt
		return nil
	})
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	ctx = WithCorrelationID(ctx, "trace-123")
	if err := client.Publish(ctx, "conversation.created", map[string]string{"id": "1"}); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	select {
	case evt := <-received:
		if evt.CorrelationID != "trace-123" {
			t.Errorf("expected correlation_id trace-123, got %s", evt.CorrelationID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestTenantIDPropagatesThroughEnvelope(t *testing.T) {
	ctx := context.Background()

	c, cleanup := startNATS(t)
	defer cleanup()

	url, err := c.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	client, err := New(url, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	received := make(chan Event, 1)

	_, err = client.Subscribe(ctx, "ai.request", func(_ context.Context, evt Event) error {
		received <- evt
		return nil
	})
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ctx = WithTenantID(ctx, tenantID)
	if err := client.Publish(ctx, "ai.request", map[string]string{"prompt": "hi"}); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	select {
	case evt := <-received:
		if evt.TenantID != tenantID {
			t.Errorf("expected tenant_id %v, got %v", tenantID, evt.TenantID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestClientClosesGracefully(t *testing.T) {
	ctx := context.Background()

	c, cleanup := startNATS(t)
	defer cleanup()

	url, err := c.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	client, err := New(url, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("expected close to succeed, got %v", err)
	}

	// After close, publish should fail because connection is drained.
	if err := client.Publish(ctx, "message.whatsapp.inbound", map[string]string{}); err == nil {
		t.Error("expected publish to fail after close")
	}
}
