package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
)

var manualTriggerRunning atomic.Bool

// A manual batch owns both callbacks until they finish. Their service context
// comes from startup; returning the accepted response must not cancel the work.
func handleTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	patrol, status := TriggerPatrolFunc, TriggerStatusFunc
	if patrol == nil || status == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Patrol and status callbacks are unavailable"})
		return
	}
	if !manualTriggerRunning.CompareAndSwap(false, true) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "A manual patrol batch is already running; check the logs for results"})
		return
	}
	go func() {
		defer manualTriggerRunning.Store(false)
		var work sync.WaitGroup
		work.Add(1)
		go func() { defer work.Done(); patrol() }()
		status()
		work.Wait()
	}()
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "accepted", "message": "Patrol and status requests accepted; check the logs for results",
	})
}
