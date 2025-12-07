package container

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// DeploymentService 部署服务接口
type DeploymentService interface {
	// 部署管理
	Deploy(ctx context.Context, projectID uint, config *DeploymentConfig) (*Deployment, error)
	GetDeployment(ctx context.Context, id uint) (*Deployment, error)
	ListDeployments(ctx context.Context, projectID uint) ([]*Deployment, error)
	
	// 部署控制
	RollbackDeployment(ctx context.Context, deploymentID uint) error
	CancelDeployment(ctx context.Context, deploymentID uint) error
	
	// 状态监控
	GetDeploymentStatus(ctx context.Context, deploymentID uint) (*DeploymentStatus, error)
	GetDeploymentEvents(ctx context.Context, deploymentID uint) ([]*DeploymentEvent, error)
	GetServiceInstances(ctx context.Context, deploymentID uint) ([]*ServiceInstance, error)
}

// deploymentServiceImpl 部署服务实现
type deploymentServiceImpl struct {
	db              *gorm.DB
	composeService  ComposeService
	dockerExecutor  DockerExecutor
	healingService  SelfHealingService
}

// NewDeploymentService 创建部署服务实例
func NewDeploymentService(db *gorm.DB, composeService ComposeService, dockerExecutor DockerExecutor) DeploymentService {
	return &deploymentServiceImpl{
		db:             db,
		composeService: composeService,
		dockerExecutor: dockerExecutor,
	}
}

// SetHealingService 设置自愈服务（用于依赖注入）
func (s *deploymentServiceImpl) SetHealingService(healingService SelfHealingService) {
	s.healingService = healingService
}

// Deploy 执行部署
func (s *deploymentServiceImpl) Deploy(ctx context.Context, projectID uint, config *DeploymentConfig) (*Deployment, error) {
	// 获取项目
	project, err := s.composeService.GetProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// 解析 Compose 配置
	composeConfig, err := s.composeService.ParseComposeFile(ctx, project.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// 设置默认配置
	if config == nil {
		config = &DeploymentConfig{
			Strategy:           DeployStrategyRecreate,
			MaxSurge:           1,
			MaxUnavailable:     0,
			HealthCheckDelay:   10,
			HealthCheckRetries: 3,
			RollbackOnFailure:  true,
		}
	}

	// 创建部署记录
	now := time.Now()
	deployment := &Deployment{
		ProjectID:  projectID,
		Version:    fmt.Sprintf("v%d", time.Now().Unix()),
		Strategy:   config.Strategy,
		Status:     DeploymentStatusPending,
		Progress:   0,
		StartedAt:  &now,
		UserID:     project.UserID,
		TenantID:   project.TenantID,
	}

	if err := s.db.WithContext(ctx).Create(deployment).Error; err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	// 记录部署开始事件
	s.recordEvent(ctx, deployment.ID, "deployment_started", "", 
		fmt.Sprintf("开始部署项目 %s，策略: %s", project.Name, config.Strategy), "")

	// 异步执行部署
	go s.executeDeployment(context.Background(), deployment, project, composeConfig, config)

	return deployment, nil
}

// executeDeployment 执行部署逻辑
func (s *deploymentServiceImpl) executeDeployment(ctx context.Context, deployment *Deployment, 
	project *ComposeProject, config *ComposeConfig, deployConfig *DeploymentConfig) {
	
	// 更新状态为进行中
	s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, 10, "开始部署...")

	var err error
	switch deployConfig.Strategy {
	case DeployStrategyRecreate:
		err = s.deployRecreate(ctx, deployment, project, config, deployConfig)
	case DeployStrategyRollingUpdate:
		err = s.deployRollingUpdate(ctx, deployment, project, config, deployConfig)
	case DeployStrategyBlueGreen:
		err = s.deployBlueGreen(ctx, deployment, project, config, deployConfig)
	default:
		err = fmt.Errorf("unsupported deployment strategy: %s", deployConfig.Strategy)
	}

	if err != nil {
		s.handleDeploymentFailure(ctx, deployment, err, deployConfig)
		return
	}

	// 部署成功
	now := time.Now()
	s.db.WithContext(ctx).Model(&Deployment{}).Where("id = ?", deployment.ID).Updates(map[string]interface{}{
		"status":       DeploymentStatusCompleted,
		"progress":     100,
		"message":      "部署成功完成",
		"completed_at": &now,
	})

	s.recordEvent(ctx, deployment.ID, "deployment_completed", "", "部署成功完成", "")
}

