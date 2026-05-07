package health

import (
	"encoding/json"
	"net/http"
	"time"
)

// Response represents the health check response.
type Response struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// Handler returns the service health status.
func Handler(w http.ResponseWriter, r *http.Request) {
	resp := Response{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Version:   "0.1.0",
		Checks: map[string]string{
			"gateway": "ok",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Ping returns a simple pong response.
func Ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
}
