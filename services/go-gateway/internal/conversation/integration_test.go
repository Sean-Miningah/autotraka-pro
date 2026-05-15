package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/contact"
	"github.com/autotraka/go-gateway/internal/eventbus"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/template"
	"github.com/autotraka/go-gateway/internal/testutil"
	"github.com/google/uuid"
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
	convSvc := NewService(queries, contactSvc, nil, nil, eb)

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

// mockChannel records SendTemplateMessage calls for verification.
type mockChannel struct {
	calls []mockChannelCall
}

type mockChannelCall struct {
	Method       string
	To           string
	TemplateName string
	Language     string
	Params       []string
}

func (m *mockChannel) ChannelType() string { return "whatsapp" }
func (m *mockChannel) SendTextMessage(ctx context.Context, to, body string) error {
	m.calls = append(m.calls, mockChannelCall{Method: "SendTextMessage", To: to})
	return nil
}
func (m *mockChannel) SendTemplateMessage(ctx context.Context, to, templateName, language string, params []string) error {
	m.calls = append(m.calls, mockChannelCall{Method: "SendTemplateMessage", To: to, TemplateName: templateName, Language: language, Params: params})
	return nil
}
func (m *mockChannel) SendMediaMessage(ctx context.Context, to, mediaType, mediaURL, caption string) error {
	return nil
}
func (m *mockChannel) MarkRead(ctx context.Context, messageID string) error {
	return nil
}
func (m *mockChannel) VerifyWebhook(mode, verifyToken, challenge string) (string, error) {
	return "", nil
}
func (m *mockChannel) VerifySignature(payload []byte, signature string) error {
	return nil
}
func (m *mockChannel) ParseWebhookEvent(payload []byte) (channel.WebhookEvent, error) {
	return channel.WebhookEvent{}, nil
}

func TestIntegrationSendTemplateMessage(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	contactSvc := contact.NewService(queries)
	templateSvc := template.NewService(queries, nil)
	mockCh := &mockChannel{}

	convSvc := NewService(queries, contactSvc, templateSvc, mockCh, nil)

	// Create tenant, channel, and contact with phone
	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test Corp", Mode: "human_first"})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Main",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123456"}`),
		Status:      "active",
	})

	_, _ = contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	// Create an approved template
	tmpl, _ := templateSvc.Create(ctx, tenant.ID, template.CreateRequest{
		ChannelID:  ch.ID,
		Name:       "welcome_v1",
		Category:   "MARKETING",
		Language:   "en",
		Body:       "Hello {{1}}, welcome to {{2}}!",
		Parameters: []template.ParameterDef{{Name: "name", DisplayName: "Name"}, {Name: "company", DisplayName: "Company"}},
	})
	// Mark as approved for sending
	approved := "approved"
	tmpl, _ = templateSvc.Update(ctx, tenant.ID, tmpl.ID, template.UpdateRequest{Status: &approved})

	// Create a conversation
	conv, _, _ := convSvc.ProcessInboundMessage(ctx, tenant.ID, ch.ID, channel.WebhookEvent{
		EventID:   "EVT_001",
		From:      "1234567890",
		MessageID: "MSG_001",
		Type:      "text",
		Content:   []byte(`{"body":"Hello"}`),
		Timestamp: time.Now().Unix(),
	})

	// Send template message
	templateID := tmpl.ID
	msg, err := convSvc.SendMessage(ctx, tenant.ID, conv.ID, SendMessageRequest{
		TemplateID: &templateID,
		Parameters: map[string]string{"name": "Alice", "company": "Acme"},
	})
	if err != nil {
		t.Fatalf("SendMessage with template failed: %v", err)
	}
	if msg.ContentType != "template" {
		t.Errorf("expected content_type template, got %s", msg.ContentType)
	}

	// Verify content stores parameter mapping
	var storedParams map[string]string
	if err := json.Unmarshal(msg.Content, &storedParams); err != nil {
		t.Fatalf("failed to unmarshal content: %v", err)
	}
	if storedParams["name"] != "Alice" || storedParams["company"] != "Acme" {
		t.Errorf("unexpected stored params: %v", storedParams)
	}

	// Verify channel received the template send with positional params
	if len(mockCh.calls) != 1 {
		t.Fatalf("expected 1 channel call, got %d", len(mockCh.calls))
	}
	call := mockCh.calls[0]
	if call.Method != "SendTemplateMessage" {
		t.Errorf("expected SendTemplateMessage, got %s", call.Method)
	}
	if call.TemplateName != "welcome_v1" {
		t.Errorf("expected template name welcome_v1, got %s", call.TemplateName)
	}
	if call.Language != "en" {
		t.Errorf("expected language en, got %s", call.Language)
	}
	if len(call.Params) != 2 || call.Params[0] != "Alice" || call.Params[1] != "Acme" {
		t.Errorf("unexpected positional params: %v", call.Params)
	}
}

