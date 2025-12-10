package notify

import (
	"context"
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

// Alert 告警信息结构（兼容容器自愈服务）
type Alert struct {
	Level       string                 `json:"level"`       // info, warning, error, critical
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	ContainerID string                 `json:"container_id"`
	ServiceName string                 `json:"service_name"`
	ProjectName string                 `json:"project_name"`
	Timestamp   time.Time              `json:"timestamp"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ContainerNotificationService 容器告警通知服务（实现容器自愈服务的接口）
type ContainerNotificationService struct {
	notifyService NotificationService
}

// NewContainerNotificationService 创建容器通知服务
func NewContainerNotificationService(notifyService NotificationService) *ContainerNotificationService {
	return &ContainerNotificationService{
		notifyService: notifyService,
	}
}

// SendAlert 发送容器告警（实现容器自愈服务的 NotificationService 接口）
func (c *ContainerNotificationService) SendAlert(ctx context.Context, alert *Alert) error {
	// 格式化告警消息
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

	return c.notifyService.SendAlert(title, content)
}

// UnifiedNotificationService 统一通知服务实现
type UnifiedNotificationService struct {
	dingTalkService *DingTalkNotificationService
	// 可以添加其他通知渠道
}

// NewUnifiedNotificationService 创建统一通知服务
func NewUnifiedNotificationService() *UnifiedNotificationService {
	service := &UnifiedNotificationService{}
	
	// 初始化钉钉服务
	if config.GlobalConfig.DingTalkWebhook != "" {
		service.dingTalkService = NewDingTalkNotificationService(config.GlobalConfig.DingTalkWebhook)
	}
	
	return service
}

// SendAlert 发送告警消息
func (u *UnifiedNotificationService) SendAlert(title, content string) error {
	var lastErr error
	sent := false

	// 发送到钉钉
	if u.dingTalkService != nil {
		if err := u.dingTalkService.SendAlert(title, content); err != nil {
			lastErr = fmt.Errorf("钉钉发送失败: %v", err)
		} else {
			sent = true
		}
	}

	// 如果没有配置任何通知渠道
	if u.dingTalkService == nil {
		return fmt.Errorf("未配置任何通知渠道")
	}

	// 如果所有渠道都失败
	if !sent && lastErr != nil {
		return lastErr
	}

	return nil
}

// SendStatusReport 发送状态报告
func (u *UnifiedNotificationService) SendStatusReport(report string) error {
	var lastErr error
	sent := false

	// 发送到钉钉
	if u.dingTalkService != nil {
		if err := u.dingTalkService.SendStatusReport(report); err != nil {
			lastErr = fmt.Errorf("钉钉发送失败: %v", err)
		} else {
			sent = true
		}
	}

	// 如果没有配置任何通知渠道
	if u.dingTalkService == nil {
		return fmt.Errorf("未配置任何通知渠道")
	}

	// 如果所有渠道都失败
	if !sent && lastErr != nil {
		return lastErr
	}

	return nil
}

// TestConnection 测试连接
func (u *UnifiedNotificationService) TestConnection() error {
	if u.dingTalkService != nil {
		return u.dingTalkService.TestConnection()
	}
	
	return fmt.Errorf("未配置任何通知渠道")
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

	if !hasValidConfig {
		return fmt.Errorf("未配置任何有效的通知渠道")
	}

	return nil
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