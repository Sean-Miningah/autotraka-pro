package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/autotraka/go-gateway/internal/automation"
	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/contact"
	"github.com/autotraka/go-gateway/internal/eventbus"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/template"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrNotFound            = errors.New("conversation not found")
	ErrTemplateNotFound    = errors.New("template not found")
	ErrInvalidTemplate     = errors.New("invalid template")
	ErrMissingParameters   = errors.New("missing required template parameters")
)

// Service provides conversation and messaging business logic.
type Service struct {
	queries     *sqlcgen.Queries
	contactSvc  *contact.Service
	templateSvc *template.Service
	autoEngine  *automation.Engine
	ch          channel.Channel
	eventbus    *eventbus.Client
}

// NewService creates a conversation service.
func NewService(queries *sqlcgen.Queries, contactSvc *contact.Service, templateSvc *template.Service, ch channel.Channel, eventbus *eventbus.Client) *Service {
	svc := &Service{
		queries:     queries,
		contactSvc:  contactSvc,
		templateSvc: templateSvc,
		ch:          ch,
		eventbus:    eventbus,
	}
	// Engine is created after service to avoid circular dependency in constructor
	svc.autoEngine = automation.NewEngine(queries, ch)
	return svc
}

// Conversation is the enriched conversation model with messages.
type Conversation struct {
	ID                     uuid.UUID  `json:"id"`
	TenantID               uuid.UUID  `json:"tenant_id"`
	ContactID              uuid.UUID  `json:"contact_id"`
	Status                 string     `json:"status"`
	AssignedMemberID       *uuid.UUID `json:"assigned_member_id,omitempty"`
	HandledBy              string     `json:"handled_by"`
	PreviousConversationID *uuid.UUID `json:"previous_conversation_id,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	Messages               []Message  `json:"messages,omitempty"`
	Contact                *contact.Contact `json:"contact,omitempty"`
}

// Message is the enriched message model.
type Message struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	ConversationID uuid.UUID       `json:"conversation_id"`
	ChannelID      *uuid.UUID      `json:"channel_id,omitempty"`
	Direction      string          `json:"direction"`
	Status         string          `json:"status"`
	ContentType    string          `json:"content_type"`
	Content        json.RawMessage `json:"content"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// MessageReceivedPayload is sent to the AI service when an inbound message
// arrives in an AI-handled conversation.
type MessageReceivedPayload struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	MessageID      uuid.UUID `json:"message_id"`
	ContactID      uuid.UUID `json:"contact_id"`
	Body           string    `json:"body"`
	Type           string    `json:"type"`
}

