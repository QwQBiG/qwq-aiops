package notify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"qwq/internal/config"
	"strings"
	"testing"
	"time"
)

func TestNotificationIntegration(t *testing.T) {
	oldConfig, oldService := config.GlobalConfig, globalNotificationService
	t.Cleanup(func() { config.GlobalConfig, globalNotificationService = oldConfig, oldService })
	config.GlobalConfig.DingTalkWebhook = ""
	config.GlobalConfig.TelegramToken = "test-token"
	config.GlobalConfig.TelegramChatID = "1234"
	messages := make(chan map[string]string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message map[string]string
		if r.Method != http.MethodPost {
			t.Error("notification must use POST")
		}
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			t.Error(err)
		}
		messages <- message
		io.WriteString(w, `{"ok":true,"result":{"message_id":1}}`)
	}))
	defer server.Close()
	InitNotificationService()
	if err := ValidateNotificationConfig(); err != nil {
		t.Fatal(err)
	}
	globalNotificationService.telegramService.endpoint = server.URL
	if err := SendStatusReportContext(context.Background(), "load_[1] = 0.2"); err != nil {
		t.Fatal(err)
	}
	message := <-messages
	if message["chat_id"] != "1234" || !strings.Contains(message["text"], "load_[1] = 0.2") || message["parse_mode"] != "" {
		t.Fatalf("unexpected Telegram payload: %#v", message)
	}
	config.GlobalConfig.TelegramToken, config.GlobalConfig.TelegramChatID = "", ""
	InitNotificationService()
	if err := SendContext(context.Background(), "title", "content"); err == nil {
		t.Fatal("missing channel succeeded")
	}
}

func TestNotificationCancelsInFlightRequest(t *testing.T) {
	for _, channel := range []string{"DingTalk", "Telegram"} {
		t.Run(channel, func(t *testing.T) {
			started := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				close(started)
				<-r.Context().Done()
			}))
			defer server.Close()
			service := &UnifiedNotificationService{}
			if channel == "DingTalk" {
				service.dingTalkService = NewDingTalkNotificationService(server.URL)
			} else {
				service.telegramService = NewTelegramNotificationService("token", "chat")
				service.telegramService.endpoint = server.URL
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- service.SendAlertContext(ctx, "title", "content") }()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("request did not arrive")
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation was lost: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("request ignored cancellation")
			}
		})
	}
}

func TestNotificationResponseErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		want       string
	}{
		{"business error", `{"errcode":310000,"errmsg":"keywords not in content"}`, 200, "310000"},
		{"missing code", `{}`, 200, "missing errcode"},
		{"invalid JSON", `not json`, 200, "invalid DingTalk response"},
		{"HTTP error", ``, 503, "503"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer server.Close()
			err := NewDingTalkNotificationService(server.URL).SendAlert("title", "content")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %s", err, tc.want)
			}
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ok":false,"error_code":400,"description":"chat not found"}`)
	}))
	defer server.Close()
	telegram := NewTelegramNotificationService("token", "chat")
	telegram.endpoint = server.URL
	service := &UnifiedNotificationService{telegramService: telegram}
	if err := service.SendAlert("title", "content"); err == nil || !strings.Contains(err.Error(), "chat not found") {
		t.Fatalf("API rejection was lost: %v", err)
	}
}

func TestNotificationCancellationAndPartialFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"errcode":0,"errmsg":"ok"}`)
	}))
	defer server.Close()
	service := &UnifiedNotificationService{dingTalkService: NewDingTalkNotificationService(server.URL), telegramService: NewTelegramNotificationService("", "chat")}
	if err := service.SendAlert("title", "content"); err == nil || !strings.Contains(err.Error(), "Telegram") {
		t.Fatalf("partial failure was hidden: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	err := service.SendAlertContext(ctx, "title", "content")
	if !errors.Is(err, context.Canceled) || time.Since(started) > time.Second {
		t.Fatalf("cancellation was lost: %v", err)
	}
	secretErr := &url.Error{Op: "Post", URL: "https://example.invalid/?access_token=secret", Err: context.DeadlineExceeded}
	err = notificationRequestError(secretErr)
	if strings.Contains(err.Error(), "secret") || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("request error exposed credentials or lost cause: %v", err)
	}
}
