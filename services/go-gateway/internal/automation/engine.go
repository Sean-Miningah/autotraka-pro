package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// AIRequestFunc is the function signature for requesting AI responses.
type AIRequestFunc func(ctx context.Context, subject string, payload map[string]interface{}) (string, error)

// Engine executes automation flows in response to inbound events.
type Engine struct {
	queries       *sqlcgen.Queries
	ch            channel.Channel
	aiRequestFunc AIRequestFunc
}

// NewEngine creates an automation engine.
func NewEngine(queries *sqlcgen.Queries, ch channel.Channel) *Engine {
	return &Engine{queries: queries, ch: ch}
}

// ProcessInboundMessage checks active automations for keyword triggers and executes matched flows.
// It also resumes any waiting automation runs for the conversation.
func (e *Engine) ProcessInboundMessage(ctx context.Context, tenantID, conversationID uuid.UUID, messageText string) error {
	if e.queries == nil || e.ch == nil {
		return nil
	}

	// First, resume any waiting runs for this conversation
	if err := e.resumeWaitingRuns(ctx, tenantID, conversationID, messageText); err != nil {
		return err
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

		// Find the first node after trigger
		nextNodeID := findNextNode(def, triggerNode.ID, "")
		if nextNodeID == "" {
			continue
		}

		// Create automation run
		run, err := e.queries.CreateAutomationRun(ctx, sqlcgen.CreateAutomationRunParams{
			AutomationID:   auto.ID,
			TenantID:       tenantID,
			ConversationID: pgtype.UUID{Bytes: conversationID, Valid: true},
			Status:         sqlcgen.AutomationRunStatusRunning,
			CurrentNodeID:  pgtype.Text{String: nextNodeID, Valid: true},
			Variables:      []byte("{}"),
		})
		if err != nil {
			continue
		}

		// Execute flow starting from the first node
		e.executeFlow(ctx, &run, def, tenantID, conversationID, messageText)
	}

	return nil
}

// resumeWaitingRuns finds waiting runs for a conversation and resumes them along the "replied" edge.
func (e *Engine) resumeWaitingRuns(ctx context.Context, tenantID, conversationID uuid.UUID, messageText string) error {
	waitingRuns, err := e.queries.GetWaitingRunsByConversation(ctx, pgtype.UUID{Bytes: conversationID, Valid: true})
	if err != nil {
		return fmt.Errorf("get waiting runs: %w", err)
	}

	for _, run := range waitingRuns {
		auto, err := e.queries.GetAutomationByID(ctx, sqlcgen.GetAutomationByIDParams{
			ID:       run.AutomationID,
			TenantID: tenantID,
		})
		if err != nil {
			continue
		}

		var def FlowDefinition
		if err := json.Unmarshal(auto.Definition, &def); err != nil {
			continue
		}

		// Find the wait_for_reply node
		node, ok := findNodeByID(def, run.CurrentNodeID.String)
		if !ok || node.Type != "wait_for_reply" {
			continue
		}

		// Get the replied edge target
		repliedTarget := findNextNode(def, node.ID, "replied")
		if repliedTarget == "" {
			continue
		}

		// Update current_node_id to replied target and resume
		_, _ = e.queries.UpdateAutomationRunState(ctx, sqlcgen.UpdateAutomationRunStateParams{
			Status:        sqlcgen.AutomationRunStatusRunning,
			CurrentNodeID: pgtype.Text{String: repliedTarget, Valid: true},
			Variables:     run.Variables,
			ResumeAt:      pgtype.Timestamptz{},
			ID:            run.ID,
		})

		// Build a compatible run object for executeFlow
		createRun := &sqlcgen.CreateAutomationRunRow{
			ID:               run.ID,
			AutomationID:     run.AutomationID,
			TenantID:         run.TenantID,
			ConversationID:   run.ConversationID,
			TriggerMessageID: run.TriggerMessageID,
			Status:           sqlcgen.AutomationRunStatusRunning,
			CurrentNodeID:    pgtype.Text{String: repliedTarget, Valid: true},
			Variables:        run.Variables,
			ResumeAt:         run.ResumeAt,
			StartedAt:        run.StartedAt,
			CompletedAt:      run.CompletedAt,
			UpdatedAt:        run.UpdatedAt,
		}

		e.executeFlow(ctx, createRun, def, tenantID, conversationID, messageText)
	}

	return nil
}

