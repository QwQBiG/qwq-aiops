package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// ConfigValidationResult 配置验证结果
// 包含验证状态、缺失项、无效项、警告和建议
type ConfigValidationResult struct {
	Valid           bool                    `json:"valid"`            // 验证是否通过
	MissingRequired []string                `json:"missing_required"` // 缺失的必需配置项
	InvalidConfigs  []ConfigValidationError `json:"invalid_configs"`  // 无效的配置项列表
	Warnings        []string                `json:"warnings"`         // 警告信息
	Suggestions     []string                `json:"suggestions"`      // 修复建议
}

// ConfigValidationError 配置验证错误详情
type ConfigValidationError struct {
	Key        string `json:"key"`        // 配置项键名
	Value      string `json:"value"`      // 配置值（敏感信息会被掩码）
	Reason     string `json:"reason"`     // 错误原因
	Suggestion string `json:"suggestion"` // 修复建议
}

// ConfigStatus 配置状态摘要
// 用于快速检查各模块配置是否完成
type ConfigStatus struct {
	EnvFileExists      bool            `json:"env_file_exists"`      // .env 文件是否存在
	RequiredVarsSet    map[string]bool `json:"required_vars_set"`    // 必需配置项设置状态
	DingTalkConfigured bool            `json:"dingtalk_configured"`  // 钉钉通知是否配置
	AIConfigured       bool            `json:"ai_configured"`        // AI 服务是否配置
	DatabaseConfigured bool            `json:"database_configured"`  // 数据库是否配置
	SecurityConfigured bool            `json:"security_configured"`  // 安全配置是否完成
}

// RequiredConfigItem 必需的配置项定义
type RequiredConfigItem struct {
	Key         string            // 配置项键名
	Description string            // 配置项描述
	Validator   func(string) bool // 验证函数
	Default     string            // 默认值（空表示必须配置）
}

// ConfigValidator 配置验证器
// 负责加载和验证系统配置
type ConfigValidator struct {
	requiredItems []RequiredConfigItem  // 必需配置项列表
	envVars       map[string]string     // 已加载的环境变量
}

// NewConfigValidator 创建新的配置验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		requiredItems: getRequiredConfigItems(),
		envVars:       make(map[string]string),
	}
}

// getRequiredConfigItems 获取必需的配置项列表
func getRequiredConfigItems() []RequiredConfigItem {
	return []RequiredConfigItem{
		{
			Key:         "PORT",
			Description: "服务端口",
			Validator:   validatePort,
			Default:     "8080",
		},
		{
			Key:         "JWT_SECRET",
			Description: "JWT 密钥",
			Validator:   validateJWTSecret,
			Default:     "",
		},
		{
			Key:         "ENCRYPTION_KEY",
			Description: "加密密钥",
			Validator:   validateEncryptionKey,
			Default:     "",
		},
	}
}


// LoadEnvVars 从环境变量和 .env 文件加载配置
func (v *ConfigValidator) LoadEnvVars() error {
	// 首先加载 .env 文件
	if err := v.loadEnvFile(".env"); err != nil {
		// .env 文件不存在不是错误，只是警告
		v.envVars["_ENV_FILE_EXISTS"] = "false"
	} else {
		v.envVars["_ENV_FILE_EXISTS"] = "true"
	}

	// 然后从系统环境变量覆盖
	for _, item := range v.requiredItems {
		if val := os.Getenv(item.Key); val != "" {
			v.envVars[item.Key] = val
		}
	}

	// 加载其他常用配置
	commonKeys := []string{
		"DINGTALK_WEBHOOK", "AI_PROVIDER", "OPENAI_API_KEY", "OPENAI_BASE_URL",
		"OLLAMA_HOST", "OLLAMA_MODEL", "DB_TYPE", "DB_PATH", "ENVIRONMENT",
		"LOG_LEVEL", "DEBUG", "DOCKER_HOST",
	}
	for _, key := range commonKeys {
		if val := os.Getenv(key); val != "" {
			v.envVars[key] = val
		}
	}

	return nil
}

// loadEnvFile 从 .env 文件加载配置
func (v *ConfigValidator) loadEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析 KEY=VALUE 格式
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// 移除引号
			value = strings.Trim(value, "\"'")
			v.envVars[key] = value
		}
	}

	return nil
}

