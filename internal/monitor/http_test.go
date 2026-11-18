package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"qwq/internal/config"
	"strings"
	"testing"
	"time"
)

func TestHTTPChecksUseCredentialsWithoutPublishingThem(t *testing.T) {
	previous := config.GlobalConfig.HTTPRules
	t.Cleanup(func() { config.GlobalConfig.HTTPRules = previous })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != "operator" || password != "probe-secret" || r.URL.Query().Get("key") != "local-token" {
			t.Error("configured credentials did not reach the check target")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	endpoint := strings.Replace(server.URL, "http://", "http://operator:probe-secret@", 1) + "/ready?key=local-token#local-fragment"
	config.GlobalConfig.HTTPRules = []config.HTTPRule{{URL: endpoint, Code: http.StatusNoContent}}
	result := RunChecksContext(context.Background())[0]
	if !result.Success || result.URL != server.URL+"/ready" || result.Error != "" {
		t.Fatal("successful check did not return a sanitized address")
	}
	server.Close()
	result = RunChecksContext(context.Background())[0]
	if result.Success || result.Error == "" || result.URL != server.URL+"/ready" {
		t.Fatal("failed check did not return an error and sanitized address")
	}
	for _, secret := range []string{"operator", "probe-secret", "local-token", "local-fragment"} {
		if strings.Contains(result.Error, secret) {
			t.Fatal("failed check disclosed sensitive URL fields")
		}
	}
}

func TestHTTPChecksObserveExpectedRedirects(t *testing.T) {
	previous := config.GlobalConfig.HTTPRules
	t.Cleanup(func() { config.GlobalConfig.HTTPRules = previous })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/ready", http.StatusFound)
	}))
	defer server.Close()
	config.GlobalConfig.HTTPRules = []config.HTTPRule{
		{URL: server.URL, Code: http.StatusFound},
		{URL: server.URL, Code: http.StatusMovedPermanently},
		{URL: server.URL},
	}
	results := RunChecksContext(context.Background())
	if len(results) != 3 || !results[0].Success || results[1].Success || !strings.Contains(results[1].Error, "302") || !results[2].Success {
		t.Fatalf("unexpected redirect check outcomes: %+v", results)
	}
}

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
