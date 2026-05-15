package broadcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNotFound = errors.New("broadcast not found")

// RateLimiter controls outbound message rate per tenant/channel.
type RateLimiter interface {
	Allow(ctx context.Context, tenantID uuid.UUID, channelID uuid.UUID) (bool, error)
}

// Broadcast is the enriched broadcast model.
type Broadcast struct {
	ID          uuid.UUID              `json:"id"`
	TenantID    uuid.UUID              `json:"tenant_id"`
	Title       string                 `json:"title"`
	TemplateID  *uuid.UUID             `json:"template_id,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
	ChannelID   *uuid.UUID             `json:"channel_id,omitempty"`
	Status      string                 `json:"status"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Recipient represents a broadcast recipient with contact info.
type Recipient struct {
	ID            uuid.UUID `json:"id"`
	BroadcastID   uuid.UUID `json:"broadcast_id"`
	ContactID     uuid.UUID `json:"contact_id"`
	ContactName   string    `json:"contact_name,omitempty"`
	ContactPhone  string    `json:"contact_phone,omitempty"`
	Status        string    `json:"status"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	Error         string    `json:"error,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// RecipientSummary holds aggregate counts per status.
type RecipientSummary struct {
	Pending int64 `json:"pending"`
	Sent    int64 `json:"sent"`
	Failed  int64 `json:"failed"`
	Total   int64 `json:"total"`
}

// CreateRequest holds data for creating a broadcast.
type CreateRequest struct {
	Title       string                 `json:"title"`
	TemplateID  *uuid.UUID             `json:"template_id,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	ChannelID   *uuid.UUID             `json:"channel_id,omitempty"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
}

// UpdateRequest holds optional fields for patching a broadcast.
type UpdateRequest struct {
	Title       *string                 `json:"title,omitempty"`
	TemplateID  *uuid.UUID              `json:"template_id,omitempty"`
	Parameters  *map[string]interface{} `json:"parameters,omitempty"`
	ChannelID   *uuid.UUID              `json:"channel_id,omitempty"`
	ScheduledAt *time.Time              `json:"scheduled_at,omitempty"`
}

// AddRecipientsRequest holds contact IDs or a tag filter.
type AddRecipientsRequest struct {
	ContactIDs []uuid.UUID `json:"contact_ids,omitempty"`
	TagFilter  string      `json:"tag_filter,omitempty"`
}

// Service handles broadcast business logic.
type Service struct {
	queries     *sqlcgen.Queries
	ch          channel.Channel
	rateLimiter RateLimiter
}

// NewService creates a broadcast service.
func NewService(queries *sqlcgen.Queries, ch channel.Channel, rateLimiter RateLimiter) *Service {
	return &Service{queries: queries, ch: ch, rateLimiter: rateLimiter}
}

// Create stores a new broadcast scoped to the tenant.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, req CreateRequest) (*Broadcast, error) {
	paramsBytes, _ := json.Marshal(req.Parameters)

	status := sqlcgen.BroadcastStatusDraft
	var scheduledAt pgtype.Timestamptz
	if req.ScheduledAt != nil {
		status = sqlcgen.BroadcastStatusScheduled
		scheduledAt = pgtype.Timestamptz{Time: *req.ScheduledAt, Valid: true}
	}

	var templateID pgtype.UUID
	if req.TemplateID != nil {
		templateID = pgtype.UUID{Bytes: *req.TemplateID, Valid: true}
	}
	var channelID pgtype.UUID
	if req.ChannelID != nil {
		channelID = pgtype.UUID{Bytes: *req.ChannelID, Valid: true}
	}

	row, err := s.queries.CreateBroadcast(ctx, sqlcgen.CreateBroadcastParams{
		TenantID:    tenantID,
		Title:       req.Title,
		TemplateID:  templateID,
		Parameters:  paramsBytes,
		ChannelID:   channelID,
		Status:      status,
		ScheduledAt: scheduledAt,
	})
	if err != nil {
		return nil, err
	}
	return fromSQLC(row), nil
}

// List returns broadcasts scoped to a tenant with pagination.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, limit, offset int32) ([]Broadcast, int64, error) {
	rows, err := s.queries.ListBroadcastsByTenant(ctx, sqlcgen.ListBroadcastsByTenantParams{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := s.queries.CountBroadcastsByTenant(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]Broadcast, len(rows))
	for i, r := range rows {
		result[i] = *fromSQLC(r)
	}
	return result, count, nil
}

// Get returns a single broadcast by ID scoped to tenant.
func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*Broadcast, error) {
	row, err := s.queries.GetBroadcastByID(ctx, sqlcgen.GetBroadcastByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return fromSQLC(row), nil
}

// Update patches a broadcast's fields.
func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateRequest) (*Broadcast, error) {
	existing, err := s.queries.GetBroadcastByID(ctx, sqlcgen.GetBroadcastByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	title := existing.Title
	if req.Title != nil {
		title = *req.Title
	}
	templateID := existing.TemplateID
	if req.TemplateID != nil {
		templateID = pgtype.UUID{Bytes: *req.TemplateID, Valid: true}
	}
	params := existing.Parameters
	if req.Parameters != nil {
		params, _ = json.Marshal(*req.Parameters)
	}
	channelID := existing.ChannelID
	if req.ChannelID != nil {
		channelID = pgtype.UUID{Bytes: *req.ChannelID, Valid: true}
	}
	status := existing.Status
	scheduledAt := existing.ScheduledAt
	if req.ScheduledAt != nil {
		scheduledAt = pgtype.Timestamptz{Time: *req.ScheduledAt, Valid: true}
		if status == sqlcgen.BroadcastStatusDraft {
			status = sqlcgen.BroadcastStatusScheduled
		}
	}

	updated, err := s.queries.UpdateBroadcast(ctx, sqlcgen.UpdateBroadcastParams{
		Title:       title,
		TemplateID:  templateID,
		Parameters:  params,
		ChannelID:   channelID,
		Status:      status,
		ScheduledAt: scheduledAt,
		ID:          id,
		TenantID:    tenantID,
	})
	if err != nil {
		return nil, err
	}
	return fromSQLC(updated), nil
}

// Delete removes a broadcast scoped to tenant.
func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.queries.DeleteBroadcast(ctx, sqlcgen.DeleteBroadcastParams{
		ID:       id,
		TenantID: tenantID,
	})
}

