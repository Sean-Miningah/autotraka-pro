package conversation

import (
	"encoding/json"
	"net/http"

	"github.com/autotraka/go-gateway/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler exposes conversation endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a conversation HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/conversations", h.List)
	r.Get("/api/v1/conversations/{id}", h.Get)
	r.Patch("/api/v1/conversations/{id}", h.Update)
	r.Post("/api/v1/conversations/{id}/messages", h.SendMessage)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	memberID := auth.GetMemberID(r.Context())

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

	conversations, count, err := h.service.ListEnriched(r.Context(), tenantID, memberID, limit, offset)
	if err != nil {
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	// Apply query filters in Go (pagination already applied at DB level)
	statusFilter := r.URL.Query().Get("status")
	handledByFilter := r.URL.Query().Get("handled_by")
	if statusFilter != "" || handledByFilter != "" {
		filtered := make([]EnrichedConversation, 0, len(conversations))
		for _, c := range conversations {
			if statusFilter != "" && c.Status != statusFilter {
				continue
			}
			if handledByFilter != "" && c.HandledBy != handledByFilter {
				continue
			}
			filtered = append(filtered, c)
		}
		conversations = filtered
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: map[string]interface{}{
		"conversations": conversations,
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
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid conversation id"})
		return
	}

	conv, err := h.service.Get(r.Context(), tenantID, id)
	if err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "conversation not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: conv})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid conversation id"})
		return
	}

	var req UpdateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid request body"})
		return
	}

	conv, err := h.service.Update(r.Context(), tenantID, id, req)
	if err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "conversation not found"})
			return
		}
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: err.Error()})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: conv})
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid conversation id"})
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid request body"})
		return
	}

	msg, err := h.service.SendMessage(r.Context(), tenantID, id, req)
	if err != nil {
		if err == ErrNotFound {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "conversation not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusCreated, auth.Envelope{Data: msg})
}

func parseInt32(s string) (int32, error) {
	var v int64
	err := json.Unmarshal([]byte(s), &v)
	return int32(v), err
}
