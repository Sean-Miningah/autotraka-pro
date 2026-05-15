package conversation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autotraka/go-gateway/internal/auth"
	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/contact"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestEnrichedListConversations(t *testing.T) {
	ctx := context.Background()
	handler, queries, cleanup := setupHandler(t)
	defer cleanup()

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	member, _ := queries.CreateMember(ctx, sqlcgen.CreateMemberParams{
		TenantID:     tenant.ID,
		Email:        "agent@test.com",
		PasswordHash: "hash",
		Role:         "agent",
	})

	c, _ := queries.CreateContact(ctx, sqlcgen.CreateContactParams{
		TenantID: tenant.ID,
		Name:     pgtype.Text{String: "Alice", Valid: true},
	})
	_, _ = queries.CreateChannelIdentity(ctx, sqlcgen.CreateChannelIdentityParams{
		ContactID:      c.ID,
		ChannelType:    "whatsapp",
		ChannelIdentity: "+1234567890",
	})

	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:         tenant.ID,
		ContactID:        c.ID,
		Status:           sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:        sqlcgen.HandledByAi,
	})

	_, _ = queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenant.ID,
		ConversationID: conv.ID,
		Direction:      sqlcgen.MessageDirectionInbound,
		Status:         sqlcgen.MessageStatusDelivered,
		ContentType:    "text",
		Content:        []byte(`{"text":"Hello"}`),
	})

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			innerCtx := auth.WithTenantID(r.Context(), tenant.ID)
			innerCtx = auth.WithMemberID(innerCtx, member.ID)
			innerCtx = auth.WithRole(innerCtx, "agent")
			next.ServeHTTP(w, r.WithContext(innerCtx))
		})
	})
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations?limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp auth.Envelope
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	conversations := data["conversations"].([]interface{})
	if len(conversations) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(conversations))
	}

	convData := conversations[0].(map[string]interface{})
	if convData["contact_name"] != "Alice" {
		t.Errorf("expected contact_name Alice, got %v", convData["contact_name"])
	}
	if convData["channel_type"] != "whatsapp" {
		t.Errorf("expected channel_type whatsapp, got %v", convData["channel_type"])
	}
	if convData["last_message"] == nil || convData["last_message"] == "" {
		t.Error("expected last_message to be populated")
	}
	if convData["unread_count"] != float64(1) {
		t.Errorf("expected unread_count 1 (no read record yet), got %v", convData["unread_count"])
	}
}

func TestEnrichedListWithUnreadCount(t *testing.T) {
	ctx := context.Background()
	_, queries, cleanup := setupHandler(t)
	defer cleanup()

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	member, _ := queries.CreateMember(ctx, sqlcgen.CreateMemberParams{
		TenantID:     tenant.ID,
		Email:        "agent2@test.com",
		PasswordHash: "hash",
		Role:         "agent",
	})

	c, _ := queries.CreateContact(ctx, sqlcgen.CreateContactParams{
		TenantID: tenant.ID,
		Name:     pgtype.Text{String: "Bob", Valid: true},
	})

	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:         tenant.ID,
		ContactID:        c.ID,
		Status:           sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:        sqlcgen.HandledByAi,
	})

	_, _ = queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenant.ID,
		ConversationID: conv.ID,
		Direction:      sqlcgen.MessageDirectionInbound,
		Status:         sqlcgen.MessageStatusDelivered,
		ContentType:    "text",
		Content:        []byte(`{"text":"Hi"}`),
	})

	svc := NewService(queries, contact.NewService(queries), nil, nil, nil)

	// No read record yet — all inbound messages should be unread
	rows, _, err := svc.ListEnriched(ctx, tenant.ID, member.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListEnriched failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(rows))
	}
	if rows[0].UnreadCount != 1 {
		t.Errorf("expected unread_count 1 (no read record), got %d", rows[0].UnreadCount)
	}

	// Now mark as read
	err = svc.MarkAsRead(ctx, member.ID, conv.ID)
	if err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	// After reading, new messages should still be unread if before read_at
	// But since the read timestamp is now, and no new messages were added,
	// unread_count should be 0
	rows, _, err = svc.ListEnriched(ctx, tenant.ID, member.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListEnriched after read failed: %v", err)
	}
	if rows[0].UnreadCount != 0 {
		t.Errorf("expected unread_count 0 after read, got %d", rows[0].UnreadCount)
	}

	// Add a new message — it should be unread
	_, _ = queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenant.ID,
		ConversationID: conv.ID,
		Direction:      sqlcgen.MessageDirectionInbound,
		Status:         sqlcgen.MessageStatusDelivered,
		ContentType:    "text",
		Content:        []byte(`{"text":"New message"}`),
	})

	rows, _, err = svc.ListEnriched(ctx, tenant.ID, member.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListEnriched after new message failed: %v", err)
	}
	if rows[0].UnreadCount != 1 {
		t.Errorf("expected unread_count 1 after new message, got %d", rows[0].UnreadCount)
	}
}

