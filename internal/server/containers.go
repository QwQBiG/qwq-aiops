package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"qwq/internal/utils"
)

func handleContainers(w http.ResponseWriter, r *http.Request) {
	serveContainerList(w, r, func(ctx context.Context) ([]byte, error) {
		return exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{json .}}").CombinedOutput()
	})
}

func serveContainerList(w http.ResponseWriter, r *http.Request, query func(context.Context) ([]byte, error)) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		jsonResponse(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), utils.CommandTimeout)
	defer cancel()
	output, err := query(ctx)
	if err != nil {
		jsonResponse(w, http.StatusBadGateway, fmt.Sprintf("Docker query failed: %v", err), nil)
		return
	}
	containers, err := parseContainerList(output)
	if err != nil {
		jsonResponse(w, http.StatusBadGateway, "Docker returned invalid container data", nil)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(containers)
}

func parseContainerList(output []byte) ([]DockerContainer, error) {
	containers := make([]DockerContainer, 0)
	decoder := json.NewDecoder(bytes.NewReader(output))
	for {
		var item struct{ ID, Image, Status, Names, State string }
		if err := decoder.Decode(&item); err == io.EOF {
			return containers, nil
		} else if err != nil {
			return nil, err
		}
		if item.ID == "" {
			return nil, fmt.Errorf("container identifier is missing")
		}
		containers = append(containers, DockerContainer{ID: item.ID, Image: item.Image, Status: item.Status, Name: item.Names, State: item.State})
	}
}
