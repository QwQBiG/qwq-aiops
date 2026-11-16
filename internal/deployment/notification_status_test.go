package deployment

import (
	"qwq/internal/config"
	"qwq/internal/notify"
	"testing"
)

func TestMissingOptionalNotificationDoesNotFailDiagnostics(t *testing.T) {
	old := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = old })
	config.GlobalConfig.DingTalkWebhook = ""
	config.GlobalConfig.TelegramToken = ""
	config.GlobalConfig.TelegramChatID = ""
	// A nil notification service also proves no connection test is attempted.
	service := &IntegrationService{}
	if result := service.validateNotificationService(); result.Status != "healthy" || len(result.Issues) != 0 {
		t.Fatalf("optional notification was treated as a broken dependency: %+v", result)
	}
}

func TestTelegramNotificationDiagnostics(t *testing.T) {
	old := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = old })
	config.GlobalConfig.DingTalkWebhook = ""
	config.GlobalConfig.TelegramToken = "test-token"
	config.GlobalConfig.TelegramChatID = ""
	service := &IntegrationService{notifyService: notify.NewUnifiedNotificationService()}
	if result := service.validateNotificationService(); result.Status != "warning" {
		t.Fatalf("incomplete Telegram configuration was not checked: %+v", result)
	}
	config.GlobalConfig.TelegramChatID = "1234"
	service.notifyService = notify.NewUnifiedNotificationService()
	if result := service.validateNotificationService(); result.Status != "healthy" {
		t.Fatalf("valid Telegram configuration was rejected: %+v", result)
	}
}
