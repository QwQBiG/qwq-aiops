// Package notify 钉钉通知服务模块
// 提供钉钉机器人消息推送功能，支持文本和 Markdown 格式消息
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"qwq/internal/logger"
	"strings"
	"time"
)

// DingTalkNotificationService 钉钉通知服务实现
// 封装钉钉机器人 API 调用，提供消息发送和配置验证功能
type DingTalkNotificationService struct {
	webhookURL string       // 钉钉机器人 Webhook URL
	httpClient *http.Client // HTTP 客户端，用于发送请求
}

// NewDingTalkNotificationService 创建钉钉通知服务实例
// 参数：webhookURL - 钉钉机器人的 Webhook URL
// 返回：配置好的钉钉通知服务实例
func NewDingTalkNotificationService(webhookURL string) *DingTalkNotificationService {
	return &DingTalkNotificationService{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // 设置 10 秒超时，防止请求阻塞
		},
	}
}

// DingTalkMessage 钉钉消息结构
// 支持文本和 Markdown 两种消息格式，可选择 @ 特定用户
type DingTalkMessage struct {
	MsgType  string            `json:"msgtype"`            // 消息类型：text 或 markdown
	Markdown *DingTalkMarkdown `json:"markdown,omitempty"` // Markdown 消息内容（可选）
	Text     *DingTalkText     `json:"text,omitempty"`     // 文本消息内容（可选）
	At       *DingTalkAt       `json:"at,omitempty"`       // @ 用户配置（可选）
}

// DingTalkMarkdown Markdown 格式消息
// 支持丰富的文本格式，适合发送告警和状态报告
type DingTalkMarkdown struct {
	Title string `json:"title"` // 消息标题
	Text  string `json:"text"`  // Markdown 格式的消息内容
}

// DingTalkText 纯文本消息
// 适合发送简单的通知信息
type DingTalkText struct {
	Content string `json:"content"` // 文本消息内容
}

// DingTalkAt @ 用户配置
// 可以 @ 特定手机号用户或 @ 所有人
type DingTalkAt struct {
	AtMobiles []string `json:"atMobiles,omitempty"` // 要 @ 的用户手机号列表
	IsAtAll   bool     `json:"isAtAll,omitempty"`   // 是否 @ 所有人
}

// SendAlert 发送告警消息
// 使用 Markdown 格式发送告警信息，支持丰富的文本格式
// 参数：title - 告警标题，content - 告警内容（支持 Markdown 语法）
// 返回：发送成功返回 nil，失败返回错误信息
func (d *DingTalkNotificationService) SendAlert(title, content string) error {
	return d.SendAlertContext(context.Background(), title, content)
}

func (d *DingTalkNotificationService) SendAlertContext(ctx context.Context, title, content string) error {
	if d.webhookURL == "" {
		return fmt.Errorf("钉钉 Webhook URL 未配置")
	}

	// 构造 Markdown 格式的告警消息
	message := &DingTalkMessage{
		MsgType: "markdown",
		Markdown: &DingTalkMarkdown{
			Title: title,
			Text:  content,
		},
	}

	return d.sendMessageContext(ctx, message)
}

// SendStatusReport 发送系统状态报告
// 自动格式化状态报告内容，添加时间戳和 Markdown 格式
// 参数：report - 状态报告内容
// 返回：发送成功返回 nil，失败返回错误信息
func (d *DingTalkNotificationService) SendStatusReport(report string) error {
	title := "系统状态报告"
	// 格式化报告内容，添加标题、内容和时间戳
	content := fmt.Sprintf("## %s\n\n%s\n\n> 报告时间: %s",
		title, report, time.Now().Format("2006-01-02 15:04:05"))

	return d.SendAlert(title, content)
}

// TestConnection 测试钉钉连接
// 发送一条测试消息验证 Webhook URL 是否可用
// 返回：连接成功返回 nil，失败返回错误信息
func (d *DingTalkNotificationService) TestConnection() error {
	// 构造简单的文本测试消息
	testMessage := &DingTalkMessage{
		MsgType: "text",
		Text: &DingTalkText{
			Content: "钉钉通知服务连接测试成功 ✅",
		},
	}

	return d.sendMessage(testMessage)
}

// ValidateConfig 验证钉钉配置
// 检查 Webhook URL 是否为空和格式是否正确
// 返回：配置有效返回 nil，无效返回错误信息
func (d *DingTalkNotificationService) ValidateConfig() error {
	if d.webhookURL == "" {
		return fmt.Errorf("钉钉 Webhook URL 不能为空")
	}

	// 简单的 URL 格式验证，确保包含钉钉域名
	if len(d.webhookURL) < 10 || !strings.Contains(d.webhookURL, "dingtalk.com") {
		return fmt.Errorf("钉钉 Webhook URL 格式不正确")
	}

	return nil
}

// sendMessage 发送消息到钉钉服务器
// 内部方法，负责实际的 HTTP 请求发送和响应处理
// 参数：message - 要发送的钉钉消息结构体
// 返回：发送成功返回 nil，失败返回错误信息
func (d *DingTalkNotificationService) sendMessage(message *DingTalkMessage) error {
	return d.sendMessageContext(context.Background(), message)
}

func (d *DingTalkNotificationService) sendMessageContext(parent context.Context, message *DingTalkMessage) error {
	// 将消息结构体序列化为 JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	// 创建带超时的上下文，防止请求长时间阻塞
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()

	// 创建 HTTP POST 请求
	req, err := http.NewRequestWithContext(ctx, "POST", d.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return notificationRequestError(err)
	}

	// 设置请求头，指定内容类型为 JSON
	req.Header.Set("Content-Type", "application/json")

	// 发送 HTTP 请求
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return notificationRequestError(err)
	}
	defer resp.Body.Close()

	// 检查响应状态码，钉钉 API 成功时返回 200
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("钉钉服务器返回错误状态码: %d", resp.StatusCode)
	}
	var result struct {
		Code    *int   `json:"errcode"`
		Message string `json:"errmsg"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result); err != nil {
		return fmt.Errorf("invalid DingTalk response: %w", err)
	}
	if result.Code == nil {
		return fmt.Errorf("invalid DingTalk response: missing errcode")
	}
	if *result.Code != 0 {
		return fmt.Errorf("DingTalk rejected notification (%d): %s", *result.Code, result.Message)
	}

	logger.Info("✅ 钉钉消息发送成功")
	return nil
}