func TestIntegrationSendTemplateMessageMissingParameter(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	contactSvc := contact.NewService(queries)
	templateSvc := template.NewService(queries, nil)
	mockCh := &mockChannel{}

	convSvc := NewService(queries, contactSvc, templateSvc, mockCh, nil)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test Corp", Mode: "human_first"})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Main",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123456"}`),
		Status:      "active",
	})

	contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	tmpl, _ := templateSvc.Create(ctx, tenant.ID, template.CreateRequest{
		ChannelID:  ch.ID,
		Name:       "welcome_v1",
		Category:   "MARKETING",
		Language:   "en",
		Body:       "Hello {{1}}!",
		Parameters: []template.ParameterDef{{Name: "name", DisplayName: "Name"}},
	})
	approved := "approved"
	tmpl, _ = templateSvc.Update(ctx, tenant.ID, tmpl.ID, template.UpdateRequest{Status: &approved})

	conv, _, _ := convSvc.ProcessInboundMessage(ctx, tenant.ID, ch.ID, channel.WebhookEvent{
		EventID:   "EVT_001",
		From:      "1234567890",
		MessageID: "MSG_001",
		Type:      "text",
		Content:   []byte(`{"body":"Hello"}`),
		Timestamp: time.Now().Unix(),
	})

	templateID := tmpl.ID
	_, err := convSvc.SendMessage(ctx, tenant.ID, conv.ID, SendMessageRequest{
		TemplateID: &templateID,
		Parameters: map[string]string{}, // missing "name"
	})
	if err == nil {
		t.Fatal("expected error for missing parameter")
	}
	if !errors.Is(err, ErrMissingParameters) {
		t.Errorf("expected ErrMissingParameters, got %v", err)
	}
}

