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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

func setupService(tb testing.TB) (*Service, *eventbus.Client, func()) {
	tb.Helper()

	ctx := context.Background()
	pool, dbCleanup := testutil.SetupTestDB(tb)

	queries := sqlcgen.New(pool)
	contactSvc := contact.NewService(queries)

	natsC, natsCleanup := startNATS(tb)
	natsURL, _ := natsC.ConnectionString(ctx)
	eb, _ := eventbus.New(natsURL, nil)

	svc := NewService(queries, contactSvc, nil, nil, eb)

	cleanup := func() {
		eb.Close()
		natsCleanup()
		dbCleanup()
	}

	return svc, eb, cleanup
}

func createTenantAndChannel(ctx context.Context, t *testing.T, queries *sqlcgen.Queries) (uuid.UUID, uuid.UUID) {
	tenant, err := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test Corp", Mode: "human_first"})
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

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

	return tenant.ID, chRow.ID
}

func TestProcessInboundMessageCreatesConversationAndContact(t *testing.T) {
	ctx := context.Background()
	svc, _, cleanup := setupService(t)
	defer cleanup()

	tenantID, channelID := createTenantAndChannel(ctx, t, svc.queries)

	evt := channel.WebhookEvent{
		EventID:   "EVT_001",
		From:      "15551234567",
		MessageID: "MSG_001",
		Type:      "text",
		Content:   []byte(`{"body":"Hello there"}`),
		Timestamp: time.Now().Unix(),
	}

	conv, msg, err := svc.ProcessInboundMessage(ctx, tenantID, channelID, evt)
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	if conv.ID == uuid.Nil {
		t.Fatal("expected conversation to be created")
	}
	if conv.Status != "open" {
		t.Errorf("expected status open, got %s", conv.Status)
	}
	if conv.HandledBy != "human" {
		t.Errorf("expected handled_by human for human_first tenant, got %s", conv.HandledBy)
	}

	if msg.ID == uuid.Nil {
		t.Fatal("expected message to be created")
	}
	if msg.Direction != "inbound" {
		t.Errorf("expected direction inbound, got %s", msg.Direction)
	}
	if msg.Status != "delivered" {
		t.Errorf("expected status delivered, got %s", msg.Status)
	}
	if msg.ChannelID == nil || *msg.ChannelID != channelID {
		t.Errorf("expected channel_id %v, got %v", channelID, msg.ChannelID)
	}

	// Verify contact was auto-created
	contacts, _, _ := svc.contactSvc.List(ctx, tenantID, 10, 0)
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(contacts))
	}
	if len(contacts[0].Phones) != 1 || contacts[0].Phones[0].Phone != "+15551234567" {
		t.Errorf("expected phone +15551234567, got %v", contacts[0].Phones)
	}
}

func TestProcessInboundMessageReusesOpenConversation(t *testing.T) {
	ctx := context.Background()
	svc, _, cleanup := setupService(t)
	defer cleanup()

	tenantID, channelID := createTenantAndChannel(ctx, t, svc.queries)

	// First message
	evt1 := channel.WebhookEvent{EventID: "EVT_001", From: "15551234567", MessageID: "MSG_001", Type: "text", Content: []byte(`{"body":"Hello"}`), Timestamp: time.Now().Unix()}
	conv1, _, _ := svc.ProcessInboundMessage(ctx, tenantID, channelID, evt1)

	// Second message from same contact
	evt2 := channel.WebhookEvent{EventID: "EVT_002", From: "15551234567", MessageID: "MSG_002", Type: "text", Content: []byte(`{"body":"World"}`), Timestamp: time.Now().Unix()}
	conv2, _, _ := svc.ProcessInboundMessage(ctx, tenantID, channelID, evt2)

	if conv1.ID != conv2.ID {
		t.Errorf("expected same conversation, got %v and %v", conv1.ID, conv2.ID)
	}

	// Verify 2 messages in conversation
	messages, _ := svc.queries.ListMessagesByConversation(ctx, sqlcgen.ListMessagesByConversationParams{
		ConversationID: conv1.ID,
		TenantID:       tenantID,
		Limit:          100,
		Offset:         0,
	})
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}

