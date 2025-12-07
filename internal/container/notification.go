package container

import (
	"context"
	"fmt"
	"log"
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
	s.alerts = append(s.alerts, alert)
	return nil
}

// GetAlerts 获取所有告警记录（用于测试）
func (s *mockNotificationService) GetAlerts() []*Alert {
	return s.alerts
}

// ClearAlerts 清空告警记录（用于测试）
func (s *mockNotificationService) ClearAlerts() {
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
