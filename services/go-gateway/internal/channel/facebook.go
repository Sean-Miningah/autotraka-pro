package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// Facebook implements the Channel interface for Facebook Messenger via the Meta
// Messenger Platform API (Graph API v19.0).
type Facebook struct {
	MetaChannel
	pageID string
}

// NewFacebook creates a Facebook Messenger channel from explicit credentials.
func NewFacebook(baseURL, accessToken, pageID, appSecret, verifyToken string) *Facebook {
	return &Facebook{
		MetaChannel: NewMetaChannel(baseURL, accessToken, appSecret, verifyToken, "facebook"),
		pageID:      pageID,
	}
}

// ChannelType returns "facebook".
func (fb *Facebook) ChannelType() string { return "facebook" }

// SendTextMessage sends a plain text message to a PSID via the Messenger API.
func (fb *Facebook) SendTextMessage(ctx context.Context, to, body string) error {
	payload := map[string]interface{}{
		"recipient": map[string]string{"id": to},
		"message":   map[string]string{"text": body},
	}

	url := fb.apiURL(fb.pageID, "messages")
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+fb.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := fb.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// SendTemplateMessage is not supported for Facebook Messenger in this implementation.
func (fb *Facebook) SendTemplateMessage(ctx context.Context, to, templateName, language string, params []string) error {
	return errors.New("facebook messenger does not support template messages in this implementation")
}

// SendMediaMessage is not yet implemented for Facebook Messenger.
func (fb *Facebook) SendMediaMessage(ctx context.Context, to, mediaType, mediaURL, caption string) error {
	return errors.New("not implemented")
}

// MarkRead marks a message as read on Facebook Messenger.
func (fb *Facebook) MarkRead(ctx context.Context, messageID string) error {
	return errors.New("not implemented")
}

// facebookPayload mirrors the webhook structure for Facebook Messenger events.
// Meta sends "object": "page" for Messenger webhooks.
type facebookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID        string `json:"id"`
		Time      int64  `json:"time"`
		Messaging []struct {
			Sender    struct {
				ID string `json:"id"`
			} `json:"sender"`
			Recipient struct {
				ID string `json:"id"`
			} `json:"recipient"`
			Timestamp int64 `json:"timestamp"`
			Message   *struct {
				Mid  string `json:"mid"`
				Text string `json:"text"`
				Attachments []struct {
					Type    string `json:"type"`
					Payload struct {
						URL string `json:"url"`
					} `json:"payload"`
				} `json:"attachments"`
			} `json:"message"`
			Postback *struct {
				Title   string `json:"title"`
				Payload string `json:"payload"`
			} `json:"postback"`
		} `json:"messaging"`
	} `json:"entry"`
}

// ParseWebhookEvent extracts inbound message events from a Facebook Messenger webhook payload.
func (fb *Facebook) ParseWebhookEvent(payload []byte) (WebhookEvent, error) {
	var body facebookPayload
	if err := json.Unmarshal(payload, &body); err != nil {
		return WebhookEvent{}, fmt.Errorf("unmarshal payload: %w", err)
	}

	if body.Object != "page" {
		return WebhookEvent{}, errors.New("unsupported webhook object: expected page")
	}

	if len(body.Entry) == 0 {
		return WebhookEvent{}, errors.New("no entries in payload")
	}

	entry := body.Entry[0]
	if len(entry.Messaging) == 0 {
		return WebhookEvent{}, errors.New("no messaging events in payload")
	}

	messaging := entry.Messaging[0]

	var evt WebhookEvent

	if messaging.Message != nil {
		msg := messaging.Message
		evt = WebhookEvent{
			EventID:     msg.Mid,
			From:        messaging.Sender.ID,
			To:          messaging.Recipient.ID,
			MessageID:   msg.Mid,
			Timestamp:   messaging.Timestamp,
			ChannelType: fb.ChannelType(),
		}

		if len(msg.Attachments) > 0 {
			att := msg.Attachments[0]
			evt.Type = att.Type
			evt.Content, _ = json.Marshal(map[string]string{
				"url":  att.Payload.URL,
				"type": att.Type,
			})
		} else {
			evt.Type = "text"
			evt.Content, _ = json.Marshal(map[string]string{"body": msg.Text})
		}
	} else if messaging.Postback != nil {
		pb := messaging.Postback
		evt = WebhookEvent{
			EventID:     messaging.Sender.ID + "_" + strconv.FormatInt(messaging.Timestamp, 10),
			From:        messaging.Sender.ID,
			To:          messaging.Recipient.ID,
			MessageID:   "postback_" + messaging.Sender.ID + "_" + strconv.FormatInt(messaging.Timestamp, 10),
			Type:        "postback",
			Timestamp:   messaging.Timestamp,
			Content:     json.RawMessage("{}"),
			ChannelType: fb.ChannelType(),
		}
		if pb.Payload != "" {
			evt.Content, _ = json.Marshal(map[string]string{
				"title":   pb.Title,
				"payload": pb.Payload,
			})
		}
	} else {
		return WebhookEvent{}, errors.New("unsupported messaging event type")
	}

	return evt, nil
}
