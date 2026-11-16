package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetRuntimeConfig(t *testing.T) {
	t.Helper()
	previous, knowledge := GlobalConfig, CachedKnowledge
	t.Cleanup(func() { GlobalConfig, CachedKnowledge = previous, knowledge })
	GlobalConfig, CachedKnowledge = Config{}, ""
	for _, key := range []string{
		"OPENAI_API_KEY", "OPENAI_BASE_URL", "OPENAI_MODEL", "AI_PROVIDER",
		"OLLAMA_HOST", "OLLAMA_MODEL", "WEB_USER", "WEB_PASSWORD", "DINGTALK_WEBHOOK",
		"TELEGRAM_TOKEN", "TELEGRAM_CHAT_ID",
	} {
		t.Setenv(key, "")
	}
}

func TestInitAppliesExplicitOverridesBeforeValidation(t *testing.T) {
	resetRuntimeConfig(t)
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("WEB_USER", "environment-user")
	if err := Init("", func(cfg *Config) {
		cfg.WebUser = "flag-user"
		cfg.WebPassword = "flag-password"
	}); err != nil {
		t.Fatal(err)
	}
	if GlobalConfig.WebUser != "flag-user" || GlobalConfig.WebPassword != "flag-password" {
		t.Fatal("explicit credentials did not override environment")
	}
}

func TestInitTelegramEnvironment(t *testing.T) {
	resetRuntimeConfig(t)
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("TELEGRAM_TOKEN", "test-token")
	if err := Init(""); err == nil || !strings.Contains(err.Error(), "configured together") {
		t.Fatalf("incomplete Telegram credentials accepted: %v", err)
	}
	t.Setenv("TELEGRAM_CHAT_ID", "test-chat")
	if err := Init(""); err != nil {
		t.Fatal(err)
	}
	if GlobalConfig.TelegramToken != "test-token" || GlobalConfig.TelegramChatID != "test-chat" {
		t.Fatal("Telegram environment was not applied")
	}
}

func TestInitEnvironmentOverridesFile(t *testing.T) {
	resetRuntimeConfig(t)
	path := filepath.Join(t.TempDir(), "config.json")
	data := `{"api_key":"file-key","base_url":"http://file/v1","model":"file-model","web_user":"file-user","web_password":"file-password"}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "env-key")
	t.Setenv("OPENAI_BASE_URL", "http://provider/v1")
	t.Setenv("OPENAI_MODEL", "env-model")
	t.Setenv("WEB_USER", "operator")
	t.Setenv("WEB_PASSWORD", "secret")
	t.Setenv("DINGTALK_WEBHOOK", "https://oapi.dingtalk.com/robot/send?access_token=env-token")
	if err := Init(path); err != nil {
		t.Fatal(err)
	}
	if GlobalConfig.ApiKey != "env-key" || GlobalConfig.BaseURL != "http://provider/v1" || GlobalConfig.Model != "env-model" {
		t.Fatal("provider environment did not override the file")
	}
	if GlobalConfig.WebUser != "operator" || GlobalConfig.WebPassword != "secret" {
		t.Fatal("web credentials were not loaded from the environment")
	}
	if GlobalConfig.DingTalkWebhook != "https://oapi.dingtalk.com/robot/send?access_token=env-token" {
		t.Fatal("notification webhook was not loaded from the environment")
	}
}

func TestInitOllama(t *testing.T) {
	for _, custom := range []bool{false, true} {
		t.Run(map[bool]string{false: "defaults", true: "custom"}[custom], func(t *testing.T) {
			resetRuntimeConfig(t)
			t.Setenv("AI_PROVIDER", "ollama")
			baseURL, model := "http://localhost:11434/v1", "qwen2.5:7b"
			if custom {
				t.Setenv("OLLAMA_HOST", "http://model-server:11434/")
				t.Setenv("OLLAMA_MODEL", "local-model")
				baseURL, model = "http://model-server:11434/v1", "local-model"
			}
			if err := Init(""); err != nil {
				t.Fatal(err)
			}
			if GlobalConfig.BaseURL != baseURL || GlobalConfig.Model != model || GlobalConfig.ApiKey == "" {
				t.Fatal("local provider configuration is incompatible with the client")
			}
		})
	}
}

func TestInitEmptyEnvironmentPreservesFileCredentials(t *testing.T) {
	resetRuntimeConfig(t)
	path := filepath.Join(t.TempDir(), "config.json")
	data := `{"api_key":"file-key","web_user":"operator","web_password":"secret"}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Init(path); err != nil {
		t.Fatal(err)
	}
	if GlobalConfig.WebUser != "operator" || GlobalConfig.WebPassword != "secret" {
		t.Fatal("empty environment variables must not disable configured authentication")
	}
}

func TestInitRejectsIncompleteCredentials(t *testing.T) {
	for _, key := range []string{"WEB_USER", "WEB_PASSWORD"} {
		t.Run(key, func(t *testing.T) {
			resetRuntimeConfig(t)
			t.Setenv("OPENAI_API_KEY", "test-key")
			t.Setenv(key, "configured")
			if err := Init(""); err == nil || !strings.Contains(err.Error(), "must be configured together") {
				t.Fatalf("expected incomplete credentials error, got %v", err)
			}
		})
	}
}

func TestInitAllowsUnconfiguredWebCredentials(t *testing.T) {
	resetRuntimeConfig(t)
	t.Setenv("OPENAI_API_KEY", "test-key")
	if err := Init(""); err != nil {
		t.Fatal(err)
	}
}

func TestInitRejectsMissingKey(t *testing.T) {
	resetRuntimeConfig(t)
	if err := Init(""); err == nil {
		t.Fatal("expected a missing key error")
	}
}

func TestInitRejectsUnknownProvider(t *testing.T) {
	resetRuntimeConfig(t)
	t.Setenv("AI_PROVIDER", "unknown")
	if err := Init(""); err == nil || !strings.Contains(err.Error(), "unsupported provider") {
		t.Fatalf("expected unsupported provider error, got %v", err)
	}
}