// deployRecreate 重建策略部署
func (s *deploymentServiceImpl) deployRecreate(ctx context.Context, deployment *Deployment, 
	project *ComposeProject, config *ComposeConfig, deployConfig *DeploymentConfig) error {
	
	s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, 20, "停止现有服务...")
	
	// 1. 停止并删除现有容器
	if err := s.dockerExecutor.StopProject(ctx, project.Name); err != nil {
		return fmt.Errorf("failed to stop project: %w", err)
	}
	
	s.recordEvent(ctx, deployment.ID, "services_stopped", "", "已停止所有现有服务", "")
	s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, 40, "删除旧容器...")
	
	if err := s.dockerExecutor.RemoveProject(ctx, project.Name); err != nil {
		return fmt.Errorf("failed to remove project: %w", err)
	}
	
	s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, 60, "启动新服务...")
	
	// 2. 启动新容器
	if err := s.dockerExecutor.StartProject(ctx, project.Name, project.Content); err != nil {
		return fmt.Errorf("failed to start project: %w", err)
	}
	
	s.recordEvent(ctx, deployment.ID, "services_started", "", "已启动所有新服务", "")
	s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, 80, "验证服务健康状态...")
	
	// 3. 健康检查
	if err := s.waitForHealthy(ctx, deployment, config, deployConfig); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	
	// 4. 记录服务实例
	if err := s.recordServiceInstances(ctx, deployment, project.Name, config); err != nil {
		return fmt.Errorf("failed to record service instances: %w", err)
	}
	
	return nil
}

// deployRollingUpdate 滚动更新策略部署
func (s *deploymentServiceImpl) deployRollingUpdate(ctx context.Context, deployment *Deployment, 
	project *ComposeProject, config *ComposeConfig, deployConfig *DeploymentConfig) error {
	
	s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, 20, "开始滚动更新...")
	
	totalServices := len(config.Services)
	completedServices := 0
	
	// 逐个服务进行滚动更新
	for serviceName, service := range config.Services {
		s.recordEvent(ctx, deployment.ID, "service_updating", serviceName, 
			fmt.Sprintf("开始更新服务: %s", serviceName), "")
		
		// 1. 启动新版本容器
		newContainerID, err := s.dockerExecutor.StartService(ctx, project.Name, serviceName, service)
		if err != nil {
			return fmt.Errorf("failed to start new service %s: %w", serviceName, err)
		}
		
		// 2. 等待新容器健康
		if err := s.waitForServiceHealthy(ctx, newContainerID, deployConfig); err != nil {
			// 清理新容器
			s.dockerExecutor.StopContainer(ctx, newContainerID)
			s.dockerExecutor.RemoveContainer(ctx, newContainerID)
			return fmt.Errorf("new service %s health check failed: %w", serviceName, err)
		}
		
		// 3. 停止旧版本容器
		oldContainers, err := s.dockerExecutor.GetServiceContainers(ctx, project.Name, serviceName)
		if err == nil {
			for _, containerID := range oldContainers {
				if containerID != newContainerID {
					s.dockerExecutor.StopContainer(ctx, containerID)
					s.dockerExecutor.RemoveContainer(ctx, containerID)
				}
			}
		}
		
		completedServices++
		progress := 20 + (60 * completedServices / totalServices)
		s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, progress, 
			fmt.Sprintf("已更新 %d/%d 个服务", completedServices, totalServices))
		
		s.recordEvent(ctx, deployment.ID, "service_updated", serviceName, 
			fmt.Sprintf("服务 %s 更新完成", serviceName), "")
	}
	
	s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, 90, "记录服务实例...")
	
	// 记录服务实例
	if err := s.recordServiceInstances(ctx, deployment, project.Name, config); err != nil {
		return fmt.Errorf("failed to record service instances: %w", err)
	}
	
	return nil
}

