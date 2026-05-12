package automation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("automation not found")

// FlowDefinition is the JSONB structure stored in the automations table.
type FlowDefinition struct {
	Nodes []FlowNode `json:"nodes"`
	Edges []FlowEdge `json:"edges"`
}

// FlowNode represents a node in the automation flow.
type FlowNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // trigger, send_message, etc.
	Position map[string]float64     `json:"position,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// FlowEdge represents a connection between nodes.
type FlowEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

// CreateRequest holds data for creating an automation.
type CreateRequest struct {
	Name       string         `json:"name"`
	Definition FlowDefinition `json:"definition"`
}

// Automation is the enriched automation model.
type Automation struct {
	ID         uuid.UUID      `json:"id"`
	TenantID   uuid.UUID      `json:"tenant_id"`
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Definition FlowDefinition `json:"definition"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

// Service handles automation business logic.
type Service struct {
	queries *sqlcgen.Queries
}

// NewService creates an automation service.
func NewService(queries *sqlcgen.Queries) *Service {
	return &Service{queries: queries}
}

// Create stores a new automation scoped to the tenant.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, req CreateRequest) (*Automation, error) {
	defBytes, err := json.Marshal(req.Definition)
	if err != nil {
		return nil, err
	}

	row, err := s.queries.CreateAutomation(ctx, sqlcgen.CreateAutomationParams{
		TenantID:   tenantID,
		Name:       req.Name,
		Status:     sqlcgen.AutomationStatusDraft,
		Definition: defBytes,
	})
	if err != nil {
		return nil, err
	}

	return fromSQLC(row)
}

// List returns automations scoped to a tenant with pagination.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, limit, offset int32) ([]Automation, int64, error) {
	rows, err := s.queries.ListAutomationsByTenant(ctx, sqlcgen.ListAutomationsByTenantParams{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := s.queries.CountAutomationsByTenant(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]Automation, len(rows))
	for i, r := range rows {
		a, err := fromSQLC(r)
		if err != nil {
			return nil, 0, err
		}
		result[i] = *a
	}
	return result, count, nil
}

// Get returns a single automation by ID scoped to tenant.
func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*Automation, error) {
	row, err := s.queries.GetAutomationByID(ctx, sqlcgen.GetAutomationByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return fromSQLC(row)
}

// UpdateRequest holds optional fields for patching an automation.
type UpdateRequest struct {
	Name       *string         `json:"name,omitempty"`
	Status     *string         `json:"status,omitempty"`
	Definition *FlowDefinition `json:"definition,omitempty"`
}

// Update patches an automation's fields.
func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateRequest) (*Automation, error) {
	existing, err := s.queries.GetAutomationByID(ctx, sqlcgen.GetAutomationByIDParams{
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
	status := existing.Status
	if req.Status != nil {
		status = sqlcgen.AutomationStatus(*req.Status)
	}
	def := existing.Definition
	if req.Definition != nil {
		def, _ = json.Marshal(*req.Definition)
	}

	// Validate flow when activating
	if status == sqlcgen.AutomationStatusActive {
		var flowDef FlowDefinition
		if err := json.Unmarshal(def, &flowDef); err != nil {
			return nil, fmt.Errorf("%w: invalid definition JSON", ErrInvalidFlow)
		}
		if err := ValidateFlow(flowDef); err != nil {
			return nil, err
		}
	}

	updated, err := s.queries.UpdateAutomation(ctx, sqlcgen.UpdateAutomationParams{
		Name:       name,
		Status:     status,
		Definition: def,
		ID:         id,
		TenantID:   tenantID,
	})
	if err != nil {
		return nil, err
	}
	return fromSQLC(updated)
}

// Delete removes an automation scoped to tenant.
func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.queries.DeleteAutomation(ctx, sqlcgen.DeleteAutomationParams{
		ID:       id,
		TenantID: tenantID,
	})
}

var ErrInvalidFlow = errors.New("invalid flow definition")

// ValidateFlow checks that a flow definition is structurally valid for activation.
func ValidateFlow(def FlowDefinition) error {
	// Must have exactly one trigger node
	triggerCount := 0
	nodeIDs := make(map[string]bool)
	for _, n := range def.Nodes {
		nodeIDs[n.ID] = true
		if n.Type == "trigger" {
			triggerCount++
		}
	}
	if triggerCount != 1 {
		return fmt.Errorf("%w: expected exactly 1 trigger node, got %d", ErrInvalidFlow, triggerCount)
	}

	// All edges must reference existing nodes
	for _, e := range def.Edges {
		if !nodeIDs[e.Source] {
			return fmt.Errorf("%w: edge references unknown source node %q", ErrInvalidFlow, e.Source)
		}
		if !nodeIDs[e.Target] {
			return fmt.Errorf("%w: edge references unknown target node %q", ErrInvalidFlow, e.Target)
		}
	}

	// Graph must be traversable from trigger (all nodes reachable)
	adj := make(map[string][]string)
	var triggerID string
	for _, n := range def.Nodes {
		if n.Type == "trigger" {
			triggerID = n.ID
		}
	}
	for _, e := range def.Edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
	}

	visited := make(map[string]bool)
	var dfs func(string)
	dfs = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true
		for _, next := range adj[id] {
			dfs(next)
		}
	}
	dfs(triggerID)

	if len(visited) != len(def.Nodes) {
		return fmt.Errorf("%w: not all nodes are reachable from trigger", ErrInvalidFlow)
	}

	return nil
}

func fromSQLC(a sqlcgen.Automation) (*Automation, error) {
	var def FlowDefinition
	if err := json.Unmarshal(a.Definition, &def); err != nil {
		return nil, err
	}
	return &Automation{
		ID:         a.ID,
		TenantID:   a.TenantID,
		Name:       a.Name,
		Status:     string(a.Status),
		Definition: def,
		CreatedAt:  a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  a.UpdatedAt.Format(time.RFC3339),
	}, nil
}
