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

func TestJWTMiddlewareRejectsUnauthenticated(t *testing.T) {
	protected := JWTMiddleware([]byte("test-jwt-secret-key-32byt"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, Envelope{Data: "secret"})
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", w.Code)
	}
}

func TestJWTMiddlewareAcceptsValidToken(t *testing.T) {
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := sqlcgen.New(pool)
	secret := []byte("test-jwt-secret-key-32byt")
	svc := NewService(queries, secret)
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
	regData := regResp.Data.(map[string]interface{})
	tenantID := regData["tenant_id"].(string)

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
	accessToken := loginData["access_token"].(string)

	protected := JWTMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := GetTenantID(r.Context())
		memberID := GetMemberID(r.Context())
		role := GetRole(r.Context())
		WriteJSON(w, http.StatusOK, Envelope{Data: map[string]interface{}{
			"tenant_id": tenantID,
			"member_id": memberID,
			"role":       role,
		}})
	}))

	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := sqlcgen.New(pool)
	svc := NewService(queries, []byte("test-jwt-secret-key-32byt"))
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", handler.Register)
	r.Post("/api/v1/auth/login", handler.Login)
	r.Post("/api/v1/auth/refresh", handler.Refresh)

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

	var regResp Envelope
	json.Unmarshal(w.Body.Bytes(), &regResp)
	regData := regResp.Data.(map[string]interface{})
	tenantID := regData["tenant_id"].(string)

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

	var loginResp Envelope
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	loginData := loginResp.Data.(map[string]interface{})
	firstRefreshToken := loginData["refresh_token"].(string)

	refreshReq := refreshRequest{RefreshToken: firstRefreshToken}
	b, _ = json.Marshal(refreshReq)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on refresh, got %d: %s", w.Code, w.Body.String())
	}

	var refreshResp Envelope
	json.Unmarshal(w.Body.Bytes(), &refreshResp)
	refreshData := refreshResp.Data.(map[string]interface{})

	newRefreshToken := refreshData["refresh_token"].(string)
	if newRefreshToken == firstRefreshToken {
		t.Error("refresh token should rotate to a new value")
	}

	b, _ = json.Marshal(refreshReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when reusing old refresh token, got %d", w.Code)
	}
}

func TestServiceTokenMiddleware(t *testing.T) {
	serviceToken := "internal-service-token"

	protected := ServiceTokenMiddleware(serviceToken)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, Envelope{Data: "ok"})
	}))

	req := httptest.NewRequest(http.MethodGet, "/internal/health", nil)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without service token, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/internal/health", nil)
	req.Header.Set("Authorization", "Bearer internal-service-token")
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with valid service token, got %d", w.Code)
	}
}