// Validate 验证所有配置
func (v *ConfigValidator) Validate() *ConfigValidationResult {
	result := &ConfigValidationResult{
		Valid:           true,
		MissingRequired: []string{},
		InvalidConfigs:  []ConfigValidationError{},
		Warnings:        []string{},
		Suggestions:     []string{},
	}

	// 检查必需的配置项
	for _, item := range v.requiredItems {
		val, exists := v.envVars[item.Key]
		if !exists || val == "" {
			if item.Default != "" {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("%s 未配置，将使用默认值: %s", item.Key, item.Default))
			} else {
				result.MissingRequired = append(result.MissingRequired, item.Key)
				result.Suggestions = append(result.Suggestions,
					fmt.Sprintf("请配置 %s: %s", item.Key, item.Description))
				result.Valid = false
			}
			continue
		}

		// 验证配置值
		if item.Validator != nil && !item.Validator(val) {
			result.InvalidConfigs = append(result.InvalidConfigs, ConfigValidationError{
				Key:        item.Key,
				Value:      maskSensitiveValue(item.Key, val),
				Reason:     fmt.Sprintf("%s 配置值无效", item.Description),
				Suggestion: getValidationSuggestion(item.Key),
			})
			result.Valid = false
		}
	}

	// 检查 AI 配置
	v.validateAIConfig(result)

	// 检查钉钉配置
	v.validateDingTalkConfig(result)

	return result
}

// validateAIConfig 验证 AI 配置
func (v *ConfigValidator) validateAIConfig(result *ConfigValidationResult) {
	provider := v.envVars["AI_PROVIDER"]

	switch provider {
	case "openai":
		if v.envVars["OPENAI_API_KEY"] == "" {
			result.Warnings = append(result.Warnings, "AI_PROVIDER 设置为 openai，但 OPENAI_API_KEY 未配置")
			result.Suggestions = append(result.Suggestions, "请配置 OPENAI_API_KEY 以启用 OpenAI 功能")
		}
	case "ollama":
		if v.envVars["OLLAMA_HOST"] == "" {
			result.Warnings = append(result.Warnings, "AI_PROVIDER 设置为 ollama，但 OLLAMA_HOST 未配置")
			result.Suggestions = append(result.Suggestions, "请配置 OLLAMA_HOST，例如: http://localhost:11434")
		}
	case "":
		result.Warnings = append(result.Warnings, "AI_PROVIDER 未配置，AI 功能将不可用")
		result.Suggestions = append(result.Suggestions, "配置 AI_PROVIDER=openai 或 AI_PROVIDER=ollama 以启用 AI 功能")
	}
}

// validateDingTalkConfig 验证钉钉配置
func (v *ConfigValidator) validateDingTalkConfig(result *ConfigValidationResult) {
	webhook := v.envVars["DINGTALK_WEBHOOK"]
	if webhook == "" {
		result.Warnings = append(result.Warnings, "DINGTALK_WEBHOOK 未配置，钉钉通知功能将不可用")
		result.Suggestions = append(result.Suggestions,
			"配置 DINGTALK_WEBHOOK=https://oapi.dingtalk.com/robot/send?access_token=xxx 以启用钉钉通知")
		return
	}

	if !validateDingTalkWebhook(webhook) {
		result.InvalidConfigs = append(result.InvalidConfigs, ConfigValidationError{
			Key:        "DINGTALK_WEBHOOK",
			Value:      maskWebhookURL(webhook),
			Reason:     "钉钉 Webhook URL 格式无效",
			Suggestion: "钉钉 Webhook URL 应该以 https://oapi.dingtalk.com/robot/send?access_token= 开头",
		})
	}
}

// GetConfigStatus 获取配置状态
func (v *ConfigValidator) GetConfigStatus() *ConfigStatus {
	status := &ConfigStatus{
		EnvFileExists:   v.envVars["_ENV_FILE_EXISTS"] == "true",
		RequiredVarsSet: make(map[string]bool),
	}

	// 检查必需配置项
	for _, item := range v.requiredItems {
		val, exists := v.envVars[item.Key]
		status.RequiredVarsSet[item.Key] = exists && val != ""
	}

	// 检查钉钉配置
	webhook := v.envVars["DINGTALK_WEBHOOK"]
	status.DingTalkConfigured = webhook != "" && validateDingTalkWebhook(webhook)

	// 检查 AI 配置
	provider := v.envVars["AI_PROVIDER"]
	switch provider {
	case "openai":
		status.AIConfigured = v.envVars["OPENAI_API_KEY"] != ""
	case "ollama":
		status.AIConfigured = v.envVars["OLLAMA_HOST"] != ""
	default:
		status.AIConfigured = false
	}

	// 检查数据库配置
	dbType := v.envVars["DB_TYPE"]
	status.DatabaseConfigured = dbType != "" || v.envVars["DB_PATH"] != ""

	// 检查安全配置
	status.SecurityConfigured = v.envVars["JWT_SECRET"] != "" &&
		v.envVars["JWT_SECRET"] != "change-this-to-a-random-secret-key-at-least-32-characters"

	return status
}


