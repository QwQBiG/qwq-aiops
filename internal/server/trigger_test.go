package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func waitTriggerIdle(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for manualTriggerRunning.Load() {
		if time.Now().After(deadline) {
			t.Fatal("manual batch did not release its running state")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestTriggerRejectsMissingCallbacks(t *testing.T) {
	patrol, status := TriggerPatrolFunc, TriggerStatusFunc
	t.Cleanup(func() {
		waitTriggerIdle(t)
		TriggerPatrolFunc, TriggerStatusFunc = patrol, status
	})
	for _, callbacks := range [][2]func(){{nil, nil}, {func() {}, nil}, {nil, func() {}}} {
		TriggerPatrolFunc, TriggerStatusFunc = callbacks[0], callbacks[1]
		response := httptest.NewRecorder()
		handleTrigger(response, httptest.NewRequest(http.MethodPost, "/api/trigger", nil))
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("missing callbacks returned %d, want 503", response.Code)
		}
		var body map[string]string
		if json.Unmarshal(response.Body.Bytes(), &body) != nil || body["error"] == "" {
			t.Fatalf("unavailable callbacks did not return an actionable JSON error: %s", response.Body.String())
		}
	}
}

func TestTriggerWaitsForBothCallbacksAndAcceptsTheNextBatch(t *testing.T) {
	patrol, status := TriggerPatrolFunc, TriggerStatusFunc
	started, finished := make(chan struct{}, 4), make(chan struct{}, 4)
	patrolRelease, statusRelease := make(chan struct{}), make(chan struct{})
	releasePatrol := sync.OnceFunc(func() { close(patrolRelease) })
	releaseStatus := sync.OnceFunc(func() { close(statusRelease) })
	callback := func(release <-chan struct{}) func() {
		return func() { started <- struct{}{}; <-release; finished <- struct{}{} }
	}
	TriggerPatrolFunc, TriggerStatusFunc = callback(patrolRelease), callback(statusRelease)
	t.Cleanup(func() {
		releasePatrol()
		releaseStatus()
		waitTriggerIdle(t)
		TriggerPatrolFunc, TriggerStatusFunc = patrol, status
	})
	post := func() *httptest.ResponseRecorder {
		requestCtx, cancel := context.WithCancel(context.Background())
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/trigger", nil).WithContext(requestCtx)
		handleTrigger(response, request)
		cancel() // The accepted HTTP exchange has ended; its batch remains active.
		return response
	}
	first := post()
	var body map[string]string
	if first.Code != http.StatusAccepted || json.Unmarshal(first.Body.Bytes(), &body) != nil || body["status"] != "accepted" || body["message"] == "" {
		t.Fatalf("accepted request did not describe its asynchronous result: %d %s", first.Code, first.Body.String())
	}
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("accepted callbacks did not start")
		}
	}
	releasePatrol()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("first callback did not finish")
	}
	busy := post()
	if busy.Code != http.StatusConflict || json.Unmarshal(busy.Body.Bytes(), &body) != nil || body["error"] == "" {
		t.Fatalf("finishing one callback released the batch: %d %s", busy.Code, busy.Body.String())
	}
	releaseStatus()
	waitTriggerIdle(t)
	if next := post(); next.Code != http.StatusAccepted {
		t.Fatalf("finished batch prevented a subsequent request: %d %s", next.Code, next.Body.String())
	}
}

func TestTriggerRejectsOverlappingBatches(t *testing.T) {
	patrol, status := TriggerPatrolFunc, TriggerStatusFunc
	started, finished, release := make(chan struct{}, 4), make(chan struct{}, 4), make(chan struct{})
	callback := func() { started <- struct{}{}; <-release; finished <- struct{}{} }
	TriggerPatrolFunc, TriggerStatusFunc = callback, callback
	t.Cleanup(func() {
		waitTriggerIdle(t)
		TriggerPatrolFunc, TriggerStatusFunc = patrol, status
	})
	post := func() int {
		response := httptest.NewRecorder()
		handleTrigger(response, httptest.NewRequest(http.MethodPost, "/api/trigger", nil))
		return response.Code
	}
	first := post()
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			t.Fatal("accepted batch did not start both callbacks")
		}
	}
	second := post()
	close(release)
	wantFinished := 2
	if second == http.StatusOK || second == http.StatusAccepted {
		wantFinished = 4
	}
	for i := 0; i < wantFinished; i++ {
		select {
		case <-finished:
		case <-time.After(time.Second):
			t.Fatal("background callbacks did not finish")
		}
	}
	if first != http.StatusAccepted || second != http.StatusConflict {
		t.Fatalf("overlapping requests returned %d then %d, want 202 then 409", first, second)
	}
}
