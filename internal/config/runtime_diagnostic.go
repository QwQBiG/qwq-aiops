package config

import (
	"net/url"
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
	if cfg.BaseURL != "" {
		u, err := url.Parse(cfg.BaseURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
			addError("The active model endpoint must be an HTTP or HTTPS URL without embedded credentials")
		}
	}
	if port := os.Getenv("PORT"); port != "" && !validatePort(port) {
		addError("PORT must be a number between 1 and 65535")
	}
	return result
}
