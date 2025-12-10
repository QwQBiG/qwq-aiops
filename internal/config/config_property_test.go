package config

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: deployment-ai-config-fix, Property 13: 配置验证完整性**
// **Validates: Requirements 4.1**
//
// Property 13: 配置验证完整性
// *对于任何* 系统启动，应该验证所有必要的配置项并提供缺失配置的指导
//
// 这个属性测试验证：
// 1. 所有必需的配置项都被检查
// 2. 缺失的配置项被正确识别
// 3. 无效的配置值被正确检测
// 4. 验证结果包含有用的建议
func TestProperty13_ConfigValidationCompleteness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// 属性 1: 缺失必需配置项时验证应该失败
	properties.Property("缺失必需配置项时验证失败", prop.ForAll(
		func(missingKey string) bool {
			// 创建一个缺少某个必需配置项的配置集
			configs := map[string]string{
				"PORT":           "8080",
				"JWT_SECRET":     "this-is-a-very-long-secret-key-for-jwt-at-least-32-chars",
				"ENCRYPTION_KEY": "12345678901234567890123456789012",
			}
			// 删除一个必需的配置项
			delete(configs, missingKey)

			result := ValidateAllRequiredConfigs(configs)

			// 验证结果应该标记为无效，并且缺失的配置项应该被记录
			return !result.Valid && result.IsMissing(missingKey)
		},
		gen.OneConstOf("PORT", "JWT_SECRET", "ENCRYPTION_KEY"),
	))

	// 属性 2: 所有必需配置项都存在且有效时验证应该通过
	properties.Property("所有必需配置项有效时验证通过", prop.ForAll(
		func(port int) bool {
			// 确保端口在有效范围内
			if port < 1 || port > 65535 {
				return true // 跳过无效端口
			}

			configs := map[string]string{
				"PORT":           fmt.Sprintf("%d", port),
				"JWT_SECRET":     "this-is-a-very-long-secret-key-for-jwt-at-least-32-chars",
				"ENCRYPTION_KEY": "12345678901234567890123456789012",
			}

			result := ValidateAllRequiredConfigs(configs)
			return result.Valid && len(result.MissingRequired) == 0
		},
		gen.IntRange(1, 65535),
	))

	// 属性 3: 无效端口号应该被检测
	properties.Property("无效端口号被检测", prop.ForAll(
		func(port int) bool {
			// 测试无效端口
			if port >= 1 && port <= 65535 {
				return true // 跳过有效端口
			}

			configs := map[string]string{
				"PORT":           fmt.Sprintf("%d", port),
				"JWT_SECRET":     "this-is-a-very-long-secret-key-for-jwt-at-least-32-chars",
				"ENCRYPTION_KEY": "12345678901234567890123456789012",
			}

			result := ValidateAllRequiredConfigs(configs)
			return !result.Valid && result.HasValidationError("PORT")
		},
		gen.OneConstOf(-1, 0, 65536, 100000),
	))

	// 属性 4: JWT 密钥长度不足应该被检测
	properties.Property("JWT密钥长度不足被检测", prop.ForAll(
		func(length int) bool {
			// 生成指定长度的密钥
			if length >= 32 {
				return true // 跳过有效长度
			}

			shortKey := strings.Repeat("a", length)
			configs := map[string]string{
				"PORT":           "8080",
				"JWT_SECRET":     shortKey,
				"ENCRYPTION_KEY": "12345678901234567890123456789012",
			}

			result := ValidateAllRequiredConfigs(configs)
			return !result.Valid && result.HasValidationError("JWT_SECRET")
		},
		gen.IntRange(1, 31),
	))

	// 属性 5: 加密密钥长度不正确应该被检测
	properties.Property("加密密钥长度不正确被检测", prop.ForAll(
		func(length int) bool {
			// 加密密钥必须是 32 字节
			if length == 32 {
				return true // 跳过正确长度
			}

			key := strings.Repeat("a", length)
			configs := map[string]string{
				"PORT":           "8080",
				"JWT_SECRET":     "this-is-a-very-long-secret-key-for-jwt-at-least-32-chars",
				"ENCRYPTION_KEY": key,
			}

			result := ValidateAllRequiredConfigs(configs)
			return !result.Valid && result.HasValidationError("ENCRYPTION_KEY")
		},
		gen.OneConstOf(1, 16, 31, 33, 64),
	))

	// 属性 6: 默认值不应该通过安全配置验证
	properties.Property("默认值不通过安全配置验证", prop.ForAll(
		func(useDefaultJWT, useDefaultEncKey bool) bool {
			configs := map[string]string{
				"PORT": "8080",
			}

			if useDefaultJWT {
				configs["JWT_SECRET"] = "change-this-to-a-random-secret-key-at-least-32-characters"
			} else {
				configs["JWT_SECRET"] = "this-is-a-valid-custom-secret-key-for-jwt-testing"
			}

			if useDefaultEncKey {
				configs["ENCRYPTION_KEY"] = "change-this-to-32-byte-key-here"
			} else {
				configs["ENCRYPTION_KEY"] = "12345678901234567890123456789012"
			}

			result := ValidateAllRequiredConfigs(configs)

			// 如果使用了任何默认值，验证应该失败
			if useDefaultJWT || useDefaultEncKey {
				return !result.Valid
			}
			return result.Valid
		},
		gen.Bool(),
		gen.Bool(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}


// **Feature: deployment-ai-config-fix, Property 14: 自动配置生成正确性**
// **Validates: Requirements 4.2**
//
// Property 14: 自动配置生成正确性
// *对于任何* 配置文件缺失的情况，系统应该自动创建有效的示例配置
//
// 这个属性测试验证：
// 1. 生成的配置包含所有必需的配置项
// 2. 生成的安全密钥是随机且有效的
// 3. 生成的配置可以通过验证
// 4. 默认值是合理的
func TestProperty14_AutoConfigGenerationCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// 属性 1: 生成的配置包含所有必需的配置项
	properties.Property("生成的配置包含所有必需项", prop.ForAll(
		func(_ int) bool {
			generator := NewConfigGenerator()
			config, err := generator.GenerateDefaultConfig()
			if err != nil {
				return false
			}

			// 检查所有必需的配置项
			requiredKeys := GetRequiredConfigKeys()
			for _, key := range requiredKeys {
				if _, exists := config.Values[key]; !exists {
					return false
				}
			}
			return true
		},
		gen.IntRange(1, 100), // 运行多次以验证一致性
	))

	// 属性 2: 生成的 JWT 密钥长度正确且唯一
	properties.Property("生成的JWT密钥有效且唯一", prop.ForAll(
		func(_ int) bool {
			generator := NewConfigGenerator()
			config1, err1 := generator.GenerateDefaultConfig()
			config2, err2 := generator.GenerateDefaultConfig()

			if err1 != nil || err2 != nil {
				return false
			}

			jwt1 := config1.Values["JWT_SECRET"]
			jwt2 := config2.Values["JWT_SECRET"]

			// 密钥长度应该至少 32 字符
			if len(jwt1) < 32 || len(jwt2) < 32 {
				return false
			}

			// 两次生成的密钥应该不同（随机性）
			return jwt1 != jwt2
		},
		gen.IntRange(1, 50),
	))

	// 属性 3: 生成的加密密钥长度正确且唯一
	properties.Property("生成的加密密钥有效且唯一", prop.ForAll(
		func(_ int) bool {
			generator := NewConfigGenerator()
			config1, err1 := generator.GenerateDefaultConfig()
			config2, err2 := generator.GenerateDefaultConfig()

			if err1 != nil || err2 != nil {
				return false
			}

			key1 := config1.Values["ENCRYPTION_KEY"]
			key2 := config2.Values["ENCRYPTION_KEY"]

			// 加密密钥必须是 32 字节
			if len(key1) != 32 || len(key2) != 32 {
				return false
			}

			// 两次生成的密钥应该不同（随机性）
			return key1 != key2
		},
		gen.IntRange(1, 50),
	))

	// 属性 4: 生成的配置可以通过验证
	properties.Property("生成的配置通过验证", prop.ForAll(
		func(_ int) bool {
			generator := NewConfigGenerator()
			config, err := generator.GenerateDefaultConfig()
			if err != nil {
				return false
			}

			result := ValidateGeneratedConfig(config)
			return result.Valid
		},
		gen.IntRange(1, 100),
	))

	// 属性 5: 默认端口值在有效范围内
	properties.Property("默认端口值有效", prop.ForAll(
		func(_ int) bool {
			generator := NewConfigGenerator()
			config, err := generator.GenerateDefaultConfig()
			if err != nil {
				return false
			}

			port := config.Values["PORT"]
			return validatePort(port)
		},
		gen.IntRange(1, 100),
	))

	// 属性 6: 使用自定义选项生成的配置保留自定义值
	properties.Property("自定义选项被保留", prop.ForAll(
		func(port int) bool {
			if port < 1 || port > 65535 {
				return true // 跳过无效端口
			}

			generator := NewConfigGenerator()
			options := map[string]string{
				"PORT": fmt.Sprintf("%d", port),
			}

			config, err := generator.GenerateConfigWithOptions(options)
			if err != nil {
				return false
			}

			return config.Values["PORT"] == fmt.Sprintf("%d", port)
		},
		gen.IntRange(1, 65535),
	))

	// 属性 7: 有默认值的配置项都有对应的默认值
	properties.Property("所有默认值配置项都有值", prop.ForAll(
		func(key string) bool {
			if !HasDefaultValue(key) {
				return true // 跳过没有默认值的配置项
			}

			defaultValue := GetDefaultConfigValue(key)
			return defaultValue != ""
		},
		gen.OneConstOf(
			"PORT", "ENVIRONMENT", "LOG_LEVEL", "TZ",
			"DB_TYPE", "DB_PATH", "AI_TIMEOUT",
			"JWT_EXPIRY", "PASSWORD_MIN_LENGTH",
			"ENABLE_CACHE", "CACHE_TTL", "ENABLE_METRICS", "PROMETHEUS_PORT", "DEBUG",
		),
	))

	// 属性 8: 生成的配置内容包含所有配置值
	properties.Property("配置内容包含所有值", prop.ForAll(
		func(_ int) bool {
			generator := NewConfigGenerator()
			config, err := generator.GenerateDefaultConfig()
			if err != nil {
				return false
			}

			// 检查配置内容是否包含关键配置项
			content := config.Content
			keysToCheck := []string{"PORT", "JWT_SECRET", "ENCRYPTION_KEY", "DB_TYPE"}

			for _, key := range keysToCheck {
				if !strings.Contains(content, key+"=") {
					return false
				}
			}
			return true
		},
		gen.IntRange(1, 50),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty_DingTalkWebhookValidation 测试钉钉 Webhook 验证
// **Feature: deployment-ai-config-fix, Property 6: 配置验证准确性**
// **Validates: Requirements 2.2**
func TestProperty_DingTalkWebhookValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// 属性 1: 有效的钉钉 Webhook URL 应该通过验证
	properties.Property("有效钉钉Webhook通过验证", prop.ForAll(
		func(token string) bool {
			if len(token) < 10 {
				return true // 跳过太短的 token
			}

			webhook := fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s", token)
			return validateDingTalkWebhook(webhook)
		},
		gen.RegexMatch("[a-f0-9]{32,64}"),
	))

	// 属性 2: 无效的钉钉 Webhook URL 应该被拒绝
	properties.Property("无效钉钉Webhook被拒绝", prop.ForAll(
		func(invalidURL string) bool {
			return !validateDingTalkWebhook(invalidURL)
		},
		gen.OneConstOf(
			"",
			"http://oapi.dingtalk.com/robot/send?access_token=xxx", // 非 HTTPS
			"https://example.com/webhook",                          // 错误域名
			"https://oapi.dingtalk.com/other/path",                 // 错误路径
			"https://oapi.dingtalk.com/robot/send",                 // 缺少 token
			"not-a-url",                                            // 非 URL
		),
	))

	// 属性 3: 通用 Webhook 验证对不同类型有效
	properties.Property("通用Webhook验证正确", prop.ForAll(
		func(webhookType string) bool {
			var validURL string
			switch webhookType {
			case "dingtalk":
				validURL = "https://oapi.dingtalk.com/robot/send?access_token=test123456"
			case "wechat":
				validURL = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test123456"
			case "slack":
				validURL = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
			default:
				return true
			}

			return ValidateWebhookURL(webhookType, validURL)
		},
		gen.OneConstOf("dingtalk", "wechat", "slack"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
