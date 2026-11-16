package config

import "time"

type DiagnosticResult struct {
	Component   string           `json:"component"`
	Status      DiagnosticStatus `json:"status"`
	Issues      []Issue          `json:"issues"`
	Suggestions []string         `json:"suggestions"`
	Timestamp   time.Time        `json:"timestamp"`
}

type DiagnosticStatus string

const (
	StatusHealthy DiagnosticStatus = "healthy"
	StatusWarning DiagnosticStatus = "warning"
	StatusError   DiagnosticStatus = "error"
)

type Issue struct {
	Type        IssueType `json:"type"`
	Description string    `json:"description"`
	Severity    Severity  `json:"severity"`
	FixCommand  string    `json:"fix_command,omitempty"`
}

type IssueType string

const IssueTypeInvalidConfig IssueType = "invalid_config"

type Severity string

const SeverityHigh Severity = "high"

// ConfigDiagnostic inspects the effective process configuration without writes
// or network probes. Platform information is reported by the deployment layer.
type ConfigDiagnostic struct{}

func NewConfigDiagnostic() *ConfigDiagnostic { return &ConfigDiagnostic{} }

func (d *ConfigDiagnostic) RunDiagnostics() []*DiagnosticResult {
	return []*DiagnosticResult{
		d.diagnoseEnvConfig(), d.diagnoseSecurityConfig(), d.diagnoseNotificationConfig(),
	}
}

func (d *ConfigDiagnostic) diagnoseEnvConfig() *DiagnosticResult {
	return diagnoseRuntimeConfig(GlobalConfig)
}

func newDiagnosticResult(component string) *DiagnosticResult {
	return &DiagnosticResult{
		Component: component, Status: StatusHealthy, Timestamp: time.Now(),
		Issues: []Issue{}, Suggestions: []string{},
	}
}

func (d *ConfigDiagnostic) diagnoseSecurityConfig() *DiagnosticResult {
	result := newDiagnosticResult("安全配置")
	if (GlobalConfig.WebUser == "") != (GlobalConfig.WebPassword == "") {
		result.Status = StatusError
		result.Issues = append(result.Issues, Issue{
			Type: IssueTypeInvalidConfig, Severity: SeverityHigh,
			Description: "WEB_USER and WEB_PASSWORD must be configured together",
		})
	} else if GlobalConfig.WebUser == "" {
		result.Status = StatusWarning
		result.Issues = append(result.Issues, Issue{
			Type: IssueTypeInvalidConfig, Severity: SeverityHigh,
			Description: "Web access is anonymous because entrypoint credentials are not configured",
		})
		result.Suggestions = append(result.Suggestions, "Configure WEB_USER and WEB_PASSWORD before allowing remote access")
	}
	return result
}

func (d *ConfigDiagnostic) diagnoseNotificationConfig() *DiagnosticResult {
	result := newDiagnosticResult("通知配置")
	if GlobalConfig.DingTalkWebhook == "" && GlobalConfig.TelegramToken == "" && GlobalConfig.TelegramChatID == "" {
		result.Suggestions = append(result.Suggestions, "Notifications are optional and currently disabled")
		return result
	}
	if GlobalConfig.DingTalkWebhook != "" && !validateDingTalkWebhook(GlobalConfig.DingTalkWebhook) {
		result.Status = StatusError
		result.Issues = append(result.Issues, Issue{
			Type: IssueTypeInvalidConfig, Severity: SeverityHigh,
			Description: "DINGTALK_WEBHOOK has an invalid URL",
		})
	}
	if (GlobalConfig.TelegramToken == "") != (GlobalConfig.TelegramChatID == "") {
		result.Status = StatusError
		result.Issues = append(result.Issues, Issue{
			Type: IssueTypeInvalidConfig, Severity: SeverityHigh,
			Description: "TELEGRAM_TOKEN and TELEGRAM_CHAT_ID must be configured together",
		})
	}
	return result
}
