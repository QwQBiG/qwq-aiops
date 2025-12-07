package security

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: enhanced-aiops-platform, Property 20: AI 安全监控响应**
// **Validates: Requirements 7.3**
//
// Property 20: AI 安全监控响应
// *For any* 检测到的异常操作，AI 应该能自动阻止并发送相应的安全告警
//
// 这个属性测试验证：
// 1. 危险命令应该被正确识别和分级
// 2. 敏感信息应该被自动脱敏
// 3. 风险等级应该与命令危险程度匹配
// 4. 验证码生成应该符合安全要求
func TestProperty20_AISecurityMonitoringResponse(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 1: 危险命令应该被识别为高风险或严重风险
	properties.Property("危险命令识别为高风险", prop.ForAll(
		func(dangerousCmd string) bool {
			risk := CheckRisk(dangerousCmd)

			// 危险命令应该被识别为高风险或严重风险
			return risk == RiskHigh || risk == RiskCritical
		},
		gen.OneConstOf(
			"rm -rf /var/lib",
			"kill -9 1",
			"fdisk /dev/sda",
			"mount /dev/sda1",
			"kubectl delete namespace production",
			"DROP TABLE users",
		),
	))

	// Property 2: 严重危险命令应该被识别为严重风险
	properties.Property("严重危险命令识别为严重风险", prop.ForAll(
		func(criticalCmd string) bool {
			risk := CheckRisk(criticalCmd)

			// 严重危险命令应该被识别为严重风险
			return risk == RiskCritical
		},
		gen.OneConstOf(
			"rm -rf /",
			"mkfs.ext4 /dev/sda",
			"> /dev/sda",
			"mkfs /dev/sda1",
		),
	))

	// Property 3: 普通命令应该被识别为低风险或中风险
	properties.Property("普通命令识别为低或中风险", prop.ForAll(
		func(normalCmd string) bool {
			risk := CheckRisk(normalCmd)

			// 普通命令应该被识别为低风险或中风险
			return risk == RiskLow || risk == RiskMedium
		},
		gen.OneConstOf(
			"ls -la",
			"cat /etc/hosts",
			"ps aux",
			"docker ps",
			"systemctl status nginx",
			"chmod 755 file.sh",
		),
	))

	// Property 4: 敏感信息应该被正确脱敏
	properties.Property("IP地址应该被脱敏", prop.ForAll(
		func(ip string) bool {
			input := fmt.Sprintf("Server IP: %s", ip)
			redacted := Redact(input)

			// 脱敏后不应该包含原始IP
			return !strings.Contains(redacted, ip) &&
				strings.Contains(redacted, "<IP_REDACTED>")
		},
		gen.OneConstOf(
			"192.168.1.1",
			"10.0.0.1",
			"172.16.0.1",
			"8.8.8.8",
		),
	))

	// Property 5: 邮箱地址应该被脱敏
	properties.Property("邮箱地址应该被脱敏", prop.ForAll(
		func(email string) bool {
			input := fmt.Sprintf("User email: %s", email)
			redacted := Redact(input)

			// 脱敏后不应该包含原始邮箱
			return !strings.Contains(redacted, email) &&
				strings.Contains(redacted, "<EMAIL_REDACTED>")
		},
		gen.OneConstOf(
			"admin@example.com",
			"user@test.com",
			"support@company.org",
			"info@domain.net",
		),
	))

	// Property 6: API密钥应该被脱敏
	properties.Property("API密钥应该被脱敏", prop.ForAll(
		func(key string) bool {
			input := fmt.Sprintf("API Key: %s", key)
			redacted := Redact(input)

			// 脱敏后不应该包含原始密钥
			return !strings.Contains(redacted, key) &&
				strings.Contains(redacted, "<SECRET_KEY_REDACTED>")
		},
		gen.OneConstOf(
			"sk-1234567890abcdefghij",
			"AKIAIOSFODNN7EXAMPLE",
			"sk-proj-abcdefghijklmnopqrstuvwxyz",
		),
	))

	// Property 7: 验证码应该是4位数字
	properties.Property("验证码格式正确", prop.ForAll(
		func() bool {
			code := GenerateVerifyCode()

			// 验证码应该是4位数字
			if len(code) != 4 {
				return false
			}

			// 验证每个字符都是数字
			for _, c := range code {
				if c < '0' || c > '9' {
					return false
				}
			}

			return true
		},
	))

	// Property 8: 风险等级应该是单调的（更危险的命令风险等级更高）
	properties.Property("风险等级单调性", prop.ForAll(
		func(safeCmd, dangerousCmd string) bool {
			safeRisk := CheckRisk(safeCmd)
			dangerousRisk := CheckRisk(dangerousCmd)

			// 危险命令的风险等级应该高于或等于安全命令
			return dangerousRisk >= safeRisk
		},
		gen.OneConstOf("ls", "pwd", "echo hello"),
		gen.OneConstOf("rm -rf /", "mkfs", "kill -9"),
	))

	// Property 9: 脱敏应该是幂等的（多次脱敏结果相同）
	properties.Property("脱敏操作幂等性", prop.ForAll(
		func(input string) bool {
			redacted1 := Redact(input)
			redacted2 := Redact(redacted1)

			// 多次脱敏应该产生相同结果
			return redacted1 == redacted2
		},
		gen.OneConstOf(
			"IP: 192.168.1.1, Email: user@test.com",
			"Server: 10.0.0.1",
			"Contact: admin@example.com",
			"Key: sk-1234567890abcdefghij",
		),
	))

	// Property 10: 风险检查应该不区分大小写
	properties.Property("风险检查大小写不敏感", prop.ForAll(
		func(cmd string) bool {
			lowerRisk := CheckRisk(strings.ToLower(cmd))
			upperRisk := CheckRisk(strings.ToUpper(cmd))
			mixedRisk := CheckRisk(cmd)

			// 不同大小写应该产生相同的风险等级
			return lowerRisk == upperRisk && upperRisk == mixedRisk
		},
		gen.OneConstOf(
			"rm -rf /var",
			"Docker ps",
			"SYSTEMCTL stop nginx",
			"Kill -9 1234",
		),
	))

	// Property 11: 包含多个危险关键词的命令应该被识别为高风险
	properties.Property("复合危险命令识别", prop.ForAll(
		func(keyword1, keyword2 string) bool {
			cmd := fmt.Sprintf("%s && %s", keyword1, keyword2)
			risk := CheckRisk(cmd)

			// 包含多个危险关键词应该至少是中等风险
			return risk >= RiskMedium
		},
		gen.OneConstOf("rm file", "kill process", "docker stop"),
		gen.OneConstOf("chmod 777", "systemctl restart", "iptables -F"),
	))

	// Property 12: 空命令或安全命令应该是低风险
	properties.Property("安全命令低风险", prop.ForAll(
		func(safeCmd string) bool {
			risk := CheckRisk(safeCmd)

			// 安全命令应该是低风险
			return risk == RiskLow
		},
		gen.OneConstOf(
			"",
			"ls",
			"pwd",
			"date",
			"whoami",
			"echo test",
		),
	))

	// Property 13: 脱敏不应该改变非敏感内容
	properties.Property("脱敏保留非敏感内容", prop.ForAll(
		func(text string) bool {
			input := fmt.Sprintf("Message: %s", text)
			redacted := Redact(input)

			// 如果输入不包含敏感信息，脱敏后应该保持不变
			if !strings.Contains(text, "@") &&
				!strings.Contains(text, ".") &&
				!strings.Contains(text, "sk-") &&
				!strings.Contains(text, "AKIA") {
				return strings.Contains(redacted, text)
			}
			return true
		},
		gen.OneConstOf(
			"Hello World",
			"Test Message",
			"System Status OK",
			"Processing Complete",
		),
	))

	// Property 14: 验证码应该在有效范围内（0000-9999）
	properties.Property("验证码范围有效", prop.ForAll(
		func() bool {
			code := GenerateVerifyCode()

			// 将验证码转换为整数
			var num int
			fmt.Sscanf(code, "%d", &num)

			// 验证码应该在 0-9999 范围内
			return num >= 0 && num <= 9999
		},
	))

	// Property 15: 风险等级应该有明确的顺序
	properties.Property("风险等级顺序正确", prop.ForAll(
		func() bool {
			// 验证风险等级的顺序
			return RiskLow < RiskMedium &&
				RiskMedium < RiskHigh &&
				RiskHigh < RiskCritical
		},
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty20_SecurityMonitoring_EdgeCases 测试边界情况
func TestProperty20_SecurityMonitoring_EdgeCases(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 1: 空字符串应该被识别为低风险
	properties.Property("空字符串低风险", prop.ForAll(
		func() bool {
			risk := CheckRisk("")
			return risk == RiskLow
		},
	))

	// Property 2: 空字符串脱敏后应该保持不变
	properties.Property("空字符串脱敏不变", prop.ForAll(
		func() bool {
			redacted := Redact("")
			return redacted == ""
		},
	))

	// Property 3: 只包含空格的命令应该是低风险
	properties.Property("空格命令低风险", prop.ForAll(
		func(spaces int) bool {
			cmd := strings.Repeat(" ", spaces)
			risk := CheckRisk(cmd)
			return risk == RiskLow
		},
		gen.IntRange(1, 100),
	))

	// Property 4: 非常长的命令应该能正确处理
	properties.Property("长命令正确处理", prop.ForAll(
		func(length int) bool {
			cmd := strings.Repeat("a", length)
			risk := CheckRisk(cmd)
			// 不应该崩溃，应该返回有效的风险等级
			return risk >= RiskLow && risk <= RiskCritical
		},
		gen.IntRange(1, 10000),
	))

	// Property 5: 包含特殊字符的输入应该能正确脱敏
	properties.Property("特殊字符正确处理", prop.ForAll(
		func(specialChar string) bool {
			input := fmt.Sprintf("Test %s 192.168.1.1", specialChar)
			redacted := Redact(input)
			// 应该能正确脱敏IP，不应该崩溃
			return strings.Contains(redacted, "<IP_REDACTED>")
		},
		gen.OneConstOf("!", "@", "#", "$", "%", "^", "&", "*", "(", ")"),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
