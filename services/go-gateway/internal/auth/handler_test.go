package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func TestRegisterEndpoint(t *testing.T) {
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := sqlcgen.New(pool)
	svc := NewService(queries, []byte("test-jwt-secret-key-32byt"))
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", handler.Register)

	body := registerRequest{
		TenantName: "Acme Corp",
		Email:      "admin@acme.com",
		Password:   "securepassword123",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("response data is not an object")
	}

	if data["tenant_id"] == nil {
		t.Error("expected tenant_id in response")
	}
	if data["member_id"] == nil {
		t.Error("expected member_id in response")
	}
	if data["email"] != "admin@acme.com" {
		t.Errorf("expected email admin@acme.com, got %v", data["email"])
	}
	if data["role"] != "admin" {
		t.Errorf("expected role admin, got %v", data["role"])
	}
}