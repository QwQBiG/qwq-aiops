package deployment

import (
	"qwq/internal/config"
	"qwq/internal/platform"
	"testing"
)

type workflowFrontend struct{}

func (workflowFrontend) ValidateResources() interface{} { return nil }
func (workflowFrontend) GetResourceStats() interface{}  { return nil }

func TestWorkflowUsesAvailableManualSteps(t *testing.T) {
	old := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = old })
	config.GlobalConfig.ApiKey = ""
	config.GlobalConfig.DingTalkWebhook = ""
	config.GlobalConfig.TelegramToken = ""
	config.GlobalConfig.TelegramChatID = ""
	service := &IntegrationService{
		configDiagnostic: config.NewConfigDiagnostic(),
		platformAdapter:  platform.NewPlatformAdapter(),
		frontendManager:  workflowFrontend{},
	}
	workflow := service.GetDeploymentWorkflow()
	steps, ok := workflow["repair_steps"].([]map[string]interface{})
	if !ok || len(steps) == 0 {
		t.Fatalf("invalid configuration has no repair guidance: %#v", workflow)
	}
	for _, step := range steps {
		if step["automated"] != false || step["command"] != "manual" {
			t.Fatalf("workflow advertised an unavailable automatic operation: %#v", step)
		}
	}
}
