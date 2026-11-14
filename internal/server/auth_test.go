package server

import (
	"net/http"
	"net/http/httptest"
	"qwq/internal/config"
	"testing"
)

func TestBasicAuth(t *testing.T) {
	previous := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = previous })
	tests := []struct {
		name, configuredUser, configuredPassword string
		user, password                           string
		status                                   int
	}{
		{"disabled", "", "", "", "", http.StatusNoContent},
		{"missing credentials", "operator", "secret", "", "", http.StatusUnauthorized},
		{"wrong credentials", "operator", "secret", "operator", "wrong", http.StatusUnauthorized},
		{"valid credentials", "operator", "secret", "operator", "secret", http.StatusNoContent},
		{"missing password configuration", "operator", "", "operator", "", http.StatusServiceUnavailable},
		{"missing username configuration", "", "secret", "", "secret", http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.GlobalConfig.WebUser = tt.configuredUser
			config.GlobalConfig.WebPassword = tt.configuredPassword
			called := false
			handler := basicAuth(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
			if tt.user != "" || tt.password != "" {
				req.SetBasicAuth(tt.user, tt.password)
			}
			response := httptest.NewRecorder()
			handler(response, req)
			if response.Code != tt.status || called != (tt.status == http.StatusNoContent) {
				t.Fatalf("status=%d, handler called=%v", response.Code, called)
			}
			if tt.status == http.StatusUnauthorized && response.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("missing authentication challenge")
			}
		})
	}
}