// ResumeRun resumes execution of a paused or waiting automation run.
func (e *Engine) ResumeRun(ctx context.Context, runID uuid.UUID) error {
	if e.queries == nil {
		return nil
	}

	run, err := e.queries.GetAutomationRunByID(ctx, runID)
	if err != nil {
		return fmt.Errorf("get run: %w", err)
	}

	if run.Status != sqlcgen.AutomationRunStatusPaused && run.Status != sqlcgen.AutomationRunStatusWaiting {
		return nil
	}

	auto, err := e.queries.GetAutomationByID(ctx, sqlcgen.GetAutomationByIDParams{
		ID:       run.AutomationID,
		TenantID: run.TenantID,
	})
	if err != nil {
		return fmt.Errorf("get automation: %w", err)
	}

	var def FlowDefinition
	if err := json.Unmarshal(auto.Definition, &def); err != nil {
		return fmt.Errorf("unmarshal definition: %w", err)
	}

	// Update run status back to running
	_, _ = e.queries.UpdateAutomationRunState(ctx, sqlcgen.UpdateAutomationRunStateParams{
		Status:        sqlcgen.AutomationRunStatusRunning,
		CurrentNodeID: run.CurrentNodeID,
		Variables:     run.Variables,
		ResumeAt:      pgtype.Timestamptz{},
		ID:            run.ID,
	})

	// Build a compatible run object for executeFlow
	createRun := &sqlcgen.CreateAutomationRunRow{
		ID:               run.ID,
		AutomationID:     run.AutomationID,
		TenantID:         run.TenantID,
		ConversationID:   run.ConversationID,
		TriggerMessageID: run.TriggerMessageID,
		Status:           run.Status,
		CurrentNodeID:    run.CurrentNodeID,
		Variables:        run.Variables,
		ResumeAt:         run.ResumeAt,
		StartedAt:        run.StartedAt,
		CompletedAt:      run.CompletedAt,
		UpdatedAt:        run.UpdatedAt,
	}

	e.executeFlow(ctx, createRun, def, run.TenantID, run.ConversationID.Bytes, "")
	return nil
}

// executeFlow runs nodes sequentially starting from the run's current_node_id.
// It updates the run state as it progresses.
func (e *Engine) executeFlow(ctx context.Context, run *sqlcgen.CreateAutomationRunRow, def FlowDefinition, tenantID, conversationID uuid.UUID, messageText string) {
	currentNodeID := run.CurrentNodeID.String

	for currentNodeID != "" {
		node, ok := findNodeByID(def, currentNodeID)
		if !ok {
			_, _ = e.queries.UpdateAutomationRunStatus(ctx, sqlcgen.UpdateAutomationRunStatusParams{
				Status: sqlcgen.AutomationRunStatusFailed,
				ID:     run.ID,
			})
			return
		}

		// Update current node in run state
		_, _ = e.queries.UpdateAutomationRunState(ctx, sqlcgen.UpdateAutomationRunStateParams{
			Status:        sqlcgen.AutomationRunStatusRunning,
			CurrentNodeID: pgtype.Text{String: currentNodeID, Valid: true},
			Variables:     run.Variables,
			ResumeAt:      pgtype.Timestamptz{},
			ID:            run.ID,
		})

		// Execute the node
		nextNodeID, paused, err := e.executeNode(ctx, run, node, def, tenantID, conversationID, messageText)
		if err != nil {
			_, _ = e.queries.UpdateAutomationRunStatus(ctx, sqlcgen.UpdateAutomationRunStatusParams{
				Status: sqlcgen.AutomationRunStatusFailed,
				ID:     run.ID,
			})
			return
		}

		if paused {
			// Flow is paused; resume_at and status already set by the node handler
			return
		}

		currentNodeID = nextNodeID
	}

	// No more nodes; mark as completed
	_, _ = e.queries.UpdateAutomationRunStatus(ctx, sqlcgen.UpdateAutomationRunStatusParams{
		Status: sqlcgen.AutomationRunStatusCompleted,
		ID:     run.ID,
	})
}

