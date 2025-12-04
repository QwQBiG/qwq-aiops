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
	Name string `json:"name"`
	URL  string `json:"url"`
	Code int    `json:"code"`
}

// Config 全局配置
type Config struct {
	ApiKey          string       `json:"api_key"`
	BaseURL         string       `json:"base_url"`
	Model           string       `json:"model"`
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

	// 环境变量覆盖
	if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" {
		GlobalConfig.ApiKey = envKey
	}
	if envBase := os.Getenv("OPENAI_BASE_URL"); envBase != "" {
		GlobalConfig.BaseURL = envBase
	}

	// 必填检查 (Ollama 模式下 ApiKey 可以随便填，但不能为空)
	if GlobalConfig.ApiKey == "" {
		return errors.New("critical: 未找到 API Key")
	}

	// 加载知识库
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