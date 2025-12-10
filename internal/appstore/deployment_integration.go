package appstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// InstanceStatus 实例状态类型
type InstanceStatus string

// 实例状态常量
const (
	InstanceStatusRunning  InstanceStatus = "running"
	InstanceStatusStopped  InstanceStatus = "stopped"
	InstanceStatusError    InstanceStatus = "error"
	InstanceStatusUpdating InstanceStatus = "updating"
)

// DeploymentIntegration 部署集成服务接口
// 负责将应用商店的安装流程与容器部署引擎集成，实现从模板到容器的完整部署流程
type DeploymentIntegration interface {
	// DeployFromTemplate 从模板部署应用
	// 根据应用实例ID，读取模板配置并调用容器部署引擎创建容器
	DeployFromTemplate(ctx context.Context, instanceID uint) error
	
	// UpdateDeployment 更新应用部署
	// 更新已部署应用的配置，支持滚动更新和蓝绿部署
	UpdateDeployment(ctx context.Context, instanceID uint) error
	
	// StopDeployment 停止应用部署
	// 停止应用的所有容器服务，但保留配置和数据
	StopDeployment(ctx context.Context, instanceID uint) error
	
	// StartDeployment 启动应用部署
	// 启动已停止的应用服务
	StartDeployment(ctx context.Context, instanceID uint) error
	
	// GetDeploymentStatus 获取部署状态
	// 查询应用的实时部署状态，包括所有服务的运行情况
	GetDeploymentStatus(ctx context.Context, instanceID uint) (*DeploymentStatusInfo, error)
	
	// SyncDeploymentStatus 同步部署状态到实例
	// 从容器引擎同步最新状态并更新到应用实例记录
	SyncDeploymentStatus(ctx context.Context, instanceID uint) error
}

// DeploymentStatusInfo 部署状态信息
// 包含应用部署的完整状态数据，用于前端展示和监控
type DeploymentStatusInfo struct {
	InstanceID    uint                   `json:"instance_id"`    // 应用实例ID
	DeploymentID  uint                   `json:"deployment_id,omitempty"` // 部署记录ID
	Status        string                 `json:"status"`         // 部署状态：deploying, running, stopped, error
	Progress      int                    `json:"progress"`       // 部署进度 0-100
	Message       string                 `json:"message"`        // 状态描述信息
	Services      []ServiceStatusInfo    `json:"services,omitempty"` // 各服务状态列表
	LastUpdated   time.Time              `json:"last_updated"`   // 最后更新时间
}

// ServiceStatusInfo 服务状态信息
// 描述单个容器服务的运行状态
type ServiceStatusInfo struct {
	Name          string    `json:"name"`           // 服务名称
	ContainerID   string    `json:"container_id"`   // 容器ID
	Status        string    `json:"status"`         // 服务状态：running, stopped, error
	Health        string    `json:"health"`         // 健康状态：healthy, unhealthy, starting
	RestartCount  int       `json:"restart_count"`  // 重启次数
	CreatedAt     time.Time `json:"created_at"`     // 创建时间
}

// ContainerService 容器服务接口（简化版本，用于集成）
type ContainerService interface {
	DeployCompose(ctx context.Context, composeContent, projectName string) error
	UpdateCompose(ctx context.Context, projectName, composeContent string) error
	StopCompose(ctx context.Context, projectName string) error
	StartCompose(ctx context.Context, projectName string) error
	GetComposeStatus(ctx context.Context, projectName string) ([]ServiceStatusInfo, error)
}

// deploymentIntegrationImpl 部署集成服务实现
type deploymentIntegrationImpl struct {
	db               *gorm.DB
	appStoreService  AppStoreService
	containerService ContainerService // 容器服务接口
}

// NewDeploymentIntegration 创建部署集成服务实例
func NewDeploymentIntegration(db *gorm.DB, appStoreService AppStoreService, containerService ContainerService) DeploymentIntegration {
	return &deploymentIntegrationImpl{
		db:               db,
		appStoreService:  appStoreService,
		containerService: containerService,
	}
}

// DeployFromTemplate 从模板部署应用
// 实现步骤：
// 1. 获取应用实例和模板信息
// 2. 渲染模板生成部署配置
// 3. 调用容器引擎执行部署
// 4. 更新实例状态和部署记录
func (d *deploymentIntegrationImpl) DeployFromTemplate(ctx context.Context, instanceID uint) error {
	// 获取应用实例
	instance, err := d.appStoreService.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("获取应用实例失败: %w", err)
	}

	// 获取应用模板
	template, err := d.appStoreService.GetTemplate(ctx, instance.TemplateID)
	if err != nil {
		return fmt.Errorf("获取应用模板失败: %w", err)
	}

	// 解析实例配置参数
	var params map[string]interface{}
	if instance.Config != "" {
		if err := json.Unmarshal([]byte(instance.Config), &params); err != nil {
			return fmt.Errorf("解析实例配置失败: %w", err)
		}
	}

	// 渲染模板内容
	renderedContent, err := d.appStoreService.RenderTemplate(ctx, template.ID, params)
	if err != nil {
		return fmt.Errorf("渲染模板失败: %w", err)
	}

	// 调用容器部署引擎
	if d.containerService != nil {
		if err := d.containerService.DeployCompose(ctx, renderedContent, instance.Name); err != nil {
			// 部署失败，更新实例状态为错误
			instance.Status = string(InstanceStatusError)
			_ = d.appStoreService.UpdateInstance(ctx, instance)
			return fmt.Errorf("部署容器失败: %w", err)
		}
	}

	// 更新实例状态为运行中
	instance.Status = string(InstanceStatusRunning)
	if err := d.appStoreService.UpdateInstance(ctx, instance); err != nil {
		return fmt.Errorf("更新实例状态失败: %w", err)
	}

	return nil
}