func TestProcessInboundMessageCreatesNewAfterClose(t *testing.T) {
	ctx := context.Background()
	svc, _, cleanup := setupService(t)
	defer cleanup()

	tenantID, channelID := createTenantAndChannel(ctx, t, svc.queries)

	// First message
	evt1 := channel.WebhookEvent{EventID: "EVT_001", From: "15551234567", MessageID: "MSG_001", Type: "text", Content: []byte(`{"body":"Hello"}`), Timestamp: time.Now().Unix()}
	conv1, _, _ := svc.ProcessInboundMessage(ctx, tenantID, channelID, evt1)

	// Close the conversation
	svc.Update(ctx, tenantID, conv1.ID, UpdateConversationRequest{Status: strPtr("closed")})

	// Second message from same contact after close
	evt2 := channel.WebhookEvent{EventID: "EVT_002", From: "15551234567", MessageID: "MSG_002", Type: "text", Content: []byte(`{"body":"World"}`), Timestamp: time.Now().Unix()}
	conv2, _, _ := svc.ProcessInboundMessage(ctx, tenantID, channelID, evt2)

	if conv1.ID == conv2.ID {
		t.Error("expected new conversation after close, got same conversation")
	}
	if conv2.PreviousConversationID == nil || *conv2.PreviousConversationID != conv1.ID {
		t.Errorf("expected previous_conversation_id %v, got %v", conv1.ID, conv2.PreviousConversationID)
	}
}

func TestSendMessageOptimistic(t *testing.T) {
	ctx := context.Background()
	svc, eb, cleanup := setupService(t)
	defer cleanup()

	tenantID, channelID := createTenantAndChannel(ctx, t, svc.queries)

	// Create contact and conversation
	contactSvc := contact.NewService(svc.queries)
	c, _ := contactSvc.Create(ctx, tenantID, contact.CreateRequest{
		Name:   "John",
		Phones: []contact.PhoneInput{{Phone: "+15551234567"}},
	})
	conv, _ := svc.queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:       tenantID,
		ContactID:      c.ID,
		Status:         sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:      sqlcgen.HandledByAi,
	})

	// Create an inbound message to set the reply channel
	_, _ = svc.queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenantID,
		ConversationID: conv.ID,
		ChannelID:      pgtype.UUID{Bytes: channelID, Valid: true},
		Direction:      sqlcgen.MessageDirectionInbound,
		Status:         sqlcgen.MessageStatusDelivered,
		ContentType:    "text",
		Content:        []byte(`{"text":"hello"}`),
	})

	// Subscribe to outbound messages
	received := make(chan eventbus.Event, 1)
	_, _ = eb.Subscribe(ctx, "message.outbound", func(_ context.Context, evt eventbus.Event) error {
		received <- evt
		return nil
	})

	// Send outbound message
	content := []byte(`{"text":"Hello from agent"}`)
	msg, err := svc.SendMessage(ctx, tenantID, conv.ID, SendMessageRequest{
		Content: content,
	})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if msg.Status != "pending" {
		t.Errorf("expected status pending, got %s", msg.Status)
	}
	if msg.Direction != "outbound" {
		t.Errorf("expected direction outbound, got %s", msg.Direction)
	}
	if msg.ChannelID == nil || *msg.ChannelID != channelID {
		t.Errorf("expected channel_id %v, got %v", channelID, msg.ChannelID)
	}

	// Assert NATS event published
	select {
	case evt := <-received:
		if evt.Subject != "message.outbound" {
			t.Errorf("expected subject message.outbound, got %s", evt.Subject)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for outbound NATS event")
	}
}

