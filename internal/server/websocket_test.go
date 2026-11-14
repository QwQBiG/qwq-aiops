package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestWebSocketOriginPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			connection.Close()
		}
	}))
	defer server.Close()
	tests := []struct {
		name, origin, host string
		allowed            bool
	}{
		{"same origin", server.URL, "", true},
		{"non-browser client", "", "", true},
		{"cross origin", "http://untrusted.example", "", false},
		{"development proxy", "http://localhost:5173", "localhost:5173", true},
		{"cross origin through proxy", "http://untrusted.example", "localhost:5173", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}
			if tt.origin != "" {
				headers.Set("Origin", tt.origin)
			}
			if tt.host != "" {
				headers.Set("Host", tt.host)
			}
			connection, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), headers)
			if connection != nil {
				connection.Close()
			}
			if response != nil {
				defer response.Body.Close()
			}
			if tt.allowed && err != nil {
				t.Fatalf("expected successful upgrade: %v", err)
			}
			if !tt.allowed && (err == nil || response == nil || response.StatusCode != http.StatusForbidden) {
				t.Fatalf("expected forbidden upgrade, response=%v error=%v", response, err)
			}
		})
	}
}
