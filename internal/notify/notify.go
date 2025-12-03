package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"qwq/internal/config"
	"qwq/internal/logger"
)

// Send 统一发送接口
func Send(title, content string) {
	// 1. 发送钉钉
	if config.GlobalConfig.DingTalkWebhook != "" {
		go sendDingTalk(title, content)
	}

	// 2. 发送 Telegram（希望我用得到是吧）
	if config.GlobalConfig.TelegramToken != "" && config.GlobalConfig.TelegramChatID != "" {
		go sendTelegram(title, content)
	}
}

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
	// Telegram 不支持 Markdown 里的某些复杂格式，简单处理一下
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