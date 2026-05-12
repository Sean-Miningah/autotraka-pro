package template

import (
	"context"
	"testing"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
)

func setupTenantAndChannel(ctx context.Context, t *testing.T, queries *sqlcgen.Queries) (tenant sqlcgen.Tenant, channel sqlcgen.CreateChannelRow) {
	t.Helper()
	tenant, _ = queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test Corp", Mode: "human_first"})
	channel, _ = queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Main",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123456"}`),
		Status:      "active",
	})
	return
}

func TestIntegrationCreateTemplate(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries, nil)
	tenant, ch := setupTenantAndChannel(ctx, t, queries)

	req := CreateRequest{
		ChannelID:  ch.ID,
		Name:       "welcome_message",
		Category:   "MARKETING",
		Language:   "en",
		Body:       "Hello {{1}}, welcome to {{2}}!",
		Parameters: []ParameterDef{{Name: "customer_name", DisplayName: "Customer Name"}, {Name: "company_name", DisplayName: "Company Name"}},
	}

	tmpl, err := svc.Create(ctx, tenant.ID, req)
	if err != nil {
		t.Fatalf("Create template failed: %v", err)
	}
	if tmpl.Name != req.Name {
		t.Errorf("expected name %q, got %q", req.Name, tmpl.Name)
	}
	if tmpl.Status != "draft" {
		t.Errorf("expected status draft, got %s", tmpl.Status)
	}
	if len(tmpl.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(tmpl.Parameters))
	}
	if tmpl.ChannelID == nil || *tmpl.ChannelID != ch.ID {
		t.Errorf("expected channel_id %v, got %v", ch.ID, tmpl.ChannelID)
	}

	// Verify it exists in DB
	stored, err := queries.GetTemplateByID(ctx, sqlcgen.GetTemplateByIDParams{
		ID:       tmpl.ID,
		TenantID: tenant.ID,
	})
	if err != nil {
		t.Fatalf("template not found in db: %v", err)
	}
	if stored.Name != req.Name {
		t.Errorf("stored name mismatch: %q vs %q", stored.Name, req.Name)
	}
}

func TestIntegrationListTemplates(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries, nil)
	tenant, ch := setupTenantAndChannel(ctx, t, queries)

	// Create two templates with different statuses
	_, _ = svc.Create(ctx, tenant.ID, CreateRequest{ChannelID: ch.ID, Name: "t1", Category: "MARKETING", Language: "en", Body: "b1"})
	_, _ = svc.Create(ctx, tenant.ID, CreateRequest{ChannelID: ch.ID, Name: "t2", Category: "UTILITY", Language: "en", Body: "b2"})

	// List all
	list, count, err := svc.List(ctx, tenant.ID, "", 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 templates, got %d", len(list))
	}

	// Filter by status
	list, count, err = svc.List(ctx, tenant.ID, "draft", 10, 0)
	if err != nil {
		t.Fatalf("List with status filter failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2 for draft, got %d", count)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 draft templates, got %d", len(list))
	}
}

func TestIntegrationGetTemplate(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries, nil)
	tenant, ch := setupTenantAndChannel(ctx, t, queries)

	tmpl, _ := svc.Create(ctx, tenant.ID, CreateRequest{ChannelID: ch.ID, Name: "t1", Category: "MARKETING", Language: "en", Body: "b1"})

	got, err := svc.Get(ctx, tenant.ID, tmpl.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != tmpl.ID {
		t.Errorf("expected id %v, got %v", tmpl.ID, got.ID)
	}

	// Wrong tenant should return not found
	otherTenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Other Corp", Mode: "human_first"})
	_, err = svc.Get(ctx, otherTenant.ID, tmpl.ID)
	if err == nil {
		t.Fatal("expected error for wrong tenant")
	}
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestIntegrationUpdateAndDeleteTemplate(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries, nil)
	tenant, ch := setupTenantAndChannel(ctx, t, queries)

	tmpl, _ := svc.Create(ctx, tenant.ID, CreateRequest{ChannelID: ch.ID, Name: "t1", Category: "MARKETING", Language: "en", Body: "b1"})

	newName := "updated_name"
	updated, err := svc.Update(ctx, tenant.ID, tmpl.ID, UpdateRequest{Name: &newName})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("expected name %q, got %q", newName, updated.Name)
	}

	// Delete
	if err := svc.Delete(ctx, tenant.ID, tmpl.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = svc.Get(ctx, tenant.ID, tmpl.ID)
	if err == nil {
		t.Fatal("expected not found after delete")
	}
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

// mockMetaTemplateClient is a test double for MetaTemplateClient.
type mockMetaTemplateClient struct {
	created    []MetaCreateTemplateReq
	statuses   map[string]string
	id         string
}

func (m *mockMetaTemplateClient) CreateTemplate(ctx context.Context, wabaID string, req MetaCreateTemplateReq) (string, error) {
	m.created = append(m.created, req)
	return m.id, nil
}

func (m *mockMetaTemplateClient) GetTemplateStatus(ctx context.Context, wabaID, templateName string) (string, error) {
	if s, ok := m.statuses[templateName]; ok {
		return s, nil
	}
	return "PENDING", nil
}

func TestIntegrationCreateTemplateSubmitsToMeta(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test Corp", Mode: "human_first"})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Main",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123456","waba_id":"WABA_001"}`),
		Status:      "active",
	})

	mock := &mockMetaTemplateClient{id: "META_123"}
	svc := NewService(queries, mock)

	req := CreateRequest{
		ChannelID:  ch.ID,
		Name:       "welcome_message",
		Category:   "MARKETING",
		Language:   "en",
		Body:       "Hello {{1}}!",
		Parameters: []ParameterDef{{Name: "name", DisplayName: "Name"}},
	}

	tmpl, err := svc.Create(ctx, tenant.ID, req)
	if err != nil {
		t.Fatalf("Create template failed: %v", err)
	}
	if tmpl.Status != "pending" {
		t.Errorf("expected status pending, got %s", tmpl.Status)
	}
	if tmpl.MetaTemplateID == nil || *tmpl.MetaTemplateID != "META_123" {
		t.Errorf("expected meta_template_id META_123, got %v", tmpl.MetaTemplateID)
	}
	if len(mock.created) != 1 {
		t.Fatalf("expected 1 meta submission, got %d", len(mock.created))
	}
}

