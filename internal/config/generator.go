package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
)

// ConfigGenerator 配置生成器
type ConfigGenerator struct {
	template string
}

// GeneratedConfig 生成的配置
type GeneratedConfig struct {
	Content     string            `json:"content"`
	Values      map[string]string `json:"values"`
	GeneratedAt time.Time         `json:"generated_at"`
}

// NewConfigGenerator 创建新的配置生成器
func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{
		template: "",
	}
}

// GenerateDefaultConfig 生成默认配置
func (g *ConfigGenerator) GenerateDefaultConfig() (*GeneratedConfig, error) {
	values := make(map[string]string)

	// 生成安全密钥
	jwtSecret, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("生成 JWT 密钥失败: %v", err)
	}
	values["JWT_SECRET"] = jwtSecret

	encryptionKey, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("生成加密密钥失败: %v", err)
	}
	values["ENCRYPTION_KEY"] = encryptionKey

	// 设置默认值
	values["PORT"] = "8080"
	values["ENVIRONMENT"] = "production"
	values["LOG_LEVEL"] = "info"
	values["TZ"] = "Asia/Shanghai"
	values["DB_TYPE"] = "sqlite"
	values["DB_PATH"] = "./data/qwq.db"
	values["AI_TIMEOUT"] = "60"
	values["JWT_EXPIRY"] = "24"
	values["PASSWORD_MIN_LENGTH"] = "8"
	values["ENABLE_CACHE"] = "true"
	values["CACHE_TTL"] = "300"
	values["ENABLE_METRICS"] = "true"
	values["PROMETHEUS_PORT"] = "9090"
	values["DEBUG"] = "false"
	values["DOCKER_HOST"] = "unix:///var/run/docker.sock"
	values["DOCKER_API_VERSION"] = "1.41"
	values["BACKUP_ENABLED"] = "true"
	values["BACKUP_SCHEDULE"] = "0 2 * * *"
	values["BACKUP_RETENTION"] = "30"
	values["BACKUP_STORAGE_TYPE"] = "local"
	values["BACKUP_PATH"] = "./backups"
	values["ALERT_EVALUATION_INTERVAL"] = "30"
	values["ALERT_COOLDOWN"] = "600"
	values["METRICS_RETENTION"] = "168"
	values["CLUSTER_ENABLED"] = "false"
	values["NODE_NAME"] = "qwq-node-1"
	values["HEALTH_CHECK_INTERVAL"] = "30"
	values["RATE_LIMIT"] = "100"
	values["LOGIN_FAIL_LIMIT"] = "5"
	values["LOGIN_FAIL_WINDOW"] = "15"
	values["CORS_ALLOWED_ORIGINS"] = "*"
	values["CORS_ALLOWED_METHODS"] = "GET,POST,PUT,DELETE,OPTIONS"
	values["CORS_ALLOWED_HEADERS"] = "Origin,Content-Type,Authorization"
	values["MAX_UPLOAD_SIZE"] = "100"
	values["SESSION_TIMEOUT"] = "30"
	values["WORKER_THREADS"] = "4"
	values["REQUEST_TIMEOUT"] = "30"

	// 生成配置内容
	content := g.generateContent(values)

	return &GeneratedConfig{
		Content:     content,
		Values:      values,
		GeneratedAt: time.Now(),
	}, nil
}


// GenerateConfigWithOptions 根据选项生成配置
func (g *ConfigGenerator) GenerateConfigWithOptions(options map[string]string) (*GeneratedConfig, error) {
	// 首先生成默认配置
	config, err := g.GenerateDefaultConfig()
	if err != nil {
		return nil, err
	}

	// 用提供的选项覆盖默认值
	for key, value := range options {
		config.Values[key] = value
	}

	// 重新生成内容
	config.Content = g.generateContent(config.Values)
	config.GeneratedAt = time.Now()

	return config, nil
}

// SaveToFile 保存配置到文件
func (g *ConfigGenerator) SaveToFile(config *GeneratedConfig, path string) error {
	return os.WriteFile(path, []byte(config.Content), 0600)
}