func TestSendMessageWithChannelOverride(t *testing.T) {
	ctx := context.Background()
	svc, _, cleanup := setupService(t)
	defer cleanup()

	tenantID, _ := createTenantAndChannel(ctx, t, svc.queries)

	// Create second channel
	ch2, _ := svc.queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenantID,
		Name:        "WhatsApp Secondary",
		ChannelType: "whatsapp",
		Config:      []byte(`{}`),
		Status:      "active",
	})

	contactSvc := contact.NewService(svc.queries)
	c, _ := contactSvc.Create(ctx, tenantID, contact.CreateRequest{
		Name:   "John",
		Phones: []contact.PhoneInput{{Phone: "+15551234567"}},
	})
	conv, _ := svc.queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:       tenantID,
		ContactID:      c.ID,
		Status:         sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:      sqlcgen.HandledByAi,
	})

	// No inbound messages - override should be used
	content := []byte(`{"text":"Hello"}`)
	msg, err := svc.SendMessage(ctx, tenantID, conv.ID, SendMessageRequest{
		Content:   content,
		ChannelID: &ch2.ID,
	})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if msg.ChannelID == nil || *msg.ChannelID != ch2.ID {
		t.Errorf("expected channel_id %v, got %v", ch2.ID, msg.ChannelID)
	}
}

func TestConversationStatusLifecycle(t *testing.T) {
	ctx := context.Background()
	svc, _, cleanup := setupService(t)
	defer cleanup()

	tenantID, _ := createTenantAndChannel(ctx, t, svc.queries)

	contactSvc := contact.NewService(svc.queries)
	c, _ := contactSvc.Create(ctx, tenantID, contact.CreateRequest{Name: "John"})
	conv, _ := svc.queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:       tenantID,
		ContactID:      c.ID,
		Status:         sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:      sqlcgen.HandledByAi,
	})

	// open -> pending (valid)
	updated, err := svc.Update(ctx, tenantID, conv.ID, UpdateConversationRequest{Status: strPtr("pending")})
	if err != nil {
		t.Fatalf("open->pending failed: %v", err)
	}
	if updated.Status != "pending" {
		t.Errorf("expected pending, got %s", updated.Status)
	}

	// pending -> escalated (valid)
	updated, err = svc.Update(ctx, tenantID, conv.ID, UpdateConversationRequest{Status: strPtr("escalated")})
	if err != nil {
		t.Fatalf("pending->escalated failed: %v", err)
	}
	if updated.Status != "escalated" {
		t.Errorf("expected escalated, got %s", updated.Status)
	}

	// escalated -> open (invalid)
	_, err = svc.Update(ctx, tenantID, conv.ID, UpdateConversationRequest{Status: strPtr("open")})
	if err == nil {
		t.Fatal("expected error for invalid transition escalated->open")
	}

	// escalated -> resolved (valid)
	updated, err = svc.Update(ctx, tenantID, conv.ID, UpdateConversationRequest{Status: strPtr("resolved")})
	if err != nil {
		t.Fatalf("escalated->resolved failed: %v", err)
	}
	if updated.Status != "resolved" {
		t.Errorf("expected resolved, got %s", updated.Status)
	}

	// resolved -> closed (valid)
	updated, err = svc.Update(ctx, tenantID, conv.ID, UpdateConversationRequest{Status: strPtr("closed")})
	if err != nil {
		t.Fatalf("resolved->closed failed: %v", err)
	}
	if updated.Status != "closed" {
		t.Errorf("expected closed, got %s", updated.Status)
	}

	// closed -> anything (invalid)
	_, err = svc.Update(ctx, tenantID, conv.ID, UpdateConversationRequest{Status: strPtr("open")})
	if err == nil {
		t.Fatal("expected error for invalid transition closed->open")
	}
}