// 验证函数

// validatePort 验证端口号
func validatePort(value string) bool {
	var port int
	_, err := fmt.Sscanf(value, "%d", &port)
	return err == nil && port > 0 && port <= 65535
}

// validateJWTSecret 验证 JWT 密钥
func validateJWTSecret(value string) bool {
	// JWT 密钥至少 32 个字符
	if len(value) < 32 {
		return false
	}
	// 不能是默认值
	if value == "change-this-to-a-random-secret-key-at-least-32-characters" {
		return false
	}
	return true
}

// validateEncryptionKey 验证加密密钥
func validateEncryptionKey(value string) bool {
	// AES-256 需要 32 字节密钥
	if len(value) != 32 {
		return false
	}
	// 不能是默认值
	if value == "change-this-to-32-byte-key-here" {
		return false
	}
	return true
}

// validateDingTalkWebhook 验证钉钉 Webhook URL
func validateDingTalkWebhook(value string) bool {
	if value == "" {
		return false
	}

	// 解析 URL
	u, err := url.Parse(value)
	if err != nil {
		return false
	}

	// 必须是 HTTPS 协议
	if u.Scheme != "https" {
		return false
	}

	// 检查是否是钉钉域名
	if u.Host != "oapi.dingtalk.com" {
		return false
	}

	// 检查路径
	if u.Path != "/robot/send" {
		return false
	}

	// 检查是否有 access_token
	token := u.Query().Get("access_token")
	return token != ""
}

// validateURL 验证 URL 格式
func validateURL(value string) bool {
	u, err := url.Parse(value)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}

// 辅助函数

// maskSensitiveValue 掩码敏感值
func maskSensitiveValue(key, value string) string {
	sensitiveKeys := []string{"SECRET", "KEY", "PASSWORD", "TOKEN"}
	for _, sk := range sensitiveKeys {
		if strings.Contains(strings.ToUpper(key), sk) {
			if len(value) <= 8 {
				return "****"
			}
			return value[:4] + "****" + value[len(value)-4:]
		}
	}
	return value
}

// maskWebhookURL 掩码 Webhook URL
func maskWebhookURL(value string) string {
	u, err := url.Parse(value)
	if err != nil {
		return "****"
	}
	token := u.Query().Get("access_token")
	if token != "" && len(token) > 8 {
		maskedToken := token[:4] + "****" + token[len(token)-4:]
		return strings.Replace(value, token, maskedToken, 1)
	}
	return value
}

// getValidationSuggestion 获取验证建议
func getValidationSuggestion(key string) string {
	suggestions := map[string]string{
		"PORT":           "端口号应该在 1-65535 之间",
		"JWT_SECRET":     "JWT 密钥至少需要 32 个字符，请使用随机字符串",
		"ENCRYPTION_KEY": "加密密钥必须是 32 字节，用于 AES-256 加密",
	}
	if s, ok := suggestions[key]; ok {
		return s
	}
	return "请检查配置值是否正确"
}

// ValidateConfigValue 验证单个配置值（用于属性测试）
func ValidateConfigValue(key, value string) bool {
	switch key {
	case "PORT":
		return validatePort(value)
	case "JWT_SECRET":
		return validateJWTSecret(value)
	case "ENCRYPTION_KEY":
		return validateEncryptionKey(value)
	case "DINGTALK_WEBHOOK":
		return validateDingTalkWebhook(value)
	default:
		return value != ""
	}
}

// IsRequiredConfig 检查是否是必需的配置项
func IsRequiredConfig(key string) bool {
	requiredKeys := []string{"PORT", "JWT_SECRET", "ENCRYPTION_KEY"}
	for _, k := range requiredKeys {
		if k == key {
			return true
		}
	}
	return false
}

// GetRequiredConfigKeys 获取所有必需的配置项键名
func GetRequiredConfigKeys() []string {
	return []string{"PORT", "JWT_SECRET", "ENCRYPTION_KEY"}
}

