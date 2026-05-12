package template

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNotFound = errors.New("template not found")

// ParameterDef represents a named parameter definition for a template.
type ParameterDef struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// CreateRequest holds data for creating a template.
type CreateRequest struct {
	ChannelID        uuid.UUID      `json:"channel_id"`
	ParentTemplateID uuid.UUID      `json:"parent_template_id"`
	Name             string         `json:"name"`
	Category         string         `json:"category"`
	Language         string         `json:"language"`
	Body             string         `json:"body"`
	Parameters       []ParameterDef `json:"parameters"`
}

// Template is the enriched template model.
type Template struct {
	ID               uuid.UUID      `json:"id"`
	TenantID         uuid.UUID      `json:"tenant_id"`
	ChannelID        *uuid.UUID     `json:"channel_id,omitempty"`
	ParentTemplateID *uuid.UUID     `json:"parent_template_id,omitempty"`
	Name             string         `json:"name"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Language         string         `json:"language"`
	Body             string         `json:"body"`
	Parameters       []ParameterDef `json:"parameters"`
	MetaTemplateID   *string        `json:"meta_template_id,omitempty"`
	CreatedAt        string         `json:"created_at"`
	UpdatedAt        string         `json:"updated_at"`
}

// MetaTemplateClient abstracts the Meta WhatsApp Templates API.
type MetaTemplateClient interface {
	CreateTemplate(ctx context.Context, wabaID string, req MetaCreateTemplateReq) (metaTemplateID string, err error)
	GetTemplateStatus(ctx context.Context, wabaID, templateName string) (status string, err error)
}

// MetaCreateTemplateReq is the payload for creating a template on Meta.
type MetaCreateTemplateReq struct {
	Name       string
	Category   string
	Language   string
	Body       string
	Parameters []ParameterDef
}

// Service handles template business logic.
type Service struct {
	queries    *sqlcgen.Queries
	metaClient MetaTemplateClient
}

// NewService creates a template service.
func NewService(queries *sqlcgen.Queries, metaClient MetaTemplateClient) *Service {
	return &Service{queries: queries, metaClient: metaClient}
}

// UpdateRequest holds optional fields for patching a template.
type UpdateRequest struct {
	Name       *string        `json:"name,omitempty"`
	Category   *string        `json:"category,omitempty"`
	Status     *string        `json:"status,omitempty"`
	Language   *string        `json:"language,omitempty"`
	Body       *string        `json:"body,omitempty"`
	Parameters []ParameterDef `json:"parameters,omitempty"`
}

// Update patches a template's fields.
func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateRequest) (*Template, error) {
	existing, err := s.queries.GetTemplateByID(ctx, sqlcgen.GetTemplateByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}
	category := existing.Category
	if req.Category != nil {
		category = *req.Category
	}
	status := existing.Status
	if req.Status != nil {
		status = sqlcgen.TemplateStatus(*req.Status)
	}
	language := existing.Language
	if req.Language != nil {
		language = *req.Language
	}
	body := existing.Body
	if req.Body != nil {
		body = *req.Body
	}
	params := existing.Parameters
	if req.Parameters != nil {
		params, _ = json.Marshal(req.Parameters)
	}

	updated, err := s.queries.UpdateTemplate(ctx, sqlcgen.UpdateTemplateParams{
		Name:           name,
		Category:       category,
		Status:         status,
		Language:       language,
		Body:           body,
		Parameters:     params,
		MetaTemplateID: existing.MetaTemplateID,
		ID:             id,
		TenantID:       tenantID,
	})
	if err != nil {
		return nil, err
	}
	return fromSQLC(updated), nil
}

// Delete removes a template scoped to tenant.
func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.queries.DeleteTemplate(ctx, sqlcgen.DeleteTemplateParams{
		ID:       id,
		TenantID: tenantID,
	})
}

// SyncPendingStatuses queries Meta for all pending templates and updates their status.
func (s *Service) SyncPendingStatuses(ctx context.Context) error {
	if s.metaClient == nil {
		return nil
	}

	pending, err := s.queries.ListPendingTemplates(ctx)
	if err != nil {
		return err
	}

	for _, t := range pending {
		if !t.ChannelID.Valid {
			continue
		}
		ch, err := s.queries.GetChannelByID(ctx, t.ChannelID.Bytes)
		if err != nil {
			continue
		}
		var cfg struct {
			WABAID string `json:"waba_id"`
		}
		if err := json.Unmarshal(ch.Config, &cfg); err != nil || cfg.WABAID == "" {
			continue
		}

		status, err := s.metaClient.GetTemplateStatus(ctx, cfg.WABAID, t.Name)
		if err != nil {
			continue
		}

		var newStatus sqlcgen.TemplateStatus
		switch status {
		case "approved":
			newStatus = sqlcgen.TemplateStatusApproved
		case "rejected":
			newStatus = sqlcgen.TemplateStatusRejected
		default:
			continue
		}

		_, _ = s.queries.UpdateTemplate(ctx, sqlcgen.UpdateTemplateParams{
			Name:           t.Name,
			Category:       t.Category,
			Status:         newStatus,
			Language:       t.Language,
			Body:           t.Body,
			Parameters:     t.Parameters,
			MetaTemplateID: t.MetaTemplateID,
			ID:             t.ID,
			TenantID:       t.TenantID,
		})
	}

	return nil
}

// Get returns a single template by ID scoped to tenant.
func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*Template, error) {
	row, err := s.queries.GetTemplateByID(ctx, sqlcgen.GetTemplateByIDParams{
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

// List returns templates scoped to a tenant with optional status filter and pagination.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int32) ([]Template, int64, error) {
	var rows []sqlcgen.Template
	var err error
	if status != "" {
		rows, err = s.queries.ListTemplatesByTenantAndStatus(ctx, sqlcgen.ListTemplatesByTenantAndStatusParams{
			TenantID: tenantID,
			Status:   sqlcgen.TemplateStatus(status),
		})
	} else {
		rows, err = s.queries.ListTemplatesByTenant(ctx, sqlcgen.ListTemplatesByTenantParams{
			TenantID: tenantID,
			Limit:    limit,
			Offset:   offset,
		})
	}
	if err != nil {
		return nil, 0, err
	}

	count, err := s.queries.CountTemplatesByTenant(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]Template, len(rows))
	for i, r := range rows {
		t := fromSQLC(r)
		result[i] = *t
	}
	return result, count, nil
}

// Create stores a new template scoped to the tenant and submits to Meta if client is configured.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, req CreateRequest) (*Template, error) {
	paramsBytes, err := json.Marshal(req.Parameters)
	if err != nil {
		return nil, err
	}

	channelID := req.ChannelID
	if channelID == uuid.Nil {
		channels, err := s.queries.ListChannelsByTenantAndType(ctx, sqlcgen.ListChannelsByTenantAndTypeParams{
			TenantID:    tenantID,
			ChannelType: "whatsapp",
		})
		if err == nil && len(channels) > 0 {
			channelID = channels[0].ID
		}
	}

	status := sqlcgen.TemplateStatusDraft
	var metaTemplateID pgtype.Text

	if s.metaClient != nil && channelID != uuid.Nil {
		ch, err := s.queries.GetChannelByID(ctx, channelID)
		if err != nil {
			return nil, fmt.Errorf("get channel: %w", err)
		}
		var cfg struct {
			WABAID string `json:"waba_id"`
		}
		if err := json.Unmarshal(ch.Config, &cfg); err == nil && cfg.WABAID != "" {
			metaReq := MetaCreateTemplateReq{
				Name:       req.Name,
				Category:   req.Category,
				Language:   req.Language,
				Body:       req.Body,
				Parameters: req.Parameters,
			}
			id, err := s.metaClient.CreateTemplate(ctx, cfg.WABAID, metaReq)
			if err != nil {
				return nil, fmt.Errorf("meta create template: %w", err)
			}
			status = sqlcgen.TemplateStatusPending
			metaTemplateID = pgtype.Text{String: id, Valid: true}
		}
	}

	row, err := s.queries.CreateTemplate(ctx, sqlcgen.CreateTemplateParams{
		TenantID:         tenantID,
		ChannelID:        pgtype.UUID{Bytes: channelID, Valid: channelID != uuid.Nil},
		ParentTemplateID: pgtype.UUID{Bytes: req.ParentTemplateID, Valid: req.ParentTemplateID != uuid.Nil},
		Name:             req.Name,
		Category:         req.Category,
		Status:           status,
		Language:         req.Language,
		Body:             req.Body,
		Parameters:       paramsBytes,
		MetaTemplateID:   metaTemplateID,
	})
	if err != nil {
		return nil, err
	}

	return fromSQLC(row), nil
}

func fromSQLC(t sqlcgen.Template) *Template {
	tmpl := &Template{
		ID:             t.ID,
		TenantID:       t.TenantID,
		Name:           t.Name,
		Category:       t.Category,
		Status:         string(t.Status),
		Language:       t.Language,
		Body:           t.Body,
		MetaTemplateID: textPtr(t.MetaTemplateID),
		CreatedAt:      t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      t.UpdatedAt.Format(time.RFC3339),
	}
	if t.ChannelID.Valid {
		id := uuid.UUID(t.ChannelID.Bytes)
		tmpl.ChannelID = &id
	}
	if t.ParentTemplateID.Valid {
		id := uuid.UUID(t.ParentTemplateID.Bytes)
		tmpl.ParentTemplateID = &id
	}
	if len(t.Parameters) > 0 {
		_ = json.Unmarshal(t.Parameters, &tmpl.Parameters)
	}
	return tmpl
}

func textPtr(t pgtype.Text) *string {
	if t.Valid {
		return &t.String
	}
	return nil
}
