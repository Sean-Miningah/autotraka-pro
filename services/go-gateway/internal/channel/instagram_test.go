package channel

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInstagramVerifySignature_Valid(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "my-secret", "verify-token")

	payload := []byte(`{"test":"data"}`)
	mac := hmac.New(sha256.New, []byte("my-secret"))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if err := ig.VerifySignature(payload, signature); err != nil {
		t.Fatalf("expected valid signature to pass, got error: %v", err)
	}
}

func TestInstagramVerifySignature_Invalid(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "my-secret", "verify-token")

	payload := []byte(`{"test":"data"}`)
	signature := "sha256=0000000000000000000000000000000000000000000000000000000000000000"

	if err := ig.VerifySignature(payload, signature); err == nil {
		t.Fatal("expected invalid signature to fail")
	}
}

func TestInstagramVerifySignature_WrongFormat(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "my-secret", "verify-token")

	if err := ig.VerifySignature([]byte("{}"), "bad-format"); err == nil {
		t.Fatal("expected bad format to fail")
	}
}

func TestInstagramVerifySignature_NoSecret(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "", "verify-token")

	if err := ig.VerifySignature([]byte("{}"), "sha256=abc"); err == nil {
		t.Fatal("expected missing secret to fail")
	}
}

func TestInstagramSendTextMessage(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v19.0/account-id/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", auth)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		recipient, ok := body["recipient"].(map[string]interface{})
		if !ok || recipient["id"] != "ig-user-123" {
			t.Errorf("unexpected recipient: %v", body["recipient"])
		}
		msg, ok := body["message"].(map[string]interface{})
		if !ok || msg["text"] != "Hello Instagram" {
			t.Errorf("unexpected message: %v", body["message"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"recipient_id":"ig-user-123","message_id":"msg_001"}`))
	}))
	defer mockServer.Close()

	ig := NewInstagram(mockServer.URL, "test-token", "account-id", "secret", "verify")
	err := ig.SendTextMessage(context.Background(), "ig-user-123", "Hello Instagram")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstagramSendTextMessage_Failure(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer mockServer.Close()

	ig := NewInstagram(mockServer.URL, "bad-token", "account-id", "secret", "verify")
	err := ig.SendTextMessage(context.Background(), "ig-user-123", "Hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestInstagramVerifyWebhook(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "my-verify-token")

	challenge, err := ig.VerifyWebhook("subscribe", "my-verify-token", "hub-challenge-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if challenge != "hub-challenge-123" {
		t.Errorf("expected challenge hub-challenge-123, got %s", challenge)
	}
}

func TestInstagramVerifyWebhook_InvalidToken(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "my-verify-token")

	_, err := ig.VerifyWebhook("subscribe", "wrong-token", "hub-challenge-123")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestInstagramVerifyWebhook_InvalidMode(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "my-verify-token")

	_, err := ig.VerifyWebhook("unsubscribe", "my-verify-token", "hub-challenge-123")
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestInstagramParseWebhookEvent_TextMessage(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "verify")

	payload := []byte(`{
		"object": "instagram",
		"entry": [{
			"id": "IG_ACCOUNT_ID",
			"time": 1234567890,
			"messaging": [{
				"sender": {"id": "IG_SENDER_001"},
				"recipient": {"id": "IG_ACCOUNT_ID"},
				"timestamp": 1234567890,
				"message": {
					"mid": "IG_MSG_001",
					"text": "Hello from Instagram"
				}
			}]
		}]
	}`)

	evt, err := ig.ParseWebhookEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.EventID != "IG_MSG_001" {
		t.Errorf("expected event_id IG_MSG_001, got %s", evt.EventID)
	}
	if evt.From != "IG_SENDER_001" {
		t.Errorf("expected from IG_SENDER_001, got %s", evt.From)
	}
	if evt.To != "IG_ACCOUNT_ID" {
		t.Errorf("expected to IG_ACCOUNT_ID, got %s", evt.To)
	}
	if evt.Type != "text" {
		t.Errorf("expected type text, got %s", evt.Type)
	}
	if evt.Timestamp != 1234567890 {
		t.Errorf("expected timestamp 1234567890, got %d", evt.Timestamp)
	}
	if evt.ChannelType != "instagram" {
		t.Errorf("expected channel_type instagram, got %s", evt.ChannelType)
	}

	var content map[string]string
	if err := json.Unmarshal(evt.Content, &content); err != nil {
		t.Fatalf("failed to unmarshal content: %v", err)
	}
	if content["body"] != "Hello from Instagram" {
		t.Errorf("unexpected content body: %s", content["body"])
	}
}

func TestInstagramParseWebhookEvent_ImageAttachment(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "verify")

	payload := []byte(`{
		"object": "instagram",
		"entry": [{
			"id": "IG_ACCOUNT_ID",
			"time": 1234567890,
			"messaging": [{
				"sender": {"id": "IG_SENDER_001"},
				"recipient": {"id": "IG_ACCOUNT_ID"},
				"timestamp": 1234567890,
				"message": {
					"mid": "IG_MSG_002",
					"attachments": [{
						"type": "image",
						"payload": {"url": "https://example.com/image.jpg"}
					}]
				}
			}]
		}]
	}`)

	evt, err := ig.ParseWebhookEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.Type != "image" {
		t.Errorf("expected type image, got %s", evt.Type)
	}

	var content map[string]string
	if err := json.Unmarshal(evt.Content, &content); err != nil {
		t.Fatalf("failed to unmarshal content: %v", err)
	}
	if content["url"] != "https://example.com/image.jpg" {
		t.Errorf("unexpected content url: %s", content["url"])
	}
}

func TestInstagramParseWebhookEvent_Postback(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "verify")

	payload := []byte(`{
		"object": "instagram",
		"entry": [{
			"id": "IG_ACCOUNT_ID",
			"time": 1234567890,
			"messaging": [{
				"sender": {"id": "IG_SENDER_001"},
				"recipient": {"id": "IG_ACCOUNT_ID"},
				"timestamp": 1234567890,
				"postback": {
					"title": "Get Started",
					"payload": "GET_STARTED_PAYLOAD"
				}
			}]
		}]
	}`)

	evt, err := ig.ParseWebhookEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.Type != "postback" {
		t.Errorf("expected type postback, got %s", evt.Type)
	}
	if evt.From != "IG_SENDER_001" {
		t.Errorf("expected from IG_SENDER_001, got %s", evt.From)
	}

	var content map[string]string
	if err := json.Unmarshal(evt.Content, &content); err != nil {
		t.Fatalf("failed to unmarshal content: %v", err)
	}
	if content["payload"] != "GET_STARTED_PAYLOAD" {
		t.Errorf("unexpected payload: %s", content["payload"])
	}
}

func TestInstagramParseWebhookEvent_NoEntries(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "verify")

	payload := []byte(`{"object": "instagram", "entry": []}`)

	_, err := ig.ParseWebhookEvent(payload)
	if err == nil {
		t.Fatal("expected error for empty entries")
	}
}

func TestInstagramParseWebhookEvent_NoMessaging(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "verify")

	payload := []byte(`{"object": "instagram", "entry": [{"id": "1", "time": 1, "messaging": []}]}`)

	_, err := ig.ParseWebhookEvent(payload)
	if err == nil {
		t.Fatal("expected error for empty messaging")
	}
}

func TestInstagramParseWebhookEvent_InvalidJSON(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "verify")

	_, err := ig.ParseWebhookEvent([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestInstagramChannelType(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "verify")
	if ig.ChannelType() != "instagram" {
		t.Errorf("expected channel type instagram, got %s", ig.ChannelType())
	}
}

func TestInstagramSendTemplateMessage_NotSupported(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "verify")
	err := ig.SendTemplateMessage(context.Background(), "to", "template", "en", nil)
	if err == nil {
		t.Fatal("expected error for unsupported template messages")
	}
}

func TestInstagramSendMediaMessage_NotImplemented(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "verify")
	err := ig.SendMediaMessage(context.Background(), "to", "image", "url", "caption")
	if err == nil {
		t.Fatal("expected not implemented error")
	}
}

func TestInstagramMarkRead_NotImplemented(t *testing.T) {
	ig := NewInstagram("http://example.com", "token", "account-id", "secret", "verify")
	err := ig.MarkRead(context.Background(), "msg-id")
	if err == nil {
		t.Fatal("expected not implemented error")
	}
}
