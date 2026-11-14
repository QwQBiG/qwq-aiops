package server

import (
	"encoding/json"
	"net/http"
)

// handleLiveness answers container probes without disclosing deployment details.
func handleLiveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodHead {
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
