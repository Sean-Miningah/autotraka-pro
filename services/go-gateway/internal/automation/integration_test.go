package automation

import (
	"context"
	"errors"
	"testing"

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
