package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"qwq/internal/agent"
	"qwq/internal/config"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	openai "github.com/sashabaranov/go-openai"
)

func TestChatCancellationReachesModel(t *testing.T) {
	for _, mode := range []string{"disconnect", "shutdown"} {
		t.Run(mode, func(t *testing.T) {
			started, cancelled := make(chan struct{}), make(chan struct{})
			model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				close(started)
				<-r.Context().Done()
				close(cancelled)
			}))
			defer model.Close()
			previousClient, previousConfig := agent.Client, config.GlobalConfig
			defer func() { agent.Client = previousClient; config.GlobalConfig = previousConfig }()
			config.GlobalConfig.WebUser, config.GlobalConfig.WebPassword = "", ""
			clientConfig := openai.DefaultConfig("local-test")
			clientConfig.BaseURL = model.URL + "/v1"
			agent.Client = openai.NewClientWithConfig(clientConfig)
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			stopped := make(chan error, 1)
			chatFinished := make(chan struct{})
			handler := NewHandler()
			go func() {
				stopped <- serve(ctx, listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					defer close(chatFinished)
					handler.ServeHTTP(w, r)
				}))
			}()
			connection, _, err := websocket.DefaultDialer.Dial("ws://"+listener.Addr().String()+"/ws/chat", nil)
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close()
			if err := connection.WriteMessage(websocket.TextMessage, []byte("Write a short story about a balloon")); err != nil {
				t.Fatal(err)
			}
			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatal("model request did not start")
			}
			if mode == "disconnect" {
				connection.Close()
			} else {
				cancel()
			}
			select {
			case <-cancelled:
			case <-time.After(2 * time.Second):
				t.Fatal("model request survived client/server cancellation")
			}
			select {
			case <-chatFinished:
			case <-time.After(2 * time.Second):
				t.Fatal("chat handler did not finish")
			}
			cancel()
			select {
			case err := <-stopped:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("server did not stop")
			}
		})
	}
}

func TestChatReportsCompletionAfterOneAnswer(t *testing.T) {
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"One answer"}}]}`)
	}))
	defer model.Close()
	previousClient, previousConfig := agent.Client, config.GlobalConfig
	defer func() { agent.Client = previousClient; config.GlobalConfig = previousConfig }()
	config.GlobalConfig.WebUser, config.GlobalConfig.WebPassword = "", ""
	clientConfig := openai.DefaultConfig("local-test")
	clientConfig.BaseURL = model.URL + "/v1"
	agent.Client = openai.NewClientWithConfig(clientConfig)
	chatFinished := make(chan struct{})
	handler := NewHandler()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(chatFinished)
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()
	connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws/chat", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		connection.Close()
		select {
		case <-chatFinished:
		case <-time.After(2 * time.Second):
			t.Error("chat handler did not finish")
		}
	}()
	connection.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := connection.WriteMessage(websocket.TextMessage, []byte("Write a short story")); err != nil {
		t.Fatal(err)
	}
	answers := 0
	for {
		var message map[string]string
		if err := connection.ReadJSON(&message); err != nil {
			t.Fatal(err)
		}
		if message["type"] == "answer" {
			answers++
		}
		if message["type"] == "status" && message["content"] == "done" {
			break
		}
	}
	if answers != 1 {
		t.Fatalf("one input received %d answers", answers)
	}
}
