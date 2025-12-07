package container

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// SelfHealingService 容器自愈服务接口
type SelfHealingService interface {
	// 启动自愈监控
	Start(ctx context.Context) error
	// 停止自愈监控
	Stop() error
	// 注册需要监控的容器
	RegisterContainer(ctx context.Context, containerID string, config *HealingConfig) error
	// 取消注册容器
	UnregisterContainer(ctx context.Context, containerID string) error
	// 获取容器健康状态
	GetContainerHealth(ctx context.Context, containerID string) (*HealthStatus, error)
	// 获取故障历史
	GetFailureHistory(ctx context.Context, containerID string, limit int) ([]*FailureRecord, error)
}

// HealingConfig 自愈配置
type HealingConfig struct {
	// 健康检查间隔（秒）
	CheckInterval int `json:"check_interval"`
	// 健康检查超时（秒）
	CheckTimeout int `json:"check_timeout"`
	// 失败阈值（连续失败多少次触发自愈）
	FailureThreshold int `json:"failure_threshold"`
	// 最大重启次数（在时间窗口内）
	MaxRestarts int `json:"max_restarts"`
	// 重启时间窗口（秒）
	RestartWindow int `json:"restart_window"`
	// 是否启用自动重启
	AutoRestart bool `json:"auto_restart"`
	// 是否发送告警通知
	SendAlert bool `json:"send_alert"`
}

// DefaultHealingConfig 默认自愈配置
func DefaultHealingConfig() *HealingConfig {
	return &HealingConfig{
		CheckInterval:    30,  // 30秒检查一次
		CheckTimeout:     10,  // 10秒超时
		FailureThreshold: 3,   // 连续失败3次触发自愈
		MaxRestarts:      5,   // 5分钟内最多重启5次
		RestartWindow:    300, // 5分钟时间窗口
		AutoRestart:      true,
		SendAlert:        true,
	}
}

