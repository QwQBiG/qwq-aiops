package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"qwq/internal/config"
	"testing"
)

func TestLivenessWithAuthenticationConfigured(t *testing.T) {
	previous := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = previous })
	config.GlobalConfig.WebUser, config.GlobalConfig.WebPassword = "operator", "secret"
	response := httptest.NewRecorder()
	handleLiveness(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	var status map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(status) != 1 || status["status"] != "ok" {
		t.Fatalf("unexpected liveness response: %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	basicAuth(handleHealthCheck)(response, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("deployment health must remain protected: %d", response.Code)
	}
}

func TestLivenessMethods(t *testing.T) {
	for _, method := range []string{http.MethodHead, http.MethodPost} {
		response := httptest.NewRecorder()
		handleLiveness(response, httptest.NewRequest(method, "/health", nil))
		if method == http.MethodHead && (response.Code != http.StatusOK || response.Body.Len() != 0) {
			t.Fatalf("unexpected HEAD response: %d %s", response.Code, response.Body.String())
		}
		if method == http.MethodPost && (response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET, HEAD") {
			t.Fatalf("unexpected POST response: %d", response.Code)
		}
	}
}