// executeNode handles a single node and returns the next node ID.
// If paused is true, the flow should stop and the run status has been updated.
func (e *Engine) executeNode(ctx context.Context, run *sqlcgen.CreateAutomationRunRow, node FlowNode, def FlowDefinition, tenantID, conversationID uuid.UUID, messageText string) (nextNodeID string, paused bool, err error) {
	switch node.Type {
	case "send_message":
		msg, _ := node.Config["message"].(string)
		if msg != "" {
			conv, err := e.queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
				ID:       conversationID,
				TenantID: tenantID,
			})
			if err != nil {
				return "", false, fmt.Errorf("get conversation: %w", err)
			}
			phones, err := e.queries.ListContactPhones(ctx, conv.ContactID)
			if err != nil || len(phones) == 0 {
				return "", false, fmt.Errorf("get contact phones: %w", err)
			}
			to := phones[0].Phone
			_ = e.ch.SendTextMessage(ctx, to, msg)
		}
		return findNextNode(def, node.ID, ""), false, nil

	case "condition":
		label, err := e.evaluateCondition(ctx, node.Config, tenantID, conversationID, messageText)
		if err != nil {
			return "", false, err
		}
		return findNextNode(def, node.ID, label), false, nil

	case "delay":
		durationRaw, ok := node.Config["duration"]
		if !ok {
			return "", false, fmt.Errorf("delay node missing duration")
		}
		var durationSeconds int
		switch v := durationRaw.(type) {
		case float64:
			durationSeconds = int(v)
		case int:
			durationSeconds = v
		default:
			return "", false, fmt.Errorf("delay node duration invalid type")
		}
		resumeAt := time.Now().UTC().Add(time.Duration(durationSeconds) * time.Second)
		_, _ = e.queries.UpdateAutomationRunState(ctx, sqlcgen.UpdateAutomationRunStateParams{
			Status:        sqlcgen.AutomationRunStatusPaused,
			CurrentNodeID: pgtype.Text{String: findNextNode(def, node.ID, ""), Valid: true},
			Variables:     run.Variables,
			ResumeAt:      pgtype.Timestamptz{Time: resumeAt, Valid: true},
			ID:            run.ID,
		})
		return "", true, nil

	case "wait_for_reply":
		var timeoutSeconds int
		switch v := node.Config["timeout"].(type) {
		case float64:
			timeoutSeconds = int(v)
		case int:
			timeoutSeconds = v
		}
		if timeoutSeconds <= 0 {
			timeoutSeconds = 300 // default 5 minutes
		}
		resumeAt := time.Now().UTC().Add(time.Duration(timeoutSeconds) * time.Second)
		// Store the wait_for_reply node ID itself so we know which node to resume from
		_, _ = e.queries.UpdateAutomationRunState(ctx, sqlcgen.UpdateAutomationRunStateParams{
			Status:        sqlcgen.AutomationRunStatusWaiting,
			CurrentNodeID: pgtype.Text{String: node.ID, Valid: true},
			Variables:     run.Variables,
			ResumeAt:      pgtype.Timestamptz{Time: resumeAt, Valid: true},
			ID:            run.ID,
		})
		return "", true, nil

	case "assign_team":
		memberIDRaw, ok := node.Config["member_id"]
		if !ok {
			return "", false, fmt.Errorf("assign_team node missing member_id")
		}
		memberIDStr, ok := memberIDRaw.(string)
		if !ok {
			return "", false, fmt.Errorf("assign_team node member_id invalid type")
		}
		memberID, err := uuid.Parse(memberIDStr)
		if err != nil {
			return "", false, fmt.Errorf("assign_team node member_id invalid uuid")
		}
		_, _ = e.queries.UpdateConversation(ctx, sqlcgen.UpdateConversationParams{
			Status:           sqlcgen.ConversationStatusOpen,
			AssignedMemberID: pgtype.UUID{Bytes: memberID, Valid: true},
			HandledBy:        sqlcgen.HandledByHuman,
			ID:               conversationID,
			TenantID:         tenantID,
		})
		return findNextNode(def, node.ID, ""), false, nil

	case "add_tag":
		tagName, ok := node.Config["tag"].(string)
		if !ok || tagName == "" {
			return "", false, fmt.Errorf("add_tag node missing tag name")
		}
		if err := e.addTagToContact(ctx, tenantID, conversationID, tagName); err != nil {
			return "", false, err
		}
		return findNextNode(def, node.ID, ""), false, nil

	case "remove_tag":
		tagName, ok := node.Config["tag"].(string)
		if !ok || tagName == "" {
			return "", false, fmt.Errorf("remove_tag node missing tag name")
		}
		if err := e.removeTagFromContact(ctx, tenantID, conversationID, tagName); err != nil {
			return "", false, err
		}
		return findNextNode(def, node.ID, ""), false, nil

	case "set_custom_field":
		fieldName, ok := node.Config["field"].(string)
		if !ok || fieldName == "" {
			return "", false, fmt.Errorf("set_custom_field node missing field name")
		}
		fieldValue, ok := node.Config["value"].(string)
		if !ok {
			return "", false, fmt.Errorf("set_custom_field node missing value")
		}
		if err := e.setCustomField(ctx, tenantID, conversationID, fieldName, fieldValue); err != nil {
			return "", false, err
		}
		return findNextNode(def, node.ID, ""), false, nil

	case "handoff_human":
		_, _ = e.queries.UpdateConversation(ctx, sqlcgen.UpdateConversationParams{
			Status:    sqlcgen.ConversationStatusEscalated,
			HandledBy: sqlcgen.HandledByHuman,
			ID:        conversationID,
			TenantID:  tenantID,
		})
		return findNextNode(def, node.ID, ""), false, nil

	case "ai_response":
		if e.aiRequestFunc == nil {
			return "", false, fmt.Errorf("ai_request_func not configured")
		}
		conv, err := e.queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
			ID:       conversationID,
			TenantID: tenantID,
		})
		if err != nil {
			return "", false, fmt.Errorf("get conversation: %w", err)
		}
		phones, err := e.queries.ListContactPhones(ctx, conv.ContactID)
		if err != nil || len(phones) == 0 {
			return "", false, fmt.Errorf("get contact phones: %w", err)
		}
		to := phones[0].Phone

		payload := map[string]interface{}{
			"conversation_id": conversationID.String(),
			"tenant_id":       tenantID.String(),
			"contact_id":      conv.ContactID.String(),
		}
		reply, err := e.aiRequestFunc(ctx, "flow.ai_request", payload)
		if err != nil {
			return "", false, fmt.Errorf("ai request: %w", err)
		}
		_ = e.ch.SendTextMessage(ctx, to, reply)
		return findNextNode(def, node.ID, ""), false, nil

	default:
		// Unknown node types are treated as passthrough for now
		return findNextNode(def, node.ID, ""), false, nil
	}
}

