package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// PatrolRule Shell 巡检规则
type PatrolRule struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// HTTPRule HTTP 监控规则
type HTTPRule struct {
	Name string `json:"name"` // 例如 "官网主页"
	URL  string `json:"url"`  // 例如 "https://google.com"
	Code int    `json:"code"` // 期望状态码，默认 200
}

// Config 全局配置
type Config struct {
	ApiKey          string       `json:"api_key"`
	DingTalkWebhook string       `json:"webhook"`
	
	// Telegram 配置（给外企用吧，我现在用不到）
	TelegramToken   string       `json:"telegram_token"`
	TelegramChatID  string       `json:"telegram_chat_id"`

	WebUser         string       `json:"web_user"`
	WebPassword     string       `json:"web_password"`
	KnowledgeFile   string       `json:"knowledge_file"`
	DebugMode       bool         `json:"debug"`
	
	PatrolRules     []PatrolRule `json:"patrol_rules"`
	HTTPRules       []HTTPRule   `json:"http_rules"`
}

var (
	GlobalConfig    Config
	CachedKnowledge string
)

func Init(configPath string) error {
	if configPath != "" {
		if err := loadFromFile(configPath); err != nil {
			return fmt.Errorf("加载配置文件失败: %v", err)
		}
	}

	if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" {
		GlobalConfig.ApiKey = envKey
	}

	if GlobalConfig.ApiKey == "" {
		return errors.New("critical: 未找到 API Key")
	}

	if GlobalConfig.KnowledgeFile != "" {
		content, err := os.ReadFile(GlobalConfig.KnowledgeFile)
		if err == nil {
			CachedKnowledge = string(content)
			fmt.Printf("📚 已加载知识库: %s (%d bytes)\n", GlobalConfig.KnowledgeFile, len(content))
		}
	}

	return nil
}

func loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &GlobalConfig)
}