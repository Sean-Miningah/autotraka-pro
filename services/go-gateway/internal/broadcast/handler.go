package broadcast

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/autotraka/go-gateway/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler exposes broadcast HTTP endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a broadcast HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/broadcasts", h.Create)
	r.Get("/api/v1/broadcasts", h.List)
	r.Get("/api/v1/broadcasts/{id}", h.Get)
	r.Patch("/api/v1/broadcasts/{id}", h.Update)
	r.Delete("/api/v1/broadcasts/{id}", h.Delete)
	r.Post("/api/v1/broadcasts/{id}/send", h.Trigger)
	r.Post("/api/v1/broadcasts/{id}/recipients", h.AddRecipients)
	r.Get("/api/v1/broadcasts/{id}/recipients", h.ListRecipients)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid request body"})
		return
	}
	if req.Title == "" {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "title is required"})
		return
	}

	tenantID := auth.GetTenantID(r.Context())
	bcast, err := h.service.Create(r.Context(), tenantID, req)
	if err != nil {
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}
	auth.WriteJSON(w, http.StatusCreated, auth.Envelope{Data: bcast})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())

	limit := int32(20)
	offset := int32(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.ParseInt(l, 10, 32); err == nil && v > 0 {
			limit = int32(v)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.ParseInt(o, 10, 32); err == nil && v >= 0 {
			offset = int32(v)
		}
	}

	broadcasts, count, err := h.service.List(r.Context(), tenantID, limit, offset)
	if err != nil {
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: map[string]interface{}{
		"broadcasts": broadcasts,
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
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid broadcast id"})
		return
	}

	bcast, err := h.service.Get(r.Context(), tenantID, id)
	if err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "broadcast not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	// Include recipient summary
	summary, _ := h.service.GetRecipientSummary(r.Context(), id)

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: map[string]interface{}{
		"broadcast":          bcast,
		"recipient_summary": summary,
	}})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid broadcast id"})
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid request body"})
		return
	}

	bcast, err := h.service.Update(r.Context(), tenantID, id, req)
	if err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "broadcast not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}
	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: bcast})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid broadcast id"})
		return
	}

	if err := h.service.Delete(r.Context(), tenantID, id); err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "broadcast not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}
	auth.WriteJSON(w, http.StatusNoContent, auth.Envelope{})
}

func (h *Handler) Trigger(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid broadcast id"})
		return
	}

	bcast, err := h.service.Trigger(r.Context(), tenantID, id)
	if err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "broadcast not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}
	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: bcast})
}

func (h *Handler) AddRecipients(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid broadcast id"})
		return
	}

	var req AddRecipientsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid request body"})
		return
	}

	if err := h.service.AddRecipients(r.Context(), tenantID, id, req); err != nil {
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}
	auth.WriteJSON(w, http.StatusNoContent, auth.Envelope{})
}

func (h *Handler) ListRecipients(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid broadcast id"})
		return
	}

	// Verify broadcast exists and belongs to tenant
	_, err = h.service.Get(r.Context(), tenantID, id)
	if err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "broadcast not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	limit := int32(50)
	offset := int32(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.ParseInt(l, 10, 32); err == nil && v > 0 {
			limit = int32(v)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.ParseInt(o, 10, 32); err == nil && v >= 0 {
			offset = int32(v)
		}
	}

	recipients, count, err := h.service.ListRecipients(r.Context(), id, limit, offset)
	if err != nil {
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: map[string]interface{}{
		"recipients": recipients,
		"pagination": map[string]interface{}{
			"total":  count,
			"limit":  limit,
			"offset": offset,
		},
	}})
}
