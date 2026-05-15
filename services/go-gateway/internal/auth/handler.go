package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

type Envelope struct {
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListTenantsByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		WriteJSON(w, http.StatusBadRequest, Envelope{Error: "email query parameter is required"})
		return
	}

	tenants, err := h.service.ListTenantsByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, ErrNoTenants) {
			WriteJSON(w, http.StatusNotFound, Envelope{Error: "no tenants found"})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, Envelope{Error: "internal error"})
		return
	}

	WriteJSON(w, http.StatusOK, Envelope{Data: tenants})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, Envelope{Error: "invalid request body"})
		return
	}

	if req.TenantName == "" || req.Email == "" || req.Password == "" {
		WriteJSON(w, http.StatusBadRequest, Envelope{Error: "tenant_name, email, and password are required"})
		return
	}

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			WriteJSON(w, http.StatusConflict, Envelope{Error: "email already taken"})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, Envelope{Error: "internal error"})
		return
	}

	WriteJSON(w, http.StatusCreated, Envelope{Data: resp})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, Envelope{Error: "invalid request body"})
		return
	}

	if req.Email == "" || req.Password == "" || req.TenantID == uuid.Nil {
		WriteJSON(w, http.StatusBadRequest, Envelope{Error: "tenant_id, email, and password are required"})
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCreds) {
			WriteJSON(w, http.StatusUnauthorized, Envelope{Error: "invalid email or password"})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, Envelope{Error: "internal error"})
		return
	}

	WriteJSON(w, http.StatusOK, Envelope{Data: resp})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, Envelope{Error: "invalid request body"})
		return
	}

	if req.RefreshToken == "" {
		WriteJSON(w, http.StatusBadRequest, Envelope{Error: "refresh_token is required"})
		return
	}

	resp, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) || errors.Is(err, ErrTokenExpired) {
			WriteJSON(w, http.StatusUnauthorized, Envelope{Error: err.Error()})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, Envelope{Error: "internal error"})
		return
	}

	WriteJSON(w, http.StatusOK, Envelope{Data: resp})
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}