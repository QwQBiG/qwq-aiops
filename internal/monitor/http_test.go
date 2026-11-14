package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"qwq/internal/config"
	"testing"
	"time"
)

func TestHTTPChecksReportStatusAndSupportCancellation(t *testing.T) {
	previous := config.GlobalConfig.HTTPRules
	defer func() { config.GlobalConfig.HTTPRules = previous }()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	config.GlobalConfig.HTTPRules = []config.HTTPRule{{Name: "matching", URL: server.URL, Code: http.StatusNoContent}, {Name: "mismatch", URL: server.URL}}
	results := RunChecksContext(context.Background())
	server.Close()
	if len(results) != 2 || !results[0].Success || results[1].Success || results[1].Error == "" {
		t.Fatalf("unexpected HTTP check results: %#v", results)
	}
	started, cancelled := make(chan struct{}), make(chan struct{})
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer server.Close()
	config.GlobalConfig.HTTPRules = []config.HTTPRule{{Name: "pending", URL: server.URL}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan []CheckResult, 1)
	go func() { done <- RunChecksContext(ctx) }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP check did not start")
	}
	cancel()
	select {
	case result := <-done:
		if len(result) != 1 || result[0].Success {
			t.Fatalf("cancelled check returned %#v", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP check ignored cancellation")
	}
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("remote check request remained active")
	}
}
