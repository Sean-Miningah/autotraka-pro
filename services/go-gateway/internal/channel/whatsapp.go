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

// WhatsApp implements the Channel interface for the Meta WhatsApp Business API.
type WhatsApp struct {
	MetaChannel
	phoneNumberID string
}

// NewWhatsApp creates a WhatsApp channel from explicit credentials.
func NewWhatsApp(baseURL, accessToken, phoneNumberID, appSecret, verifyToken string) *WhatsApp {
	return &WhatsApp{
		MetaChannel:   NewMetaChannel(baseURL, accessToken, appSecret, verifyToken, "whatsapp"),
		phoneNumberID: phoneNumberID,
	}
}

// ChannelType returns "whatsapp".
func (w *WhatsApp) ChannelType() string { return "whatsapp" }

// SendTextMessage sends a text message via the WhatsApp API.
func (w *WhatsApp) SendTextMessage(ctx context.Context, to, body string) error {
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              "text",
		"text":              map[string]string{"body": body},
	}

	url := w.apiURL(w.phoneNumberID, "messages")
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// SendTemplateMessage sends a WhatsApp template message with positional parameters.
func (w *WhatsApp) SendTemplateMessage(ctx context.Context, to, templateName, language string, params []string) error {
	templateParams := make([]map[string]string, len(params))
	for i, p := range params {
		templateParams[i] = map[string]string{
			"type": "text",
			"text": p,
		}
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              "template",
		"template": map[string]interface{}{
			"name":     templateName,
			"language": map[string]string{"code": language},
			"components": []map[string]interface{}{
				{
					"type":       "body",
					"parameters": templateParams,
				},
			},
		},
	}

	url := w.apiURL(w.phoneNumberID, "messages")
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// SendMediaMessage sends a WhatsApp media message.
func (w *WhatsApp) SendMediaMessage(ctx context.Context, to, mediaType, mediaURL, caption string) error {
	return errors.New("not implemented")
}

// MarkRead marks a message as read on WhatsApp.
func (w *WhatsApp) MarkRead(ctx context.Context, messageID string) error {
	return errors.New("not implemented")
}

// ParseWebhookEvent extracts inbound message events from a Meta webhook payload.
func (w *WhatsApp) ParseWebhookEvent(payload []byte) (WebhookEvent, error) {
	var body struct {
		Entry []struct {
			Changes []struct {
				Value struct {
					Messages []struct {
						From      string `json:"from"`
						ID        string `json:"id"`
						Timestamp string `json:"timestamp"`
						Type      string `json:"type"`
						Text      struct {
							Body string `json:"body"`
						} `json:"text"`
						Image struct {
							ID       string `json:"id"`
							MimeType string `json:"mime_type"`
							Caption  string `json:"caption"`
						} `json:"image"`
						Audio struct {
							ID       string `json:"id"`
							MimeType string `json:"mime_type"`
						} `json:"audio"`
						Document struct {
							ID       string `json:"id"`
							MimeType string `json:"mime_type"`
							Caption  string `json:"caption"`
							Filename string `json:"filename"`
						} `json:"document"`
					} `json:"messages"`
					Metadata struct {
						PhoneNumberID string `json:"phone_number_id"`
					} `json:"metadata"`
				} `json:"value"`
			} `json:"changes"`
		} `json:"entry"`
	}

	if err := json.Unmarshal(payload, &body); err != nil {
		return WebhookEvent{}, fmt.Errorf("unmarshal payload: %w", err)
	}

	if len(body.Entry) == 0 || len(body.Entry[0].Changes) == 0 {
		return WebhookEvent{}, errors.New("no events in payload")
	}

	change := body.Entry[0].Changes[0].Value
	if len(change.Messages) == 0 {
		return WebhookEvent{}, errors.New("no messages in payload")
	}

	msg := change.Messages[0]
	ts, _ := strconv.ParseInt(msg.Timestamp, 10, 64)

	evt := WebhookEvent{
		EventID:     msg.ID,
		From:        msg.From,
		To:          change.Metadata.PhoneNumberID,
		MessageID:   msg.ID,
		Type:        msg.Type,
		Timestamp:   ts,
		ChannelType: "whatsapp",
	}

	switch msg.Type {
	case "text":
		evt.Content, _ = json.Marshal(map[string]string{"body": msg.Text.Body})
	case "image":
		evt.Content, _ = json.Marshal(msg.Image)
	case "audio":
		evt.Content, _ = json.Marshal(msg.Audio)
	case "document":
		evt.Content, _ = json.Marshal(msg.Document)
	default:
		evt.Content = json.RawMessage("{}")
	}

	return evt, nil
}
