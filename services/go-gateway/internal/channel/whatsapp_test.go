package channel

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWhatsAppVerifySignature_Valid(t *testing.T) {
	wa := NewWhatsApp("http://example.com", "token", "phone-id", "my-secret", "verify-token")

	payload := []byte(`{"test":"data"}`)
	mac := hmac.New(sha256.New, []byte("my-secret"))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if err := wa.VerifySignature(payload, signature); err != nil {
		t.Fatalf("expected valid signature to pass, got error: %v", err)
	}
}

func TestWhatsAppVerifySignature_Invalid(t *testing.T) {
	wa := NewWhatsApp("http://example.com", "token", "phone-id", "my-secret", "verify-token")

	payload := []byte(`{"test":"data"}`)
	signature := "sha256=0000000000000000000000000000000000000000000000000000000000000000"

	if err := wa.VerifySignature(payload, signature); err == nil {
		t.Fatal("expected invalid signature to fail")
	}
}

func TestWhatsAppVerifySignature_WrongFormat(t *testing.T) {
	wa := NewWhatsApp("http://example.com", "token", "phone-id", "my-secret", "verify-token")

	if err := wa.VerifySignature([]byte("{}"), "bad-format"); err == nil {
		t.Fatal("expected bad format to fail")
	}
}

func TestWhatsAppVerifySignature_NoSecret(t *testing.T) {
	wa := NewWhatsApp("http://example.com", "token", "phone-id", "", "verify-token")

	if err := wa.VerifySignature([]byte("{}"), "sha256=abc"); err == nil {
		t.Fatal("expected missing secret to fail")
	}
}

func TestWhatsAppSendTextMessage(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v19.0/123456789/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", auth)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"messages":[{"id":"mock_msg_id"}]}`))
	}))
	defer mockServer.Close()

	wa := NewWhatsApp(mockServer.URL, "test-token", "123456789", "secret", "verify")
	err := wa.SendTextMessage(context.Background(), "+1234567890", "Hello from unit test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWhatsAppSendTextMessage_Failure(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer mockServer.Close()

	wa := NewWhatsApp(mockServer.URL, "bad-token", "123456789", "secret", "verify")
	err := wa.SendTextMessage(context.Background(), "+1234567890", "Hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWhatsAppVerifyWebhook(t *testing.T) {
	wa := NewWhatsApp("http://example.com", "token", "phone-id", "secret", "my-verify-token")

	challenge, err := wa.VerifyWebhook("subscribe", "my-verify-token", "hub-challenge-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if challenge != "hub-challenge-123" {
		t.Errorf("expected challenge hub-challenge-123, got %s", challenge)
	}
}

func TestWhatsAppVerifyWebhook_InvalidToken(t *testing.T) {
	wa := NewWhatsApp("http://example.com", "token", "phone-id", "secret", "my-verify-token")

	_, err := wa.VerifyWebhook("subscribe", "wrong-token", "hub-challenge-123")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestWhatsAppVerifyWebhook_InvalidMode(t *testing.T) {
	wa := NewWhatsApp("http://example.com", "token", "phone-id", "secret", "my-verify-token")

	_, err := wa.VerifyWebhook("unsubscribe", "my-verify-token", "hub-challenge-123")
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestWhatsAppParseWebhookEvent_TextMessage(t *testing.T) {
	wa := NewWhatsApp("http://example.com", "token", "phone-id", "secret", "verify")

	payload := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "BUSINESS_ID",
			"changes": [{
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {"display_phone_number": "PHONE", "phone_number_id": "PHONE_ID"},
					"contacts": [{"profile": {"name": "John"}, "wa_id": "12345"}],
					"messages": [{
						"from": "12345",
						"id": "MSG_ID",
						"timestamp": "1234567890",
						"text": {"body": "Hello there"},
						"type": "text"
					}]
				},
				"field": "messages"
			}]
		}]
	}`)

	evt, err := wa.ParseWebhookEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.EventID != "MSG_ID" {
		t.Errorf("expected event_id MSG_ID, got %s", evt.EventID)
	}
	if evt.From != "12345" {
		t.Errorf("expected from 12345, got %s", evt.From)
	}
	if evt.To != "PHONE_ID" {
		t.Errorf("expected to PHONE_ID, got %s", evt.To)
	}
	if evt.Type != "text" {
		t.Errorf("expected type text, got %s", evt.Type)
	}
	if evt.Timestamp != 1234567890 {
		t.Errorf("expected timestamp 1234567890, got %d", evt.Timestamp)
	}
}

func TestWhatsAppParseWebhookEvent_NoMessages(t *testing.T) {
	wa := NewWhatsApp("http://example.com", "token", "phone-id", "secret", "verify")

	payload := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "BUSINESS_ID",
			"changes": [{
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {},
					"messages": []
				},
				"field": "messages"
			}]
		}]
	}`)

	_, err := wa.ParseWebhookEvent(payload)
	if err == nil {
		t.Fatal("expected error for empty messages")
	}
}
