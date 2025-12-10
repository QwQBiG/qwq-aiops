package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"qwq/internal/config"
	"qwq/internal/logger"
)

// 全局统一通知服务实例
var globalNotificationService *UnifiedNotificationService

// InitNotificationService 初始化全局通知服务
func InitNotificationService() {
	globalNotificationService = NewUnifiedNotificationService()
}

// Send 发送通知消息（保持向后兼容）
func Send(title, content string) {
	// 如果全局服务未初始化，使用原有逻辑
	if globalNotificationService == nil {
		if config.GlobalConfig.DingTalkWebhook != "" {
			go sendDingTalk(title, content)
		}
		if config.GlobalConfig.TelegramToken != "" && config.GlobalConfig.TelegramChatID != "" {
			go sendTelegram(title, content)
		}
		return
	}

	// 使用新的统一通知服务
	go func() {
		if err := globalNotificationService.SendAlert(title, content); err != nil {
			logger.Info("❌ 通知发送失败: %v", err)
		}
	}()
}

// SendStatusReport 发送状态报告
func SendStatusReport(report string) error {
	if globalNotificationService == nil {
		InitNotificationService()
	}
	return globalNotificationService.SendStatusReport(report)
}

// TestNotificationConnection 测试通知连接
func TestNotificationConnection() error {
	if globalNotificationService == nil {
		InitNotificationService()
	}
	return globalNotificationService.TestConnection()
}

// ValidateNotificationConfig 验证通知配置
func ValidateNotificationConfig() error {
	if globalNotificationService == nil {
		InitNotificationService()
	}
	return globalNotificationService.ValidateConfig()
}

// GetNotificationService 获取全局通知服务实例
func GetNotificationService() NotificationService {
	if globalNotificationService == nil {
		InitNotificationService()
	}
	return globalNotificationService
}

// 原有的发送函数（保持向后兼容）
func sendDingTalk(title, msg string) {
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  msg,
		},
	}
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(config.GlobalConfig.DingTalkWebhook, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Info("❌ 钉钉发送失败: %v", err)
		return
	}
	defer resp.Body.Close()
}

func sendTelegram(title, msg string) {
	text := fmt.Sprintf("*%s*\n\n%s", title, msg)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.GlobalConfig.TelegramToken)
	payload := map[string]string{
		"chat_id":    config.GlobalConfig.TelegramChatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Info("❌ Telegram 发送失败: %v", err)
		return
	}
	defer resp.Body.Close()
}