// ValidateAllRequiredConfigs 验证所有必需的配置项
func ValidateAllRequiredConfigs(configs map[string]string) *ConfigValidationResult {
	result := &ConfigValidationResult{
		Valid:           true,
		MissingRequired: []string{},
		InvalidConfigs:  []ConfigValidationError{},
		Warnings:        []string{},
		Suggestions:     []string{},
	}

	requiredItems := getRequiredConfigItems()
	for _, item := range requiredItems {
		val, exists := configs[item.Key]
		if !exists || val == "" {
			result.MissingRequired = append(result.MissingRequired, item.Key)
			result.Valid = false
			continue
		}

		if item.Validator != nil && !item.Validator(val) {
			result.InvalidConfigs = append(result.InvalidConfigs, ConfigValidationError{
				Key:    item.Key,
				Value:  maskSensitiveValue(item.Key, val),
				Reason: fmt.Sprintf("%s 配置值无效", item.Description),
			})
			result.Valid = false
		}
	}

	return result
}

// HasValidationError 检查验证结果是否有特定类型的错误
func (r *ConfigValidationResult) HasValidationError(key string) bool {
	for _, err := range r.InvalidConfigs {
		if err.Key == key {
			return true
		}
	}
	return false
}

// IsMissing 检查配置项是否缺失
func (r *ConfigValidationResult) IsMissing(key string) bool {
	for _, k := range r.MissingRequired {
		if k == key {
			return true
		}
	}
	return false
}

// GetAllConfigKeys 获取所有配置项键名（用于测试）
func GetAllConfigKeys() []string {
	return []string{
		"PORT", "ENVIRONMENT", "LOG_LEVEL", "TZ",
		"DB_TYPE", "DB_PATH", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"AI_PROVIDER", "OPENAI_API_KEY", "OPENAI_BASE_URL", "OPENAI_MODEL",
		"OLLAMA_HOST", "OLLAMA_MODEL", "AI_TIMEOUT",
		"JWT_SECRET", "ENCRYPTION_KEY", "JWT_EXPIRY", "PASSWORD_MIN_LENGTH",
		"ENABLE_CACHE", "CACHE_TTL", "ENABLE_METRICS", "PROMETHEUS_PORT", "DEBUG",
		"DOCKER_HOST", "DOCKER_API_VERSION",
		"BACKUP_ENABLED", "BACKUP_SCHEDULE", "BACKUP_RETENTION", "BACKUP_STORAGE_TYPE", "BACKUP_PATH",
		"ALERT_EVALUATION_INTERVAL", "ALERT_COOLDOWN", "METRICS_RETENTION",
		"CLUSTER_ENABLED", "NODE_NAME", "HEALTH_CHECK_INTERVAL",
		"DINGTALK_WEBHOOK", "WECHAT_WEBHOOK", "SLACK_WEBHOOK",
		"RATE_LIMIT", "LOGIN_FAIL_LIMIT", "LOGIN_FAIL_WINDOW",
		"CORS_ALLOWED_ORIGINS", "CORS_ALLOWED_METHODS", "CORS_ALLOWED_HEADERS",
		"MAX_UPLOAD_SIZE", "SESSION_TIMEOUT", "WORKER_THREADS", "REQUEST_TIMEOUT",
	}
}

// ValidateWebhookURL 验证 Webhook URL（通用）
func ValidateWebhookURL(webhookType, value string) bool {
	if value == "" {
		return false
	}

	u, err := url.Parse(value)
	if err != nil {
		return false
	}

	// 必须是 HTTPS
	if u.Scheme != "https" {
		return false
	}

	switch webhookType {
	case "dingtalk":
		return u.Host == "oapi.dingtalk.com" && u.Path == "/robot/send" && u.Query().Get("access_token") != ""
	case "wechat":
		return u.Host == "qyapi.weixin.qq.com" && strings.HasPrefix(u.Path, "/cgi-bin/webhook/send") && u.Query().Get("key") != ""
	case "slack":
		return u.Host == "hooks.slack.com" && strings.HasPrefix(u.Path, "/services/")
	default:
		return u.Host != ""
	}
}

// ValidateEmailConfig 验证邮件配置
func ValidateEmailConfig(host, port, user, password string) bool {
	if host == "" || port == "" {
		return false
	}

	// 验证端口
	if !validatePort(port) {
		return false
	}

	// 验证邮箱格式
	if user != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(user) {
			return false
		}
	}

	return true
}
