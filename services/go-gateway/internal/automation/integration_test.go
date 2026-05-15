package automation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/contact"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
)

func setupTenant(ctx context.Context, t *testing.T, queries *sqlcgen.Queries) sqlcgen.Tenant {
	t.Helper()
	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test Corp", Mode: "human_first"})
	return tenant
}

func TestIntegrationCreateAutomation(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries)
	tenant := setupTenant(ctx, t, queries)

	req := CreateRequest{
		Name: "Welcome Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger", Config: map[string]interface{}{"keywords": []string{"hi"}}},
				{ID: "n2", Type: "send_message", Config: map[string]interface{}{"message": "Hello!"}},
			},
			Edges: []FlowEdge{
				{Source: "n1", Target: "n2"},
			},
		},
	}

	auto, err := svc.Create(ctx, tenant.ID, req)
	if err != nil {
		t.Fatalf("Create automation failed: %v", err)
	}
	if auto.Name != req.Name {
		t.Errorf("expected name %q, got %q", req.Name, auto.Name)
	}
	if auto.Status != "draft" {
		t.Errorf("expected status draft, got %s", auto.Status)
	}
	if len(auto.Definition.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(auto.Definition.Nodes))
	}

	// Verify it exists in DB
	stored, err := queries.GetAutomationByID(ctx, sqlcgen.GetAutomationByIDParams{
		ID:       auto.ID,
		TenantID: tenant.ID,
	})
	if err != nil {
		t.Fatalf("automation not found in db: %v", err)
	}
	if stored.Name != req.Name {
		t.Errorf("stored name mismatch: %q vs %q", stored.Name, req.Name)
	}
}

