package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type PatrolRule struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type HTTPRule struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Code int    `json:"code"`
}

type Config struct {
	ApiKey          string       `json:"api_key"`
	DingTalkWebhook string       `json:"webhook"`
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