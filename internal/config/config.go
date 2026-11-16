package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
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

// Init applies file values, nonempty environment variables, then explicit overrides.
func Init(configPath string, overrides ...func(*Config)) error {
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
	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		GlobalConfig.Model = model
	}
	if webhook := os.Getenv("DINGTALK_WEBHOOK"); webhook != "" {
		GlobalConfig.DingTalkWebhook = webhook
	}
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		GlobalConfig.TelegramToken = token
	}
	if chatID := os.Getenv("TELEGRAM_CHAT_ID"); chatID != "" {
		GlobalConfig.TelegramChatID = chatID
	}
	if user := os.Getenv("WEB_USER"); user != "" {
		GlobalConfig.WebUser = user
	}
	if password := os.Getenv("WEB_PASSWORD"); password != "" {
		GlobalConfig.WebPassword = password
	}
	switch provider := os.Getenv("AI_PROVIDER"); provider {
	case "", "openai":
	case "ollama":
		host := strings.TrimRight(os.Getenv("OLLAMA_HOST"), "/")
		if host == "" {
			host = "http://localhost:11434"
		}
		GlobalConfig.BaseURL = host + "/v1"
		GlobalConfig.ApiKey = "ollama"
		GlobalConfig.Model = os.Getenv("OLLAMA_MODEL")
		if GlobalConfig.Model == "" {
			GlobalConfig.Model = "qwen2.5:7b"
		}
	default:
		return fmt.Errorf("unsupported provider: %q", provider)
	}
	for _, override := range overrides {
		override(&GlobalConfig)
	}
	if (GlobalConfig.WebUser == "") != (GlobalConfig.WebPassword == "") {
		return errors.New("WEB_USER and WEB_PASSWORD must be configured together")
	}
	if (GlobalConfig.TelegramToken == "") != (GlobalConfig.TelegramChatID == "") {
		return errors.New("TELEGRAM_TOKEN and TELEGRAM_CHAT_ID must be configured together")
	}

	// The client requires a key; local Ollama uses a placeholder.
	if GlobalConfig.ApiKey == "" {
		return errors.New("critical: 未找到 API Key")
	}
	if failures := activeConfigurationErrors(GlobalConfig); len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}

	// 加载知识库
	if GlobalConfig.KnowledgeFile != "" {
		content, err := os.ReadFile(GlobalConfig.KnowledgeFile)
		if err != nil {
			return fmt.Errorf("knowledge_file: %w", err)
		}
		CachedKnowledge = string(content)
		fmt.Printf("📚 已加载知识库: %s (%d bytes)\n", GlobalConfig.KnowledgeFile, len(content))
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
