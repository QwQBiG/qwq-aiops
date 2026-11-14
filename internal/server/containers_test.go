package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContainerListReportsQueryFailuresAndEmptyResults(t *testing.T) {
	for _, tc := range []struct {
		name, output  string
		err           error
		status, count int
	}{
		{"empty", "", nil, http.StatusOK, 0},
		{"containers", "{\"ID\":\"123\",\"Image\":\"nginx\",\"Names\":\"web\",\"Status\":\"Up 2 hours\",\"State\":\"running\"}\n", nil, http.StatusOK, 1},
		{"unavailable daemon", "", errors.New("daemon is unavailable"), http.StatusBadGateway, 0},
		{"invalid daemon output", "not JSON", nil, http.StatusBadGateway, 0},
		{"missing identifier", "{}", nil, http.StatusBadGateway, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			serveContainerList(response, httptest.NewRequest(http.MethodGet, "/api/containers", nil), func(ctx context.Context) ([]byte, error) {
				if _, ok := ctx.Deadline(); !ok {
					t.Error("Docker request has no timeout")
				}
				return []byte(tc.output), tc.err
			})
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if tc.status == http.StatusOK {
				var rows []DockerContainer
				if err := json.Unmarshal(response.Body.Bytes(), &rows); err != nil {
					t.Fatal(err)
				}
				if rows == nil || len(rows) != tc.count {
					t.Fatalf("unexpected containers: %#v", rows)
				}
				if tc.count > 0 && (rows[0].Name != "web" || rows[0].State != "running") {
					t.Fatalf("container fields were lost: %#v", rows[0])
				}
			}
		})
	}
}

func TestContainerListRejectsWriteMethods(t *testing.T) {
	response := httptest.NewRecorder()
	serveContainerList(response, httptest.NewRequest(http.MethodPost, "/api/containers", nil), func(context.Context) ([]byte, error) {
		t.Fatal("write request executed Docker")
		return nil, nil
	})
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", response.Code)
	}
}