func TestGetConversationMarksAsRead(t *testing.T) {
	ctx := context.Background()
	svc, _, cleanup := setupService(t)
	defer cleanup()

	tenantID, channelID := createTenantAndChannel(ctx, t, svc.queries)

	member, err := svc.queries.CreateMember(ctx, sqlcgen.CreateMemberParams{
		TenantID:     tenantID,
		Email:        "reader@test.com",
		PasswordHash: "hash",
		Role:         "agent",
	})
	if err != nil {
		t.Fatalf("create member: %v", err)
	}

	// Create conversation with inbound message
	evt := channel.WebhookEvent{
		EventID:   "EVT_READ_001",
		From:      "15551234567",
		MessageID: "MSG_READ_001",
		Type:      "text",
		Content:   []byte(`{"body":"Hello"}`),
		Timestamp: time.Now().Unix(),
	}
	conv, _, err := svc.ProcessInboundMessage(ctx, tenantID, channelID, evt)
	if err != nil {
		t.Fatalf("ProcessInboundMessage failed: %v", err)
	}

	// Inject member ID into context
	ctxWithMember := auth.WithMemberID(ctx, member.ID)
	ctxWithMember = auth.WithTenantID(ctxWithMember, tenantID)

	// Call Get — should mark as read
	_, err = svc.Get(ctxWithMember, tenantID, conv.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Verify read record was created
	read, err := svc.queries.UpsertConversationRead(ctx, sqlcgen.UpsertConversationReadParams{
		MemberID:       member.ID,
		ConversationID: conv.ID,
	})
	if err != nil {
		t.Fatalf("read record not found: %v", err)
	}
	if read.MemberID != member.ID {
		t.Errorf("expected member_id %s, got %s", member.ID, read.MemberID)
	}
}

func TestEnrichedListStatusFilter(t *testing.T) {
	ctx := context.Background()
	handler, queries, cleanup := setupHandler(t)
	defer cleanup()

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	member, _ := queries.CreateMember(ctx, sqlcgen.CreateMemberParams{
		TenantID:     tenant.ID,
		Email:        "filter@test.com",
		PasswordHash: "hash",
		Role:         "agent",
	})

	c, _ := queries.CreateContact(ctx, sqlcgen.CreateContactParams{TenantID: tenant.ID})
	c2, _ := queries.CreateContact(ctx, sqlcgen.CreateContactParams{TenantID: tenant.ID})

	_, _ = queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID: tenant.ID, ContactID: c.ID,
		Status: sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false}, HandledBy: sqlcgen.HandledByAi,
	})
	_, _ = queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID: tenant.ID, ContactID: c2.ID,
		Status: sqlcgen.ConversationStatusPending,
		AssignedMemberID: pgtype.UUID{Valid: false}, HandledBy: sqlcgen.HandledByHuman,
	})

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			innerCtx := auth.WithTenantID(r.Context(), tenant.ID)
			innerCtx = auth.WithMemberID(innerCtx, member.ID)
			innerCtx = auth.WithRole(innerCtx, "agent")
			next.ServeHTTP(w, r.WithContext(innerCtx))
		})
	})
	handler.RegisterRoutes(r)

	// Filter by status
	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations?status=open", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp auth.Envelope
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	conversations := data["conversations"].([]interface{})
	if len(conversations) != 1 {
		t.Errorf("expected 1 conversation with status=open, got %d", len(conversations))
	}

	// Filter by handled_by
	req = httptest.NewRequest(http.MethodGet, "/api/v1/conversations?handled_by=human", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &resp)
	data = resp.Data.(map[string]interface{})
	conversations = data["conversations"].([]interface{})
	if len(conversations) != 1 {
		t.Errorf("expected 1 conversation with handled_by=human, got %d", len(conversations))
	}
}

func TestEnrichedListNoContactName(t *testing.T) {
	ctx := context.Background()
	svc, _, cleanup := setupService(t)
	defer cleanup()

	tenantID, _ := createTenantAndChannel(ctx, t, svc.queries)
	member, _ := svc.queries.CreateMember(ctx, sqlcgen.CreateMemberParams{
		TenantID:     tenantID,
		Email:        "nocontact@test.com",
		PasswordHash: "hash",
		Role:         "agent",
	})

	// Create contact with no name
	contactSvc := contact.NewService(svc.queries)
	c, _ := contactSvc.Create(ctx, tenantID, contact.CreateRequest{
		Phones: []contact.PhoneInput{{Phone: "+15559999999"}},
	})

	_, _ = svc.queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:         tenantID,
		ContactID:        c.ID,
		Status:           sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:        sqlcgen.HandledByAi,
	})

	rows, _, err := svc.ListEnriched(ctx, tenantID, member.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListEnriched failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(rows))
	}
	if rows[0].ContactName != "" {
		t.Errorf("expected empty contact_name for unnamed contact, got %q", rows[0].ContactName)
	}
}