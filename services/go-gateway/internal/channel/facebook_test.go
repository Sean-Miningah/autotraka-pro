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

func TestFacebookVerifySignature_Valid(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "my-secret", "verify-token")

	payload := []byte(`{"test":"data"}`)
	mac := hmac.New(sha256.New, []byte("my-secret"))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if err := fb.VerifySignature(payload, signature); err != nil {
		t.Fatalf("expected valid signature to pass, got error: %v", err)
	}
}

func TestFacebookVerifySignature_Invalid(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "my-secret", "verify-token")

	payload := []byte(`{"test":"data"}`)
	signature := "sha256=0000000000000000000000000000000000000000000000000000000000000000"

	if err := fb.VerifySignature(payload, signature); err == nil {
		t.Fatal("expected invalid signature to fail")
	}
}

func TestFacebookVerifySignature_WrongFormat(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "my-secret", "verify-token")

	if err := fb.VerifySignature([]byte("{}"), "bad-format"); err == nil {
		t.Fatal("expected bad format to fail")
	}
}

func TestFacebookVerifySignature_NoSecret(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "", "verify-token")

	if err := fb.VerifySignature([]byte("{}"), "sha256=abc"); err == nil {
		t.Fatal("expected missing secret to fail")
	}
}

func TestFacebookSendTextMessage(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v19.0/page-id/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", auth)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		recipient, ok := body["recipient"].(map[string]interface{})
		if !ok || recipient["id"] != "fb-user-123" {
			t.Errorf("unexpected recipient: %v", body["recipient"])
		}
		msg, ok := body["message"].(map[string]interface{})
		if !ok || msg["text"] != "Hello Facebook" {
			t.Errorf("unexpected message: %v", body["message"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"recipient_id":"fb-user-123","message_id":"msg_001"}`))
	}))
	defer mockServer.Close()

	fb := NewFacebook(mockServer.URL, "test-token", "page-id", "secret", "verify")
	err := fb.SendTextMessage(context.Background(), "fb-user-123", "Hello Facebook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFacebookSendTextMessage_Failure(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer mockServer.Close()

	fb := NewFacebook(mockServer.URL, "bad-token", "page-id", "secret", "verify")
	err := fb.SendTextMessage(context.Background(), "fb-user-123", "Hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFacebookVerifyWebhook(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "my-verify-token")

	challenge, err := fb.VerifyWebhook("subscribe", "my-verify-token", "hub-challenge-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if challenge != "hub-challenge-123" {
		t.Errorf("expected challenge hub-challenge-123, got %s", challenge)
	}
}

func TestFacebookVerifyWebhook_InvalidToken(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "my-verify-token")

	_, err := fb.VerifyWebhook("subscribe", "wrong-token", "hub-challenge-123")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestFacebookVerifyWebhook_InvalidMode(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "my-verify-token")

	_, err := fb.VerifyWebhook("unsubscribe", "my-verify-token", "hub-challenge-123")
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestFacebookParseWebhookEvent_TextMessage(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "verify")

	payload := []byte(`{
		"object": "page",
		"entry": [{
			"id": "PAGE_ID",
			"time": 1234567890,
			"messaging": [{
				"sender": {"id": "FB_SENDER_001"},
				"recipient": {"id": "PAGE_ID"},
				"timestamp": 1234567890,
				"message": {
					"mid": "FB_MSG_001",
					"text": "Hello from Facebook"
				}
			}]
		}]
	}`)

	evt, err := fb.ParseWebhookEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.EventID != "FB_MSG_001" {
		t.Errorf("expected event_id FB_MSG_001, got %s", evt.EventID)
	}
	if evt.From != "FB_SENDER_001" {
		t.Errorf("expected from FB_SENDER_001, got %s", evt.From)
	}
	if evt.To != "PAGE_ID" {
		t.Errorf("expected to PAGE_ID, got %s", evt.To)
	}
	if evt.Type != "text" {
		t.Errorf("expected type text, got %s", evt.Type)
	}
	if evt.Timestamp != 1234567890 {
		t.Errorf("expected timestamp 1234567890, got %d", evt.Timestamp)
	}
	if evt.ChannelType != "facebook" {
		t.Errorf("expected channel_type facebook, got %s", evt.ChannelType)
	}

	var content map[string]string
	if err := json.Unmarshal(evt.Content, &content); err != nil {
		t.Fatalf("failed to unmarshal content: %v", err)
	}
	if content["body"] != "Hello from Facebook" {
		t.Errorf("unexpected content body: %s", content["body"])
	}
}

func TestFacebookParseWebhookEvent_ImageAttachment(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "verify")

	payload := []byte(`{
		"object": "page",
		"entry": [{
			"id": "PAGE_ID",
			"time": 1234567890,
			"messaging": [{
				"sender": {"id": "FB_SENDER_001"},
				"recipient": {"id": "PAGE_ID"},
				"timestamp": 1234567890,
				"message": {
					"mid": "FB_MSG_002",
					"attachments": [{
						"type": "image",
						"payload": {"url": "https://example.com/image.jpg"}
					}]
				}
			}]
		}]
	}`)

	evt, err := fb.ParseWebhookEvent(payload)
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

func TestFacebookParseWebhookEvent_Postback(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "verify")

	payload := []byte(`{
		"object": "page",
		"entry": [{
			"id": "PAGE_ID",
			"time": 1234567890,
			"messaging": [{
				"sender": {"id": "FB_SENDER_001"},
				"recipient": {"id": "PAGE_ID"},
				"timestamp": 1234567890,
				"postback": {
					"title": "Get Started",
					"payload": "GET_STARTED_PAYLOAD"
				}
			}]
		}]
	}`)

	evt, err := fb.ParseWebhookEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.Type != "postback" {
		t.Errorf("expected type postback, got %s", evt.Type)
	}
	if evt.From != "FB_SENDER_001" {
		t.Errorf("expected from FB_SENDER_001, got %s", evt.From)
	}

	var content map[string]string
	if err := json.Unmarshal(evt.Content, &content); err != nil {
		t.Fatalf("failed to unmarshal content: %v", err)
	}
	if content["payload"] != "GET_STARTED_PAYLOAD" {
		t.Errorf("unexpected payload: %s", content["payload"])
	}
}

func TestFacebookParseWebhookEvent_WrongObject(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "verify")

	payload := []byte(`{"object": "instagram", "entry": []}`)

	_, err := fb.ParseWebhookEvent(payload)
	if err == nil {
		t.Fatal("expected error for wrong object type")
	}
}

func TestFacebookParseWebhookEvent_NoEntries(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "verify")

	payload := []byte(`{"object": "page", "entry": []}`)

	_, err := fb.ParseWebhookEvent(payload)
	if err == nil {
		t.Fatal("expected error for empty entries")
	}
}

func TestFacebookParseWebhookEvent_NoMessaging(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "verify")

	payload := []byte(`{"object": "page", "entry": [{"id": "1", "time": 1, "messaging": []}]}`)

	_, err := fb.ParseWebhookEvent(payload)
	if err == nil {
		t.Fatal("expected error for empty messaging")
	}
}

func TestFacebookParseWebhookEvent_InvalidJSON(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "verify")

	_, err := fb.ParseWebhookEvent([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestFacebookChannelType(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "verify")
	if fb.ChannelType() != "facebook" {
		t.Errorf("expected channel type facebook, got %s", fb.ChannelType())
	}
}

func TestFacebookSendTemplateMessage_NotSupported(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "verify")
	err := fb.SendTemplateMessage(context.Background(), "to", "template", "en", nil)
	if err == nil {
		t.Fatal("expected error for unsupported template messages")
	}
}

func TestFacebookSendMediaMessage_NotImplemented(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "verify")
	err := fb.SendMediaMessage(context.Background(), "to", "image", "url", "caption")
	if err == nil {
		t.Fatal("expected not implemented error")
	}
}

func TestFacebookMarkRead_NotImplemented(t *testing.T) {
	fb := NewFacebook("http://example.com", "token", "page-id", "secret", "verify")
	err := fb.MarkRead(context.Background(), "msg-id")
	if err == nil {
		t.Fatal("expected not implemented error")
	}
}
