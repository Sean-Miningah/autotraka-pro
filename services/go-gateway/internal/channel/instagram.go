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

// Instagram implements the Channel interface for Instagram DMs via the Meta
// Conversational API (Instagram Graph API).
type Instagram struct {
	MetaChannel
	instagramAccountID string
}

// NewInstagram creates an Instagram channel from explicit credentials.
func NewInstagram(baseURL, accessToken, instagramAccountID, appSecret, verifyToken string) *Instagram {
	return &Instagram{
		MetaChannel:        NewMetaChannel(baseURL, accessToken, appSecret, verifyToken, "instagram"),
		instagramAccountID: instagramAccountID,
	}
}

// ChannelType returns "instagram".
func (ig *Instagram) ChannelType() string { return "instagram" }

// SendTextMessage sends a plain text DM via the Instagram API.
func (ig *Instagram) SendTextMessage(ctx context.Context, to, body string) error {
	payload := map[string]interface{}{
		"recipient": map[string]string{"id": to},
		"message":   map[string]string{"text": body},
	}

	url := ig.apiURL(ig.instagramAccountID, "messages")
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+ig.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ig.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// SendTemplateMessage is not supported for Instagram DMs.
func (ig *Instagram) SendTemplateMessage(ctx context.Context, to, templateName, language string, params []string) error {
	return errors.New("instagram does not support template messages")
}

// SendMediaMessage sends an Instagram media message.
func (ig *Instagram) SendMediaMessage(ctx context.Context, to, mediaType, mediaURL, caption string) error {
	return errors.New("not implemented")
}

// MarkRead marks a message as read on Instagram.
func (ig *Instagram) MarkRead(ctx context.Context, messageID string) error {
	return errors.New("not implemented")
}

// instagramPayload mirrors the webhook structure for Instagram messaging events.
type instagramPayload struct {
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

// ParseWebhookEvent extracts inbound DM events from an Instagram webhook payload.
func (ig *Instagram) ParseWebhookEvent(payload []byte) (WebhookEvent, error) {
	var body instagramPayload
	if err := json.Unmarshal(payload, &body); err != nil {
		return WebhookEvent{}, fmt.Errorf("unmarshal payload: %w", err)
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
			ChannelType: "instagram",
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
			ChannelType: "instagram",
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