// ProcessInboundMessage handles an inbound webhook event: resolves contact,
// finds or creates conversation, stores message.
func (s *Service) ProcessInboundMessage(ctx context.Context, tenantID, channelID uuid.UUID, evt channel.WebhookEvent) (*Conversation, *Message, error) {
	// Resolve or create contact
	c, err := s.contactSvc.ResolveOrCreate(ctx, tenantID, contact.ResolveRequest{
		ChannelType:     "whatsapp",
		ChannelIdentity: evt.From,
		Name:            "", // name may come from webhook payload but we don't have it in WebhookEvent
		Phone:           evt.From,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("resolve contact: %w", err)
	}

	// Find existing open conversation for this contact
	conv, err := s.queries.GetOpenConversationByContact(ctx, sqlcgen.GetOpenConversationByContactParams{
		TenantID:  tenantID,
		ContactID: c.ID,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, fmt.Errorf("find open conversation: %w", err)
	}

	// If no open conversation, check if there's a closed one to link
	var prevConvID pgtype.UUID
	if errors.Is(err, pgx.ErrNoRows) {
		lastConv, err := s.queries.GetLastConversationByContact(ctx, sqlcgen.GetLastConversationByContactParams{
			TenantID:  tenantID,
			ContactID: c.ID,
		})
		if err == nil && lastConv.Status == sqlcgen.ConversationStatusClosed {
			prevConvID = pgtype.UUID{Bytes: lastConv.ID, Valid: true}
		}

		// Determine handled_by based on tenant mode
		handledBy := sqlcgen.HandledByAi
		if tenant, terr := s.queries.GetTenant(ctx, tenantID); terr == nil {
			switch tenant.Mode {
			case "human_first":
				handledBy = sqlcgen.HandledByHuman
			case "hybrid":
				handledBy = sqlcgen.HandledByHybrid
			default:
				handledBy = sqlcgen.HandledByAi
			}
		}

		// Create new conversation
		conv, err = s.queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
			TenantID:               tenantID,
			ContactID:              c.ID,
			Status:                 sqlcgen.ConversationStatusOpen,
			AssignedMemberID:       pgtype.UUID{Valid: false},
			HandledBy:              handledBy,
			PreviousConversationID: prevConvID,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("create conversation: %w", err)
		}
	}

	// Store the inbound message
	var textBody string
	if len(evt.Content) > 0 {
		var parsed struct {
			Body string `json:"body"`
		}
		if err := json.Unmarshal(evt.Content, &parsed); err == nil && parsed.Body != "" {
			textBody = parsed.Body
		} else {
			textBody = string(evt.Content)
		}
	}
	content, _ := json.Marshal(map[string]interface{}{"text": textBody})
	msg, err := s.queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenantID,
		ConversationID: conv.ID,
		ChannelID:      pgtype.UUID{Bytes: channelID, Valid: true},
		Direction:      sqlcgen.MessageDirectionInbound,
		Status:         sqlcgen.MessageStatusDelivered,
		ContentType:    "text",
		Content:        content,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create message: %w", err)
	}

	conversation := s.toConversation(conv)
	message := s.toMessage(msg)

	// In AI-first mode, publish message to AI service
	if s.eventbus != nil && conv.HandledBy == sqlcgen.HandledByAi {
		var inboundText string
		var parsed struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(msg.Content, &parsed); err == nil {
			inboundText = parsed.Text
		} else {
			inboundText = string(msg.Content)
		}
		ctx = eventbus.WithTenantID(ctx, tenantID)
		_ = s.eventbus.Publish(ctx, "message.received", &MessageReceivedPayload{
			ConversationID: conv.ID,
			MessageID:      msg.ID,
			ContactID:      c.ID,
			Body:           inboundText,
			Type:           "text",
		})
	}

	// Check automation triggers
	var textContent string
	if s.autoEngine != nil {
		var parsed struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(msg.Content, &parsed); err == nil {
			textContent = parsed.Text
		} else {
			// Fallback: try direct string content
			textContent = string(msg.Content)
		}
		_ = s.autoEngine.ProcessInboundMessage(ctx, tenantID, conv.ID, textContent)
	}

	// Publish events
	if s.eventbus != nil {
		ctx = eventbus.WithTenantID(ctx, tenantID)
		_ = s.eventbus.Publish(ctx, "message.inbound", map[string]interface{}{
			"message_id":      msg.ID,
			"conversation_id": conv.ID,
			"tenant_id":       tenantID,
			"direction":       "inbound",
			"status":          "delivered",
		})
		_ = s.eventbus.Publish(ctx, "conversation.updated", map[string]interface{}{
			"conversation_id": conv.ID,
			"contact_id":      c.ID,
			"tenant_id":       tenantID,
			"status":          string(conv.Status),
		})
	}

	return &conversation, &message, nil
}

