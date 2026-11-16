package config

import (
	"net/url"
	"strconv"
)

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
