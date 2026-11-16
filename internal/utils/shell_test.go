package utils

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestAutomaticCommandsStayReadOnly(t *testing.T) {
	for _, command := range []string{"ps aux", "free -m", "df -h", "docker ps -a", "docker logs service", "kubectl get pods", "systemctl status nginx", "ip route show", "hostname -I", "ss -tan", "date -u", "grep error /var/log/example.log"} {
		if !IsReadOnlyCommand(command) {
			t.Errorf("diagnostic rejected: %q", command)
		}
	}
	for _, command := range []string{
		"", "pwd; touch marker", "echo $(touch marker)", "ls `touch marker`", "pwd\ntouch marker", "ls && touch marker", "ls | sh", "cat <(touch marker)", "cat file > other", "ls\\;touch marker",
		"docker rm useful-container", "docker exec service sh", "kubectl delete pod service", "systemctl restart nginx", "find /tmp -delete", "find /tmp -exec touch marker", "awk 'BEGIN {system(x)}'", "hostname changed", "date --set=tomorrow", "ss -K", "ss -tanK", "ip link set eth0 down", "lsanything", "sudo ls", "curl https://example.com", "cat file\x00ignored",
		"kubectl cluster-info dump --output-directory=/tmp/cluster", "kubectl cluster-info 'dump'", `kubectl cluster-info d"ump"`,
	} {
		if IsReadOnlyCommand(command) {
			t.Errorf("automatic execution accepted: %q", command)
		}
	}
}

func TestCommandOutputIsBoundedUTF8(t *testing.T) {
	output := &CommandOutput{}
	data := strings.Repeat("x", 3999) + "中文" + strings.Repeat("y", 10000)
	if n, err := output.Write([]byte(data)); n != len(data) || err != nil {
		t.Fatalf("pipe not drained: n=%d err=%v", n, err)
	}
	got := output.String()
	if len(output.data) != 4000 || !utf8.ValidString(got) || !strings.Contains(got, "Output truncated") {
		t.Fatalf("invalid bounded output: len=%d utf8=%v", len(output.data), utf8.ValidString(got))
	}
}

func TestCanceledShellReturnsPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	got := ExecuteShellContext(ctx, "printf unused")
	if !strings.Contains(got, "Command canceled") || time.Since(started) > 2*time.Second {
		t.Fatalf("cancellation not propagated: %q", got)
	}
}

func TestShellResultAndFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Bash system commands are verified on Linux")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	if got := ExecuteShell("printf normal-result"); got != "normal-result" {
		t.Fatalf("normal command: %q", got)
	}
	if got := ExecuteShell("printf failure-result; exit 7"); !strings.Contains(got, "failure-result") || !strings.Contains(got, "Command failed") {
		t.Fatalf("exit status lost: %q", got)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	if got := ExecuteShellContext(ctx, "sleep 5"); !strings.Contains(got, "Command timed out") || time.Since(started) > 2*time.Second {
		t.Fatalf("deadline not propagated: %q", got)
	}
}

func TestStructuredShellPreservesPipelineFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Bash system commands are verified on Linux")
	}
	output, err := ExecuteShellResultContext(context.Background(), "printf partial; false | cat")
	if output != "partial" || err == nil {
		t.Fatalf("pipeline failure was lost: output=%q err=%v", output, err)
	}
	output, err = ExecuteShellResultContext(context.Background(), "printf normal-result")
	if output != "normal-result" || err != nil {
		t.Fatalf("successful structured command failed: output=%q err=%v", output, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = ExecuteShellResultContext(ctx, "printf unused")
	if err != context.Canceled {
		t.Fatalf("structured cancellation was lost: %v", err)
	}
}