func TestIntegrationListGetUpdateDeleteAutomation(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries)
	tenant := setupTenant(ctx, t, queries)

	// Create two automations
	auto1, _ := svc.Create(ctx, tenant.ID, CreateRequest{Name: "Flow 1", Definition: FlowDefinition{}})
	auto2, _ := svc.Create(ctx, tenant.ID, CreateRequest{Name: "Flow 2", Definition: FlowDefinition{}})

	// List
	list, count, err := svc.List(ctx, tenant.ID, 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 automations, got %d", len(list))
	}

	// Get
	got, err := svc.Get(ctx, tenant.ID, auto1.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != auto1.ID {
		t.Errorf("expected id %v, got %v", auto1.ID, got.ID)
	}

	// Wrong tenant returns not found
	otherTenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Other Corp", Mode: "human_first"})
	_, err = svc.Get(ctx, otherTenant.ID, auto1.ID)
	if err == nil {
		t.Fatal("expected error for wrong tenant")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Update
	newName := "Updated Flow"
	updated, err := svc.Update(ctx, tenant.ID, auto1.ID, UpdateRequest{Name: &newName})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("expected name %q, got %q", newName, updated.Name)
	}

	// Activation requires valid flow
	// Create a valid flow first
	validAuto, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name: "Valid Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger", Config: map[string]interface{}{"keywords": []string{"hi"}}},
				{ID: "n2", Type: "send_message", Config: map[string]interface{}{"message": "Hello!"}},
			},
			Edges: []FlowEdge{{Source: "n1", Target: "n2"}},
		},
	})
	active := "active"
	_, err = svc.Update(ctx, tenant.ID, validAuto.ID, UpdateRequest{Status: &active})
	if err != nil {
		t.Fatalf("Update to active failed for valid flow: %v", err)
	}

	// Invalid flow cannot be activated
	invalidAuto, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name: "Invalid Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger"},
				{ID: "n2", Type: "send_message"},
			},
			Edges: []FlowEdge{}, // n2 unreachable
		},
	})
	_, err = svc.Update(ctx, tenant.ID, invalidAuto.ID, UpdateRequest{Status: &active})
	if err == nil {
		t.Fatal("expected error when activating invalid flow")
	}
	if !errors.Is(err, ErrInvalidFlow) {
		t.Errorf("expected ErrInvalidFlow, got %v", err)
	}

	// Delete
	if err := svc.Delete(ctx, tenant.ID, auto2.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = svc.Get(ctx, tenant.ID, auto2.ID)
	if err == nil {
		t.Fatal("expected not found after delete")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

// mockAutoChannel records SendTextMessage calls for automation engine tests.
type mockAutoChannel struct {
	calls []struct {
		To   string
		Body string
	}
}

func (m *mockAutoChannel) ChannelType() string { return "whatsapp" }
func (m *mockAutoChannel) SendTextMessage(ctx context.Context, to, body string) error {
	m.calls = append(m.calls, struct{ To, Body string }{to, body})
	return nil
}
func (m *mockAutoChannel) SendTemplateMessage(ctx context.Context, to, templateName, language string, params []string) error {
	return nil
}
func (m *mockAutoChannel) SendMediaMessage(ctx context.Context, to, mediaType, mediaURL, caption string) error {
	return nil
}
func (m *mockAutoChannel) MarkRead(ctx context.Context, messageID string) error {
	return nil
}
func (m *mockAutoChannel) VerifyWebhook(mode, verifyToken, challenge string) (string, error) {
	return "", nil
}
func (m *mockAutoChannel) VerifySignature(payload []byte, signature string) error {
	return nil
}
func (m *mockAutoChannel) ParseWebhookEvent(payload []byte) (channel.WebhookEvent, error) {
	return channel.WebhookEvent{}, nil
}

func TestIntegrationEngineKeywordTrigger(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries)
	mockCh := &mockAutoChannel{}
	engine := NewEngine(queries, mockCh)

	tenant := setupTenant(ctx, t, queries)

	// Create a contact with phone
	contactSvc := contact.NewService(queries)
	c, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	// Create an active automation with keyword trigger
	auto, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name: "Welcome Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger", Config: map[string]interface{}{"keywords": []string{"help"}}},
				{ID: "n2", Type: "send_message", Config: map[string]interface{}{"message": "How can I help you?"}},
			},
			Edges: []FlowEdge{{Source: "n1", Target: "n2"}},
		},
	})
	active := "active"
	_, _ = svc.Update(ctx, tenant.ID, auto.ID, UpdateRequest{Status: &active})

	// Create a conversation
	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:  tenant.ID,
		ContactID: c.ID,
		Status:    sqlcgen.ConversationStatusOpen,
		HandledBy: sqlcgen.HandledByAi,
	})

	// Process inbound message that matches keyword
	err := engine.ProcessInboundMessage(ctx, tenant.ID, conv.ID, "I need help please")
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	// Verify channel received the automated reply
	if len(mockCh.calls) != 1 {
		t.Fatalf("expected 1 channel call, got %d", len(mockCh.calls))
	}
	if mockCh.calls[0].Body != "How can I help you?" {
		t.Errorf("unexpected reply: %q", mockCh.calls[0].Body)
	}
	if mockCh.calls[0].To != "+11234567890" {
		t.Errorf("unexpected recipient: %q", mockCh.calls[0].To)
	}

	// Verify automation run was created and completed
	runs, _ := queries.ListAutomationRunsByAutomation(ctx, auto.ID)
	if len(runs) != 1 {
		t.Fatalf("expected 1 automation run, got %d", len(runs))
	}
	if runs[0].Status != "completed" {
		t.Errorf("expected run status completed, got %s", runs[0].Status)
	}
}

