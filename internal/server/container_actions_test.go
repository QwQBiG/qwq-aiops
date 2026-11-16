package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestContainerActionCommandUsesArguments(t *testing.T) {
	id := "example; echo unexpected"
	command, err := containerActionCommand(context.Background(), "restart", id)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(command.Args, []string{"docker", "restart", id}) {
		t.Fatalf("unexpected command arguments: %q", command.Args)
	}
}

func TestContainerActionRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name, method, query string
		status              int
	}{
		{"GET cannot mutate", http.MethodGet, "?action=stop&id=example", http.StatusMethodNotAllowed},
		{"missing action", http.MethodPost, "?id=example", http.StatusBadRequest},
		{"unsupported action", http.MethodPost, "?action=remove&id=example", http.StatusBadRequest},
		{"missing ID", http.MethodPost, "?action=start", http.StatusBadRequest},
		{"empty ID", http.MethodPost, "?action=start&id=%20", http.StatusBadRequest},
		{"option as ID", http.MethodPost, "?action=start&id=--help", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handleContainerAction(response, httptest.NewRequest(tt.method, "/api/container/action"+tt.query, nil))
			if response.Code != tt.status {
				t.Fatalf("unexpected status: %d %s", response.Code, response.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] == "" {
				t.Fatalf("missing JSON error: %s", response.Body.String())
			}
		})
	}
}

func TestContainerActionReportsExecutionError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	response := httptest.NewRecorder()
	handleContainerAction(response, httptest.NewRequest(http.MethodPost, "/api/container/action?action=start&id=example", nil))
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusBadGateway || body["error"] == "" || body["status"] == "success" {
		t.Fatalf("execution error reported as success: %d %s", response.Code, response.Body.String())
	}
}

func TestTriggerRequiresPost(t *testing.T) {
	patrol, status := TriggerPatrolFunc, TriggerStatusFunc
	t.Cleanup(func() {
		waitTriggerIdle(t)
		TriggerPatrolFunc, TriggerStatusFunc = patrol, status
	})
	called := make(chan string, 2)
	TriggerPatrolFunc = func() { called <- "patrol" }
	TriggerStatusFunc = func() { called <- "status" }
	response := httptest.NewRecorder()
	handleTrigger(response, httptest.NewRequest(http.MethodGet, "/api/trigger", nil))
	if response.Code != http.StatusMethodNotAllowed || len(called) != 0 {
		t.Fatal("GET must not trigger background work")
	}
	response = httptest.NewRecorder()
	handleTrigger(response, httptest.NewRequest(http.MethodPost, "/api/trigger", nil))
	if response.Code != http.StatusAccepted {
		t.Fatalf("POST returned %d", response.Code)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-called:
		case <-time.After(time.Second):
			t.Fatal("POST did not trigger both callbacks")
		}
	}
}
