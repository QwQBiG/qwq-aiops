package notify

import (
	"context"
	"errors"
	"fmt"
	"qwq/internal/config"
	"time"
)

// NotificationService 统一的通知服务接口
type NotificationService interface {
	// 发送告警消息
	SendAlert(title, content string) error

	// 发送状态报告
	SendStatusReport(report string) error

	// 测试连接
	TestConnection() error

	// 验证配置
	ValidateConfig() error
}

// UnifiedNotificationService 统一通知服务实现
type UnifiedNotificationService struct {
	dingTalkService *DingTalkNotificationService
	telegramService *TelegramNotificationService
}

// NewUnifiedNotificationService 创建统一通知服务
func NewUnifiedNotificationService() *UnifiedNotificationService {
	service := &UnifiedNotificationService{}

	// 初始化钉钉服务
	if config.GlobalConfig.DingTalkWebhook != "" {
		service.dingTalkService = NewDingTalkNotificationService(config.GlobalConfig.DingTalkWebhook)
	}
	if config.GlobalConfig.TelegramToken != "" || config.GlobalConfig.TelegramChatID != "" {
		service.telegramService = NewTelegramNotificationService(config.GlobalConfig.TelegramToken, config.GlobalConfig.TelegramChatID)
	}

	return service
}

// SendAlert 发送告警消息
func (u *UnifiedNotificationService) SendAlert(title, content string) error {
	return u.SendAlertContext(context.Background(), title, content)
}

// SendAlertContext waits for every configured channel and reports partial failures.
func (u *UnifiedNotificationService) SendAlertContext(ctx context.Context, title, content string) error {
	if u.dingTalkService == nil && u.telegramService == nil {
		return fmt.Errorf("未配置任何通知渠道")
	}
	var failures []error
	if u.dingTalkService != nil {
		if err := u.dingTalkService.SendAlertContext(ctx, title, content); err != nil {
			failures = append(failures, fmt.Errorf("DingTalk: %w", err))
		}
	}
	if u.telegramService != nil {
		if err := u.telegramService.SendAlertContext(ctx, title, content); err != nil {
			failures = append(failures, fmt.Errorf("Telegram: %w", err))
		}
	}
	return errors.Join(failures...)
}

// SendStatusReport 发送状态报告
func (u *UnifiedNotificationService) SendStatusReport(report string) error {
	return u.SendStatusReportContext(context.Background(), report)
}

func (u *UnifiedNotificationService) SendStatusReportContext(ctx context.Context, report string) error {
	title := "系统状态报告"
	content := fmt.Sprintf("## %s\n\n%s\n\n> 报告时间: %s", title, report, time.Now().Format("2006-01-02 15:04:05"))
	return u.SendAlertContext(ctx, title, content)
}

// TestConnection 测试连接
func (u *UnifiedNotificationService) TestConnection() error {
	return u.SendAlert("通知连接测试", "通知服务连接测试")
}

// ValidateConfig 验证配置
func (u *UnifiedNotificationService) ValidateConfig() error {
	hasValidConfig := false

	// 验证钉钉配置
	if u.dingTalkService != nil {
		if err := u.dingTalkService.ValidateConfig(); err != nil {
			return fmt.Errorf("钉钉配置验证失败: %v", err)
		}
		hasValidConfig = true
	}
	if u.telegramService != nil {
		if err := u.telegramService.ValidateConfig(); err != nil {
			return err
		}
		hasValidConfig = true
	}

	if !hasValidConfig {
		return fmt.Errorf("未配置任何有效的通知渠道")
	}

	return nil
}
