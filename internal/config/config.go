package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// PatrolRule 定义单条巡检规则
type PatrolRule struct {
	Name    string `json:"name"`    // 规则名称，如 "Nginx检查"
	Command string `json:"command"` // Shell命令，有输出则报警，无输出则正常
}

// Config 定义全局配置结构
type Config struct {
	ApiKey          string       `json:"api_key"`
	DingTalkWebhook string       `json:"webhook"`
	WebUser         string       `json:"web_user"`
	WebPassword     string       `json:"web_password"`
	KnowledgeFile   string       `json:"knowledge_file"`
	DebugMode       bool         `json:"debug"`
	PatrolRules     []PatrolRule `json:"patrol_rules"` // [新增] 自定义规则列表
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
		} else {
			fmt.Printf("⚠️ 警告: 无法读取知识库文件: %v\n", err)
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