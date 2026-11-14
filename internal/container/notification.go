package container

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// simpleNotificationService 简单的通知服务实现
// 实际生产环境中应该集成真实的通知系统（邮件、短信、Webhook等）
type simpleNotificationService struct {
	// 可以添加邮件服务、短信服务等依赖
}

// NewSimpleNotificationService 创建简单通知服务实例
func NewSimpleNotificationService() NotificationService {
	return &simpleNotificationService{}
}

// NewDingTalkNotificationService 创建钉钉通知服务实例（用于容器自愈）
func NewDingTalkNotificationService() NotificationService {
	// 导入 notify 包并使用统一通知服务
	return &dingTalkContainerNotificationService{}
}

// SendAlert 发送告警
func (s *simpleNotificationService) SendAlert(ctx context.Context, alert *Alert) error {
	// 这里简化实现，只打印日志
	// 实际应该发送邮件、短信、Webhook等
	log.Printf("[ALERT] Level: %s, Title: %s, Message: %s, Container: %s, Service: %s, Project: %s",
		alert.Level, alert.Title, alert.Message, alert.ContainerID, alert.ServiceName, alert.ProjectName)

	// 可以在这里添加实际的通知逻辑
	// 例如：
	// - 发送邮件
	// - 发送短信
	// - 调用 Webhook
	// - 推送到消息队列
	// - 集成 Slack/钉钉/企业微信等

	return nil
}

// mockNotificationService 用于测试的 Mock 通知服务
type mockNotificationService struct {
	mu     sync.Mutex
	alerts []*Alert
}

// NewMockNotificationService 创建 Mock 通知服务实例
func NewMockNotificationService() *mockNotificationService {
	return &mockNotificationService{
		alerts: make([]*Alert, 0),
	}
}

// SendAlert 发送告警（记录到内存）
func (s *mockNotificationService) SendAlert(ctx context.Context, alert *Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = append(s.alerts, alert)
	return nil
}

// GetAlerts 获取所有告警记录（用于测试）
func (s *mockNotificationService) GetAlerts() []*Alert {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*Alert(nil), s.alerts...)
}

// ClearAlerts 清空告警记录（用于测试）
func (s *mockNotificationService) ClearAlerts() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = make([]*Alert, 0)
}

// webhookNotificationService Webhook 通知服务
type webhookNotificationService struct {
	webhookURL string
	// 可以添加 HTTP 客户端等
}

// NewWebhookNotificationService 创建 Webhook 通知服务实例
func NewWebhookNotificationService(webhookURL string) NotificationService {
	return &webhookNotificationService{
		webhookURL: webhookURL,
	}
}

// SendAlert 发送告警到 Webhook
func (s *webhookNotificationService) SendAlert(ctx context.Context, alert *Alert) error {
	// TODO: 实现实际的 Webhook 调用
	// 1. 将 alert 序列化为 JSON
	// 2. 发送 POST 请求到 webhookURL
	// 3. 处理响应和错误

	fmt.Printf("Sending alert to webhook: %s\n", s.webhookURL)
	return nil
}

// dingTalkContainerNotificationService 钉钉容器通知服务
type dingTalkContainerNotificationService struct{}

// SendAlert 发送容器告警到钉钉
func (d *dingTalkContainerNotificationService) SendAlert(ctx context.Context, alert *Alert) error {
	// 导入 notify 包需要在文件顶部添加
	// 这里我们直接调用 notify.Send 函数
	title := fmt.Sprintf("🚨 容器告警 - %s", alert.Title)

	content := fmt.Sprintf(`## %s

**告警级别**: %s
**容器ID**: %s
**服务名称**: %s
**项目名称**: %s
**告警时间**: %s

### 详细信息
%s

---
> 系统自动发送，请及时处理`,
		alert.Title,
		getLevelEmoji(alert.Level),
		alert.ContainerID,
		alert.ServiceName,
		alert.ProjectName,
		alert.Timestamp.Format("2006-01-02 15:04:05"),
		alert.Message,
	)

	// 这里需要调用 notify 包的 Send 函数
	// 由于循环导入问题，我们使用接口方式
	return sendNotificationMessage(title, content)
}

// getLevelEmoji 获取告警级别对应的表情符号
func getLevelEmoji(level string) string {
	switch level {
	case "info":
		return "ℹ️ 信息"
	case "warning":
		return "⚠️ 警告"
	case "error":
		return "❌ 错误"
	case "critical":
		return "🔥 严重"
	default:
		return "📢 " + level
	}
}

// sendNotificationMessage 发送通知消息（避免循环导入）
func sendNotificationMessage(title, content string) error {
	// 这里可以通过接口或者回调函数的方式来避免循环导入
	// 暂时使用简单的日志输出
	log.Printf("[DINGTALK ALERT] %s: %s", title, content)
	return nil
}
