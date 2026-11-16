package notify

import (
	"context"
	"qwq/internal/logger"
)

// 全局统一通知服务实例
var globalNotificationService *UnifiedNotificationService

// InitNotificationService 初始化全局通知服务
func InitNotificationService() {
	globalNotificationService = NewUnifiedNotificationService()
}

// Send 发送通知消息（保持向后兼容）
func Send(title, content string) {
	service := GetNotificationService().(*UnifiedNotificationService)
	go func() {
		if err := service.SendAlert(title, content); err != nil {
			logger.Info("❌ 通知发送失败: %v", err)
		}
	}()
}

// SendContext sends synchronously so callers can wait for delivery or cancellation.
func SendContext(ctx context.Context, title, content string) error {
	return GetNotificationService().(*UnifiedNotificationService).SendAlertContext(ctx, title, content)
}

// SendStatusReport 发送状态报告
func SendStatusReport(report string) error {
	return SendStatusReportContext(context.Background(), report)
}

func SendStatusReportContext(ctx context.Context, report string) error {
	return GetNotificationService().(*UnifiedNotificationService).SendStatusReportContext(ctx, report)
}

// TestNotificationConnection 测试通知连接
func TestNotificationConnection() error {
	if globalNotificationService == nil {
		InitNotificationService()
	}
	return globalNotificationService.TestConnection()
}

// ValidateNotificationConfig 验证通知配置
func ValidateNotificationConfig() error {
	if globalNotificationService == nil {
		InitNotificationService()
	}
	return globalNotificationService.ValidateConfig()
}

// GetNotificationService 获取全局通知服务实例
func GetNotificationService() NotificationService {
	if globalNotificationService == nil {
		InitNotificationService()
	}
	return globalNotificationService
}
