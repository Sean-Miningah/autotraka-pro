package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Engine executes automation flows in response to inbound events.
type Engine struct {
	queries *sqlcgen.Queries
	ch      channel.Channel
}

// NewEngine creates an automation engine.
func NewEngine(queries *sqlcgen.Queries, ch channel.Channel) *Engine {
	return &Engine{queries: queries, ch: ch}
}

// ProcessInboundMessage checks active automations for keyword triggers and executes matched flows.
func (e *Engine) ProcessInboundMessage(ctx context.Context, tenantID, conversationID uuid.UUID, messageText string) error {
	if e.queries == nil || e.ch == nil {
		return nil
	}

	automations, err := e.queries.ListActiveAutomationsByTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("list active automations: %w", err)
	}

	for _, auto := range automations {
		var def FlowDefinition
		if err := json.Unmarshal(auto.Definition, &def); err != nil {
			continue
		}

		triggerNode, matched := matchTrigger(def, messageText)
		if !matched {
			continue
		}

		// Find the next node from the trigger
		nextNodeID := findNextNode(def, triggerNode.ID)
		if nextNodeID == "" {
			continue
		}

		nextNode, ok := findNodeByID(def, nextNodeID)
		if !ok {
			continue
		}

		// Create automation run
		run, err := e.queries.CreateAutomationRun(ctx, sqlcgen.CreateAutomationRunParams{
			AutomationID: auto.ID,
			TenantID:     tenantID,
			ConversationID: pgtype.UUID{Bytes: conversationID, Valid: true},
			Status:       sqlcgen.AutomationRunStatusRunning,
		})
		if err != nil {
			continue
		}

		// Execute the next node (in this slice: only send_message)
		if nextNode.Type == "send_message" {
			msg, _ := nextNode.Config["message"].(string)
			if msg != "" {
				// Get conversation to find contact, then get phone
				conv, err := e.queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
					ID:       conversationID,
					TenantID: tenantID,
				})
				if err != nil {
					_, _ = e.queries.UpdateAutomationRunStatus(ctx, sqlcgen.UpdateAutomationRunStatusParams{
						Status: sqlcgen.AutomationRunStatusFailed,
						ID:     run.ID,
					})
					continue
				}
				phones, err := e.queries.ListContactPhones(ctx, conv.ContactID)
				if err != nil || len(phones) == 0 {
					_, _ = e.queries.UpdateAutomationRunStatus(ctx, sqlcgen.UpdateAutomationRunStatusParams{
						Status: sqlcgen.AutomationRunStatusFailed,
						ID:     run.ID,
					})
					continue
				}
				to := phones[0].Phone
				_ = e.ch.SendTextMessage(ctx, to, msg)
			}
		}

		_, _ = e.queries.UpdateAutomationRunStatus(ctx, sqlcgen.UpdateAutomationRunStatusParams{
			Status: sqlcgen.AutomationRunStatusCompleted,
			ID:     run.ID,
		})
	}

	return nil
}

func matchTrigger(def FlowDefinition, text string) (FlowNode, bool) {
	for _, n := range def.Nodes {
		if n.Type != "trigger" {
			continue
		}
		keywordsRaw, ok := n.Config["keywords"]
		if !ok {
			continue
		}
		keywords, ok := keywordsRaw.([]interface{})
		if !ok {
			continue
		}
		for _, k := range keywords {
			kw, ok := k.(string)
			if !ok {
				continue
			}
			// Simple case-insensitive contains match
			if strings.Contains(strings.ToLower(text), strings.ToLower(kw)) {
				return n, true
			}
		}
	}
	return FlowNode{}, false
}

func findNextNode(def FlowDefinition, fromID string) string {
	for _, e := range def.Edges {
		if e.Source == fromID {
			return e.Target
		}
	}
	return ""
}

func findNodeByID(def FlowDefinition, id string) (FlowNode, bool) {
	for _, n := range def.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return FlowNode{}, false
}