// HealthStatus 健康状态
type HealthStatus struct {
	ContainerID      string    `json:"container_id"`
	Status           string    `json:"status"` // healthy, unhealthy, unknown
	LastCheckTime    time.Time `json:"last_check_time"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	TotalRestarts    int       `json:"total_restarts"`
	LastRestartTime  *time.Time `json:"last_restart_time,omitempty"`
	Message          string    `json:"message"`
}

// FailureRecord 故障记录
type FailureRecord struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	ContainerID  string    `json:"container_id" gorm:"index;not null"`
	ServiceName  string    `json:"service_name"`
	ProjectName  string    `json:"project_name"`
	FailureType  string    `json:"failure_type"` // health_check_failed, container_stopped, container_error
	ErrorMessage string    `json:"error_message" gorm:"type:text"`
	Details      string    `json:"details" gorm:"type:text"` // JSON格式的详细信息
	Action       string    `json:"action"` // restart, alert, none
	ActionResult string    `json:"action_result"` // success, failed
	DetectedAt   time.Time `json:"detected_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	TenantID     uint      `json:"tenant_id" gorm:"index"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName 指定表名
func (FailureRecord) TableName() string {
	return "container_failure_records"
}

// monitoredContainer 被监控的容器
type monitoredContainer struct {
	containerID string
	config      *HealingConfig
	health      *HealthStatus
	restartTimes []time.Time // 重启时间记录
	mu          sync.RWMutex
}

// selfHealingServiceImpl 自愈服务实现
type selfHealingServiceImpl struct {
	db             *gorm.DB
	executor       DockerExecutor
	containers     map[string]*monitoredContainer
	mu             sync.RWMutex
	stopChan       chan struct{}
	wg             sync.WaitGroup
	notifyService  NotificationService
}

// NotificationService 通知服务接口（用于发送告警）
type NotificationService interface {
	SendAlert(ctx context.Context, alert *Alert) error
}

// Alert 告警信息
type Alert struct {
	Level       string    `json:"level"` // info, warning, error, critical
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	ContainerID string    `json:"container_id"`
	ServiceName string    `json:"service_name"`
	ProjectName string    `json:"project_name"`
	Timestamp   time.Time `json:"timestamp"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// NewSelfHealingService 创建自愈服务实例
func NewSelfHealingService(db *gorm.DB, executor DockerExecutor, notifyService NotificationService) SelfHealingService {
	return &selfHealingServiceImpl{
		db:            db,
		executor:      executor,
		containers:    make(map[string]*monitoredContainer),
		stopChan:      make(chan struct{}),
		notifyService: notifyService,
	}
}

// Start 启动自愈监控
func (s *selfHealingServiceImpl) Start(ctx context.Context) error {
	s.wg.Add(1)
	go s.monitorLoop(ctx)
	return nil
}

// Stop 停止自愈监控
func (s *selfHealingServiceImpl) Stop() error {
	close(s.stopChan)
	s.wg.Wait()
	return nil
}

// RegisterContainer 注册需要监控的容器
func (s *selfHealingServiceImpl) RegisterContainer(ctx context.Context, containerID string, config *HealingConfig) error {
	if config == nil {
		config = DefaultHealingConfig()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.containers[containerID] = &monitoredContainer{
		containerID: containerID,
		config:      config,
		health: &HealthStatus{
			ContainerID:   containerID,
			Status:        "unknown",
			LastCheckTime: time.Now(),
		},
		restartTimes: make([]time.Time, 0),
	}

	return nil
}

// UnregisterContainer 取消注册容器
func (s *selfHealingServiceImpl) UnregisterContainer(ctx context.Context, containerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.containers, containerID)
	return nil
}

// GetContainerHealth 获取容器健康状态
func (s *selfHealingServiceImpl) GetContainerHealth(ctx context.Context, containerID string) (*HealthStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	container, exists := s.containers[containerID]
	if !exists {
		return nil, fmt.Errorf("container %s not registered", containerID)
	}

	container.mu.RLock()
	defer container.mu.RUnlock()

	// 返回健康状态的副本
	health := *container.health
	return &health, nil
}

// GetFailureHistory 获取故障历史
func (s *selfHealingServiceImpl) GetFailureHistory(ctx context.Context, containerID string, limit int) ([]*FailureRecord, error) {
	if limit <= 0 {
		limit = 50
	}

	var records []*FailureRecord
	err := s.db.WithContext(ctx).
		Where("container_id = ?", containerID).
		Order("detected_at DESC").
		Limit(limit).
		Find(&records).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get failure history: %w", err)
	}

	return records, nil
}

// monitorLoop 监控循环
func (s *selfHealingServiceImpl) monitorLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAllContainers(ctx)
		}
	}
}

// checkAllContainers 检查所有容器
func (s *selfHealingServiceImpl) checkAllContainers(ctx context.Context) {
	s.mu.RLock()
	containers := make([]*monitoredContainer, 0, len(s.containers))
	for _, container := range s.containers {
		containers = append(containers, container)
	}
	s.mu.RUnlock()

	for _, container := range containers {
		s.checkContainer(ctx, container)
	}
}

// checkContainer 检查单个容器
func (s *selfHealingServiceImpl) checkContainer(ctx context.Context, container *monitoredContainer) {
	container.mu.Lock()
	defer container.mu.Unlock()

	// 检查是否到了检查时间
	if time.Since(container.health.LastCheckTime) < time.Duration(container.config.CheckInterval)*time.Second {
		return
	}

	// 创建带超时的上下文
	checkCtx, cancel := context.WithTimeout(ctx, time.Duration(container.config.CheckTimeout)*time.Second)
	defer cancel()

	// 获取容器状态
	status, err := s.executor.GetContainerStatus(checkCtx, container.containerID)
	
	container.health.LastCheckTime = time.Now()

	if err != nil {
		// 检查失败
		container.health.ConsecutiveFailures++
		container.health.Status = "unhealthy"
		container.health.Message = fmt.Sprintf("health check failed: %v", err)

		// 记录故障
		s.recordFailure(ctx, container, "health_check_failed", err.Error(), nil)

		// 检查是否需要触发自愈
		if container.health.ConsecutiveFailures >= container.config.FailureThreshold {
			s.healContainer(ctx, container)
		}
		return
	}

	// 检查容器状态
	if status != "running" {
		// 容器未运行
		container.health.ConsecutiveFailures++
		container.health.Status = "unhealthy"
		container.health.Message = fmt.Sprintf("container status: %s", status)

		// 记录故障
		s.recordFailure(ctx, container, "container_stopped", fmt.Sprintf("container status: %s", status), nil)

		// 检查是否需要触发自愈
		if container.health.ConsecutiveFailures >= container.config.FailureThreshold {
			s.healContainer(ctx, container)
		}
		return
	}

	// 容器健康
	if container.health.Status != "healthy" {
		// 从不健康恢复到健康
		container.health.Status = "healthy"
		container.health.Message = "container is healthy"
		
		// 如果之前有未解决的故障记录，标记为已解决
		s.resolveFailures(ctx, container.containerID)
	}
	
	container.health.ConsecutiveFailures = 0
}

// healContainer 自愈容器
func (s *selfHealingServiceImpl) healContainer(ctx context.Context, container *monitoredContainer) {
	if !container.config.AutoRestart {
		// 不自动重启，只发送告警
		if container.config.SendAlert {
			s.sendAlert(ctx, container, "critical", "Container unhealthy", 
				fmt.Sprintf("Container %s is unhealthy but auto-restart is disabled", container.containerID))
		}
		return
	}

	// 检查重启次数限制
	now := time.Now()
	windowStart := now.Add(-time.Duration(container.config.RestartWindow) * time.Second)
	
	// 清理时间窗口外的重启记录
	validRestarts := make([]time.Time, 0)
	for _, t := range container.restartTimes {
		if t.After(windowStart) {
			validRestarts = append(validRestarts, t)
		}
	}
	container.restartTimes = validRestarts

	// 检查是否超过最大重启次数
	if len(container.restartTimes) >= container.config.MaxRestarts {
		// 超过最大重启次数，发送告警
		if container.config.SendAlert {
			s.sendAlert(ctx, container, "critical", "Container restart limit exceeded",
				fmt.Sprintf("Container %s has exceeded max restart limit (%d restarts in %d seconds)",
					container.containerID, container.config.MaxRestarts, container.config.RestartWindow))
		}
		
		// 记录故障
		s.recordFailure(ctx, container, "restart_limit_exceeded",
			fmt.Sprintf("exceeded max restart limit: %d restarts in %d seconds",
				container.config.MaxRestarts, container.config.RestartWindow),
			map[string]interface{}{
				"action": "none",
				"reason": "restart_limit_exceeded",
			})
		
		return
	}

	// 尝试重启容器
	err := s.executor.StartContainer(ctx, container.containerID)
	
	actionResult := "success"
	if err != nil {
		actionResult = "failed"
		
		// 重启失败，发送告警
		if container.config.SendAlert {
			s.sendAlert(ctx, container, "error", "Container restart failed",
				fmt.Sprintf("Failed to restart container %s: %v", container.containerID, err))
		}
		
		// 记录故障
		s.recordFailure(ctx, container, "restart_failed", err.Error(), map[string]interface{}{
			"action": "restart",
			"result": "failed",
		})
	} else {
		// 重启成功
		container.restartTimes = append(container.restartTimes, now)
		container.health.TotalRestarts++
		container.health.LastRestartTime = &now
		container.health.ConsecutiveFailures = 0
		
		// 发送成功通知
		if container.config.SendAlert {
			s.sendAlert(ctx, container, "warning", "Container restarted",
				fmt.Sprintf("Container %s has been automatically restarted", container.containerID))
		}
		
		// 记录故障和恢复
		s.recordFailure(ctx, container, "auto_restart", "container automatically restarted", map[string]interface{}{
			"action": "restart",
			"result": "success",
			"restart_count": len(container.restartTimes),
		})
	}

	// 更新故障记录的操作结果
	s.updateFailureActionResult(ctx, container.containerID, actionResult)
}

// recordFailure 记录故障
func (s *selfHealingServiceImpl) recordFailure(ctx context.Context, container *monitoredContainer, 
	failureType, errorMessage string, details map[string]interface{}) {
	
	// 获取容器信息
	info, err := s.executor.GetContainerInfo(ctx, container.containerID)
	
	serviceName := ""
	projectName := ""
	if err == nil && info != nil {
		serviceName = info.Name
		// 可以从容器标签中提取项目名称
	}

	action := "none"
	if container.config.AutoRestart && failureType != "restart_limit_exceeded" {
		action = "restart"
	} else if container.config.SendAlert {
		action = "alert"
	}

	detailsJSON := ""
	if details != nil {
		// 将 details 转换为 JSON 字符串
		// 这里简化处理，实际应该使用 json.Marshal
		detailsJSON = fmt.Sprintf("%v", details)
	}

	record := &FailureRecord{
		ContainerID:  container.containerID,
		ServiceName:  serviceName,
		ProjectName:  projectName,
		FailureType:  failureType,
		ErrorMessage: errorMessage,
		Details:      detailsJSON,
		Action:       action,
		ActionResult: "pending",
		DetectedAt:   time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		// 记录失败，但不影响主流程
		fmt.Printf("failed to record failure: %v\n", err)
	}
}

// updateFailureActionResult 更新故障记录的操作结果
func (s *selfHealingServiceImpl) updateFailureActionResult(ctx context.Context, containerID, result string) {
	// 更新最近的未解决故障记录
	s.db.WithContext(ctx).
		Model(&FailureRecord{}).
		Where("container_id = ? AND resolved_at IS NULL AND action_result = ?", containerID, "pending").
		Order("detected_at DESC").
		Limit(1).
		Update("action_result", result)
}

// resolveFailures 解决故障记录
func (s *selfHealingServiceImpl) resolveFailures(ctx context.Context, containerID string) {
	now := time.Now()
	s.db.WithContext(ctx).
		Model(&FailureRecord{}).
		Where("container_id = ? AND resolved_at IS NULL", containerID).
		Update("resolved_at", now)
}

// sendAlert 发送告警
func (s *selfHealingServiceImpl) sendAlert(ctx context.Context, container *monitoredContainer, 
	level, title, message string) {
	
	if s.notifyService == nil {
		return
	}

	// 获取容器信息
	info, _ := s.executor.GetContainerInfo(ctx, container.containerID)
	
	serviceName := ""
	projectName := ""
	if info != nil {
		serviceName = info.Name
	}

	alert := &Alert{
		Level:       level,
		Title:       title,
		Message:     message,
		ContainerID: container.containerID,
		ServiceName: serviceName,
		ProjectName: projectName,
		Timestamp:   time.Now(),
		Details: map[string]interface{}{
			"consecutive_failures": container.health.ConsecutiveFailures,
			"total_restarts":      container.health.TotalRestarts,
			"health_status":       container.health.Status,
		},
	}

	if err := s.notifyService.SendAlert(ctx, alert); err != nil {
		fmt.Printf("failed to send alert: %v\n", err)
	}
}