// deployBlueGreen 蓝绿部署策略
func (s *deploymentServiceImpl) deployBlueGreen(ctx context.Context, deployment *Deployment, 
	project *ComposeProject, config *ComposeConfig, deployConfig *DeploymentConfig) error {
	
	s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, 20, "准备绿色环境...")
	
	greenProjectName := fmt.Sprintf("%s-green-%d", project.Name, time.Now().Unix())
	
	// 1. 部署绿色环境
	s.recordEvent(ctx, deployment.ID, "green_deployment_started", "", "开始部署绿色环境", "")
	
	if err := s.dockerExecutor.StartProject(ctx, greenProjectName, project.Content); err != nil {
		return fmt.Errorf("failed to start green environment: %w", err)
	}
	
	s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, 50, "验证绿色环境...")
	
	// 2. 验证绿色环境健康
	if err := s.waitForHealthy(ctx, deployment, config, deployConfig); err != nil {
		// 清理绿色环境
		s.dockerExecutor.StopProject(ctx, greenProjectName)
		s.dockerExecutor.RemoveProject(ctx, greenProjectName)
		return fmt.Errorf("green environment health check failed: %w", err)
	}
	
	s.recordEvent(ctx, deployment.ID, "green_deployment_healthy", "", "绿色环境健康检查通过", "")
	s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, 70, "切换流量到绿色环境...")
	
	// 3. 切换流量（这里简化处理，实际需要负载均衡器配置）
	// TODO: 实现流量切换逻辑
	
	s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, 80, "停止蓝色环境...")
	
	// 4. 停止并删除蓝色环境
	if err := s.dockerExecutor.StopProject(ctx, project.Name); err != nil {
		// 记录警告但不失败
		s.recordEvent(ctx, deployment.ID, "blue_cleanup_warning", "", 
			fmt.Sprintf("停止蓝色环境时出现警告: %v", err), "")
	}
	
	if err := s.dockerExecutor.RemoveProject(ctx, project.Name); err != nil {
		s.recordEvent(ctx, deployment.ID, "blue_cleanup_warning", "", 
			fmt.Sprintf("删除蓝色环境时出现警告: %v", err), "")
	}
	
	// 5. 重命名绿色环境为主环境
	// TODO: 实现容器重命名逻辑
	
	s.recordEvent(ctx, deployment.ID, "traffic_switched", "", "流量已切换到新环境", "")
	s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusInProgress, 90, "记录服务实例...")
	
	// 记录服务实例
	if err := s.recordServiceInstances(ctx, deployment, project.Name, config); err != nil {
		return fmt.Errorf("failed to record service instances: %w", err)
	}
	
	return nil
}

// waitForHealthy 等待所有服务健康
func (s *deploymentServiceImpl) waitForHealthy(ctx context.Context, deployment *Deployment, 
	config *ComposeConfig, deployConfig *DeploymentConfig) error {
	
	time.Sleep(time.Duration(deployConfig.HealthCheckDelay) * time.Second)
	
	for serviceName := range config.Services {
		containers, err := s.dockerExecutor.GetServiceContainers(ctx, "", serviceName)
		if err != nil {
			return fmt.Errorf("failed to get containers for service %s: %w", serviceName, err)
		}
		
		for _, containerID := range containers {
			if err := s.waitForServiceHealthy(ctx, containerID, deployConfig); err != nil {
				return fmt.Errorf("service %s (container %s) is not healthy: %w", 
					serviceName, containerID, err)
			}
		}
	}
	
	return nil
}

// waitForServiceHealthy 等待单个服务健康
func (s *deploymentServiceImpl) waitForServiceHealthy(ctx context.Context, containerID string, 
	deployConfig *DeploymentConfig) error {
	
	for i := 0; i < deployConfig.HealthCheckRetries; i++ {
		status, err := s.dockerExecutor.GetContainerStatus(ctx, containerID)
		if err != nil {
			return err
		}
		
		if status == "running" || status == "healthy" {
			return nil
		}
		
		if status == "exited" || status == "dead" {
			return fmt.Errorf("container is in %s state", status)
		}
		
		time.Sleep(5 * time.Second)
	}
	
	return fmt.Errorf("health check timeout after %d retries", deployConfig.HealthCheckRetries)
}

// recordServiceInstances 记录服务实例
func (s *deploymentServiceImpl) recordServiceInstances(ctx context.Context, deployment *Deployment, 
	projectName string, config *ComposeConfig) error {
	
	for serviceName, service := range config.Services {
		containers, err := s.dockerExecutor.GetServiceContainers(ctx, projectName, serviceName)
		if err != nil {
			continue // 跳过错误，继续处理其他服务
		}
		
		for _, containerID := range containers {
			info, err := s.dockerExecutor.GetContainerInfo(ctx, containerID)
			if err != nil {
				continue
			}
			
			instance := &ServiceInstance{
				DeploymentID:  deployment.ID,
				ServiceName:   serviceName,
				ContainerID:   containerID,
				ContainerName: info.Name,
				Image:         info.Image,
				Status:        info.Status,
				Health:        info.Health,
				StartedAt:     info.StartedAt,
			}
			
			s.db.WithContext(ctx).Create(instance)
			
			// 注册到自愈服务
			if s.healingService != nil {
				healingConfig := s.buildHealingConfig(service)
				if err := s.healingService.RegisterContainer(ctx, containerID, healingConfig); err != nil {
					// 记录警告但不失败
					s.recordEvent(ctx, deployment.ID, "healing_registration_warning", serviceName,
						fmt.Sprintf("注册容器到自愈服务时出现警告: %v", err), "")
				}
			}
		}
	}
	
	return nil
}

