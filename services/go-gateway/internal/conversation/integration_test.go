package conversation

import (
	"context"
	"testing"
	"time"

	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/contact"
	"github.com/autotraka/go-gateway/internal/eventbus"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
)

func TestIntegrationWebhookToStatusUpdate(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	natsC, natsCleanup := startNATS(t)
	defer natsCleanup()
	natsURL, _ := natsC.ConnectionString(ctx)
	eb, _ := eventbus.New(natsURL, nil)
	defer eb.Close()

	contactSvc := contact.NewService(queries)
	convSvc := NewService(queries, contactSvc, eb)

	// Create tenant and channel
	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test Corp", Mode: "human_first"})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Main",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123456"}`),
		Status:      "active",
	})

	// Subscribe to outbound and status update events
	outboundReceived := make(chan eventbus.Event, 1)
	statusReceived := make(chan eventbus.Event, 1)
	_, _ = eb.Subscribe(ctx, "message.outbound", func(_ context.Context, evt eventbus.Event) error {
		outboundReceived <- evt
		return nil
	})
	_, _ = eb.Subscribe(ctx, "message.status_updated", func(_ context.Context, evt eventbus.Event) error {
		statusReceived <- evt
		return nil
	})

	// Wait for NATS consumers to be ready
	time.Sleep(500 * time.Millisecond)

	// Step 1: Webhook inbound message → contact auto-merge → conversation created
	evt := channel.WebhookEvent{
		EventID:   "EVT_001",
		From:      "15551234567",
		MessageID: "MSG_001",
		Type:      "text",
		Content:   []byte(`{"body":"Hello there"}`),
		Timestamp: time.Now().Unix(),
	}

	conv, msg, err := convSvc.ProcessInboundMessage(ctx, tenant.ID, ch.ID, evt)
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}
	if conv.Status != "open" {
		t.Errorf("expected conversation status open, got %s", conv.Status)
	}
	if msg.Direction != "inbound" {
		t.Errorf("expected message direction inbound, got %s", msg.Direction)
	}

	// Verify contact was auto-created
	contacts, _, _ := contactSvc.List(ctx, tenant.ID, 10, 0)
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(contacts))
	}

	// Step 2: Agent sends message → optimistic store → NATS publish
	content := []byte(`{"text":"Hello from agent"}`)
	outboundMsg, err := convSvc.SendMessage(ctx, tenant.ID, conv.ID, SendMessageRequest{
		Content: content,
	})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if outboundMsg.Status != "pending" {
		t.Errorf("expected outbound status pending, got %s", outboundMsg.Status)
	}

	select {
	case evt := <-outboundReceived:
		if evt.Subject != "message.outbound" {
			t.Errorf("expected subject message.outbound, got %s", evt.Subject)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for outbound NATS event")
	}

	// Step 3: Simulate webhook status update (delivered)
	updated, err := convSvc.UpdateMessageStatus(ctx, tenant.ID, outboundMsg.ID, "delivered")
	if err != nil {
		t.Fatalf("UpdateMessageStatus failed: %v", err)
	}
	if updated.Status != "delivered" {
		t.Errorf("expected status delivered, got %s", updated.Status)
	}

	select {
	case evt := <-statusReceived:
		if evt.Subject != "message.status_updated" {
			t.Errorf("expected subject message.status_updated, got %s", evt.Subject)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for status update NATS event")
	}

	// Step 4: Verify conversation PATCH works
	updatedConv, err := convSvc.Update(ctx, tenant.ID, conv.ID, UpdateConversationRequest{
		Status:    strPtr("pending"),
		HandledBy: strPtr("human"),
	})
	if err != nil {
		t.Fatalf("Update conversation failed: %v", err)
	}
	if updatedConv.Status != "pending" {
		t.Errorf("expected status pending, got %s", updatedConv.Status)
	}
	if updatedConv.HandledBy != "human" {
		t.Errorf("expected handled_by human, got %s", updatedConv.HandledBy)
	}

	// Step 5: Verify GET with messages and contact
	fullConv, err := convSvc.Get(ctx, tenant.ID, conv.ID)
	if err != nil {
		t.Fatalf("Get conversation failed: %v", err)
	}
	if len(fullConv.Messages) < 2 {
		t.Errorf("expected at least 2 messages, got %d", len(fullConv.Messages))
	}
	if fullConv.Contact == nil {
		t.Fatal("expected contact to be loaded")
	}

	// Step 6: Close conversation and verify new one created on re-message
	_, _ = convSvc.Update(ctx, tenant.ID, conv.ID, UpdateConversationRequest{Status: strPtr("closed")})

	evt2 := channel.WebhookEvent{
		EventID:   "EVT_002",
		From:      "15551234567",
		MessageID: "MSG_002",
		Type:      "text",
		Content:   []byte(`{"body":"Follow up"}`),
		Timestamp: time.Now().Unix(),
	}
	conv2, _, _ := convSvc.ProcessInboundMessage(ctx, tenant.ID, ch.ID, evt2)
	if conv2.ID == conv.ID {
		t.Error("expected new conversation after close")
	}
	if conv2.PreviousConversationID == nil || *conv2.PreviousConversationID != conv.ID {
		t.Errorf("expected previous_conversation_id link, got %v", conv2.PreviousConversationID)
	}
}
