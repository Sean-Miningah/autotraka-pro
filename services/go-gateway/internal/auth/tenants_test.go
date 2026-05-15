package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func TestListTenantsByEmailOneTenant(t *testing.T) {
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := sqlcgen.New(pool)
	svc := NewService(queries, []byte("test-jwt-secret-key-32byt"))
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Get("/api/v1/auth/tenants", handler.ListTenantsByEmail)

	regReq := registerRequest{
		TenantName: "Acme Corp",
		Email:      "admin@acme.com",
		Password:   "securepassword123",
	}
	_, err := svc.Register(t.Context(), regReq)
	if err != nil {
		t.Fatalf("failed to register member: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/tenants?email=admin@acme.com", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("response data is not an array")
	}

	if len(data) != 1 {
		t.Fatalf("expected 1 tenant, got %d", len(data))
	}

	tenant := data[0].(map[string]interface{})
	if tenant["tenant_name"] != "Acme Corp" {
		t.Errorf("expected tenant_name 'Acme Corp', got %v", tenant["tenant_name"])
	}
	if tenant["tenant_id"] == nil {
		t.Error("expected tenant_id in response")
	}
}

func TestListTenantsByEmailMultipleTenants(t *testing.T) {
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := sqlcgen.New(pool)
	svc := NewService(queries, []byte("test-jwt-secret-key-32byt"))
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Get("/api/v1/auth/tenants", handler.ListTenantsByEmail)

	// Register the same email in two different tenants
	_, err := svc.Register(t.Context(), registerRequest{
		TenantName: "Acme Corp",
		Email:      "admin@shared.com",
		Password:   "securepassword123",
	})
	if err != nil {
		t.Fatalf("failed to register first member: %v", err)
	}

	_, err = svc.Register(t.Context(), registerRequest{
		TenantName: "Beta Inc",
		Email:      "admin@shared.com",
		Password:   "securepassword123",
	})
	if err != nil {
		t.Fatalf("failed to register second member: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/tenants?email=admin@shared.com", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("response data is not an array")
	}

	if len(data) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(data))
	}

	names := map[string]bool{}
	for _, item := range data {
		tenant := item.(map[string]interface{})
		names[tenant["tenant_name"].(string)] = true
	}
	if !names["Acme Corp"] || !names["Beta Inc"] {
		t.Errorf("expected both Acme Corp and Beta Inc, got %v", names)
	}
}

func TestListTenantsByEmailNotFound(t *testing.T) {
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := sqlcgen.New(pool)
	svc := NewService(queries, []byte("test-jwt-secret-key-32byt"))
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Get("/api/v1/auth/tenants", handler.ListTenantsByEmail)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/tenants?email=nobody@example.com", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}

	var resp Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "no tenants found" {
		t.Errorf("expected error 'no tenants found', got %q", resp.Error)
	}
}

func TestListTenantsByEmailMissingParam(t *testing.T) {
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := sqlcgen.New(pool)
	svc := NewService(queries, []byte("test-jwt-secret-key-32byt"))
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Get("/api/v1/auth/tenants", handler.ListTenantsByEmail)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/tenants", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "email query parameter is required" {
		t.Errorf("expected error 'email query parameter is required', got %q", resp.Error)
	}
}