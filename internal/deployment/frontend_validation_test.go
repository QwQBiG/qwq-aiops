package deployment

import (
	"qwq/internal/config"
	"strings"
	"testing"
)

func TestFrontendDiagnosticsRejectMissingValidationResult(t *testing.T) {
	service := &IntegrationService{frontendManager: workflowFrontend{}}
	if result := service.validateFrontendResources(); result.Status != "error" || len(result.Issues) == 0 {
		t.Fatalf("missing resource validation result was silently accepted: %+v", result)
	}
}

func TestConfigurationWarningsRemainVisible(t *testing.T) {
	previous := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = previous })
	config.GlobalConfig = config.Config{ApiKey: "test-key"}
	service := &IntegrationService{configDiagnostic: config.NewConfigDiagnostic()}
	result := service.validateConfiguration()
	if result.Status != "warning" || !strings.Contains(strings.Join(result.Issues, " "), "anonymous") {
		t.Fatalf("warning status lost its actionable issue: %+v", result)
	}
}
