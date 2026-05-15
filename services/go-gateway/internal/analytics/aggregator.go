package analytics

import (
	"context"
	"log/slog"
	"time"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// AggregatorTask computes daily analytics snapshots from conversations and messages.
// It runs as a scheduled task (typically at midnight) and upserts aggregated
// metrics into analytics_daily for the previous day.
type AggregatorTask struct {
	queries *sqlcgen.Queries
}

// NewAggregatorTask creates a new aggregation scheduler task.
func NewAggregatorTask(queries *sqlcgen.Queries) *AggregatorTask {
	return &AggregatorTask{queries: queries}
}

// Run aggregates previous-day metrics for all tenants.
// This is the entry point called by the scheduler.
func (t *AggregatorTask) Run(ctx context.Context) error {
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	return t.RunForDate(ctx, yesterday)
}

// RunForDate aggregates metrics for a specific date for all tenants.
// This is useful for testing and backfilling.
func (t *AggregatorTask) RunForDate(ctx context.Context, date time.Time) error {
	tenants, err := t.queries.ListTenants(ctx)
	if err != nil {
		return err
	}

	for _, tenant := range tenants {
		if err := t.aggregateTenant(ctx, tenant.ID, date); err != nil {
			slog.Error("failed to aggregate analytics for tenant", "tenant_id", tenant.ID, "error", err)
			continue
		}
	}

	return nil
}

func (t *AggregatorTask) aggregateTenant(ctx context.Context, tenantID uuid.UUID, date time.Time) error {
	datePG := pgtype.Date{Time: date, Valid: true}
	dayStart := date
	dayEnd := date.AddDate(0, 0, 1)

	// --- Conversation metrics ---
	statusRows, err := t.queries.AggregateConversationsByStatus(ctx, sqlcgen.AggregateConversationsByStatusParams{
		TenantID:    tenantID,
		CreatedAt:   dayStart,
		CreatedAt_2: dayEnd,
	})
	if err != nil {
		return err
	}

	for _, row := range statusRows {
		metricType := conversationMetricForStatus(string(row.Status))
		if err := t.queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
			TenantID:    tenantID,
			Date:        datePG,
			ChannelType: pgtype.Text{Valid: false},
			MetricType:  metricType,
			Value:       float64(row.Count),
		}); err != nil {
			return err
		}
	}

	// Also store overall conversation total (sum across all statuses)
	var totalConvs int64
	for _, row := range statusRows {
		totalConvs += row.Count
	}
	if totalConvs > 0 {
		if err := t.queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
			TenantID:    tenantID,
			Date:        datePG,
			ChannelType: pgtype.Text{Valid: false},
			MetricType:  "conversations_total",
			Value:       float64(totalConvs),
		}); err != nil {
			return err
		}
	}

	// Per-channel conversation counts
	channelRows, err := t.queries.AggregateConversationsByChannel(ctx, sqlcgen.AggregateConversationsByChannelParams{
		TenantID:    tenantID,
		CreatedAt:   dayStart,
		CreatedAt_2: dayEnd,
	})
	if err != nil {
		return err
	}

	for _, row := range channelRows {
		metricType := conversationMetricForStatus(string(row.Status))
		if err := t.queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
			TenantID:    tenantID,
			Date:        datePG,
			ChannelType: pgtype.Text{String: row.ChannelType, Valid: true},
			MetricType:  metricType,
			Value:       float64(row.Count),
		}); err != nil {
			return err
		}
	}

	// Overall conversation total per channel
	channelTotalRows, err := t.queries.AggregateConversationsTotalByChannel(ctx, sqlcgen.AggregateConversationsTotalByChannelParams{
		TenantID:    tenantID,
		CreatedAt:   dayStart,
		CreatedAt_2: dayEnd,
	})
	if err != nil {
		return err
	}
	for _, row := range channelTotalRows {
		if err := t.queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
			TenantID:    tenantID,
			Date:        datePG,
			ChannelType: pgtype.Text{String: row.ChannelType, Valid: true},
			MetricType:  "conversations_total",
			Value:       float64(row.Count),
		}); err != nil {
			return err
		}
	}

	// Handled_by breakdown
	handledByRows, err := t.queries.AggregateConversationsByHandledBy(ctx, sqlcgen.AggregateConversationsByHandledByParams{
		TenantID:    tenantID,
		CreatedAt:   dayStart,
		CreatedAt_2: dayEnd,
	})
	if err != nil {
		return err
	}

	for _, row := range handledByRows {
		metricType := "conversations_handled_" + string(row.HandledBy)
		if err := t.queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
			TenantID:    tenantID,
			Date:        datePG,
			ChannelType: pgtype.Text{Valid: false},
			MetricType:  metricType,
			Value:       float64(row.Count),
		}); err != nil {
			return err
		}
	}

	// Per-agent conversation counts
	agentRows, err := t.queries.AggregateConversationsByAgent(ctx, sqlcgen.AggregateConversationsByAgentParams{
		TenantID:    tenantID,
		CreatedAt:   dayStart,
		CreatedAt_2: dayEnd,
	})
	if err != nil {
		return err
	}
	for _, row := range agentRows {
		metricType := "conversations_assigned"
		if row.AssignedMemberID.Valid {
			metricType = "conversations_assigned_" + uuid.UUID(row.AssignedMemberID.Bytes).String()
		} else {
			metricType = "conversations_unassigned"
		}
		if err := t.queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
			TenantID:    tenantID,
			Date:        datePG,
			ChannelType: pgtype.Text{Valid: false},
			MetricType:  metricType,
			Value:       float64(row.Count),
		}); err != nil {
			return err
		}
	}

	// --- Message metrics ---
	msgRows, err := t.queries.AggregateMessagesByDirection(ctx, sqlcgen.AggregateMessagesByDirectionParams{
		TenantID:    tenantID,
		CreatedAt:   dayStart,
		CreatedAt_2: dayEnd,
	})
	if err != nil {
		return err
	}

	for _, row := range msgRows {
		metricType := "messages_inbound"
		if row.Direction == sqlcgen.MessageDirectionOutbound {
			metricType = "messages_outbound"
		}
		if err := t.queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
			TenantID:    tenantID,
			Date:        datePG,
			ChannelType: pgtype.Text{Valid: false},
			MetricType:  metricType,
			Value:       float64(row.Count),
		}); err != nil {
			return err
		}
	}

	// Per-channel message counts
	msgChannelRows, err := t.queries.AggregateMessagesByChannel(ctx, sqlcgen.AggregateMessagesByChannelParams{
		TenantID:    tenantID,
		CreatedAt:   dayStart,
		CreatedAt_2: dayEnd,
	})
	if err != nil {
		return err
	}

	for _, row := range msgChannelRows {
		metricType := "messages_inbound"
		if row.Direction == sqlcgen.MessageDirectionOutbound {
			metricType = "messages_outbound"
		}
		if err := t.queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
			TenantID:    tenantID,
			Date:        datePG,
			ChannelType: pgtype.Text{String: row.ChannelType, Valid: true},
			MetricType:  metricType,
			Value:       float64(row.Count),
		}); err != nil {
			return err
		}
	}

	return nil
}

func conversationMetricForStatus(status string) string {
	switch status {
	case "open":
		return "conversations_open"
	case "pending":
		return "conversations_pending"
	case "escalated":
		return "conversations_escalated"
	case "resolved":
		return "conversations_resolved"
	case "closed":
		return "conversations_closed"
	default:
		return "conversations_total"
	}
}