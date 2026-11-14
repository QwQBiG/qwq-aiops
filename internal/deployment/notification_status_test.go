package deployment

import (
	"qwq/internal/config"
	"testing"
)

func TestMissingOptionalNotificationDoesNotFailDiagnostics(t *testing.T) {
	old := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = old })
	config.GlobalConfig.DingTalkWebhook = ""
	// A nil notification service also proves no connection test is attempted.
	service := &IntegrationService{}
	if result := service.validateNotificationService(); result.Status != "healthy" || len(result.Issues) != 0 {
		t.Fatalf("optional notification was treated as a broken dependency: %+v", result)
	}
}