func TestIntegrationEngineNoMatch(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries)
	mockCh := &mockAutoChannel{}
	engine := NewEngine(queries, mockCh)

	tenant := setupTenant(ctx, t, queries)

	contactSvc := contact.NewService(queries)
	c, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	auto, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name: "Welcome Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger", Config: map[string]interface{}{"keywords": []string{"help"}}},
				{ID: "n2", Type: "send_message", Config: map[string]interface{}{"message": "How can I help you?"}},
			},
			Edges: []FlowEdge{{Source: "n1", Target: "n2"}},
		},
	})
	active := "active"
	_, _ = svc.Update(ctx, tenant.ID, auto.ID, UpdateRequest{Status: &active})

	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:  tenant.ID,
		ContactID: c.ID,
		Status:    sqlcgen.ConversationStatusOpen,
		HandledBy: sqlcgen.HandledByAi,
	})

	// Process inbound message that does NOT match keyword
	_ = engine.ProcessInboundMessage(ctx, tenant.ID, conv.ID, "Just saying hello")

	// Verify no channel call was made
	if len(mockCh.calls) != 0 {
		t.Fatalf("expected 0 channel calls, got %d", len(mockCh.calls))
	}
}

func TestIntegrationEngineSequentialExecution(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries)
	mockCh := &mockAutoChannel{}
	engine := NewEngine(queries, mockCh)

	tenant := setupTenant(ctx, t, queries)

	contactSvc := contact.NewService(queries)
	c, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	auto, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name: "Sequential Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger", Config: map[string]interface{}{"keywords": []string{"start"}}},
				{ID: "n2", Type: "send_message", Config: map[string]interface{}{"message": "First message"}},
				{ID: "n3", Type: "send_message", Config: map[string]interface{}{"message": "Second message"}},
			},
			Edges: []FlowEdge{
				{Source: "n1", Target: "n2"},
				{Source: "n2", Target: "n3"},
			},
		},
	})
	active := "active"
	_, _ = svc.Update(ctx, tenant.ID, auto.ID, UpdateRequest{Status: &active})

	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:  tenant.ID,
		ContactID: c.ID,
		Status:    sqlcgen.ConversationStatusOpen,
		HandledBy: sqlcgen.HandledByAi,
	})

	// Process inbound message that matches keyword
	err := engine.ProcessInboundMessage(ctx, tenant.ID, conv.ID, "start")
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	// Verify both messages were sent
	if len(mockCh.calls) != 2 {
		t.Fatalf("expected 2 channel calls, got %d", len(mockCh.calls))
	}
	if mockCh.calls[0].Body != "First message" {
		t.Errorf("unexpected first message: %q", mockCh.calls[0].Body)
	}
	if mockCh.calls[1].Body != "Second message" {
		t.Errorf("unexpected second message: %q", mockCh.calls[1].Body)
	}

	// Verify automation run was created and completed
	runs, _ := queries.ListAutomationRunsByAutomation(ctx, auto.ID)
	if len(runs) != 1 {
		t.Fatalf("expected 1 automation run, got %d", len(runs))
	}
	if runs[0].Status != "completed" {
		t.Errorf("expected run status completed, got %s", runs[0].Status)
	}
}

func TestIntegrationEngineConditionNode(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries)
	mockCh := &mockAutoChannel{}
	engine := NewEngine(queries, mockCh)

	tenant := setupTenant(ctx, t, queries)

	contactSvc := contact.NewService(queries)
	c, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:     "Alice",
		Language: "en",
		Phones:   []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	// Flow: trigger → condition (language == "en") → send_message "English!" (true branch)
	auto, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name: "Condition Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger", Config: map[string]interface{}{"keywords": []string{"test"}}},
				{ID: "n2", Type: "condition", Config: map[string]interface{}{
					"field":    "contact.language",
					"operator": "equals",
					"value":    "en",
				}},
				{ID: "n3", Type: "send_message", Config: map[string]interface{}{"message": "English!"}},
				{ID: "n4", Type: "send_message", Config: map[string]interface{}{"message": "Not English!"}},
			},
			Edges: []FlowEdge{
				{Source: "n1", Target: "n2"},
				{Source: "n2", Target: "n3", Label: "true"},
				{Source: "n2", Target: "n4", Label: "false"},
			},
		},
	})
	active := "active"
	_, _ = svc.Update(ctx, tenant.ID, auto.ID, UpdateRequest{Status: &active})

	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:  tenant.ID,
		ContactID: c.ID,
		Status:    sqlcgen.ConversationStatusOpen,
		HandledBy: sqlcgen.HandledByAi,
	})

	// Trigger the flow
	err := engine.ProcessInboundMessage(ctx, tenant.ID, conv.ID, "test")
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	// Verify the true branch was taken
	if len(mockCh.calls) != 1 {
		t.Fatalf("expected 1 channel call, got %d", len(mockCh.calls))
	}
	if mockCh.calls[0].Body != "English!" {
		t.Errorf("expected 'English!', got %q", mockCh.calls[0].Body)
	}
}

