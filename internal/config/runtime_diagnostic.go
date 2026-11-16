package config

import (
	"os"
	"time"
)

// Inspect the effective startup configuration, regardless of whether it came
// from process environment variables, a JSON file, or command-line flags.
func diagnoseRuntimeConfig(cfg Config) *DiagnosticResult {
	result := &DiagnosticResult{
		Component: "环境配置", Status: StatusHealthy, Timestamp: time.Now(),
		Issues: []Issue{}, Suggestions: []string{},
	}
	addError := func(description string) {
		result.Status = StatusError
		result.Issues = append(result.Issues, Issue{Type: IssueTypeInvalidConfig, Description: description, Severity: SeverityHigh})
	}
	if cfg.ApiKey == "" {
		addError("The active model endpoint requires a credential")
	}
	for _, description := range activeConfigurationErrors(cfg) {
		addError(description)
	}
	if port := os.Getenv("PORT"); port != "" && !validatePort(port) {
		addError("PORT must be a number between 1 and 65535")
	}
	return result
}
