package analytics

import (
	"context"
	"time"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Service provides analytics business logic.
type Service struct {
	queries *sqlcgen.Queries
}

// NewService creates an analytics service.
func NewService(queries *sqlcgen.Queries) *Service {
	return &Service{queries: queries}
}

// Overview contains aggregated analytics totals for a date range.
type Overview struct {
	ConversationsOpen      float64 `json:"conversations_open"`
	ConversationsPending   float64 `json:"conversations_pending"`
	ConversationsEscalated float64 `json:"conversations_escalated"`
	ConversationsResolved  float64 `json:"conversations_resolved"`
	ConversationsClosed    float64 `json:"conversations_closed"`
	MessagesInbound        float64 `json:"messages_inbound"`
	MessagesOutbound       float64 `json:"messages_outbound"`
}

// ConversationBreakdown is a single row of conversation analytics.
type ConversationBreakdown struct {
	MetricType  string  `json:"metric_type"`
	ChannelType string  `json:"channel_type"`
	Value       float64 `json:"value"`
}

// MessageVolume is a single row of message volume analytics.
type MessageVolume struct {
	ID          uuid.UUID `json:"id"`
	Date        string    `json:"date"`
	ChannelType string    `json:"channel_type"`
	MetricType  string    `json:"metric_type"`
	Value       float64   `json:"value"`
}

// GetOverview returns aggregated totals for a date range.
func (s *Service) GetOverview(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*Overview, error) {
	rows, err := s.queries.GetAnalyticsOverview(ctx, sqlcgen.GetAnalyticsOverviewParams{
		TenantID: tenantID,
		Date:     pgtype.Date{Time: from, Valid: true},
		Date_2:   pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	overview := &Overview{}
	for _, row := range rows {
		switch row.MetricType {
		case "conversations_open":
			overview.ConversationsOpen += row.TotalValue
		case "conversations_pending":
			overview.ConversationsPending += row.TotalValue
		case "conversations_escalated":
			overview.ConversationsEscalated += row.TotalValue
		case "conversations_resolved":
			overview.ConversationsResolved += row.TotalValue
		case "conversations_closed":
			overview.ConversationsClosed += row.TotalValue
		case "messages_inbound":
			overview.MessagesInbound += row.TotalValue
		case "messages_outbound":
			overview.MessagesOutbound += row.TotalValue
		}
	}

	return overview, nil
}

// GetConversations returns conversation analytics broken down by status and channel.
func (s *Service) GetConversations(ctx context.Context, tenantID uuid.UUID, from, to time.Time, channelType string, limit, offset int32) ([]ConversationBreakdown, error) {
	rows, err := s.queries.GetAnalyticsConversationsByStatus(ctx, sqlcgen.GetAnalyticsConversationsByStatusParams{
		TenantID: tenantID,
		Date:     pgtype.Date{Time: from, Valid: true},
		Date_2:   pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	result := make([]ConversationBreakdown, 0, len(rows))
	for _, row := range rows {
		channel := ""
		if row.ChannelType.Valid {
			channel = row.ChannelType.String
		}
		// Filter by channel type if specified
		if channelType != "" && channel != channelType {
			continue
		}
		result = append(result, ConversationBreakdown{
			MetricType:  row.MetricType,
			ChannelType: channel,
			Value:       float64(row.TotalValue),
		})
	}

	// Apply offset/limit (in-memory since the aggregation is already done at DB level)
	if offset > 0 {
		if int(offset) >= len(result) {
			return nil, nil
		}
		result = result[offset:]
	}
	if limit > 0 && int(limit) < len(result) {
		result = result[:limit]
	}

	return result, nil
}

// GetMessages returns message volume with cursor-based pagination.
func (s *Service) GetMessages(ctx context.Context, tenantID uuid.UUID, from, to time.Time, cursor uuid.UUID, limit int32) ([]MessageVolume, uuid.UUID, error) {
	if limit <= 0 {
		limit = 20
	}

	params := sqlcgen.GetMessageVolumeCursorParams{
		TenantID: tenantID,
		Date:     pgtype.Date{Time: from, Valid: true},
		Date_2:   pgtype.Date{Time: to, Valid: true},
		ID:       cursor,
		Limit:    limit + 1, // fetch one extra to determine if there's a next page
	}

	rows, err := s.queries.GetMessageVolumeCursor(ctx, params)
	if err != nil {
		return nil, uuid.Nil, err
	}

	var nextCursor uuid.UUID
	hasMore := len(rows) > int(limit)
	if hasMore {
		nextCursor = rows[len(rows)-2].ID // last item before the extra one
		rows = rows[:len(rows)-1]
	}

	result := make([]MessageVolume, 0, len(rows))
	for _, row := range rows {
		channel := ""
		if row.ChannelType.Valid {
			channel = row.ChannelType.String
		}
		result = append(result, MessageVolume{
			ID:          row.ID,
			Date:        row.Date.Time.Format("2006-01-02"),
			ChannelType: channel,
			MetricType:  row.MetricType,
			Value:       row.Value,
		})
	}

	return result, nextCursor, nil
}