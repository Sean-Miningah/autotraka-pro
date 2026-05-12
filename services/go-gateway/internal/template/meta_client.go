package template

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MetaTemplateAPI implements MetaTemplateClient using the Meta WhatsApp Business API.
type MetaTemplateAPI struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewMetaTemplateAPI creates a new Meta Templates API client.
func NewMetaTemplateAPI(baseURL, token string) *MetaTemplateAPI {
	return &MetaTemplateAPI{
		httpClient: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		baseURL: baseURL,
		token:   token,
	}
}

// CreateTemplate submits a template to the Meta WhatsApp Templates API.
func (c *MetaTemplateAPI) CreateTemplate(ctx context.Context, wabaID string, req MetaCreateTemplateReq) (string, error) {
	components := []map[string]interface{}{
		{
			"type": "BODY",
			"text": req.Body,
		},
	}

	payload := map[string]interface{}{
		"name":       req.Name,
		"category":   req.Category,
		"language":   req.Language,
		"components": components,
	}

	url := fmt.Sprintf("%s/v19.0/%s/message_templates", c.baseURL, wabaID)
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return result.ID, nil
}

// GetTemplateStatus queries Meta for a template's approval status.
func (c *MetaTemplateAPI) GetTemplateStatus(ctx context.Context, wabaID, templateName string) (string, error) {
	url := fmt.Sprintf("%s/v19.0/%s/message_templates?name=%s", c.baseURL, wabaID, templateName)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	for _, t := range result.Data {
		if t.Name == templateName {
			return t.Status, nil
		}
	}
	return "PENDING", nil
}