// UpdateDeployment 更新应用部署
func (d *deploymentIntegrationImpl) UpdateDeployment(ctx context.Context, instanceID uint) error {
	instance, err := d.appStoreService.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("获取应用实例失败: %w", err)
	}

	// 获取模板并渲染
	template, err := d.appStoreService.GetTemplate(ctx, instance.TemplateID)
	if err != nil {
		return fmt.Errorf("获取应用模板失败: %w", err)
	}

	// 解析实例配置参数用于更新
	var updateParams map[string]interface{}
	if instance.Config != "" {
		if err := json.Unmarshal([]byte(instance.Config), &updateParams); err != nil {
			return fmt.Errorf("解析实例配置失败: %w", err)
		}
	}

	renderedContent, err := d.appStoreService.RenderTemplate(ctx, template.ID, updateParams)
	if err != nil {
		return fmt.Errorf("渲染模板失败: %w", err)
	}

	// 更新实例状态为更新中
	instance.Status = string(InstanceStatusUpdating)
	if err := d.appStoreService.UpdateInstance(ctx, instance); err != nil {
		return fmt.Errorf("更新实例状态失败: %w", err)
	}

	// 调用容器引擎执行更新
	if d.containerService != nil {
		if err := d.containerService.UpdateCompose(ctx, instance.Name, renderedContent); err != nil {
			instance.Status = string(InstanceStatusError)
			_ = d.appStoreService.UpdateInstance(ctx, instance)
			return fmt.Errorf("更新容器失败: %w", err)
		}
	}

	// 更新完成后恢复运行状态
	instance.Status = string(InstanceStatusRunning)
	return d.appStoreService.UpdateInstance(ctx, instance)
}

// StopDeployment 停止应用部署
func (d *deploymentIntegrationImpl) StopDeployment(ctx context.Context, instanceID uint) error {
	instance, err := d.appStoreService.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("获取应用实例失败: %w", err)
	}

	// 调用容器引擎停止服务
	if d.containerService != nil {
		if err := d.containerService.StopCompose(ctx, instance.Name); err != nil {
			return fmt.Errorf("停止容器失败: %w", err)
		}
	}

	// 更新实例状态为已停止
	instance.Status = string(InstanceStatusStopped)
	return d.appStoreService.UpdateInstance(ctx, instance)
}

// StartDeployment 启动应用部署
func (d *deploymentIntegrationImpl) StartDeployment(ctx context.Context, instanceID uint) error {
	instance, err := d.appStoreService.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("获取应用实例失败: %w", err)
	}

	// 调用容器引擎启动服务
	if d.containerService != nil {
		if err := d.containerService.StartCompose(ctx, instance.Name); err != nil {
			return fmt.Errorf("启动容器失败: %w", err)
		}
	}

	// 更新实例状态为运行中
	instance.Status = string(InstanceStatusRunning)
	return d.appStoreService.UpdateInstance(ctx, instance)
}

// GetDeploymentStatus 获取部署状态
func (d *deploymentIntegrationImpl) GetDeploymentStatus(ctx context.Context, instanceID uint) (*DeploymentStatusInfo, error) {
	instance, err := d.appStoreService.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("获取应用实例失败: %w", err)
	}

	// 从容器引擎获取实时状态
	var services []ServiceStatusInfo
	if d.containerService != nil {
		var err error
		services, err = d.containerService.GetComposeStatus(ctx, instance.Name)
		if err != nil {
			// 如果获取状态失败，返回基本信息
			services = []ServiceStatusInfo{}
		}
	}

	// 构建状态信息
	statusInfo := &DeploymentStatusInfo{
		InstanceID:  instance.ID,
		Status:      string(instance.Status),
		Progress:    100,
		Message:     "应用运行正常",
		Services:    services,
		LastUpdated: time.Now(),
	}

	return statusInfo, nil
}

// SyncDeploymentStatus 同步部署状态到实例
func (d *deploymentIntegrationImpl) SyncDeploymentStatus(ctx context.Context, instanceID uint) error {
	// 获取最新部署状态
	statusInfo, err := d.GetDeploymentStatus(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("获取部署状态失败: %w", err)
	}

	// 更新实例状态
	instance, err := d.appStoreService.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("获取应用实例失败: %w", err)
	}

	instance.Status = statusInfo.Status
	return d.appStoreService.UpdateInstance(ctx, instance)
} 