// buildHealingConfig 根据服务配置构建自愈配置
func (s *deploymentServiceImpl) buildHealingConfig(service *Service) *HealingConfig {
	config := DefaultHealingConfig()
	
	// 根据服务的重启策略调整自愈配置
	if service.Restart != "" {
		switch service.Restart {
		case "no":
			config.AutoRestart = false
		case "always", "unless-stopped":
			config.AutoRestart = true
			config.MaxRestarts = 10 // 增加重启次数
		case "on-failure":
			config.AutoRestart = true
		}
	}
	
	// 如果服务有健康检查配置，使用它
	if service.HealthCheck != nil {
		if service.HealthCheck.Interval != "" {
			// 解析间隔时间（简化处理）
			// 实际应该解析 "30s", "1m" 等格式
			config.CheckInterval = 30
		}
		if service.HealthCheck.Retries > 0 {
			config.FailureThreshold = service.HealthCheck.Retries
		}
	}
	
	return config
}

// handleDeploymentFailure 处理部署失败
func (s *deploymentServiceImpl) handleDeploymentFailure(ctx context.Context, deployment *Deployment, 
	err error, config *DeploymentConfig) {
	
	s.recordEvent(ctx, deployment.ID, "deployment_failed", "", 
		fmt.Sprintf("部署失败: %v", err), "")
	
	if config.RollbackOnFailure {
		s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusRollingBack, 0, 
			"部署失败，开始自动回滚...")
		
		if rollbackErr := s.performRollback(ctx, deployment); rollbackErr != nil {
			s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusFailed, 0, 
				fmt.Sprintf("部署失败且回滚失败: %v, 回滚错误: %v", err, rollbackErr))
		} else {
			s.updateDeploymentStatus(ctx, deployment.ID, DeploymentStatusRolledBack, 0, 
				fmt.Sprintf("部署失败，已自动回滚: %v", err))
		}
	} else {
		now := time.Now()
		s.db.WithContext(ctx).Model(&Deployment{}).Where("id = ?", deployment.ID).Updates(map[string]interface{}{
			"status":       DeploymentStatusFailed,
			"message":      fmt.Sprintf("部署失败: %v", err),
			"completed_at": &now,
		})
	}
}

// RollbackDeployment 回滚部署
func (s *deploymentServiceImpl) RollbackDeployment(ctx context.Context, deploymentID uint) error {
	deployment, err := s.GetDeployment(ctx, deploymentID)
	if err != nil {
		return err
	}
	
	if deployment.Status != DeploymentStatusCompleted && deployment.Status != DeploymentStatusFailed {
		return fmt.Errorf("cannot rollback deployment in %s status", deployment.Status)
	}
	
	s.updateDeploymentStatus(ctx, deploymentID, DeploymentStatusRollingBack, 0, "开始回滚...")
	s.recordEvent(ctx, deploymentID, "rollback_started", "", "手动触发回滚", "")
	
	if err := s.performRollback(ctx, deployment); err != nil {
		s.updateDeploymentStatus(ctx, deploymentID, DeploymentStatusFailed, 0, 
			fmt.Sprintf("回滚失败: %v", err))
		return err
	}
	
	s.updateDeploymentStatus(ctx, deploymentID, DeploymentStatusRolledBack, 100, "回滚完成")
	s.recordEvent(ctx, deploymentID, "rollback_completed", "", "回滚成功完成", "")
	
	return nil
}

