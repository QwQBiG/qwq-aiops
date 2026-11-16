package executor

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"qwq/internal/agent"
	"runtime"
	"strings"
	"testing"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

func requireBash(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Bash execution is checked on Linux")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
}

func TestSmartRunFailureExecutesOnce(t *testing.T) {
	requireBash(t)
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"Check the command arguments."}}]}`)
	}))
	defer model.Close()
	previous := agent.Client
	t.Cleanup(func() { agent.Client = previous })
	cfg := openai.DefaultConfig("test")
	cfg.BaseURL = model.URL
	agent.Client = openai.NewClientWithConfig(cfg)
	marker := filepath.Join(t.TempDir(), "attempts")
	t.Setenv("QWQ_TEST_MARKER", marker)
	err := SmartRunContext(context.Background(), `printf attempt >> "$QWQ_TEST_MARKER"; printf failure >&2; exit 7`)
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) || exitError.ExitCode() != 7 {
		t.Fatalf("failure was not returned: %v", err)
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "attempt" {
		t.Fatalf("failed command executed more than once: %q, %v", data, err)
	}
}

func TestRunCommandStreamsAndBoundsDiagnostics(t *testing.T) {
	requireBash(t)
	var stdout, stderr strings.Builder
	output, err := runCommand(context.Background(), "printf normal; printf error >&2", &stdout, &stderr)
	if err != nil || stdout.String() != "normal" || stderr.String() != "error" || !strings.Contains(output, "normal") || !strings.Contains(output, "error") {
		t.Fatalf("streams lost: stdout=%q stderr=%q captured=%q err=%v", stdout.String(), stderr.String(), output, err)
	}
	output, err = runCommand(context.Background(), "printf '%5000s' ''", io.Discard, io.Discard)
	if err != nil || len(output) > 4050 || !strings.Contains(output, "Output truncated") {
		t.Fatalf("diagnostic output is not bounded: len=%d err=%v", len(output), err)
	}
}

func TestSmartRunCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	if err := SmartRunContext(ctx, "sleep 30"); !errors.Is(err, context.Canceled) || time.Since(started) > time.Second {
		t.Fatalf("cancellation was not returned promptly: %v", err)
	}
}
