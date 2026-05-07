package messaging

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetaClient_SendTextMessage(t *testing.T) {
	// Create a mock server to simulate Meta API
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

	client := NewMetaClient(mockServer.URL, "test-token")
	err := client.SendTextMessage("123456789", "+1234567890", "Hello from unit test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMetaClient_SendTextMessage_Failure(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer mockServer.Close()

	client := NewMetaClient(mockServer.URL, "bad-token")
	err := client.SendTextMessage("123456789", "+1234567890", "Hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
