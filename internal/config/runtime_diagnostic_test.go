package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnosticsUseEffectiveJSONConfiguration(t *testing.T) {
	old := GlobalConfig
	t.Cleanup(func() { GlobalConfig = old })
	GlobalConfig = Config{}
	for _, key := range []string{"PORT", "AI_PROVIDER", "OPENAI_API_KEY", "OPENAI_BASE_URL", "OPENAI_MODEL", "WEB_USER", "WEB_PASSWORD", "DINGTALK_WEBHOOK", "TELEGRAM_TOKEN", "TELEGRAM_CHAT_ID", "JWT_SECRET", "ENCRYPTION_KEY"} {
		t.Setenv(key, "")
	}
	configFile := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(configFile, []byte(`{"api_key":"private-test-key","base_url":"http://localhost:11434/v1","model":"installed-model","web_user":"operator","web_password":"private-test-password"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Init(configFile); err != nil {
		t.Fatal(err)
	}
	diagnostic := NewConfigDiagnostic()
	for _, result := range []*DiagnosticResult{diagnostic.diagnoseEnvConfig(), diagnostic.diagnoseSecurityConfig(), diagnostic.diagnoseNotificationConfig()} {
		if result.Status != StatusHealthy {
			t.Fatalf("valid effective configuration failed diagnostics: %+v", result)
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), "private-test") || strings.Contains(string(encoded), "JWT_SECRET") || strings.Contains(string(encoded), ".env") {
			t.Fatalf("diagnostics exposed secrets or required an unused setting: %s", encoded)
		}
	}
}

func TestRuntimeDiagnosticsRejectInvalidActiveConfiguration(t *testing.T) {
	t.Setenv("PORT", "8080")
	for _, cfg := range []Config{{}, {ApiKey: "key", BaseURL: "not-a-url"}, {ApiKey: "key", BaseURL: "file:///tmp/model"}} {
		if result := diagnoseRuntimeConfig(cfg); result.Status != StatusError || len(result.Issues) == 0 {
			t.Fatalf("invalid configuration accepted: %+v", result)
		}
	}
	t.Setenv("PORT", "70000")
	if result := diagnoseRuntimeConfig(Config{ApiKey: "key"}); result.Status != StatusError {
		t.Fatal("invalid port accepted")
	}
}

func TestRuntimePortRejectsTrailingCharacters(t *testing.T) {
	for _, port := range []string{"8080suffix", "8080 123", "+8080", "8080\n"} {
		t.Setenv("PORT", port)
		if result := diagnoseRuntimeConfig(Config{ApiKey: "key"}); result.Status != StatusError {
			t.Errorf("invalid listener port was accepted: %q", port)
		}
	}
}

func TestTelegramOnlyConfigurationIsNotDisabled(t *testing.T) {
	old := GlobalConfig
	t.Cleanup(func() { GlobalConfig = old })
	GlobalConfig = Config{TelegramToken: "test-token", TelegramChatID: "1234"}
	result := NewConfigDiagnostic().diagnoseNotificationConfig()
	if result.Status != StatusHealthy || strings.Contains(strings.Join(result.Suggestions, " "), "disabled") {
		t.Fatalf("configured Telegram was marked disabled: %+v", result)
	}
	GlobalConfig.TelegramChatID = ""
	if result := NewConfigDiagnostic().diagnoseNotificationConfig(); result.Status != StatusError {
		t.Fatalf("incomplete Telegram credentials passed diagnostics: %+v", result)
	}
}
