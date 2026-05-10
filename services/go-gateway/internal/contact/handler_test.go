package contact

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autotraka/go-gateway/internal/auth"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func setupContactTest(t *testing.T) (*Service, uuid.UUID, func()) {
	pool, cleanup := testutil.SetupTestDB(t)
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	ctx := context.Background()
	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Acme", Mode: "human_first"})

	return svc, tenant.ID, cleanup
}

func injectTenant(tenantID uuid.UUID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.WithTenantID(r.Context(), tenantID)
			ctx = auth.WithMemberID(ctx, uuid.New())
			ctx = auth.WithRole(ctx, "admin")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TestCreateContactEndpoint(t *testing.T) {
	svc, tenantID, cleanup := setupContactTest(t)
	defer cleanup()

	handler := NewHandler(svc)
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(injectTenant(tenantID))
		handler.RegisterRoutes(r)
	})

	reqBody := CreateRequest{
		Name:   "John Doe",
		Email:  "john@example.com",
		Phones: []PhoneInput{{Phone: "+15551234567"}},
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp auth.Envelope
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	if data["name"] != "John Doe" {
		t.Errorf("expected name John Doe, got %v", data["name"])
	}
}

func TestListContactsEnforcesTenantScope(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	tenantA, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "A", Mode: "human_first"})
	tenantB, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "B", Mode: "human_first"})

	svc.Create(ctx, tenantA.ID, CreateRequest{Name: "Contact A"})
	svc.Create(ctx, tenantB.ID, CreateRequest{Name: "Contact B"})

	handler := NewHandler(svc)
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(injectTenant(tenantA.ID))
		handler.RegisterRoutes(r)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp auth.Envelope
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	contacts := data["contacts"].([]interface{})
	if len(contacts) != 1 {
		t.Errorf("expected 1 contact for tenant A, got %d", len(contacts))
	}
}

func TestGetContactWrongTenantReturns404(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	tenantA, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "A", Mode: "human_first"})
	tenantB, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "B", Mode: "human_first"})
	contact, _ := svc.Create(ctx, tenantA.ID, CreateRequest{Name: "John"})

	handler := NewHandler(svc)
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(injectTenant(tenantB.ID))
		handler.RegisterRoutes(r)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/"+contact.ID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestMergeContactsEndpoint(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Acme", Mode: "human_first"})
	target, _ := svc.Create(ctx, tenant.ID, CreateRequest{Name: "Target"})
	source, _ := svc.Create(ctx, tenant.ID, CreateRequest{Name: "Source"})

	handler := NewHandler(svc)
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(injectTenant(tenant.ID))
		handler.RegisterRoutes(r)
	})

	body := map[string]uuid.UUID{"source_contact_id": source.ID}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/"+target.ID.String()+"/merge", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