// SendMessageRequest holds data for sending an outbound message.
type SendMessageRequest struct {
	Content    json.RawMessage   `json:"content"`
	ChannelID  *uuid.UUID        `json:"channel_id,omitempty"`
	TemplateID *uuid.UUID        `json:"template_id,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// SendMessage stores an outbound message optimistically and publishes to NATS.
// When TemplateID is provided, resolves the template, maps named parameters to
// positional values, and sends directly via the WhatsApp channel.
func (s *Service) SendMessage(ctx context.Context, tenantID, conversationID uuid.UUID, req SendMessageRequest) (*Message, error) {
	// Verify conversation exists and belongs to tenant
	conv, err := s.queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
		ID:       conversationID,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Determine channel: use override or last inbound message's channel
	var channelID pgtype.UUID
	if req.ChannelID != nil {
		channelID = pgtype.UUID{Bytes: *req.ChannelID, Valid: true}
	} else {
		lastMsg, err := s.queries.GetLastMessageByConversation(ctx, conversationID)
		if err == nil && lastMsg.ChannelID.Valid {
			channelID = lastMsg.ChannelID
		}
	}

	var content json.RawMessage
	var contentType string

	// Template message flow
	if req.TemplateID != nil {
		if s.templateSvc == nil || s.ch == nil {
			return nil, fmt.Errorf("template messaging not configured")
		}

		tmpl, err := s.templateSvc.Get(ctx, tenantID, *req.TemplateID)
		if err != nil {
			if errors.Is(err, template.ErrNotFound) {
				return nil, ErrTemplateNotFound
			}
			return nil, err
		}
		if tmpl.Status != "approved" {
			return nil, fmt.Errorf("%w: template status is %s", ErrInvalidTemplate, tmpl.Status)
		}

		// Validate all required parameters are present
		positional := make([]string, 0, len(tmpl.Parameters))
		for _, p := range tmpl.Parameters {
			val, ok := req.Parameters[p.Name]
			if !ok {
				return nil, fmt.Errorf("%w: missing parameter %q", ErrMissingParameters, p.Name)
			}
			positional = append(positional, val)
		}

		// Store parameter mapping as content
		paramBytes, _ := json.Marshal(req.Parameters)
		content = paramBytes
		contentType = "template"

		// Get contact phone for the "to" field
		phones, err := s.queries.ListContactPhones(ctx, conv.ContactID)
		if err != nil || len(phones) == 0 {
			return nil, fmt.Errorf("contact has no phone number")
		}
		to := phones[0].Phone

		// Send directly via channel
		if err := s.ch.SendTemplateMessage(ctx, to, tmpl.Name, tmpl.Language, positional); err != nil {
			return nil, fmt.Errorf("send template: %w", err)
		}
	} else {
		content = req.Content
		contentType = "text"
	}

	// Store message as pending
	msg, err := s.queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenantID,
		ConversationID: conversationID,
		ChannelID:      channelID,
		Direction:      sqlcgen.MessageDirectionOutbound,
		Status:         sqlcgen.MessageStatusPending,
		ContentType:    contentType,
		Content:        content,
	})
	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	message := s.toMessage(msg)

	// Publish to NATS for delivery (only for text; templates are already sent)
	if s.eventbus != nil && req.TemplateID == nil {
		ctx = eventbus.WithTenantID(ctx, tenantID)
		_ = s.eventbus.Publish(ctx, "message.outbound", map[string]interface{}{
			"message_id":      msg.ID,
			"conversation_id": conversationID,
			"tenant_id":       tenantID,
			"channel_id":      msg.ChannelID,
			"contact_id":      conv.ContactID,
			"content":         req.Content,
		})
	}

	return &message, nil
}

// UpdateConversationRequest holds optional fields for patching a conversation.
type UpdateConversationRequest struct {
	Status           *string    `json:"status,omitempty"`
	AssignedMemberID *uuid.UUID `json:"assigned_member_id,omitempty"`
	HandledBy        *string    `json:"handled_by,omitempty"`
}

// validTransitions defines allowed status transitions.
var validTransitions = map[string][]string{
	"open":       {"pending", "escalated", "resolved", "closed"},
	"pending":    {"escalated", "resolved", "closed"},
	"escalated":  {"resolved", "closed"},
	"resolved":   {"closed"},
	"closed":     {},
}

func isValidTransition(from, to string) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == to {
			return true
		}
	}
	return false
}

// Update patches a conversation's fields.
func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateConversationRequest) (*Conversation, error) {
	existing, err := s.queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	status := existing.Status
	if req.Status != nil {
		if !isValidTransition(string(existing.Status), *req.Status) {
			return nil, fmt.Errorf("invalid status transition from %s to %s", existing.Status, *req.Status)
		}
		status = sqlcgen.ConversationStatus(*req.Status)
	}

	assignedMemberID := existing.AssignedMemberID
	if req.AssignedMemberID != nil {
		assignedMemberID = pgtype.UUID{Bytes: *req.AssignedMemberID, Valid: true}
	}

	handledBy := existing.HandledBy
	if req.HandledBy != nil {
		handledBy = sqlcgen.HandledBy(*req.HandledBy)
	}

	updated, err := s.queries.UpdateConversation(ctx, sqlcgen.UpdateConversationParams{
		Status:           status,
		AssignedMemberID: assignedMemberID,
		HandledBy:        handledBy,
		ID:               id,
		TenantID:         tenantID,
	})
	if err != nil {
		return nil, err
	}

	conv := s.toConversation(updated)

	if s.eventbus != nil {
		ctx = eventbus.WithTenantID(ctx, tenantID)
		_ = s.eventbus.Publish(ctx, "conversation.updated", map[string]interface{}{
			"conversation_id": conv.ID,
			"tenant_id":       tenantID,
			"status":          conv.Status,
			"handled_by":      conv.HandledBy,
		})
		if conv.Status == "escalated" {
			_ = s.eventbus.Publish(ctx, "conversation.escalated", map[string]interface{}{
				"conversation_id": conv.ID,
				"tenant_id":       tenantID,
			})
		}
	}

	return &conv, nil
}

// List returns conversations scoped to a tenant with pagination.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, limit, offset int32) ([]Conversation, int64, error) {
	conversations, err := s.queries.ListConversationsByTenant(ctx, sqlcgen.ListConversationsByTenantParams{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := s.queries.CountConversationsByTenant(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]Conversation, 0, len(conversations))
	for _, c := range conversations {
		result = append(result, s.toConversation(c))
	}
	return result, count, nil
}

// Get returns a single conversation with messages.
func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*Conversation, error) {
	conv, err := s.queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	conversation := s.toConversation(conv)

	// Load messages
	messages, err := s.queries.ListMessagesByConversation(ctx, sqlcgen.ListMessagesByConversationParams{
		ConversationID: id,
		TenantID:       tenantID,
		Limit:          100,
		Offset:         0,
	})
	if err != nil {
		return nil, err
	}
	conversation.Messages = make([]Message, len(messages))
	for i, m := range messages {
		conversation.Messages[i] = s.toMessage(m)
	}

	// Load contact
	contact, err := s.contactSvc.Get(ctx, tenantID, conv.ContactID)
	if err == nil {
		conversation.Contact = contact
	}

	return &conversation, nil
}

// UpdateMessageStatus updates a message's delivery status.
func (s *Service) UpdateMessageStatus(ctx context.Context, tenantID, messageID uuid.UUID, status string) (*Message, error) {
	msg, err := s.queries.UpdateMessageStatus(ctx, sqlcgen.UpdateMessageStatusParams{
		Status:   sqlcgen.MessageStatus(status),
		ID:       messageID,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	message := s.toMessage(msg)

	// Publish status update
	if s.eventbus != nil {
		ctx = eventbus.WithTenantID(ctx, tenantID)
		_ = s.eventbus.Publish(ctx, "message.status_updated", map[string]interface{}{
			"message_id": messageID,
			"status":     status,
			"tenant_id":  tenantID,
		})
	}

	return &message, nil
}

func (s *Service) toConversation(row sqlcgen.Conversation) Conversation {
	c := Conversation{
		ID:        row.ID,
		TenantID:  row.TenantID,
		ContactID: row.ContactID,
		Status:    string(row.Status),
		HandledBy: string(row.HandledBy),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.AssignedMemberID.Valid {
		uid := uuid.UUID(row.AssignedMemberID.Bytes)
		c.AssignedMemberID = &uid
	}
	if row.PreviousConversationID.Valid {
		uid := uuid.UUID(row.PreviousConversationID.Bytes)
		c.PreviousConversationID = &uid
	}
	return c
}

func (s *Service) toMessage(row sqlcgen.Message) Message {
	m := Message{
		ID:             row.ID,
		TenantID:       row.TenantID,
		ConversationID: row.ConversationID,
		Direction:      string(row.Direction),
		Status:         string(row.Status),
		ContentType:    row.ContentType,
		Content:        row.Content,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
	if row.ChannelID.Valid {
		uid := uuid.UUID(row.ChannelID.Bytes)
		m.ChannelID = &uid
	}
	return m
}

// AIResponsePayload is the expected shape of a message.ai_response event.
type AIResponsePayload struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Body           string    `json:"body"`
}

// AIHandoffPayload is the expected shape of an ai.handoff_request event.
type AIHandoffPayload struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Reason         string    `json:"reason,omitempty"`
}

// HandleAIResponse stores an AI-generated outbound message and sends it.
// In hybrid mode the message is stored as pending and not sent immediately.
func (s *Service) HandleAIResponse(ctx context.Context, payload AIResponsePayload) (*Message, error) {
	// Verify conversation exists and belongs to tenant
	conv, err := s.queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
		ID:       payload.ConversationID,
		TenantID: payload.TenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Resolve primary channel
	channels, err := s.queries.ListChannelsByTenantAndType(ctx, sqlcgen.ListChannelsByTenantAndTypeParams{
		TenantID:    payload.TenantID,
		ChannelType: "whatsapp",
	})
	if err != nil || len(channels) == 0 {
		return nil, fmt.Errorf("no whatsapp channel found")
	}
	channelID := pgtype.UUID{Bytes: channels[0].ID, Valid: true}

	content, _ := json.Marshal(map[string]interface{}{"text": payload.Body})

	status := sqlcgen.MessageStatusSent
	if conv.HandledBy == sqlcgen.HandledByHybrid {
		status = sqlcgen.MessageStatusPending
	}

	msg, err := s.queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       payload.TenantID,
		ConversationID: payload.ConversationID,
		ChannelID:      channelID,
		Direction:      sqlcgen.MessageDirectionOutbound,
		Status:         status,
		ContentType:    "text",
		Content:        content,
	})
	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	message := s.toMessage(msg)

	// Send via channel only in AI-first mode; hybrid stays pending
	if conv.HandledBy == sqlcgen.HandledByAi {
		phones, err := s.queries.ListContactPhones(ctx, conv.ContactID)
		if err != nil || len(phones) == 0 {
			return nil, fmt.Errorf("contact has no phone number")
		}
		to := phones[0].Phone
		if err := s.ch.SendTextMessage(ctx, to, payload.Body); err != nil {
			return nil, fmt.Errorf("send text: %w", err)
		}
	}

	// Publish outbound event
	if s.eventbus != nil {
		ctx = eventbus.WithTenantID(ctx, payload.TenantID)
		_ = s.eventbus.Publish(ctx, "message.outbound", map[string]interface{}{
			"message_id":      msg.ID,
			"conversation_id": payload.ConversationID,
			"tenant_id":       payload.TenantID,
			"direction":       "outbound",
			"status":          string(status),
		})
	}

	return &message, nil
}

// HandleAIHandoff escalates a conversation from AI to human.
func (s *Service) HandleAIHandoff(ctx context.Context, payload AIHandoffPayload) error {
	existing, err := s.queries.GetConversationByID(ctx, sqlcgen.GetConversationByIDParams{
		ID:       payload.ConversationID,
		TenantID: payload.TenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	updated, err := s.queries.UpdateConversation(ctx, sqlcgen.UpdateConversationParams{
		Status:           existing.Status,
		AssignedMemberID: existing.AssignedMemberID,
		HandledBy:        sqlcgen.HandledByHuman,
		ID:               payload.ConversationID,
		TenantID:         payload.TenantID,
	})
	if err != nil {
		return err
	}

	if s.eventbus != nil {
		ctx = eventbus.WithTenantID(ctx, payload.TenantID)
		_ = s.eventbus.Publish(ctx, "conversation.updated", map[string]interface{}{
			"conversation_id": payload.ConversationID,
			"tenant_id":       payload.TenantID,
			"status":          string(updated.Status),
			"handled_by":      string(updated.HandledBy),
		})
		_ = s.eventbus.Publish(ctx, "conversation.escalated", map[string]interface{}{
			"conversation_id": payload.ConversationID,
			"tenant_id":       payload.TenantID,
			"reason":          payload.Reason,
		})
	}

	return nil
}

// StartAIConsumers sets up NATS subscriptions for AI integration events.
func (s *Service) StartAIConsumers(ctx context.Context) error {
	if s.eventbus == nil {
		return nil
	}

	_, err := s.eventbus.Subscribe(ctx, "message.ai_response", func(_ context.Context, evt eventbus.Event) error {
		var payload AIResponsePayload
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal ai response: %w", err)
		}
		_, err := s.HandleAIResponse(context.Background(), payload)
		return err
	})
	if err != nil {
		return fmt.Errorf("subscribe message.ai_response: %w", err)
	}

	_, err = s.eventbus.Subscribe(ctx, "ai.handoff_request", func(_ context.Context, evt eventbus.Event) error {
		var payload AIHandoffPayload
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal handoff request: %w", err)
		}
		return s.HandleAIHandoff(context.Background(), payload)
	})
	if err != nil {
		return fmt.Errorf("subscribe ai.handoff_request: %w", err)
	}

	return nil
}
