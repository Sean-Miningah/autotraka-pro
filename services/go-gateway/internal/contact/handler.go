package contact

import (
	"encoding/json"
	"net/http"

	"github.com/autotraka/go-gateway/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler exposes contact CRUD endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a contact HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/contacts", h.Create)
	r.Get("/api/v1/contacts", h.List)
	r.Get("/api/v1/contacts/{id}", h.Get)
	r.Patch("/api/v1/contacts/{id}", h.Update)
	r.Post("/api/v1/contacts/{id}/merge", h.Merge)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid request body"})
		return
	}

	tenantID := auth.GetTenantID(r.Context())
	contact, err := h.service.Create(r.Context(), tenantID, req)
	if err != nil {
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusCreated, auth.Envelope{Data: contact})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())

	limit := int32(20)
	offset := int32(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := parseInt32(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := parseInt32(o); err == nil && v >= 0 {
			offset = v
		}
	}

	contacts, count, err := h.service.List(r.Context(), tenantID, limit, offset)
	if err != nil {
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: map[string]interface{}{
		"contacts": contacts,
		"pagination": map[string]interface{}{
			"total":  count,
			"limit":  limit,
			"offset": offset,
		},
	}})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid contact id"})
		return
	}

	contact, err := h.service.Get(r.Context(), tenantID, id)
	if err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "contact not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: contact})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid contact id"})
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid request body"})
		return
	}

	contact, err := h.service.Update(r.Context(), tenantID, id, req)
	if err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "contact not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: contact})
}

func (h *Handler) Merge(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	targetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid target contact id"})
		return
	}

	var req struct {
		SourceContactID uuid.UUID `json:"source_contact_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid request body"})
		return
	}

	contact, err := h.service.Merge(r.Context(), tenantID, targetID, req.SourceContactID)
	if err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "contact not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: contact})
}

func parseInt32(s string) (int32, error) {
	var v int64
	err := json.Unmarshal([]byte(s), &v)
	return int32(v), err
}
