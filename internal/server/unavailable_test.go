package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnconnectedDashboardOperations(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		methods []string
	}{
		{"applications", handleAppStoreInstances, []string{http.MethodPost, http.MethodDelete}},
		{"databases", handleDatabaseConnections, []string{http.MethodPost, http.MethodPut, http.MethodDelete}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, method := range tt.methods {
				response := httptest.NewRecorder()
				tt.handler(response, httptest.NewRequest(method, "/", strings.NewReader(`{"name":"example"}`)))
				var body map[string]string
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if response.Code != http.StatusNotImplemented || body["error"] == "" || body["status"] != "" {
					t.Fatalf("%s returned a false success: %d %s", method, response.Code, response.Body.String())
				}
			}
			response := httptest.NewRecorder()
			tt.handler(response, httptest.NewRequest(http.MethodGet, "/", nil))
			if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != "[]" {
				t.Fatalf("unexpected empty list response: %d %s", response.Code, response.Body.String())
			}
		})
	}
}
