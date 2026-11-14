package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"qwq/internal/logger"
	"qwq/internal/utils"
	"strings"
)

func containerActionCommand(ctx context.Context, action, id string) (*exec.Cmd, error) {
	if action != "start" && action != "stop" && action != "restart" {
		return nil, fmt.Errorf("invalid container action")
	}
	if strings.TrimSpace(id) == "" || strings.HasPrefix(id, "-") {
		return nil, fmt.Errorf("invalid container ID or name")
	}
	return exec.CommandContext(ctx, "docker", action, id), nil
}

func handleContainerAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}
	action, id := r.URL.Query().Get("action"), r.URL.Query().Get("id")
	ctx, cancel := context.WithTimeout(r.Context(), utils.CommandTimeout)
	defer cancel()
	command, err := containerActionCommand(ctx, action, id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	logger.Info("Container action %s for %s", action, id)
	output, err := command.CombinedOutput()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Docker action failed: %v", err), "output": string(output)})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
