package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

func useModelServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	previous := Client
	cfg := openai.DefaultConfig("local-test")
	cfg.BaseURL = server.URL + "/v1"
	Client = openai.NewClientWithConfig(cfg)
	t.Cleanup(func() { Client = previous; server.Close() })
}

func TestPlainReplyCompletesOneModelTurn(t *testing.T) {
	var calls atomic.Int32
	useModelServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var request struct {
			Messages []openai.ChatCompletionMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		for _, message := range request.Messages {
			if len(message.ToolCalls) != 0 {
				t.Error("conversation seed contains unanswered tool calls")
			}
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"One response."},"finish_reason":"stop"}]}`)
	})
	messages := GetBaseMessages()
	messages = append(messages, openai.ChatCompletionMessage{Role: "user", Content: "Write a short story"})
	answers := 0
	for step := 0; step < 5; step++ {
		response, more := ProcessAgentStepForWebContext(context.Background(), &messages, func(log string) { t.Log(log) })
		if response.Content != "" {
			answers++
		}
		if !more {
			break
		}
	}
	if calls.Load() != 1 || answers != 1 {
		t.Fatalf("one input produced %d model requests and %d answers", calls.Load(), answers)
	}
}

func TestEmptyChoicesReportError(t *testing.T) {
	useModelServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[]}`)
	})
	messages := GetBaseMessages()
	var logs []string
	response, more := ProcessAgentStepForWebContext(context.Background(), &messages, func(log string) { logs = append(logs, log) })
	if more || !strings.Contains(response.Content, "no choices") || len(logs) != 0 {
		t.Fatalf("unexpected empty response handling: %#v %v %v", response, more, logs)
	}
	if result := AnalyzeWithAIContext(context.Background(), "test"); !strings.Contains(result, "no choices") {
		t.Fatalf("analysis did not report empty choices: %s", result)
	}
}

func TestModelFailureIsReturnedOnce(t *testing.T) {
	useModelServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, `{"error":{"message":"temporarily unavailable","type":"server_error"}}`)
	})
	messages := GetBaseMessages()
	originalCount := len(messages)
	var logs []string
	response, more := ProcessAgentStepForWebContext(context.Background(), &messages, func(log string) { logs = append(logs, log) })
	if more || response.Role != openai.ChatMessageRoleAssistant || !strings.Contains(response.Content, "temporarily unavailable") || len(logs) != 0 {
		t.Fatalf("model error was lost or duplicated: response=%#v more=%v logs=%v", response, more, logs)
	}
	if len(messages) != originalCount {
		t.Fatal("transport error was added to model conversation history")
	}
}

func TestModelRequestHonorsCancellation(t *testing.T) {
	started, cancelled := make(chan struct{}), make(chan struct{})
	useModelServer(t, func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		close(started)
		<-r.Context().Done()
		close(cancelled)
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		messages := GetBaseMessages()
		ProcessAgentStepForWebContext(ctx, &messages, func(string) {})
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("model request did not start")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("model call ignored cancellation")
	}
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("remote request remained active")
	}
}

func TestInvalidToolsPreserveResponsePairing(t *testing.T) {
	for _, tc := range []struct{ name, tool, arguments string }{
		{"unknown", "other_tool", `{}`},
		{"invalid JSON", "execute_shell_command", `{`},
		{"missing command", "execute_shell_command", `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var messages []openai.ChatCompletionMessage
			handleToolCallContext(context.Background(), openai.ToolCall{ID: "tool-1", Function: openai.FunctionCall{Name: tc.tool, Arguments: tc.arguments}}, &messages, func(string) {})
			if len(messages) != 1 || messages[0].Role != openai.ChatMessageRoleTool || messages[0].ToolCallID != "tool-1" {
				t.Fatalf("missing paired tool error: %#v", messages)
			}
		})
	}
}

func TestGenerationRequestsReachTheModel(t *testing.T) {
	for _, input := range []string{
		"写一个 CPU 监控脚本", "帮我写个磁盘检查脚本", "生成 Docker 镜像配置", "生成网站介绍页面",
		"Write a Docker image configuration", "Please generate a memory monitoring script",
	} {
		if response := CheckStaticResponse(input); response != "" {
			t.Errorf("generation request received a static answer: %q", input)
		}
		if command := GetQuickCommand(input); command != "" {
			t.Errorf("generation request executed %q instead of reaching the model: %q", command, input)
		}
	}
	for input, expected := range map[string]string{"看看内存": "free -h", "看 Docker 容器状态": "docker ps -a", "查磁盘": "df -hT | grep -v tmpfs"} {
		if command := GetQuickCommand(input); command != expected {
			t.Errorf("diagnostic %q changed: got %q, want %q", input, command, expected)
		}
	}
	if CheckStaticResponse("介绍") == "" {
		t.Error("explicit product introduction should remain available")
	}
}
