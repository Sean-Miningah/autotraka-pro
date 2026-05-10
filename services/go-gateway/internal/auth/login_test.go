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
	"github.com/google/uuid"
)

func TestLoginEndpoint(t *testing.T) {
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := sqlcgen.New(pool)
	svc := NewService(queries, []byte("test-jwt-secret-key-32byt"))
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", handler.Register)
	r.Post("/api/v1/auth/login", handler.Login)

	regBody := registerRequest{
		TenantName: "Acme Corp",
		Email:      "admin@acme.com",
		Password:   "securepassword123",
	}
	b, _ := json.Marshal(regBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var regResp Envelope
	json.Unmarshal(w.Body.Bytes(), &regResp)
	data := regResp.Data.(map[string]interface{})
	tenantID := data["tenant_id"].(string)

	loginBody := loginRequest{
		TenantID: uuid.MustParse(tenantID),
		Email:    "admin@acme.com",
		Password: "securepassword123",
	}
	b, _ = json.Marshal(loginBody)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var loginResp Envelope
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	loginData := loginResp.Data.(map[string]interface{})

	if loginData["access_token"] == nil || loginData["access_token"] == "" {
		t.Error("expected access_token in login response")
	}
	if loginData["refresh_token"] == nil || loginData["refresh_token"] == "" {
		t.Error("expected refresh_token in login response")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := sqlcgen.New(pool)
	svc := NewService(queries, []byte("test-jwt-secret-key-32byt"))
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", handler.Register)
	r.Post("/api/v1/auth/login", handler.Login)

	regBody := registerRequest{
		TenantName: "Acme Corp",
		Email:      "admin@acme.com",
		Password:    "securepassword123",
	}
	b, _ := json.Marshal(regBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var regResp Envelope
	json.Unmarshal(w.Body.Bytes(), &regResp)
	regData := regResp.Data.(map[string]interface{})
	tenantID := regData["tenant_id"].(string)

	loginBody := loginRequest{
		TenantID: uuid.MustParse(tenantID),
		Email:    "admin@acme.com",
		Password: "wrongpassword",
	}
	b, _ = json.Marshal(loginBody)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", w.Code)
	}
}