func (e *Engine) evaluateCondition(ctx context.Context, config map[string]interface{}, tenantID, conversationID uuid.UUID, messageText string) (string, error) {
	fieldRaw, ok := config["field"]
	if !ok {
		return "false", nil
	}
	field, ok := fieldRaw.(string)
	if !ok {
		return "false", nil
	}

	operatorRaw, ok := config["operator"]
	if !ok {
		return "false", nil
	}
	operator, ok := operatorRaw.(string)
	if !ok {
		return "false", nil
	}

	expectedValueRaw, ok := config["value"]
	if !ok {
		return "false", nil
	}
	expectedValue := fmt.Sprintf("%v", expectedValueRaw)

	// Get context
	conv, err := e.queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
		ID:       conversationID,
		TenantID: tenantID,
	})
	if err != nil {
		return "false", fmt.Errorf("get conversation: %w", err)
	}

	contact, err := e.queries.GetContactByID(ctx, sqlcgen.GetContactByIDParams{
		ID:       conv.ContactID,
		TenantID: tenantID,
	})
	if err != nil {
		return "false", fmt.Errorf("get contact: %w", err)
	}

	var actualValue string
	switch field {
	case "contact.language":
		if contact.Language.Valid {
			actualValue = contact.Language.String
		}
	case "contact.first_time":
		// Check if contact has any previous conversations (excluding current)
		lastConv, err := e.queries.GetLastConversationByContact(ctx, sqlcgen.GetLastConversationByContactParams{
			TenantID:  tenantID,
			ContactID: conv.ContactID,
		})
		actualValue = "false"
		if err == nil && lastConv.ID == conversationID {
			actualValue = "true"
		}
	case "message.text":
		actualValue = messageText
	case "conversation.handled_by":
		actualValue = string(conv.HandledBy)
	case "channel.type":
		// Get channel type from the most recent message or contact identities
		identities, err := e.queries.ListChannelIdentities(ctx, conv.ContactID)
		if err == nil && len(identities) > 0 {
			actualValue = identities[0].ChannelType
		}
	default:
		actualValue = ""
	}

	result := compareValues(actualValue, operator, expectedValue)
	if result {
		return "true", nil
	}
	return "false", nil
}

