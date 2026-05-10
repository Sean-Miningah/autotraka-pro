package conversation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autotraka/go-gateway/internal/auth"
	"github.com/autotraka/go-gateway/internal/contact"
	"github.com/autotraka/go-gateway/internal/eventbus"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func setupHandler(tb testing.TB) (*Handler, *sqlcgen.Queries, func()) {
	tb.Helper()

	ctx := context.Background()
	pool, dbCleanup := testutil.SetupTestDB(tb)
	queries := sqlcgen.New(pool)

	natsC, natsCleanup := startNATS(tb)
	natsURL, _ := natsC.ConnectionString(ctx)
	eb, _ := eventbus.New(natsURL, nil)

	contactSvc := contact.NewService(queries)
	svc := NewService(queries, contactSvc, eb)
	handler := NewHandler(svc)

	cleanup := func() {
		eb.Close()
		natsCleanup()
		dbCleanup()
	}

	return handler, queries, cleanup
}

func injectTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" {
			tenantIDStr = uuid.MustParse("00000000-0000-0000-0000-000000000001").String()
		}
		tenantID := uuid.MustParse(tenantIDStr)
		ctx := auth.WithTenantID(r.Context(), tenantID)
		ctx = auth.WithMemberID(ctx, uuid.New())
		ctx = auth.WithRole(ctx, "admin")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TestListConversationsEndpoint(t *testing.T) {
	ctx := context.Background()
	handler, queries, cleanup := setupHandler(t)
	defer cleanup()

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	c, _ := queries.CreateContact(ctx, sqlcgen.CreateContactParams{TenantID: tenant.ID})
	_, _ = queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:       tenant.ID,
		ContactID:      c.ID,
		Status:         sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:      sqlcgen.HandledByAi,
	})

	r := chi.NewRouter()
	r.Use(injectTenant)
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations?limit=10&offset=0", nil)
	req.Header.Set("X-Tenant-ID", tenant.ID.String())
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
		t.Errorf("expected 1 conversation, got %d", len(conversations))
	}
}

func TestGetConversationEndpoint(t *testing.T) {
	ctx := context.Background()
	handler, queries, cleanup := setupHandler(t)
	defer cleanup()

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	c, _ := queries.CreateContact(ctx, sqlcgen.CreateContactParams{TenantID: tenant.ID})
	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:       tenant.ID,
		ContactID:      c.ID,
		Status:         sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:      sqlcgen.HandledByAi,
	})

	r := chi.NewRouter()
	r.Use(injectTenant)
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+conv.ID.String(), nil)
	req.Header.Set("X-Tenant-ID", tenant.ID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp auth.Envelope
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	if data["id"] != conv.ID.String() {
		t.Errorf("expected conversation %v, got %v", conv.ID, data["id"])
	}
}

func TestPatchConversationStatusEndpoint(t *testing.T) {
	ctx := context.Background()
	handler, queries, cleanup := setupHandler(t)
	defer cleanup()

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	c, _ := queries.CreateContact(ctx, sqlcgen.CreateContactParams{TenantID: tenant.ID})
	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:       tenant.ID,
		ContactID:      c.ID,
		Status:         sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:      sqlcgen.HandledByAi,
	})

	r := chi.NewRouter()
	r.Use(injectTenant)
	handler.RegisterRoutes(r)

	body, _ := json.Marshal(map[string]interface{}{"status": "pending"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/conversations/"+conv.ID.String(), bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", tenant.ID.String())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp auth.Envelope
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	if data["status"] != "pending" {
		t.Errorf("expected status pending, got %v", data["status"])
	}
}

func TestSendMessageEndpoint(t *testing.T) {
	ctx := context.Background()
	handler, queries, cleanup := setupHandler(t)
	defer cleanup()

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID: tenant.ID, Name: "WA", ChannelType: "whatsapp",
		Config: []byte(`{}`), Status: "active",
	})
	c, _ := queries.CreateContact(ctx, sqlcgen.CreateContactParams{TenantID: tenant.ID})
	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:       tenant.ID,
		ContactID:      c.ID,
		Status:         sqlcgen.ConversationStatusOpen,
		AssignedMemberID: pgtype.UUID{Valid: false},
		HandledBy:      sqlcgen.HandledByAi,
	})

	// Add inbound message to set reply channel
	_, _ = queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenant.ID,
		ConversationID: conv.ID,
		ChannelID:      pgtype.UUID{Bytes: ch.ID, Valid: true},
		Direction:      sqlcgen.MessageDirectionInbound,
		Status:         sqlcgen.MessageStatusDelivered,
		ContentType:    "text",
		Content:        []byte(`{"text":"hello"}`),
	})

	r := chi.NewRouter()
	r.Use(injectTenant)
	handler.RegisterRoutes(r)

	body, _ := json.Marshal(map[string]interface{}{"content": map[string]string{"text": "Reply"}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/"+conv.ID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", tenant.ID.String())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp auth.Envelope
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	if data["status"] != "pending" {
		t.Errorf("expected status pending, got %v", data["status"])
	}
	if data["direction"] != "outbound" {
		t.Errorf("expected direction outbound, got %v", data["direction"])
	}
}
