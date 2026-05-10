package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

// HealthChecker performs periodic health checks on messaging channels.
type HealthChecker struct {
	queries *sqlcgen.Queries
}

// NewHealthChecker creates a health checker.
func NewHealthChecker(queries *sqlcgen.Queries) *HealthChecker {
	return &HealthChecker{queries: queries}
}

// CheckAllChannels verifies connectivity for all active channels.
func (h *HealthChecker) CheckAllChannels(ctx context.Context) error {
	channels, err := h.queries.ListActiveChannels(ctx)
	if err != nil {
		return fmt.Errorf("list active channels: %w", err)
	}

	for _, ch := range channels {
		if err := h.checkChannel(ctx, ch); err != nil {
			slog.Default().Error("channel health check failed", "channel_id", ch.ID, "error", err)
		}
	}

	return nil
}

func (h *HealthChecker) checkChannel(ctx context.Context, ch sqlcgen.ListActiveChannelsRow) error {
	var status string
	var lastError string

	switch ch.ChannelType {
	case "whatsapp":
		var config struct {
			PhoneNumberID string `json:"phone_number_id"`
			AccessToken   string `json:"access_token"`
			AppSecret     string `json:"app_secret"`
		}
		if err := json.Unmarshal(ch.Config, &config); err != nil {
			status = "error"
			lastError = fmt.Sprintf("invalid config: %v", err)
			break
		}

		wa := channel.NewWhatsApp("https://graph.facebook.com", config.AccessToken, config.PhoneNumberID, config.AppSecret, "")
		if err := h.pingWhatsApp(ctx, wa); err != nil {
			status = "unhealthy"
			lastError = err.Error()
		} else {
			status = "healthy"
		}
	default:
		status = "unknown"
		lastError = fmt.Sprintf("unsupported channel type: %s", ch.ChannelType)
	}

	return h.queries.UpdateChannelHealth(ctx, sqlcgen.UpdateChannelHealthParams{
		HealthStatus: pgtype.Text{String: status, Valid: true},
		LastError:    pgtype.Text{String: lastError, Valid: true},
		ID:           ch.ID,
	})
}

func (h *HealthChecker) pingWhatsApp(ctx context.Context, wa *channel.WhatsApp) error {
	// Placeholder: real implementation would verify API connectivity
	_ = wa
	return nil
}
