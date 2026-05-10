package channel

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WhatsApp implements the Channel interface for the Meta WhatsApp Business API.
type WhatsApp struct {
	httpClient      *http.Client
	baseURL         string
	accessToken     string
	phoneNumberID   string
	appSecret       string
	verifyToken     string
}

// NewWhatsApp creates a WhatsApp channel from explicit credentials.
func NewWhatsApp(baseURL, accessToken, phoneNumberID, appSecret, verifyToken string) *WhatsApp {
	return &WhatsApp{
		httpClient: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		baseURL:       baseURL,
		accessToken:   accessToken,
		phoneNumberID: phoneNumberID,
		appSecret:     appSecret,
		verifyToken:   verifyToken,
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

	url := fmt.Sprintf("%s/v19.0/%s/messages", w.baseURL, w.phoneNumberID)
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

// SendTemplateMessage sends a WhatsApp template message.
func (w *WhatsApp) SendTemplateMessage(ctx context.Context, to, templateID string, params map[string]string) error {
	return errors.New("not implemented")
}

// SendMediaMessage sends a WhatsApp media message.
func (w *WhatsApp) SendMediaMessage(ctx context.Context, to, mediaType, mediaURL, caption string) error {
	return errors.New("not implemented")
}

// MarkRead marks a message as read on WhatsApp.
func (w *WhatsApp) MarkRead(ctx context.Context, messageID string) error {
	return errors.New("not implemented")
}

// VerifyWebhook checks the subscription challenge.
func (w *WhatsApp) VerifyWebhook(mode, verifyToken, challenge string) (string, error) {
	if mode != "subscribe" {
		return "", errors.New("invalid mode")
	}
	if verifyToken != w.verifyToken {
		return "", errors.New("invalid verify token")
	}
	return challenge, nil
}

// VerifySignature validates the HMAC-SHA256 signature on a webhook payload.
func (w *WhatsApp) VerifySignature(payload []byte, signature string) error {
	if w.appSecret == "" {
		return errors.New("app secret not configured")
	}

	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return errors.New("invalid signature format")
	}

	expectedMAC, err := hex.DecodeString(strings.TrimPrefix(signature, prefix))
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(w.appSecret))
	mac.Write(payload)
	computedMAC := mac.Sum(nil)

	if !hmac.Equal(expectedMAC, computedMAC) {
		return errors.New("signature mismatch")
	}
	return nil
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
		EventID:   msg.ID,
		From:      msg.From,
		To:        change.Metadata.PhoneNumberID,
		MessageID: msg.ID,
		Type:      msg.Type,
		Timestamp: ts,
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