func compareValues(actual, operator, expected string) bool {
	switch operator {
	case "equals", "==":
		return strings.EqualFold(actual, expected)
	case "not_equals", "!=":
		return !strings.EqualFold(actual, expected)
	case "contains":
		return strings.Contains(strings.ToLower(actual), strings.ToLower(expected))
	case "starts_with":
		return strings.HasPrefix(strings.ToLower(actual), strings.ToLower(expected))
	case "ends_with":
		return strings.HasSuffix(strings.ToLower(actual), strings.ToLower(expected))
	default:
		return strings.EqualFold(actual, expected)
	}
}

func (e *Engine) addTagToContact(ctx context.Context, tenantID, conversationID uuid.UUID, tagName string) error {
	conv, err := e.queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
		ID:       conversationID,
		TenantID: tenantID,
	})
	if err != nil {
		return fmt.Errorf("get conversation: %w", err)
	}

	tag, err := e.queries.GetContactTagByName(ctx, sqlcgen.GetContactTagByNameParams{
		TenantID: tenantID,
		Name:     tagName,
	})
	if err != nil {
		// Create tag if it doesn't exist
		tag, err = e.queries.CreateContactTag(ctx, sqlcgen.CreateContactTagParams{
			TenantID: tenantID,
			Name:     tagName,
		})
		if err != nil {
			return fmt.Errorf("create tag: %w", err)
		}
	}

	return e.queries.AddTagToContact(ctx, sqlcgen.AddTagToContactParams{
		ContactID: conv.ContactID,
		TagID:     tag.ID,
	})
}

func (e *Engine) removeTagFromContact(ctx context.Context, tenantID, conversationID uuid.UUID, tagName string) error {
	conv, err := e.queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
		ID:       conversationID,
		TenantID: tenantID,
	})
	if err != nil {
		return fmt.Errorf("get conversation: %w", err)
	}

	tag, err := e.queries.GetContactTagByName(ctx, sqlcgen.GetContactTagByNameParams{
		TenantID: tenantID,
		Name:     tagName,
	})
	if err != nil {
		return nil // tag doesn't exist, nothing to remove
	}

	return e.queries.RemoveTagFromContact(ctx, sqlcgen.RemoveTagFromContactParams{
		ContactID: conv.ContactID,
		TagID:     tag.ID,
	})
}

func (e *Engine) setCustomField(ctx context.Context, tenantID, conversationID uuid.UUID, fieldName, fieldValue string) error {
	conv, err := e.queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
		ID:       conversationID,
		TenantID: tenantID,
	})
	if err != nil {
		return fmt.Errorf("get conversation: %w", err)
	}

	field, err := e.queries.GetCustomFieldByName(ctx, sqlcgen.GetCustomFieldByNameParams{
		TenantID: tenantID,
		Name:     fieldName,
	})
	if err != nil {
		// Create custom field if it doesn't exist
		field, err = e.queries.CreateCustomField(ctx, sqlcgen.CreateCustomFieldParams{
			TenantID:  tenantID,
			Name:      fieldName,
			FieldType: "text",
			Options:   nil,
		})
		if err != nil {
			return fmt.Errorf("create custom field: %w", err)
		}
	}

	// Validate field value against field definition
	if err := validateCustomFieldValue(fieldValue, field.FieldType, field.Options); err != nil {
		return fmt.Errorf("validate custom field: %w", err)
	}

	_, err = e.queries.SetContactCustomField(ctx, sqlcgen.SetContactCustomFieldParams{
		ContactID: conv.ContactID,
		FieldID:   field.ID,
		Value:     fieldValue,
	})
	return err
}

func validateCustomFieldValue(value, fieldType string, options []byte) error {
	switch fieldType {
	case "text":
		return nil
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("value %q is not a valid number", value)
		}
		return nil
	case "date":
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return fmt.Errorf("value %q is not a valid date", value)
			}
		}
		return nil
	case "boolean":
		lower := strings.ToLower(value)
		if lower != "true" && lower != "false" {
			return fmt.Errorf("value %q is not a valid boolean", value)
		}
		return nil
	case "select":
		if len(options) == 0 {
			return nil
		}
		var opts []string
		if err := json.Unmarshal(options, &opts); err != nil {
			return nil // can't validate, allow through
		}
		for _, opt := range opts {
			if opt == value {
				return nil
			}
		}
		return fmt.Errorf("value %q is not one of the allowed options", value)
	default:
		return nil
	}
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

func findNextNode(def FlowDefinition, fromID, label string) string {
	for _, e := range def.Edges {
		if e.Source == fromID {
			if label == "" || e.Label == label {
				return e.Target
			}
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
