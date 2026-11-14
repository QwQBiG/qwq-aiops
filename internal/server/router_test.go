package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"qwq/internal/config"
	"strings"
	"testing"
	"testing/fstest"
)

func TestIndependentRoutersKeepUnknownEndpointsOutOfSPA(t *testing.T) {
	previous := config.GlobalConfig
	config.GlobalConfig.WebUser, config.GlobalConfig.WebPassword = "", ""
	t.Cleanup(func() { config.GlobalConfig = previous })
	defaultMux := http.DefaultServeMux
	for i := 0; i < 2; i++ {
		handler := NewHandler()
		for _, path := range []string{"/api", "/api/not-implemented", "/ws", "/ws/not-implemented"} {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			var body FileResponse
			if response.Code != http.StatusNotFound || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.Code != http.StatusNotFound {
				t.Fatalf("%s returned %d %s", path, response.Code, response.Body.String())
			}
		}
	}
	if http.DefaultServeMux != defaultMux {
		t.Fatal("constructing a handler replaced the global router")
	}
}

func TestFrontendRoutesCacheAndHead(t *testing.T) {
	assets := fstest.MapFS{
		"index.html":             {Data: []byte("<!doctype html><html>console</html>")},
		"assets/app-abcdefgh.js": {Data: []byte("console.log('ready')")},
		"assets/shared.js":       {Data: []byte("export default {}")},
	}
	handler := newFrontendHandler(assets)
	for _, tc := range []struct {
		method, path       string
		status             int
		contentType, cache string
		empty              bool
	}{
		{http.MethodGet, "/", 200, "text/html", "no-cache", false},
		{http.MethodGet, "/containers", 200, "text/html", "no-cache", false},
		{http.MethodGet, "/assets/app-abcdefgh.js", 200, "application/javascript", "immutable", false},
		{http.MethodGet, "/assets/shared.js", 200, "application/javascript", "no-cache", false},
		{http.MethodHead, "/assets/app-abcdefgh.js", 200, "application/javascript", "immutable", true},
		{http.MethodGet, "/assets/missing.js", 404, "text/plain", "", false},
		{http.MethodPost, "/", 405, "text/plain", "", false},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
		if response.Code != tc.status || !strings.Contains(response.Header().Get("Content-Type"), tc.contentType) || !strings.Contains(response.Header().Get("Cache-Control"), tc.cache) {
			t.Errorf("%s %s: status=%d headers=%v", tc.method, tc.path, response.Code, response.Header())
		}
		if tc.empty && response.Body.Len() != 0 {
			t.Errorf("HEAD returned %d body bytes", response.Body.Len())
		}
	}
}