func TestIntegrationSyncPendingTemplates(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test Corp", Mode: "human_first"})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Main",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123456","waba_id":"WABA_001"}`),
		Status:      "active",
	})

	mock := &mockMetaTemplateClient{
		id:       "META_123",
		statuses: map[string]string{"welcome_message": "approved"},
	}
	svc := NewService(queries, mock)

	// Create a template that submits to Meta and becomes pending
	_, _ = svc.Create(ctx, tenant.ID, CreateRequest{
		ChannelID: ch.ID,
		Name:      "welcome_message",
		Category:  "MARKETING",
		Language:  "en",
		Body:      "Hello {{1}}!",
		Parameters: []ParameterDef{{Name: "name", DisplayName: "Name"}},
	})

	// Run sync
	if err := svc.SyncPendingStatuses(ctx); err != nil {
		t.Fatalf("SyncPendingStatuses failed: %v", err)
	}

	// Verify status updated to approved
	list, _, _ := svc.List(ctx, tenant.ID, "approved", 10, 0)
	if len(list) != 1 {
		t.Fatalf("expected 1 approved template, got %d", len(list))
	}
	if list[0].Status != "approved" {
		t.Errorf("expected status approved, got %s", list[0].Status)
	}
}

func TestIntegrationLocalizationTemplate(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	svc := NewService(queries, nil)
	tenant, ch := setupTenantAndChannel(ctx, t, queries)

	parent, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		ChannelID: ch.ID,
		Name:      "welcome_message",
		Category:  "MARKETING",
		Language:  "en",
		Body:      "Hello {{1}}!",
		Parameters: []ParameterDef{{Name: "name", DisplayName: "Name"}},
	})

	child, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		ChannelID:        ch.ID,
		ParentTemplateID: parent.ID,
		Name:             "welcome_message_es",
		Category:         "MARKETING",
		Language:         "es",
		Body:             "Hola {{1}}!",
		Parameters:       []ParameterDef{{Name: "name", DisplayName: "Nombre"}},
	})

	if child.ParentTemplateID == nil || *child.ParentTemplateID != parent.ID {
		t.Fatalf("expected parent_template_id %v, got %v", parent.ID, child.ParentTemplateID)
	}

	got, _ := svc.Get(ctx, tenant.ID, child.ID)
	if got.ParentTemplateID == nil || *got.ParentTemplateID != parent.ID {
		t.Errorf("expected parent_template_id on get %v, got %v", parent.ID, got.ParentTemplateID)
	}
}
