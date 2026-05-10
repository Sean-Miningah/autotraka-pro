package channel

import (
	"errors"
	"net/http"

	"github.com/autotraka/go-gateway/internal/auth"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// HealthHandler exposes channel health endpoints.
type HealthHandler struct {
	queries *sqlcgen.Queries
}

// NewHealthHandler creates a channel health handler.
func NewHealthHandler(queries *sqlcgen.Queries) *HealthHandler {
	return &HealthHandler{queries: queries}
}

func (h *HealthHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/channels/{id}/health", h.GetHealth)
}

func (h *HealthHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		auth.WriteJSON(w, http.StatusBadRequest, auth.Envelope{Error: "invalid channel id"})
		return
	}

	ch, err := h.queries.GetChannelHealth(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "channel not found"})
			return
		}
		auth.WriteJSON(w, http.StatusInternalServerError, auth.Envelope{Error: "internal error"})
		return
	}

	// Verify tenant scope
	if ch.TenantID != tenantID {
		auth.WriteJSON(w, http.StatusNotFound, auth.Envelope{Error: "channel not found"})
		return
	}

	auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: map[string]interface{}{
		"id":                ch.ID,
		"name":              ch.Name,
		"channel_type":      ch.ChannelType,
		"health_status":     ch.HealthStatus,
		"health_checked_at": ch.HealthCheckedAt,
		"last_error":        ch.LastError,
	}})
}
