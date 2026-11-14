package registry

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegistryManagementIsLocalOnly(t *testing.T) {
	registry := NewServiceRegistry()
	defer registry.Stop()
	server := newRegistryHTTPServer(9080, registry, nil)
	if server.Addr != "127.0.0.1:9080" {
		t.Fatalf("management API exposed on %s", server.Addr)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/registry/services", nil)
	r.Header.Set("Origin", "https://foreign.example")
	server.Handler.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("registry API granted browser access to an unrelated origin")
	}
}