func TestIntegrationAIFirstMode(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	natsC, natsCleanup := startNATS(t)
	defer natsCleanup()
	natsURL, _ := natsC.ConnectionString(ctx)
	eb, _ := eventbus.New(natsURL, nil)
	defer eb.Close()

	mockCh := &mockChannel{}
	contactSvc := contact.NewService(queries)
	convSvc := NewService(queries, contactSvc, nil, mockCh, eb)

	// Start AI consumers
	if err := convSvc.StartAIConsumers(ctx); err != nil {
		t.Fatalf("StartAIConsumers failed: %v", err)
	}

	// Create tenant in AI-first mode
	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "AI Corp", Mode: "ai_first"})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Main",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123456"}`),
		Status:      "active",
	})

	// Subscribe to message.received events
	receivedCh := make(chan eventbus.Event, 1)
	_, _ = eb.Subscribe(ctx, "message.received", func(_ context.Context, evt eventbus.Event) error {
		receivedCh <- evt
		return nil
	})

	// Wait for NATS consumers to be ready
	time.Sleep(500 * time.Millisecond)

	// Process inbound message in AI-first mode
	evt := channel.WebhookEvent{
		EventID:   "EVT_AI_001",
		From:      "15551234567",
		MessageID: "MSG_AI_001",
		Type:      "text",
		Content:   []byte(`{"body":"Hello AI"}`),
		Timestamp: time.Now().Unix(),
	}

	conv, msg, err := convSvc.ProcessInboundMessage(ctx, tenant.ID, ch.ID, evt)
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}
	if conv.HandledBy != "ai" {
		t.Errorf("expected handled_by ai, got %s", conv.HandledBy)
	}
	if msg.Direction != "inbound" {
		t.Errorf("expected message direction inbound, got %s", msg.Direction)
	}

	// Verify message.received was published
	select {
	case evt := <-receivedCh:
		if evt.Subject != "message.received" {
			t.Errorf("expected subject message.received, got %s", evt.Subject)
		}
		var payload MessageReceivedPayload
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		if payload.ConversationID != conv.ID {
			t.Errorf("expected conversation_id %s, got %s", conv.ID, payload.ConversationID)
		}
		if payload.Body != "Hello AI" {
			t.Errorf("expected body 'Hello AI', got %s", payload.Body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for message.received event")
	}

	// Simulate AI response
	aiResponse := AIResponsePayload{
		ConversationID: conv.ID,
		TenantID:       tenant.ID,
		Body:           "Hello from AI",
	}
	aiRespBytes, _ := json.Marshal(aiResponse)
	eb.Publish(ctx, "message.ai_response", json.RawMessage(aiRespBytes))

	// Wait for AI response to be processed
	time.Sleep(500 * time.Millisecond)

	// Verify AI response message was stored and sent
	fullConv, err := convSvc.Get(ctx, tenant.ID, conv.ID)
	if err != nil {
		t.Fatalf("Get conversation failed: %v", err)
	}
	if len(fullConv.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(fullConv.Messages))
	}

	var aiMsg *Message
	for i := range fullConv.Messages {
		if fullConv.Messages[i].Direction == "outbound" {
			aiMsg = &fullConv.Messages[i]
			break
		}
	}
	if aiMsg == nil {
		t.Fatal("expected AI outbound message")
	}
	if aiMsg.Status != "sent" {
		t.Errorf("expected AI message status sent, got %s", aiMsg.Status)
	}

	// Verify channel received the send
	if len(mockCh.calls) != 1 {
		t.Fatalf("expected 1 channel call, got %d", len(mockCh.calls))
	}
	if mockCh.calls[0].Method != "SendTextMessage" {
		t.Errorf("expected SendTextMessage, got %s", mockCh.calls[0].Method)
	}

	// Simulate AI handoff
	handoffPayload := AIHandoffPayload{
		ConversationID: conv.ID,
		TenantID:       tenant.ID,
		Reason:         "complex query",
	}
	handoffBytes, _ := json.Marshal(handoffPayload)
	eb.Publish(ctx, "ai.handoff_request", json.RawMessage(handoffBytes))

	time.Sleep(500 * time.Millisecond)

	// Verify conversation escalated to human
	escalated, err := convSvc.Get(ctx, tenant.ID, conv.ID)
	if err != nil {
		t.Fatalf("Get conversation after handoff failed: %v", err)
	}
	if escalated.HandledBy != "human" {
		t.Errorf("expected handled_by human after handoff, got %s", escalated.HandledBy)
	}
}

func TestIntegrationHumanFirstMode(t *testing.T) {
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
	convSvc := NewService(queries, contactSvc, nil, nil, eb)
	if err := convSvc.StartAIConsumers(ctx); err != nil {
		t.Fatalf("StartAIConsumers failed: %v", err)
	}

	// Create tenant in human-first mode (default)
	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Human Corp", Mode: "human_first"})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Main",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123456"}`),
		Status:      "active",
	})

	// Subscribe to message.received (should NOT be published)
	receivedCh := make(chan eventbus.Event, 1)
	_, _ = eb.Subscribe(ctx, "message.received", func(_ context.Context, evt eventbus.Event) error {
		receivedCh <- evt
		return nil
	})

	time.Sleep(500 * time.Millisecond)

	evt := channel.WebhookEvent{
		EventID:   "EVT_HUMAN_001",
		From:      "15551234567",
		MessageID: "MSG_HUMAN_001",
		Type:      "text",
		Content:   []byte(`{"body":"Hello human"}`),
		Timestamp: time.Now().Unix(),
	}

	conv, _, err := convSvc.ProcessInboundMessage(ctx, tenant.ID, ch.ID, evt)
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}
	if conv.HandledBy != "human" {
		t.Errorf("expected handled_by human, got %s", conv.HandledBy)
	}

	// Verify message.received was NOT published
	select {
	case <-receivedCh:
		t.Fatal("unexpected message.received event in human-first mode")
	case <-time.After(1 * time.Second):
		// expected — no event
	}
}

