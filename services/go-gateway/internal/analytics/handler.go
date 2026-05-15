package analytics

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/autotraka/go-gateway/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler exposes analytics HTTP endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates an analytics HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/analytics/overview", h.Overview)
	r.Get("/api/v1/analytics/conversations", h.Conversations)
	r.Get("/api/v1/analytics/messages", h.Messages)
}

func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())

	from, to, err := parseDateRange(r)
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: err.Error()})
		return
	}

	overview, err := h.service.GetOverview(r.Context(), tenantID, from, to)
	if err != nil {
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: overview})
}

func (h *Handler) Conversations(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())

	from, to, err := parseDateRange(r)
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: err.Error()})
		return
	}

	channelType := r.URL.Query().Get("channel")
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

	convs, err := h.service.GetConversations(r.Context(), tenantID, from, to, channelType, limit, offset)
	if err != nil {
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: convs})
}

func (h *Handler) Messages(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())

	from, to, err := parseDateRange(r)
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: err.Error()})
		return
	}

	cursor := uuid.Nil
	if c := r.URL.Query().Get("cursor"); c != "" {
		if parsed, err := uuid.Parse(c); err == nil {
			cursor = parsed
		}
	}

	limit := int32(20)
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.ParseInt(l, 10, 32); err == nil && v > 0 {
			limit = int32(v)
		}
	}

	msgs, nextCursor, err := h.service.GetMessages(r.Context(), tenantID, from, to, cursor, limit)
	if err != nil {
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	response := map[string]interface{}{
		"messages": msgs,
	}
	if nextCursor != uuid.Nil {
		response["next_cursor"] = nextCursor.String()
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: response})
}

// parseDateRange extracts from and to date query parameters (format: 2006-01-02).
// Defaults to last 7 days if not provided.
func parseDateRange(r *http.Request) (from, to time.Time, err error) {
	now := time.Now().UTC().Truncate(24 * time.Hour)

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" && toStr == "" {
		// Default: last 7 days
		return now.AddDate(0, 0, -7), now, nil
	}

	if fromStr != "" {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' date format, use YYYY-MM-DD")
		}
	}
	if toStr != "" {
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	if fromStr == "" {
		from = to.AddDate(0, 0, -7)
	}
	if toStr == "" {
		to = from.AddDate(0, 0, 7)
	}

	return from, to, nil
}