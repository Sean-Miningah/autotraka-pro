package messaging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// MetaClient handles communication with the Meta WhatsApp API.
type MetaClient struct {
	HTTPClient *http.Client
	BaseURL    string
	Token      string
}

// NewMetaClient creates a new Meta API client.
func NewMetaClient(baseURL, token string) *MetaClient {
	return &MetaClient{
		HTTPClient: &http.Client{},
		BaseURL:    baseURL,
		Token:      token,
	}
}

// SendTextMessage sends a text message via the WhatsApp API.
func (c *MetaClient) SendTextMessage(phoneNumberID, to, body string) error {
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              "text",
		"text":              map[string]string{"body": body},
	}

	url := fmt.Sprintf("%s/v19.0/%s/messages", c.BaseURL, phoneNumberID)
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