func TestIntegrationHybridMode(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	natsC, natsCleanup := startNATS(t)
	defer natsCleanup()
	natsURL, _ := natsC.ConnectionString(ctx)
	eb, _ := eventbus.New(natsURL, nil)
	defer eb.Close()

	mockCh := &mockChannel{}
	contactSvc := contact.NewService(queries)
	convSvc := NewService(queries, contactSvc, nil, mockCh, eb)
	if err := convSvc.StartAIConsumers(ctx); err != nil {
		t.Fatalf("StartAIConsumers failed: %v", err)
	}

	// Create tenant in hybrid mode
	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Hybrid Corp", Mode: "hybrid"})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Main",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123456"}`),
		Status:      "active",
	})

	time.Sleep(500 * time.Millisecond)

	evt := channel.WebhookEvent{
		EventID:   "EVT_HYB_001",
		From:      "15551234567",
		MessageID: "MSG_HYB_001",
		Type:      "text",
		Content:   []byte(`{"body":"Hello hybrid"}`),
		Timestamp: time.Now().Unix(),
	}

	conv, _, err := convSvc.ProcessInboundMessage(ctx, tenant.ID, ch.ID, evt)
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}
	if conv.HandledBy != "hybrid" {
		t.Errorf("expected handled_by hybrid, got %s", conv.HandledBy)
	}

	// Simulate AI response in hybrid mode
	aiResponse := AIResponsePayload{
		ConversationID: conv.ID,
		TenantID:       tenant.ID,
		Body:           "Hybrid AI suggestion",
	}
	aiRespBytes, _ := json.Marshal(aiResponse)
	eb.Publish(ctx, "message.ai_response", json.RawMessage(aiRespBytes))

	time.Sleep(500 * time.Millisecond)

	// Verify AI response stored as pending (draft) and NOT sent
	fullConv, err := convSvc.Get(ctx, tenant.ID, conv.ID)
	if err != nil {
		t.Fatalf("Get conversation failed: %v", err)
	}
	if len(fullConv.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(fullConv.Messages))
	}

	var aiMsg *Message
	for i := range fullConv.Messages {
		if fullConv.Messages[i].Direction == "outbound" {
			aiMsg = &fullConv.Messages[i]
			break
		}
	}
	if aiMsg == nil {
		t.Fatal("expected AI outbound message")
	}
	if aiMsg.Status != "pending" {
		t.Errorf("expected AI message status pending (draft), got %s", aiMsg.Status)
	}

	// Verify channel did NOT receive any send
	if len(mockCh.calls) != 0 {
		t.Errorf("expected 0 channel calls in hybrid mode, got %d", len(mockCh.calls))
	}
}

func TestIntegrationSendTemplateMessageInvalidTemplate(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	contactSvc := contact.NewService(queries)
	templateSvc := template.NewService(queries, nil)
	mockCh := &mockChannel{}

	convSvc := NewService(queries, contactSvc, templateSvc, mockCh, nil)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test Corp", Mode: "human_first"})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Main",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123456"}`),
		Status:      "active",
	})

	contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	conv, _, _ := convSvc.ProcessInboundMessage(ctx, tenant.ID, ch.ID, channel.WebhookEvent{
		EventID:   "EVT_001",
		From:      "1234567890",
		MessageID: "MSG_001",
		Type:      "text",
		Content:   []byte(`{"body":"Hello"}`),
		Timestamp: time.Now().Unix(),
	})

	// Use a random UUID that doesn't exist
	fakeID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	_, err := convSvc.SendMessage(ctx, tenant.ID, conv.ID, SendMessageRequest{
		TemplateID: &fakeID,
		Parameters: map[string]string{"name": "Alice"},
	})
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
	if err != ErrTemplateNotFound {
		t.Errorf("expected ErrTemplateNotFound, got %v", err)
	}
}
