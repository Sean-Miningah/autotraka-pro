package broadcast

import (
	"context"
	"testing"
	"time"

	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/contact"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/template"
	"github.com/autotraka/go-gateway/internal/testutil"
	"github.com/google/uuid"
)

func setupTenant(ctx context.Context, t *testing.T, queries *sqlcgen.Queries) sqlcgen.Tenant {
	t.Helper()
	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test Corp", Mode: "human_first"})
	return tenant
}

// mockBroadcastChannel records SendTemplateMessage calls for broadcast tests.
type mockBroadcastChannel struct {
	calls []struct {
		To           string
		TemplateName string
		Language     string
		Params       []string
	}
}

func (m *mockBroadcastChannel) ChannelType() string { return "whatsapp" }
func (m *mockBroadcastChannel) SendTextMessage(ctx context.Context, to, body string) error {
	return nil
}
func (m *mockBroadcastChannel) SendTemplateMessage(ctx context.Context, to, templateName, language string, params []string) error {
	m.calls = append(m.calls, struct {
		To           string
		TemplateName string
		Language     string
		Params       []string
	}{to, templateName, language, params})
	return nil
}
func (m *mockBroadcastChannel) SendMediaMessage(ctx context.Context, to, mediaType, mediaURL, caption string) error {
	return nil
}
func (m *mockBroadcastChannel) MarkRead(ctx context.Context, messageID string) error {
	return nil
}
func (m *mockBroadcastChannel) VerifyWebhook(mode, verifyToken, challenge string) (string, error) {
	return "", nil
}
func (m *mockBroadcastChannel) VerifySignature(payload []byte, signature string) error {
	return nil
}
func (m *mockBroadcastChannel) ParseWebhookEvent(payload []byte) (channel.WebhookEvent, error) {
	return channel.WebhookEvent{}, nil
}

func TestIntegrationBroadcastLifecycle(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	tenant := setupTenant(ctx, t, queries)

	// Create a channel
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Test",
		ChannelType: "whatsapp",
		Config:      []byte(`{"waba_id":"123"}`),
		Status:      "active",
	})

	// Create a template
	templateSvc := template.NewService(queries, nil)
	tmpl, _ := templateSvc.Create(ctx, tenant.ID, template.CreateRequest{
		Name:     "welcome",
		Category: "MARKETING",
		Language: "en",
		Body:     "Hello {{1}}!",
		Parameters: []template.ParameterDef{
			{Name: "name", DisplayName: "Customer Name"},
		},
	})

	// Create contacts
	contactSvc := contact.NewService(queries)
	c1, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1111111111", Label: "mobile"}},
	})
	c2, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Bob",
		Phones: []contact.PhoneInput{{Phone: "+2222222222", Label: "mobile"}},
	})

	mockCh := &mockBroadcastChannel{}
	svc := NewService(queries, mockCh, &NoopRateLimiter{})

	// 1. Create broadcast as draft
	req := CreateRequest{
		Title:      "Welcome Campaign",
		TemplateID: &tmpl.ID,
		Parameters: map[string]interface{}{"name": "Customer"},
		ChannelID:  &ch.ID,
	}
	bcast, err := svc.Create(ctx, tenant.ID, req)
	if err != nil {
		t.Fatalf("Create broadcast failed: %v", err)
	}
	if bcast.Title != req.Title {
		t.Errorf("expected title %q, got %q", req.Title, bcast.Title)
	}
	if bcast.Status != "draft" {
		t.Errorf("expected status draft, got %s", bcast.Status)
	}

	// 2. Add recipients individually
	err = svc.AddRecipients(ctx, tenant.ID, bcast.ID, AddRecipientsRequest{
		ContactIDs: []uuid.UUID{c1.ID, c2.ID},
	})
	if err != nil {
		t.Fatalf("AddRecipients failed: %v", err)
	}

	// Verify recipients
	recipients, count, err := svc.ListRecipients(ctx, bcast.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListRecipients failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 recipients, got %d", count)
	}
	if len(recipients) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(recipients))
	}

	// 3. Schedule the broadcast
	scheduledTime := time.Now().UTC().Add(1 * time.Hour)
	updated, err := svc.Update(ctx, tenant.ID, bcast.ID, UpdateRequest{
		ScheduledAt: &scheduledTime,
	})
	if err != nil {
		t.Fatalf("Update broadcast failed: %v", err)
	}
	if updated.Status != "scheduled" {
		t.Errorf("expected status scheduled, got %s", updated.Status)
	}

	// 4. Trigger the broadcast immediately
	triggered, err := svc.Trigger(ctx, tenant.ID, bcast.ID)
	if err != nil {
		t.Fatalf("Trigger broadcast failed: %v", err)
	}
	if triggered.Status != "sending" {
		t.Errorf("expected status sending, got %s", triggered.Status)
	}

	// Wait for background send to complete
	time.Sleep(2 * time.Second)

	// 5. Verify messages were sent via mock channel
	if len(mockCh.calls) != 2 {
		t.Fatalf("expected 2 channel calls, got %d", len(mockCh.calls))
	}
	if mockCh.calls[0].TemplateName != "welcome" {
		t.Errorf("unexpected template name: %q", mockCh.calls[0].TemplateName)
	}

	// 6. Verify broadcast status is completed
	completed, err := svc.Get(ctx, tenant.ID, bcast.ID)
	if err != nil {
		t.Fatalf("Get broadcast failed: %v", err)
	}
	if completed.Status != "completed" {
		t.Errorf("expected status completed, got %s", completed.Status)
	}

	// 7. Verify recipient summary
	summary, err := svc.GetRecipientSummary(ctx, bcast.ID)
	if err != nil {
		t.Fatalf("GetRecipientSummary failed: %v", err)
	}
	if summary.Total != 2 {
		t.Errorf("expected total 2, got %d", summary.Total)
	}
	if summary.Sent != 2 {
		t.Errorf("expected sent 2, got %d", summary.Sent)
	}
	if summary.Pending != 0 {
		t.Errorf("expected pending 0, got %d", summary.Pending)
	}
}