func TestIntegrationEngineDelayNode(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries)
	mockCh := &mockAutoChannel{}
	engine := NewEngine(queries, mockCh)

	tenant := setupTenant(ctx, t, queries)

	contactSvc := contact.NewService(queries)
	c, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	auto, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name: "Delay Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger", Config: map[string]interface{}{"keywords": []string{"wait"}}},
				{ID: "n2", Type: "delay", Config: map[string]interface{}{"duration": 1}},
				{ID: "n3", Type: "send_message", Config: map[string]interface{}{"message": "After delay!"}},
			},
			Edges: []FlowEdge{
				{Source: "n1", Target: "n2"},
				{Source: "n2", Target: "n3"},
			},
		},
	})
	active := "active"
	_, _ = svc.Update(ctx, tenant.ID, auto.ID, UpdateRequest{Status: &active})

	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:  tenant.ID,
		ContactID: c.ID,
		Status:    sqlcgen.ConversationStatusOpen,
		HandledBy: sqlcgen.HandledByAi,
	})

	// Trigger the flow
	err := engine.ProcessInboundMessage(ctx, tenant.ID, conv.ID, "wait")
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	// Verify run is paused and resume_at is set
	runs, _ := queries.ListAutomationRunsByAutomation(ctx, auto.ID)
	if len(runs) != 1 {
		t.Fatalf("expected 1 automation run, got %d", len(runs))
	}
	if runs[0].Status != "paused" {
		t.Errorf("expected run status paused, got %s", runs[0].Status)
	}
	if !runs[0].ResumeAt.Valid {
		t.Errorf("expected resume_at to be set")
	}

	// No message should have been sent yet
	if len(mockCh.calls) != 0 {
		t.Fatalf("expected 0 channel calls before delay, got %d", len(mockCh.calls))
	}

	// Wait for delay to expire
	time.Sleep(2 * time.Second)

	// Resume the run
	err = engine.ResumeRun(ctx, runs[0].ID)
	if err != nil {
		t.Fatalf("ResumeRun failed: %v", err)
	}

	// Verify message was sent after resume
	if len(mockCh.calls) != 1 {
		t.Fatalf("expected 1 channel call after resume, got %d", len(mockCh.calls))
	}
	if mockCh.calls[0].Body != "After delay!" {
		t.Errorf("unexpected message: %q", mockCh.calls[0].Body)
	}

	// Verify run is completed
	runs, _ = queries.ListAutomationRunsByAutomation(ctx, auto.ID)
	if runs[0].Status != "completed" {
		t.Errorf("expected run status completed after resume, got %s", runs[0].Status)
	}
}

