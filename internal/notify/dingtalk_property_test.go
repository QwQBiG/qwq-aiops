package notify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: deployment-ai-config-fix, Property 5: 告警消息发送可靠性**
// **Validates: Requirements 2.1**
//
// Property 5: 告警消息发送可靠性
// *对于任何* 检测到的异常情况，如果配置了有效的通知渠道，系统应该成功发送格式化的告警消息
//
// 这个属性测试验证：
// 1. 有效配置下的消息发送成功
// 2. 消息格式正确性
// 3. 错误处理的正确性
// 4. 网络异常时的行为
func TestProperty5_AlertMessageSendingReliability(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// 属性 1: 有效配置下消息发送成功
	properties.Property("有效配置下消息发送成功", prop.ForAll(
		func(title, content string) bool {
			// 创建模拟服务器
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
			}))
			defer server.Close()

			// 创建钉钉服务
			service := NewDingTalkNotificationService(server.URL)

			// 发送消息
			err := service.SendAlert(title, content)

			// 应该成功发送
			return err == nil
		},
		genValidTitle(),
		genValidContent(),
	))

	// 属性 2: 无效 Webhook URL 时发送失败
	properties.Property("无效Webhook URL时发送失败", prop.ForAll(
		func(invalidURL, title, content string) bool {
			// 创建带有无效 URL 的服务
			service := NewDingTalkNotificationService(invalidURL)

			// 发送消息
			err := service.SendAlert(title, content)

			// 应该失败
			return err != nil
		},
		genInvalidURL(),
		genValidTitle(),
		genValidContent(),
	))

	// 属性 3: 服务器错误时发送失败
	properties.Property("服务器错误时发送失败", prop.ForAll(
		func(statusCode int, title, content string) bool {
			// 创建返回错误状态码的模拟服务器
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
			}))
			defer server.Close()

			// 创建钉钉服务
			service := NewDingTalkNotificationService(server.URL)

			// 发送消息
			err := service.SendAlert(title, content)

			// 应该失败
			return err != nil
		},
		gen.IntRange(400, 599), // HTTP 错误状态码
		genValidTitle(),
		genValidContent(),
	))

	// 属性 4: 配置验证正确性
	properties.Property("配置验证正确性", prop.ForAll(
		func(webhookURL string) bool {
			service := NewDingTalkNotificationService(webhookURL)
			err := service.ValidateConfig()

			// 如果 URL 包含 dingtalk.com 且长度足够，应该验证通过
			if len(webhookURL) >= 10 && strings.Contains(webhookURL, "dingtalk.com") {
				return err == nil
			}
			// 否则应该验证失败
			return err != nil
		},
		genWebhookURL(),
	))

	// 属性 5: 消息格式一致性
	properties.Property("消息格式一致性", prop.ForAll(
		func(title, content string) bool {
			// 创建模拟服务器来检查消息格式
			var receivedMessage *DingTalkMessage
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 解析接收到的消息
				var msg DingTalkMessage
				if err := parseJSONBody(r, &msg); err == nil {
					receivedMessage = &msg
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
			}))
			defer server.Close()

			// 创建钉钉服务并发送消息
			service := NewDingTalkNotificationService(server.URL)
			err := service.SendAlert(title, content)

			if err != nil {
				return false
			}

			// 验证消息格式
			if receivedMessage == nil {
				return false
			}

			// 检查消息类型和内容
			return receivedMessage.MsgType == "markdown" &&
				receivedMessage.Markdown != nil &&
				receivedMessage.Markdown.Title == title &&
				receivedMessage.Markdown.Text == content
		},
		genValidTitle(),
		genValidContent(),
	))

	properties.TestingRun(t)
}

// **Feature: deployment-ai-config-fix, Property 6: 配置验证准确性**
// **Validates: Requirements 2.2**
//
// Property 6: 配置验证准确性
// *对于任何* 钉钉 Webhook URL 配置，系统应该能够准确验证其有效性
func TestProperty6_ConfigValidationAccuracy(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// 属性 1: 有效 URL 格式验证通过
	properties.Property("有效URL格式验证通过", prop.ForAll(
		func(validURL string) bool {
			service := NewDingTalkNotificationService(validURL)
			err := service.ValidateConfig()
			return err == nil
		},
		genValidDingTalkURL(),
	))

	// 属性 2: 无效 URL 格式验证失败
	properties.Property("无效URL格式验证失败", prop.ForAll(
		func(invalidURL string) bool {
			service := NewDingTalkNotificationService(invalidURL)
			err := service.ValidateConfig()
			return err != nil
		},
		genInvalidDingTalkURL(),
	))

	// 属性 3: 空 URL 验证失败
	properties.Property("空URL验证失败", prop.ForAll(
		func() bool {
			service := NewDingTalkNotificationService("")
			err := service.ValidateConfig()
			return err != nil
		},
	))

	properties.TestingRun(t)
}

// 生成器函数

// genValidTitle 生成有效的标题
func genValidTitle() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0 && len(s) <= 100
	})
}

// genValidContent 生成有效的内容
func genValidContent() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0 && len(s) <= 1000
	})
}

// genInvalidURL 生成无效的 URL
func genInvalidURL() gopter.Gen {
	return gen.OneConstOf(
		"",
		"invalid-url",
		"http://",
		"not-a-url",
	)
}

// genWebhookURL 生成各种 Webhook URL（有效和无效的）
func genWebhookURL() gopter.Gen {
	return gen.OneConstOf(
		"https://oapi.dingtalk.com/robot/send?access_token=valid_token_12345",
		"https://oapi.dingtalk.com/robot/send?access_token=another_valid_token",
		"",
		"invalid-url",
		"http://example.com",
	)
}

// genValidDingTalkURL 生成有效的钉钉 Webhook URL
func genValidDingTalkURL() gopter.Gen {
	return gen.AlphaString().Map(func(token string) string {
		if len(token) < 10 {
			token = "abcdefghijklmnop" // 确保足够长
		}
		return fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s", token)
	})
}

// genInvalidDingTalkURL 生成无效的钉钉 Webhook URL
func genInvalidDingTalkURL() gopter.Gen {
	return gen.OneConstOf(
		"",
		"http://invalid.com",
		"https://example.com/webhook",
		"not-a-url",
		"dingtalk", // 太短
	)
}

// 辅助函数

// parseJSONBody 解析 JSON 请求体
func parseJSONBody(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}

// 测试统一通知服务的属性
func TestUnifiedNotificationServiceProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	// 属性: 未配置通知渠道时应该返回错误
	properties.Property("未配置通知渠道时返回错误", prop.ForAll(
		func(title, content string) bool {
			// 创建没有配置任何通知渠道的服务
			service := &UnifiedNotificationService{}

			// 发送消息应该失败
			err := service.SendAlert(title, content)
			return err != nil && strings.Contains(err.Error(), "未配置任何通知渠道")
		},
		genValidTitle(),
		genValidContent(),
	))

	properties.TestingRun(t)
}