func TestIntegrationBroadcastTagFilter(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	tenant := setupTenant(ctx, t, queries)

	// Create contacts with a tag
	contactSvc := contact.NewService(queries)
	c1, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1111111111", Label: "mobile"}},
	})
	_, _ = contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Bob",
		Phones: []contact.PhoneInput{{Phone: "+2222222222", Label: "mobile"}},
	})

	// Add tag to only one contact
	_, _ = queries.CreateContactTag(ctx, sqlcgen.CreateContactTagParams{TenantID: tenant.ID, Name: "vip"})
	tag, _ := queries.GetContactTagByName(ctx, sqlcgen.GetContactTagByNameParams{TenantID: tenant.ID, Name: "vip"})
	_ = queries.AddTagToContact(ctx, sqlcgen.AddTagToContactParams{ContactID: c1.ID, TagID: tag.ID})

	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Test",
		ChannelType: "whatsapp",
		Config:      []byte(`{"waba_id":"123"}`),
		Status:      "active",
	})

	templateSvc := template.NewService(queries, nil)
	tmpl, _ := templateSvc.Create(ctx, tenant.ID, template.CreateRequest{
		Name:     "announcement",
		Category: "MARKETING",
		Language: "en",
		Body:     "Announcement!",
	})

	mockCh := &mockBroadcastChannel{}
	svc := NewService(queries, mockCh, &NoopRateLimiter{})

	bcast, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Title:      "VIP Announcement",
		TemplateID: &tmpl.ID,
		ChannelID:  &ch.ID,
	})

	// Add recipients by tag filter
	err := svc.AddRecipients(ctx, tenant.ID, bcast.ID, AddRecipientsRequest{
		TagFilter: "vip",
	})
	if err != nil {
		t.Fatalf("AddRecipients by tag failed: %v", err)
	}

	recipients, count, _ := svc.ListRecipients(ctx, bcast.ID, 10, 0)
	if count != 1 {
		t.Fatalf("expected 1 recipient from tag filter, got %d", count)
	}
	if recipients[0].ContactID != c1.ID {
		t.Errorf("expected contact Alice, got %v", recipients[0].ContactID)
	}
}

func TestIntegrationBroadcastSchedulerTask(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	tenant := setupTenant(ctx, t, queries)

	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Test",
		ChannelType: "whatsapp",
		Config:      []byte(`{"waba_id":"123"}`),
		Status:      "active",
	})

	templateSvc := template.NewService(queries, nil)
	tmpl, _ := templateSvc.Create(ctx, tenant.ID, template.CreateRequest{
		Name:     "test",
		Category: "MARKETING",
		Language: "en",
		Body:     "Test!",
	})

	contactSvc := contact.NewService(queries)
	c1, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1111111111", Label: "mobile"}},
	})

	mockCh := &mockBroadcastChannel{}
	svc := NewService(queries, mockCh, &NoopRateLimiter{})

	// Create a broadcast scheduled in the past (so it's ready immediately)
	pastTime := time.Now().UTC().Add(-1 * time.Minute)
	bcast, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Title:       "Scheduled Campaign",
		TemplateID:  &tmpl.ID,
		ChannelID:   &ch.ID,
		ScheduledAt: &pastTime,
	})

	_ = svc.AddRecipients(ctx, tenant.ID, bcast.ID, AddRecipientsRequest{
		ContactIDs: []uuid.UUID{c1.ID},
	})

	// Run scheduler task
	sched := NewSchedulerTask(svc)
	err := sched.Run(ctx)
	if err != nil {
		t.Fatalf("Scheduler task failed: %v", err)
	}

	// Wait for background send
	time.Sleep(2 * time.Second)

	// Verify message was sent
	if len(mockCh.calls) != 1 {
		t.Fatalf("expected 1 channel call, got %d", len(mockCh.calls))
	}

	// Verify broadcast is completed
	completed, _ := svc.Get(ctx, tenant.ID, bcast.ID)
	if completed.Status != "completed" {
		t.Errorf("expected status completed, got %s", completed.Status)
	}
}
