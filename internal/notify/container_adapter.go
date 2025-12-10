package notify

import (
	"context"
	"fmt"
	"time"
)

// ContainerAlert 容器告警信息（避免循环导入）
type ContainerAlert struct {
	Level       string                 `json:"level"`       // info, warning, error, critical
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	ContainerID string                 `json:"container_id"`
	ServiceName string                 `json:"service_name"`
	ProjectName string                 `json:"project_name"`
	Timestamp   time.Time              `json:"timestamp"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ContainerNotificationAdapter 容器通知适配器
type ContainerNotificationAdapter struct {
	notifyService NotificationService
}

// NewContainerNotificationAdapter 创建容器通知适配器
func NewContainerNotificationAdapter() *ContainerNotificationAdapter {
	return &ContainerNotificationAdapter{
		notifyService: GetNotificationService(),
	}
}

// SendAlert 发送容器告警
func (c *ContainerNotificationAdapter) SendAlert(ctx context.Context, alert *ContainerAlert) error {
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

// CreateContainerNotificationService 创建容器通知服务（供容器包使用）
func CreateContainerNotificationService() interface{} {
	return NewContainerNotificationAdapter()
}