func TestIntegrationEngineWaitForReplyNode(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries)
	mockCh := &mockAutoChannel{}
	engine := NewEngine(queries, mockCh)

	tenant := setupTenant(ctx, t, queries)

	contactSvc := contact.NewService(queries)
	c, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	auto, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name: "Wait For Reply Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger", Config: map[string]interface{}{"keywords": []string{"question"}}},
				{ID: "n2", Type: "send_message", Config: map[string]interface{}{"message": "What is your name?"}},
				{ID: "n3", Type: "wait_for_reply", Config: map[string]interface{}{"timeout": 60}},
				{ID: "n4", Type: "send_message", Config: map[string]interface{}{"message": "Thanks for replying!"}},
			},
			Edges: []FlowEdge{
				{Source: "n1", Target: "n2"},
				{Source: "n2", Target: "n3"},
				{Source: "n3", Target: "n4", Label: "replied"},
			},
		},
	})
	active := "active"
	_, _ = svc.Update(ctx, tenant.ID, auto.ID, UpdateRequest{Status: &active})

	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:  tenant.ID,
		ContactID: c.ID,
		Status:    sqlcgen.ConversationStatusOpen,
		HandledBy: sqlcgen.HandledByAi,
	})

	// First message triggers the flow
	err := engine.ProcessInboundMessage(ctx, tenant.ID, conv.ID, "question")
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	// Verify first message sent and run is waiting
	if len(mockCh.calls) != 1 {
		t.Fatalf("expected 1 channel call, got %d", len(mockCh.calls))
	}
	if mockCh.calls[0].Body != "What is your name?" {
		t.Errorf("unexpected message: %q", mockCh.calls[0].Body)
	}

	runs, _ := queries.ListAutomationRunsByAutomation(ctx, auto.ID)
	if len(runs) != 1 {
		t.Fatalf("expected 1 automation run, got %d", len(runs))
	}
	if runs[0].Status != "waiting" {
		t.Errorf("expected run status waiting, got %s", runs[0].Status)
	}

	// Second message should resume the flow
	err = engine.ProcessInboundMessage(ctx, tenant.ID, conv.ID, "My name is Bob")
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	// Verify second message sent after reply
	if len(mockCh.calls) != 2 {
		t.Fatalf("expected 2 channel calls, got %d", len(mockCh.calls))
	}
	if mockCh.calls[1].Body != "Thanks for replying!" {
		t.Errorf("unexpected message: %q", mockCh.calls[1].Body)
	}

	// Verify run is completed
	runs, _ = queries.ListAutomationRunsByAutomation(ctx, auto.ID)
	if runs[0].Status != "completed" {
		t.Errorf("expected run status completed, got %s", runs[0].Status)
	}
}

func TestIntegrationEngineHandoffHuman(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries)
	mockCh := &mockAutoChannel{}
	engine := NewEngine(queries, mockCh)

	tenant := setupTenant(ctx, t, queries)

	contactSvc := contact.NewService(queries)
	c, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	auto, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name: "Handoff Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger", Config: map[string]interface{}{"keywords": []string{"help"}}},
				{ID: "n2", Type: "handoff_human"},
			},
			Edges: []FlowEdge{
				{Source: "n1", Target: "n2"},
			},
		},
	})
	active := "active"
	_, _ = svc.Update(ctx, tenant.ID, auto.ID, UpdateRequest{Status: &active})

	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:  tenant.ID,
		ContactID:  c.ID,
		Status:    sqlcgen.ConversationStatusOpen,
		HandledBy: sqlcgen.HandledByAi,
	})

	err := engine.ProcessInboundMessage(ctx, tenant.ID, conv.ID, "help")
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	// Verify conversation was escalated
	updatedConv, _ := queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
		ID:       conv.ID,
		TenantID: tenant.ID,
	})
	if updatedConv.Status != "escalated" {
		t.Errorf("expected conversation status escalated, got %s", updatedConv.Status)
	}
	if updatedConv.HandledBy != "human" {
		t.Errorf("expected handled_by human, got %s", updatedConv.HandledBy)
	}

	// Verify run completed
	runs, _ := queries.ListAutomationRunsByAutomation(ctx, auto.ID)
	if len(runs) != 1 {
		t.Fatalf("expected 1 automation run, got %d", len(runs))
	}
	if runs[0].Status != "completed" {
		t.Errorf("expected run status completed, got %s", runs[0].Status)
	}
}