// CreateEnvFileIfNotExists 如果 .env 文件不存在则创建
func (g *ConfigGenerator) CreateEnvFileIfNotExists(path string) (bool, error) {
	// 检查文件是否存在
	if _, err := os.Stat(path); err == nil {
		return false, nil // 文件已存在
	}

	// 生成默认配置
	config, err := g.GenerateDefaultConfig()
	if err != nil {
		return false, err
	}

	// 保存到文件
	if err := g.SaveToFile(config, path); err != nil {
		return false, err
	}

	return true, nil
}

// generateContent 生成配置文件内容
func (g *ConfigGenerator) generateContent(values map[string]string) string {
	var sb strings.Builder

	sb.WriteString("# ============================================\n")
	sb.WriteString("# qwq AIOps Platform - 环境变量配置\n")
	sb.WriteString(fmt.Sprintf("# 自动生成于: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("# ============================================\n\n")

	// 基础配置
	sb.WriteString("# ============================================\n")
	sb.WriteString("# 基础配置\n")
	sb.WriteString("# ============================================\n\n")
	writeConfigLine(&sb, "PORT", values["PORT"], "服务端口")
	writeConfigLine(&sb, "ENVIRONMENT", values["ENVIRONMENT"], "运行环境")
	writeConfigLine(&sb, "LOG_LEVEL", values["LOG_LEVEL"], "日志级别")
	writeConfigLine(&sb, "TZ", values["TZ"], "时区")
	sb.WriteString("\n")

	// 数据库配置
	sb.WriteString("# ============================================\n")
	sb.WriteString("# 数据库配置\n")
	sb.WriteString("# ============================================\n\n")
	writeConfigLine(&sb, "DB_TYPE", values["DB_TYPE"], "数据库类型")
	writeConfigLine(&sb, "DB_PATH", values["DB_PATH"], "SQLite 数据库路径")
	sb.WriteString("\n")

	// AI 配置
	sb.WriteString("# ============================================\n")
	sb.WriteString("# AI 配置\n")
	sb.WriteString("# ============================================\n\n")
	if values["AI_PROVIDER"] != "" {
		writeConfigLine(&sb, "AI_PROVIDER", values["AI_PROVIDER"], "AI 提供商")
	} else {
		sb.WriteString("# AI_PROVIDER=openai  # 或 ollama\n")
	}
	if values["OPENAI_API_KEY"] != "" {
		writeConfigLine(&sb, "OPENAI_API_KEY", values["OPENAI_API_KEY"], "OpenAI API Key")
	} else {
		sb.WriteString("# OPENAI_API_KEY=sk-your-api-key-here\n")
	}
	if values["OPENAI_BASE_URL"] != "" {
		writeConfigLine(&sb, "OPENAI_BASE_URL", values["OPENAI_BASE_URL"], "OpenAI Base URL")
	} else {
		sb.WriteString("# OPENAI_BASE_URL=https://api.openai.com/v1\n")
	}
	if values["OLLAMA_HOST"] != "" {
		writeConfigLine(&sb, "OLLAMA_HOST", values["OLLAMA_HOST"], "Ollama Host")
	} else {
		sb.WriteString("# OLLAMA_HOST=http://localhost:11434\n")
	}
	writeConfigLine(&sb, "AI_TIMEOUT", values["AI_TIMEOUT"], "AI 请求超时（秒）")
	sb.WriteString("\n")

	// 安全配置
	sb.WriteString("# ============================================\n")
	sb.WriteString("# 安全配置\n")
	sb.WriteString("# ============================================\n\n")
	writeConfigLine(&sb, "JWT_SECRET", values["JWT_SECRET"], "JWT 密钥")
	writeConfigLine(&sb, "ENCRYPTION_KEY", values["ENCRYPTION_KEY"], "加密密钥")
	writeConfigLine(&sb, "JWT_EXPIRY", values["JWT_EXPIRY"], "JWT 过期时间（小时）")
	writeConfigLine(&sb, "PASSWORD_MIN_LENGTH", values["PASSWORD_MIN_LENGTH"], "密码最小长度")
	sb.WriteString("\n")

	// 通知配置
	sb.WriteString("# ============================================\n")
	sb.WriteString("# 通知配置\n")
	sb.WriteString("# ============================================\n\n")
	if values["DINGTALK_WEBHOOK"] != "" {
		writeConfigLine(&sb, "DINGTALK_WEBHOOK", values["DINGTALK_WEBHOOK"], "钉钉 Webhook")
	} else {
		sb.WriteString("# DINGTALK_WEBHOOK=https://oapi.dingtalk.com/robot/send?access_token=xxx\n")
	}
	if values["WECHAT_WEBHOOK"] != "" {
		writeConfigLine(&sb, "WECHAT_WEBHOOK", values["WECHAT_WEBHOOK"], "企业微信 Webhook")
	} else {
		sb.WriteString("# WECHAT_WEBHOOK=https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx\n")
	}
	if values["SLACK_WEBHOOK"] != "" {
		writeConfigLine(&sb, "SLACK_WEBHOOK", values["SLACK_WEBHOOK"], "Slack Webhook")
	} else {
		sb.WriteString("# SLACK_WEBHOOK=https://hooks.slack.com/services/xxx\n")
	}
	sb.WriteString("\n")

	// Docker 配置
	sb.WriteString("# ============================================\n")
	sb.WriteString("# Docker 配置\n")
	sb.WriteString("# ============================================\n\n")
	writeConfigLine(&sb, "DOCKER_HOST", values["DOCKER_HOST"], "Docker 守护进程地址")
	writeConfigLine(&sb, "DOCKER_API_VERSION", values["DOCKER_API_VERSION"], "Docker API 版本")
	sb.WriteString("\n")

	// 备份配置
	sb.WriteString("# ============================================\n")
	sb.WriteString("# 备份配置\n")
	sb.WriteString("# ============================================\n\n")
	writeConfigLine(&sb, "BACKUP_ENABLED", values["BACKUP_ENABLED"], "启用自动备份")
	writeConfigLine(&sb, "BACKUP_SCHEDULE", values["BACKUP_SCHEDULE"], "备份调度")
	writeConfigLine(&sb, "BACKUP_RETENTION", values["BACKUP_RETENTION"], "备份保留天数")
	writeConfigLine(&sb, "BACKUP_STORAGE_TYPE", values["BACKUP_STORAGE_TYPE"], "备份存储类型")
	writeConfigLine(&sb, "BACKUP_PATH", values["BACKUP_PATH"], "备份存储路径")
	sb.WriteString("\n")

	// 监控告警配置
	sb.WriteString("# ============================================\n")
	sb.WriteString("# 监控告警配置\n")
	sb.WriteString("# ============================================\n\n")
	writeConfigLine(&sb, "ENABLE_METRICS", values["ENABLE_METRICS"], "启用 Prometheus 指标")
	writeConfigLine(&sb, "PROMETHEUS_PORT", values["PROMETHEUS_PORT"], "Prometheus 端口")
	writeConfigLine(&sb, "ALERT_EVALUATION_INTERVAL", values["ALERT_EVALUATION_INTERVAL"], "告警评估间隔（秒）")
	writeConfigLine(&sb, "ALERT_COOLDOWN", values["ALERT_COOLDOWN"], "告警冷却期（秒）")
	writeConfigLine(&sb, "METRICS_RETENTION", values["METRICS_RETENTION"], "指标保留时间（小时）")
	sb.WriteString("\n")

	// 其他配置
	sb.WriteString("# ============================================\n")
	sb.WriteString("# 其他配置\n")
	sb.WriteString("# ============================================\n\n")
	writeConfigLine(&sb, "ENABLE_CACHE", values["ENABLE_CACHE"], "启用 API 缓存")
	writeConfigLine(&sb, "CACHE_TTL", values["CACHE_TTL"], "缓存过期时间（秒）")
	writeConfigLine(&sb, "DEBUG", values["DEBUG"], "调试模式")
	writeConfigLine(&sb, "RATE_LIMIT", values["RATE_LIMIT"], "API 限流")
	writeConfigLine(&sb, "MAX_UPLOAD_SIZE", values["MAX_UPLOAD_SIZE"], "最大上传文件大小（MB）")
	writeConfigLine(&sb, "SESSION_TIMEOUT", values["SESSION_TIMEOUT"], "会话超时时间（分钟）")
	writeConfigLine(&sb, "REQUEST_TIMEOUT", values["REQUEST_TIMEOUT"], "请求超时时间（秒）")

	return sb.String()
}

// writeConfigLine 写入配置行
func writeConfigLine(sb *strings.Builder, key, value, comment string) {
	if comment != "" {
		sb.WriteString(fmt.Sprintf("# %s\n", comment))
	}
	sb.WriteString(fmt.Sprintf("%s=%s\n", key, value))
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateSecureKey 生成安全密钥
func GenerateSecureKey(length int) (string, error) {
	return generateRandomString(length)
}

// ValidateGeneratedConfig 验证生成的配置
func ValidateGeneratedConfig(config *GeneratedConfig) *ConfigValidationResult {
	return ValidateAllRequiredConfigs(config.Values)
}

// GetDefaultConfigValue 获取默认配置值
func GetDefaultConfigValue(key string) string {
	defaults := map[string]string{
		"PORT":                      "8080",
		"ENVIRONMENT":               "production",
		"LOG_LEVEL":                 "info",
		"TZ":                        "Asia/Shanghai",
		"DB_TYPE":                   "sqlite",
		"DB_PATH":                   "./data/qwq.db",
		"AI_TIMEOUT":                "60",
		"JWT_EXPIRY":                "24",
		"PASSWORD_MIN_LENGTH":       "8",
		"ENABLE_CACHE":              "true",
		"CACHE_TTL":                 "300",
		"ENABLE_METRICS":            "true",
		"PROMETHEUS_PORT":           "9090",
		"DEBUG":                     "false",
		"DOCKER_HOST":               "unix:///var/run/docker.sock",
		"DOCKER_API_VERSION":        "1.41",
		"BACKUP_ENABLED":            "true",
		"BACKUP_SCHEDULE":           "0 2 * * *",
		"BACKUP_RETENTION":          "30",
		"BACKUP_STORAGE_TYPE":       "local",
		"BACKUP_PATH":               "./backups",
		"ALERT_EVALUATION_INTERVAL": "30",
		"ALERT_COOLDOWN":            "600",
		"METRICS_RETENTION":         "168",
		"CLUSTER_ENABLED":           "false",
		"NODE_NAME":                 "qwq-node-1",
		"HEALTH_CHECK_INTERVAL":     "30",
		"RATE_LIMIT":                "100",
		"LOGIN_FAIL_LIMIT":          "5",
		"LOGIN_FAIL_WINDOW":         "15",
		"CORS_ALLOWED_ORIGINS":      "*",
		"CORS_ALLOWED_METHODS":      "GET,POST,PUT,DELETE,OPTIONS",
		"CORS_ALLOWED_HEADERS":      "Origin,Content-Type,Authorization",
		"MAX_UPLOAD_SIZE":           "100",
		"SESSION_TIMEOUT":           "30",
		"WORKER_THREADS":            "4",
		"REQUEST_TIMEOUT":           "30",
	}
	return defaults[key]
}

// HasDefaultValue 检查配置项是否有默认值
func HasDefaultValue(key string) bool {
	return GetDefaultConfigValue(key) != ""
}

// GetConfigKeysWithDefaults 获取有默认值的配置项列表
func GetConfigKeysWithDefaults() []string {
	return []string{
		"PORT", "ENVIRONMENT", "LOG_LEVEL", "TZ",
		"DB_TYPE", "DB_PATH", "AI_TIMEOUT",
		"JWT_EXPIRY", "PASSWORD_MIN_LENGTH",
		"ENABLE_CACHE", "CACHE_TTL", "ENABLE_METRICS", "PROMETHEUS_PORT", "DEBUG",
		"DOCKER_HOST", "DOCKER_API_VERSION",
		"BACKUP_ENABLED", "BACKUP_SCHEDULE", "BACKUP_RETENTION", "BACKUP_STORAGE_TYPE", "BACKUP_PATH",
		"ALERT_EVALUATION_INTERVAL", "ALERT_COOLDOWN", "METRICS_RETENTION",
		"CLUSTER_ENABLED", "NODE_NAME", "HEALTH_CHECK_INTERVAL",
		"RATE_LIMIT", "LOGIN_FAIL_LIMIT", "LOGIN_FAIL_WINDOW",
		"CORS_ALLOWED_ORIGINS", "CORS_ALLOWED_METHODS", "CORS_ALLOWED_HEADERS",
		"MAX_UPLOAD_SIZE", "SESSION_TIMEOUT", "WORKER_THREADS", "REQUEST_TIMEOUT",
	}
}