func TestUpdateMessageStatus(t *testing.T) {
	ctx := context.Background()
	svc, eb, cleanup := setupService(t)
	defer cleanup()

	tenantID, channelID := createTenantAndChannel(ctx, t, svc.queries)
	contactSvc := contact.NewService(svc.queries)
	c, _ := contactSvc.Create(ctx, tenantID, contact.CreateRequest{Name: "John"})
	conv, _ := svc.queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:       tenantID,
		ContactID:      c.ID,
		Status:         sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:      sqlcgen.HandledByAi,
	})

	msg, _ := svc.queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenantID,
		ConversationID: conv.ID,
		ChannelID:      pgtype.UUID{Bytes: channelID, Valid: true},
		Direction:      sqlcgen.MessageDirectionOutbound,
		Status:         sqlcgen.MessageStatusPending,
		ContentType:    "text",
		Content:        []byte(`{"text":"hello"}`),
	})

	// Subscribe to status updates
	received := make(chan eventbus.Event, 1)
	_, _ = eb.Subscribe(ctx, "message.status_updated", func(_ context.Context, evt eventbus.Event) error {
		received <- evt
		return nil
	})

	updated, err := svc.UpdateMessageStatus(ctx, tenantID, msg.ID, "delivered")
	if err != nil {
		t.Fatalf("UpdateMessageStatus failed: %v", err)
	}
	if updated.Status != "delivered" {
		t.Errorf("expected delivered, got %s", updated.Status)
	}

	select {
	case evt := <-received:
		if evt.Subject != "message.status_updated" {
			t.Errorf("expected subject message.status_updated, got %s", evt.Subject)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for status update NATS event")
	}
}

func TestListConversationsScopedToTenant(t *testing.T) {
	ctx := context.Background()
	svc, _, cleanup := setupService(t)
	defer cleanup()

	tenantA, _ := createTenantAndChannel(ctx, t, svc.queries)
	tenantB, _ := createTenantAndChannel(ctx, t, svc.queries)

	contactSvc := contact.NewService(svc.queries)
	cA, _ := contactSvc.Create(ctx, tenantA, contact.CreateRequest{Name: "TenantA"})
	cB, _ := contactSvc.Create(ctx, tenantB, contact.CreateRequest{Name: "TenantB"})

	_, _ = svc.queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:       tenantA,
		ContactID:      cA.ID,
		Status:         sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:      sqlcgen.HandledByAi,
	})
	_, _ = svc.queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:       tenantB,
		ContactID:      cB.ID,
		Status:         sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:      sqlcgen.HandledByAi,
	})

	convs, count, err := svc.List(ctx, tenantA, 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1 for tenant A, got %d", count)
	}
	if len(convs) != 1 {
		t.Errorf("expected 1 conversation for tenant A, got %d", len(convs))
	}
}

func TestGetConversationWithMessagesAndContact(t *testing.T) {
	ctx := context.Background()
	svc, _, cleanup := setupService(t)
	defer cleanup()

	tenantID, channelID := createTenantAndChannel(ctx, t, svc.queries)

	contactSvc := contact.NewService(svc.queries)
	c, _ := contactSvc.Create(ctx, tenantID, contact.CreateRequest{
		Name:   "John",
		Phones: []contact.PhoneInput{{Phone: "+15551234567"}},
	})
	conv, _ := svc.queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:       tenantID,
		ContactID:      c.ID,
		Status:         sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:      sqlcgen.HandledByAi,
	})

	_, _ = svc.queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenantID,
		ConversationID: conv.ID,
		ChannelID:      pgtype.UUID{Bytes: channelID, Valid: true},
		Direction:      sqlcgen.MessageDirectionInbound,
		Status:         sqlcgen.MessageStatusDelivered,
		ContentType:    "text",
		Content:        []byte(`{"text":"hello"}`),
	})

	result, err := svc.Get(ctx, tenantID, conv.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}
	if result.Contact == nil {
		t.Fatal("expected contact to be loaded")
	}
	if result.Contact.Name != "John" {
		t.Errorf("expected contact name John, got %s", result.Contact.Name)
	}
}

func TestGetConversationWrongTenantReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	svc, _, cleanup := setupService(t)
	defer cleanup()

	tenantA, _ := createTenantAndChannel(ctx, t, svc.queries)
	tenantB, _ := createTenantAndChannel(ctx, t, svc.queries)

	contactSvc := contact.NewService(svc.queries)
	c, _ := contactSvc.Create(ctx, tenantA, contact.CreateRequest{Name: "John"})
	conv, _ := svc.queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:       tenantA,
		ContactID:      c.ID,
		Status:         sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:      sqlcgen.HandledByAi,
	})

	_, err := svc.Get(ctx, tenantB, conv.ID)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func strPtr(s string) *string {
	return &s
}