func TestIntegrationEngineAddTagAndSetCustomField(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries)
	mockCh := &mockAutoChannel{}
	engine := NewEngine(queries, mockCh)

	tenant := setupTenant(ctx, t, queries)

	contactSvc := contact.NewService(queries)
	c, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	auto, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name: "Tag and Field Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger", Config: map[string]interface{}{"keywords": []string{"tag"}}},
				{ID: "n2", Type: "add_tag", Config: map[string]interface{}{"tag": "vip"}},
				{ID: "n3", Type: "set_custom_field", Config: map[string]interface{}{"field": "region", "value": "us-east"}},
				{ID: "n4", Type: "send_message", Config: map[string]interface{}{"message": "Done!"}},
			},
			Edges: []FlowEdge{
				{Source: "n1", Target: "n2"},
				{Source: "n2", Target: "n3"},
				{Source: "n3", Target: "n4"},
			},
		},
	})
	active := "active"
	_, _ = svc.Update(ctx, tenant.ID, auto.ID, UpdateRequest{Status: &active})

	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:  tenant.ID,
		ContactID: c.ID,
		Status:    sqlcgen.ConversationStatusOpen,
		HandledBy: sqlcgen.HandledByAi,
	})

	err := engine.ProcessInboundMessage(ctx, tenant.ID, conv.ID, "tag")
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	// Verify tag was added
	tags, _ := queries.ListContactTags(ctx, c.ID)
	foundVIP := false
	for _, tag := range tags {
		if tag.Name == "vip" {
			foundVIP = true
			break
		}
	}
	if !foundVIP {
		t.Errorf("expected contact to have vip tag")
	}

	// Verify custom field was set
	fields, _ := queries.ListContactCustomFields(ctx, c.ID)
	foundRegion := false
	for _, f := range fields {
		if f.Name == "region" && f.Value == "us-east" {
			foundRegion = true
			break
		}
	}
	if !foundRegion {
		t.Errorf("expected contact to have region=us-east custom field, got %+v", fields)
	}

	// Verify message sent
	if len(mockCh.calls) != 1 || mockCh.calls[0].Body != "Done!" {
		t.Errorf("expected 1 'Done!' message, got %+v", mockCh.calls)
	}
}

func TestIntegrationEngineAIResponse(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries)
	mockCh := &mockAutoChannel{}
	engine := NewEngine(queries, mockCh)
	engine.aiRequestFunc = func(ctx context.Context, subject string, payload map[string]interface{}) (string, error) {
		return "Hello from AI!", nil
	}

	tenant := setupTenant(ctx, t, queries)

	contactSvc := contact.NewService(queries)
	c, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	auto, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name: "AI Response Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger", Config: map[string]interface{}{"keywords": []string{"ai"}}},
				{ID: "n2", Type: "ai_response"},
			},
			Edges: []FlowEdge{
				{Source: "n1", Target: "n2"},
			},
		},
	})
	active := "active"
	_, _ = svc.Update(ctx, tenant.ID, auto.ID, UpdateRequest{Status: &active})

	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:  tenant.ID,
		ContactID: c.ID,
		Status:    sqlcgen.ConversationStatusOpen,
		HandledBy: sqlcgen.HandledByAi,
	})

	err := engine.ProcessInboundMessage(ctx, tenant.ID, conv.ID, "ai")
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	// Verify AI response was sent
	if len(mockCh.calls) != 1 {
		t.Fatalf("expected 1 channel call, got %d", len(mockCh.calls))
	}
	if mockCh.calls[0].Body != "Hello from AI!" {
		t.Errorf("unexpected AI message: %q", mockCh.calls[0].Body)
	}
}

