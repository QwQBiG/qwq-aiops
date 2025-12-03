package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Config 定义全局配置结构
type Config struct {
	ApiKey          string `json:"api_key"`
	DingTalkWebhook string `json:"webhook"`
	WebUser         string `json:"web_user"`
	WebPassword     string `json:"web_password"`
	KnowledgeFile   string `json:"knowledge_file"`
	DebugMode       bool   `json:"debug"`
}

var (
	// GlobalConfig 存储运行时配置
	GlobalConfig Config
	// CachedKnowledge 存储加载的知识库内容
	CachedKnowledge string
)

// Init 初始化配置
func Init(configPath string) error {
	// 1. 如果指定了配置文件，先加载文件
	if configPath != "" {
		if err := loadFromFile(configPath); err != nil {
			return fmt.Errorf("加载配置文件失败: %v", err)
		}
	}

	// 2. 环境变量覆盖 (优先级最高)
	if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" {
		GlobalConfig.ApiKey = envKey
	}

	// 3. 检查必要参数
	if GlobalConfig.ApiKey == "" {
		return errors.New("critical: 未找到 API Key (请配置 config.json 或环境变量 OPENAI_API_KEY)")
	}

	// 4. 加载知识库
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