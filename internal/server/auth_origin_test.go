package server

import (
	"net/http"
	"net/http/httptest"
	"qwq/internal/config"
	"testing"
)

func TestWriteOriginPolicy(t *testing.T) {
	previous := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = previous })
	config.GlobalConfig.WebUser, config.GlobalConfig.WebPassword = "operator", "secret"
	tests := []struct {
		name, target, origin string
		allowed              bool
	}{
		{"same origin", "http://localhost:8080/api/files/action", "http://localhost:8080", true},
		{"missing origin", "http://localhost:8080/api/files/action", "", true},
		{"foreign origin", "http://localhost:8080/api/files/action", "http://untrusted.example", false},
		{"wrong port", "http://localhost:8080/api/files/action", "http://localhost:5173", false},
		{"TLS terminating proxy", "http://ops.example/api/files/action", "https://ops.example", true},
		{"unsupported scheme", "http://localhost:8080/api/files/action", "file://localhost:8080", false},
		{"malformed origin", "http://localhost:8080/api/files/action", "http://%", false},
		{"null origin", "http://localhost:8080/api/files/action", "null", false},
		{"origin with path", "http://localhost:8080/api/files/action", "http://localhost:8080/path", false},
		{"origin with credentials", "http://localhost:8080/api/files/action", "http://user@localhost:8080", false},
		{"TLS origin", "https://ops.example/api/files/action", "https://ops.example", true},
		{"IPv6 origin", "http://[::1]:8080/api/files/action", "http://[::1]:8080", true},
		{"development proxy", "http://localhost:5173/api/files/action", "http://localhost:5173", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
				request := httptest.NewRequest(method, tt.target, nil)
				request.SetBasicAuth("operator", "secret")
				if tt.origin != "" {
					request.Header.Set("Origin", tt.origin)
				}
				response := httptest.NewRecorder()
				called := false
				basicAuth(func(w http.ResponseWriter, r *http.Request) {
					called = true
					w.WriteHeader(http.StatusNoContent)
				})(response, request)
				if called != tt.allowed || (!tt.allowed && response.Code != http.StatusForbidden) {
					t.Fatalf("%s allowed=%v, status=%d", method, called, response.Code)
				}
			}
		})
	}
}

func TestOriginPolicyAllowsReadMethods(t *testing.T) {
	previous := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = previous })
	config.GlobalConfig.WebUser, config.GlobalConfig.WebPassword = "", ""
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		request := httptest.NewRequest(method, "http://localhost:8080/api/stats", nil)
		request.Header.Set("Origin", "http://untrusted.example")
		response := httptest.NewRecorder()
		basicAuth(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("%s returned %d", method, response.Code)
		}
	}
}

func TestOriginPolicyRejectsMultipleOrigins(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://localhost:8080/", nil)
	request.Header.Add("Origin", "http://localhost:8080")
	request.Header.Add("Origin", "http://untrusted.example")
	if hasSameOrigin(request) {
		t.Fatal("multiple origins must be rejected")
	}
}