func TestIntegrationWorkerPoolResumesDelayedRun(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries)
	mockCh := &mockAutoChannel{}
	engine := NewEngine(queries, mockCh)
	workerPool := NewWorkerPool(queries, engine, nil, 2)

	tenant := setupTenant(ctx, t, queries)

	contactSvc := contact.NewService(queries)
	c, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	auto, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name: "Worker Pool Delay Flow",
		Definition: FlowDefinition{
			Nodes: []FlowNode{
				{ID: "n1", Type: "trigger", Config: map[string]interface{}{"keywords": []string{"worker"}}},
				{ID: "n2", Type: "delay", Config: map[string]interface{}{"duration": 1}},
				{ID: "n3", Type: "send_message", Config: map[string]interface{}{"message": "Worker resumed!"}},
			},
			Edges: []FlowEdge{
				{Source: "n1", Target: "n2"},
				{Source: "n2", Target: "n3"},
			},
		},
	})
	active := "active"
	_, _ = svc.Update(ctx, tenant.ID, auto.ID, UpdateRequest{Status: &active})

	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:  tenant.ID,
		ContactID: c.ID,
		Status:    sqlcgen.ConversationStatusOpen,
		HandledBy: sqlcgen.HandledByAi,
	})

	// Start worker pool
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	workerPool.Start(workerCtx)
	defer workerPool.Stop()

	// Trigger the flow
	err := engine.ProcessInboundMessage(ctx, tenant.ID, conv.ID, "worker")
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	// Verify run is paused
	runs, _ := queries.ListAutomationRunsByAutomation(ctx, auto.ID)
	if len(runs) != 1 {
		t.Fatalf("expected 1 automation run, got %d", len(runs))
	}
	if runs[0].Status != "paused" {
		t.Errorf("expected run status paused, got %s", runs[0].Status)
	}

	// Wait for worker to resume (delay is 1s, poll interval is 5s)
	time.Sleep(7 * time.Second)

	// Verify message was sent and run completed
	if len(mockCh.calls) != 1 {
		t.Fatalf("expected 1 channel call, got %d", len(mockCh.calls))
	}
	if mockCh.calls[0].Body != "Worker resumed!" {
		t.Errorf("unexpected message: %q", mockCh.calls[0].Body)
	}

	runs, _ = queries.ListAutomationRunsByAutomation(ctx, auto.ID)
	if runs[0].Status != "completed" {
		t.Errorf("expected run status completed, got %s", runs[0].Status)
	}
}

func TestValidateFlow(t *testing.T) {
	// Valid flow
	valid := FlowDefinition{
		Nodes: []FlowNode{
			{ID: "n1", Type: "trigger"},
			{ID: "n2", Type: "send_message"},
		},
		Edges: []FlowEdge{{Source: "n1", Target: "n2"}},
	}
	if err := ValidateFlow(valid); err != nil {
		t.Errorf("expected valid flow, got error: %v", err)
	}

	// No trigger
	noTrigger := FlowDefinition{
		Nodes: []FlowNode{{ID: "n1", Type: "send_message"}},
		Edges: []FlowEdge{},
	}
	if err := ValidateFlow(noTrigger); err == nil {
		t.Error("expected error for missing trigger")
	}

	// Multiple triggers
	multiTrigger := FlowDefinition{
		Nodes: []FlowNode{
			{ID: "n1", Type: "trigger"},
			{ID: "n2", Type: "trigger"},
		},
		Edges: []FlowEdge{},
	}
	if err := ValidateFlow(multiTrigger); err == nil {
		t.Error("expected error for multiple triggers")
	}

	// Edge references unknown node
	badEdge := FlowDefinition{
		Nodes: []FlowNode{
			{ID: "n1", Type: "trigger"},
			{ID: "n2", Type: "send_message"},
		},
		Edges: []FlowEdge{{Source: "n1", Target: "n3"}},
	}
	if err := ValidateFlow(badEdge); err == nil {
		t.Error("expected error for edge referencing unknown node")
	}

	// Unreachable node
	unreachable := FlowDefinition{
		Nodes: []FlowNode{
			{ID: "n1", Type: "trigger"},
			{ID: "n2", Type: "send_message"},
			{ID: "n3", Type: "send_message"},
		},
		Edges: []FlowEdge{{Source: "n1", Target: "n2"}},
	}
	if err := ValidateFlow(unreachable); err == nil {
		t.Error("expected error for unreachable node")
	}
}