// Trigger immediately starts sending a broadcast.
func (s *Service) Trigger(ctx context.Context, tenantID, id uuid.UUID) (*Broadcast, error) {
	existing, err := s.queries.GetBroadcastByID(ctx, sqlcgen.GetBroadcastByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if existing.Status != sqlcgen.BroadcastStatusDraft && existing.Status != sqlcgen.BroadcastStatusScheduled {
		return nil, fmt.Errorf("cannot trigger broadcast with status %s", existing.Status)
	}

	updated, err := s.queries.UpdateBroadcastStatus(ctx, sqlcgen.UpdateBroadcastStatusParams{
		Status:      sqlcgen.BroadcastStatusSending,
		StartedAt:   pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		CompletedAt: pgtype.Timestamptz{},
		ID:          id,
		TenantID:    tenantID,
	})
	if err != nil {
		return nil, err
	}

	// Start sending in background
	go s.sendBroadcast(context.Background(), id)

	return fromSQLC(updated), nil
}

// AddRecipients adds contacts to a broadcast individually or by tag filter.
func (s *Service) AddRecipients(ctx context.Context, tenantID, broadcastID uuid.UUID, req AddRecipientsRequest) error {
	// Verify broadcast exists and belongs to tenant
	_, err := s.queries.GetBroadcastByID(ctx, sqlcgen.GetBroadcastByIDParams{
		ID:       broadcastID,
		TenantID: tenantID,
	})
	if err != nil {
		return err
	}

	var contactIDs []uuid.UUID
	if len(req.ContactIDs) > 0 {
		contactIDs = req.ContactIDs
	} else if req.TagFilter != "" {
		contacts, err := s.queries.FindContactsByTag(ctx, sqlcgen.FindContactsByTagParams{
			TenantID: tenantID,
			Name:     req.TagFilter,
		})
		if err != nil {
			return err
		}
		contactIDs = make([]uuid.UUID, len(contacts))
		for i, c := range contacts {
			contactIDs[i] = c.ID
		}
	}

	if len(contactIDs) == 0 {
		return nil
	}

	// Use copyfrom for bulk insert
	params := make([]sqlcgen.AddBroadcastRecipientsParams, len(contactIDs))
	for i, cid := range contactIDs {
		params[i] = sqlcgen.AddBroadcastRecipientsParams{
			BroadcastID: broadcastID,
			ContactID:   cid,
			Status:      sqlcgen.BroadcastRecipientStatusPending,
		}
	}
	_, err = s.queries.AddBroadcastRecipients(ctx, params)
	return err
}

// ListRecipients returns recipients for a broadcast with pagination.
func (s *Service) ListRecipients(ctx context.Context, broadcastID uuid.UUID, limit, offset int32) ([]Recipient, int64, error) {
	rows, err := s.queries.ListBroadcastRecipients(ctx, sqlcgen.ListBroadcastRecipientsParams{
		BroadcastID: broadcastID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := s.queries.CountBroadcastRecipients(ctx, broadcastID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]Recipient, len(rows))
	for i, r := range rows {
		result[i] = Recipient{
			ID:          r.ID,
			BroadcastID: r.BroadcastID,
			ContactID:   r.ContactID,
			Status:      string(r.Status),
			CreatedAt:   r.CreatedAt,
		}
		if r.ContactName.Valid {
			result[i].ContactName = r.ContactName.String
		}
		if r.ContactPhone.Valid {
			result[i].ContactPhone = r.ContactPhone.String
		}
		if r.SentAt.Valid {
			t := r.SentAt.Time
			result[i].SentAt = &t
		}
		if r.Error.Valid {
			result[i].Error = r.Error.String
		}
	}
	return result, count, nil
}

// GetRecipientSummary returns aggregate recipient counts.
func (s *Service) GetRecipientSummary(ctx context.Context, broadcastID uuid.UUID) (*RecipientSummary, error) {
	total, err := s.queries.CountBroadcastRecipients(ctx, broadcastID)
	if err != nil {
		return nil, err
	}
	pending, _ := s.queries.CountBroadcastRecipientsByStatus(ctx, sqlcgen.CountBroadcastRecipientsByStatusParams{
		BroadcastID: broadcastID,
		Status:      sqlcgen.BroadcastRecipientStatusPending,
	})
	sent, _ := s.queries.CountBroadcastRecipientsByStatus(ctx, sqlcgen.CountBroadcastRecipientsByStatusParams{
		BroadcastID: broadcastID,
		Status:      sqlcgen.BroadcastRecipientStatusSent,
	})
	failed, _ := s.queries.CountBroadcastRecipientsByStatus(ctx, sqlcgen.CountBroadcastRecipientsByStatusParams{
		BroadcastID: broadcastID,
		Status:      sqlcgen.BroadcastRecipientStatusFailed,
	})

	return &RecipientSummary{
		Pending: pending,
		Sent:    sent,
		Failed:  failed,
		Total:   total,
	}, nil
}

// sendBroadcast sends messages to all pending recipients.
func (s *Service) sendBroadcast(ctx context.Context, broadcastID uuid.UUID) {
	// Fetch broadcast to get template and channel info
	bcast, err := s.queries.GetBroadcastByIDForWorker(ctx, broadcastID)
	if err != nil {
		return
	}

	if !bcast.ChannelID.Valid || !bcast.TemplateID.Valid {
		_, _ = s.queries.UpdateBroadcastStatus(ctx, sqlcgen.UpdateBroadcastStatusParams{
			Status:      sqlcgen.BroadcastStatusFailed,
			StartedAt:   bcast.StartedAt,
			CompletedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			ID:          broadcastID,
			TenantID:    bcast.TenantID,
		})
		return
	}

	// Get template info
	tmpl, err := s.queries.GetTemplateByID(ctx, sqlcgen.GetTemplateByIDParams{
		ID:       bcast.TemplateID.Bytes,
		TenantID: bcast.TenantID,
	})
	if err != nil {
		_, _ = s.queries.UpdateBroadcastStatus(ctx, sqlcgen.UpdateBroadcastStatusParams{
			Status:      sqlcgen.BroadcastStatusFailed,
			StartedAt:   bcast.StartedAt,
			CompletedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			ID:          broadcastID,
			TenantID:    bcast.TenantID,
		})
		return
	}

	var params []string
	if len(bcast.Parameters) > 0 {
		var paramMap map[string]interface{}
		_ = json.Unmarshal(bcast.Parameters, &paramMap)
		// Map parameters by template parameter names
		if len(tmpl.Parameters) > 0 {
			var defs []struct{ Name string `json:"name"` }
			_ = json.Unmarshal(tmpl.Parameters, &defs)
			for _, d := range defs {
				if v, ok := paramMap[d.Name]; ok {
					params = append(params, fmt.Sprintf("%v", v))
				} else {
					params = append(params, "")
				}
			}
		}
	}

	for {
		// Get pending recipients in batches
		recipients, err := s.queries.GetPendingRecipientsForBroadcast(ctx, sqlcgen.GetPendingRecipientsForBroadcastParams{
			BroadcastID: broadcastID,
			Limit:         100,
		})
		if err != nil || len(recipients) == 0 {
			break
		}

		for _, r := range recipients {
			// Rate limit check
			if s.rateLimiter != nil {
				allowed, err := s.rateLimiter.Allow(ctx, bcast.TenantID, bcast.ChannelID.Bytes)
				if err != nil || !allowed {
					// Skip this batch if rate limited; they'll be retried
					break
				}
			}

			phone := ""
			if r.ContactPhone.Valid {
				phone = r.ContactPhone.String
			}
			err := s.ch.SendTemplateMessage(ctx, phone, tmpl.Name, tmpl.Language, params)
			if err != nil {
				_ = s.queries.UpdateBroadcastRecipientStatus(ctx, sqlcgen.UpdateBroadcastRecipientStatusParams{
					Status: sqlcgen.BroadcastRecipientStatusFailed,
					SentAt: pgtype.Timestamptz{},
					Error:  pgtype.Text{String: err.Error(), Valid: true},
					ID:     r.ID,
				})
			} else {
				_ = s.queries.UpdateBroadcastRecipientStatus(ctx, sqlcgen.UpdateBroadcastRecipientStatusParams{
					Status: sqlcgen.BroadcastRecipientStatusSent,
					SentAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
					Error:  pgtype.Text{},
					ID:     r.ID,
				})
			}
		}
	}

	// Mark broadcast as completed
	_, _ = s.queries.UpdateBroadcastStatus(ctx, sqlcgen.UpdateBroadcastStatusParams{
		Status:      sqlcgen.BroadcastStatusCompleted,
		StartedAt:   bcast.StartedAt,
		CompletedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		ID:          broadcastID,
		TenantID:    bcast.TenantID,
	})
}

func fromSQLC(b sqlcgen.Broadcast) *Broadcast {
	bc := &Broadcast{
		ID:        b.ID,
		TenantID:  b.TenantID,
		Title:     b.Title,
		Status:    string(b.Status),
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
	if b.TemplateID.Valid {
		id := uuid.UUID(b.TemplateID.Bytes)
		bc.TemplateID = &id
	}
	if b.ChannelID.Valid {
		id := uuid.UUID(b.ChannelID.Bytes)
		bc.ChannelID = &id
	}
	if b.ScheduledAt.Valid {
		t := b.ScheduledAt.Time
		bc.ScheduledAt = &t
	}
	if b.StartedAt.Valid {
		t := b.StartedAt.Time
		bc.StartedAt = &t
	}
	if b.CompletedAt.Valid {
		t := b.CompletedAt.Time
		bc.CompletedAt = &t
	}
	if len(b.Parameters) > 0 {
		_ = json.Unmarshal(b.Parameters, &bc.Parameters)
	}
	return bc
}
