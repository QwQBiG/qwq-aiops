package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Static checks are shared by startup and diagnostics; they do not probe services.
func activeConfigurationErrors(cfg Config) []string {
	var failures []string
	if cfg.BaseURL != "" {
		u := parseHTTPURL(cfg.BaseURL)
		if u == nil || u.User != nil || strings.ContainsAny(cfg.BaseURL, "?#") {
			failures = append(failures, "The active model endpoint must be an HTTP or HTTPS URL without embedded credentials, query, or fragment")
		}
	}
	for index, rule := range cfg.HTTPRules {
		if parseHTTPURL(rule.URL) == nil {
			failures = append(failures, fmt.Sprintf("http_rules[%d].url must be an absolute HTTP or HTTPS URL", index))
		}
		if rule.Code != 0 && (rule.Code < 100 || rule.Code > 599) {
			failures = append(failures, fmt.Sprintf("http_rules[%d].code must be 0 or between 100 and 599", index))
		}
	}
	for index, rule := range cfg.PatrolRules {
		if strings.TrimSpace(rule.Command) == "" {
			failures = append(failures, fmt.Sprintf("patrol_rules[%d].command must not be empty", index))
		}
	}
	return failures
}

func parseHTTPURL(value string) *url.URL {
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil
	}
	return u
}

func validatePort(value string) bool {
	if value == "" {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	port, err := strconv.Atoi(value)
	return err == nil && port >= 1 && port <= 65535
}

func validateDingTalkWebhook(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Scheme == "https" && u.Host == "oapi.dingtalk.com" &&
		u.User == nil && u.Path == "/robot/send" && u.Query().Get("access_token") != ""
}
