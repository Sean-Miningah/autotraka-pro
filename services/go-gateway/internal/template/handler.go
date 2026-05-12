package template

import (
	"encoding/json"
	"net/http"

	"github.com/autotraka/go-gateway/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler exposes template CRUD endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a template HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/templates", h.Create)
	r.Get("/api/v1/templates", h.List)
	r.Get("/api/v1/templates/{id}", h.Get)
	r.Patch("/api/v1/templates/{id}", h.Update)
	r.Delete("/api/v1/templates/{id}", h.Delete)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid request body"})
		return
	}

	if req.Name == "" || req.Body == "" {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "name and body are required"})
		return
	}

	tenantID := auth.GetTenantID(r.Context())
	tmpl, err := h.service.Create(r.Context(), tenantID, req)
	if err != nil {
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusCreated, auth.Envelope{Data: tmpl})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())

	status := r.URL.Query().Get("status")

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

	templates, count, err := h.service.List(r.Context(), tenantID, status, limit, offset)
	if err != nil {
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: map[string]interface{}{
		"templates": templates,
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
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid template id"})
		return
	}

	tmpl, err := h.service.Get(r.Context(), tenantID, id)
	if err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "template not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: tmpl})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid template id"})
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid request body"})
		return
	}

	tmpl, err := h.service.Update(r.Context(), tenantID, id, req)
	if err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "template not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: tmpl})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid template id"})
		return
	}

	if err := h.service.Delete(r.Context(), tenantID, id); err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "template not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusNoContent, auth.Envelope{})
}

func parseInt32(s string) (int32, error) {
	var v int64
	err := json.Unmarshal([]byte(s), &v)
	return int32(v), err
}