// performRollback 执行回滚
func (s *deploymentServiceImpl) performRollback(ctx context.Context, deployment *Deployment) error {
	// 查找上一个成功的部署
	var previousDeployment Deployment
	err := s.db.WithContext(ctx).
		Where("project_id = ? AND id < ? AND status = ?", 
			deployment.ProjectID, deployment.ID, DeploymentStatusCompleted).
		Order("id DESC").
		First(&previousDeployment).Error
	
	if err != nil {
		return fmt.Errorf("no previous successful deployment found: %w", err)
	}
	
	// 获取项目
	project, err := s.composeService.GetProject(ctx, deployment.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}
	
	// 停止当前部署
	if err := s.dockerExecutor.StopProject(ctx, project.Name); err != nil {
		return fmt.Errorf("failed to stop current deployment: %w", err)
	}
	
	if err := s.dockerExecutor.RemoveProject(ctx, project.Name); err != nil {
		return fmt.Errorf("failed to remove current deployment: %w", err)
	}
	
	// 启动上一个版本
	// 注意：这里简化处理，实际应该保存每个部署的完整配置
	if err := s.dockerExecutor.StartProject(ctx, project.Name, project.Content); err != nil {
		return fmt.Errorf("failed to start previous version: %w", err)
	}
	
	return nil
}

// CancelDeployment 取消部署
func (s *deploymentServiceImpl) CancelDeployment(ctx context.Context, deploymentID uint) error {
	deployment, err := s.GetDeployment(ctx, deploymentID)
	if err != nil {
		return err
	}
	
	if deployment.Status != DeploymentStatusPending && deployment.Status != DeploymentStatusInProgress {
		return fmt.Errorf("cannot cancel deployment in %s status", deployment.Status)
	}
	
	// 更新状态
	now := time.Now()
	s.db.WithContext(ctx).Model(&Deployment{}).Where("id = ?", deploymentID).Updates(map[string]interface{}{
		"status":       DeploymentStatusFailed,
		"message":      "部署已被用户取消",
		"completed_at": &now,
	})
	
	s.recordEvent(ctx, deploymentID, "deployment_cancelled", "", "部署已被用户取消", "")
	
	return nil
}

// GetDeployment 获取部署
func (s *deploymentServiceImpl) GetDeployment(ctx context.Context, id uint) (*Deployment, error) {
	var deployment Deployment
	if err := s.db.WithContext(ctx).Preload("Project").First(&deployment, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}
	return &deployment, nil
}

// ListDeployments 列出部署
func (s *deploymentServiceImpl) ListDeployments(ctx context.Context, projectID uint) ([]*Deployment, error) {
	var deployments []*Deployment
	query := s.db.WithContext(ctx).Model(&Deployment{})
	
	if projectID > 0 {
		query = query.Where("project_id = ?", projectID)
	}
	
	if err := query.Order("created_at DESC").Find(&deployments).Error; err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	
	return deployments, nil
}

// GetDeploymentStatus 获取部署状态
func (s *deploymentServiceImpl) GetDeploymentStatus(ctx context.Context, deploymentID uint) (*DeploymentStatus, error) {
	deployment, err := s.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	
	return &deployment.Status, nil
}

// GetDeploymentEvents 获取部署事件
func (s *deploymentServiceImpl) GetDeploymentEvents(ctx context.Context, deploymentID uint) ([]*DeploymentEvent, error) {
	var events []*DeploymentEvent
	if err := s.db.WithContext(ctx).
		Where("deployment_id = ?", deploymentID).
		Order("created_at ASC").
		Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to get deployment events: %w", err)
	}
	return events, nil
}

// GetServiceInstances 获取服务实例
func (s *deploymentServiceImpl) GetServiceInstances(ctx context.Context, deploymentID uint) ([]*ServiceInstance, error) {
	var instances []*ServiceInstance
	if err := s.db.WithContext(ctx).
		Where("deployment_id = ?", deploymentID).
		Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("failed to get service instances: %w", err)
	}
	return instances, nil
}

// updateDeploymentStatus 更新部署状态
func (s *deploymentServiceImpl) updateDeploymentStatus(ctx context.Context, deploymentID uint, 
	status DeploymentStatus, progress int, message string) {
	s.db.WithContext(ctx).Model(&Deployment{}).Where("id = ?", deploymentID).Updates(map[string]interface{}{
		"status":   status,
		"progress": progress,
		"message":  message,
	})
}

// recordEvent 记录部署事件
func (s *deploymentServiceImpl) recordEvent(ctx context.Context, deploymentID uint, 
	eventType, serviceName, message, details string) {
	event := &DeploymentEvent{
		DeploymentID: deploymentID,
		EventType:    eventType,
		ServiceName:  serviceName,
		Message:      message,
		Details:      details,
	}
	s.db.WithContext(ctx).Create(event)
}
