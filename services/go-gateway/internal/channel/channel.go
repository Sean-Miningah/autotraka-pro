package channel

import (
	"context"
	"encoding/json"
)

// Channel is the abstraction for all messaging platforms.
type Channel interface {
	// ChannelType returns the platform identifier (whatsapp, instagram, facebook).
	ChannelType() string

	// SendTextMessage sends a plain text message to the recipient.
	SendTextMessage(ctx context.Context, to, body string) error

	// SendTemplateMessage sends a template message with positional parameters.
	SendTemplateMessage(ctx context.Context, to, templateName, language string, params []string) error

	// SendMediaMessage sends a media message (image, audio, document, video).
	SendMediaMessage(ctx context.Context, to, mediaType, mediaURL, caption string) error

	// MarkRead marks a message as read on the platform.
	MarkRead(ctx context.Context, messageID string) error

	// VerifyWebhook checks the subscription challenge from the platform.
	VerifyWebhook(mode, verifyToken, challenge string) (string, error)

	// VerifySignature validates the HMAC signature on a webhook payload.
	VerifySignature(payload []byte, signature string) error

	// ParseWebhookEvent extracts the platform-specific event from raw payload.
	ParseWebhookEvent(payload []byte) (WebhookEvent, error)
}

// WebhookEvent is the platform-agnostic representation of an inbound event.
type WebhookEvent struct {
	EventID     string          `json:"event_id"`
	From        string          `json:"from"`
	To          string          `json:"to"`
	MessageID   string          `json:"message_id"`
	Type        string          `json:"type"` // text, image, audio, document, template, etc.
	Content     json.RawMessage `json:"content"`
	Timestamp   int64           `json:"timestamp"`
	ChannelType string          `json:"channel_type"`
}
