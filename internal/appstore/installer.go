package appstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrInstallationInProgress 安装正在进行中
	ErrInstallationInProgress = errors.New("installation already in progress")
	// ErrInstallationFailed 安装失败
	ErrInstallationFailed = errors.New("installation failed")
	// ErrDependencyNotMet 依赖未满足
	ErrDependencyNotMet = errors.New("dependency not met")
	// ErrPortConflict 端口冲突
	ErrPortConflict = errors.New("port conflict detected")
	// ErrVolumeConflict 数据卷冲突
	ErrVolumeConflict = errors.New("volume conflict detected")
	// ErrResourceInsufficient 资源不足
	ErrResourceInsufficient = errors.New("insufficient resources")
)

// InstallationStatus 安装状态
type InstallationStatus string

const (
	StatusPending    InstallationStatus = "pending"     // 等待中
	StatusValidating InstallationStatus = "validating"  // 验证中
	StatusInstalling InstallationStatus = "installing"  // 安装中
	StatusCompleted  InstallationStatus = "completed"   // 已完成
	StatusFailed     InstallationStatus = "failed"      // 失败
	StatusRollingBack InstallationStatus = "rolling_back" // 回滚中
	StatusRolledBack InstallationStatus = "rolled_back"  // 已回滚
)

// InstallationProgress 安装进度
type InstallationProgress struct {
	ID            string             `json:"id"`
	InstanceID    uint               `json:"instance_id"`
	Status        InstallationStatus `json:"status"`
	CurrentStep   string             `json:"current_step"`
	TotalSteps    int                `json:"total_steps"`
	CompletedSteps int               `json:"completed_steps"`
	Message       string             `json:"message"`
	Error         string             `json:"error,omitempty"`
	StartTime     time.Time          `json:"start_time"`
	EndTime       *time.Time         `json:"end_time,omitempty"`
	RollbackInfo  *RollbackInfo      `json:"rollback_info,omitempty"`
}

// RollbackInfo 回滚信息
type RollbackInfo struct {
	Reason       string    `json:"reason"`
	FailedStep   string    `json:"failed_step"`
	RollbackTime time.Time `json:"rollback_time"`
}

// ConflictInfo 冲突信息
type ConflictInfo struct {
	Type        string   `json:"type"`         // port, volume, service
	Resource    string   `json:"resource"`     // 冲突的资源
	ExistingApp string   `json:"existing_app"` // 已存在的应用
	Resolvable  bool     `json:"resolvable"`   // 是否可自动解决
	Suggestions []string `json:"suggestions"`  // 解决建议
}

// DependencyCheck 依赖检查结果
type DependencyCheck struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Required  bool   `json:"required"`
	Satisfied bool   `json:"satisfied"`
	Message   string `json:"message,omitempty"`
}

// InstallRequest 安装请求
type InstallRequest struct {
	TemplateID   uint                   `json:"template_id"`
	InstanceName string                 `json:"instance_name"`
	Parameters   map[string]interface{} `json:"parameters"`
	UserID       uint                   `json:"user_id"`
	TenantID     uint                   `json:"tenant_id"`
	AutoResolve  bool                   `json:"auto_resolve"` // 是否自动解决冲突
}

// InstallResult 安装结果
type InstallResult struct {
	InstanceID   uint                   `json:"instance_id"`
	ProgressID   string                 `json:"progress_id"`
	Status       InstallationStatus     `json:"status"`
	Message      string                 `json:"message"`
	Conflicts    []ConflictInfo         `json:"conflicts,omitempty"`
	Dependencies []DependencyCheck      `json:"dependencies,omitempty"`
}

// UninstallRequest 卸载请求
type UninstallRequest struct {
	InstanceID uint `json:"instance_id"`
	Force      bool `json:"force"` // 强制卸载，忽略依赖检查
}

// InstallerService 安装器服务接口
type InstallerService interface {
	// 安装应用
	Install(ctx context.Context, req *InstallRequest) (*InstallResult, error)
	
	// 卸载应用
	Uninstall(ctx context.Context, req *UninstallRequest) error
	
	// 检查依赖
	CheckDependencies(ctx context.Context, templateID uint) ([]DependencyCheck, error)
	
	// 检测冲突
	DetectConflicts(ctx context.Context, templateID uint, params map[string]interface{}) ([]ConflictInfo, error)
	
	// 获取安装进度
	GetProgress(ctx context.Context, progressID string) (*InstallationProgress, error)
	
	// 回滚安装
	Rollback(ctx context.Context, instanceID uint) error
}

// installerServiceImpl 安装器服务实现
type installerServiceImpl struct {
	appStoreService AppStoreService
	progressStore   *ProgressStore
	conflictChecker *ConflictChecker
	dependencyMgr   *DependencyManager
	mu              sync.RWMutex
}

// NewInstallerService 创建安装器服务实例
func NewInstallerService(appStoreService AppStoreService) InstallerService {
	return &installerServiceImpl{
		appStoreService: appStoreService,
		progressStore:   NewProgressStore(),
		conflictChecker: NewConflictChecker(appStoreService),
		dependencyMgr:   NewDependencyManager(appStoreService),
	}
}

// Install 安装应用
func (s *installerServiceImpl) Install(ctx context.Context, req *InstallRequest) (*InstallResult, error) {
	if req == nil {
		return nil, errors.New("install request is nil")
	}

	// 获取模板
	template, err := s.appStoreService.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// 检查依赖
	depChecks, err := s.CheckDependencies(ctx, req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to check dependencies: %w", err)
	}

	// 检查是否有未满足的必需依赖
	for _, dep := range depChecks {
		if dep.Required && !dep.Satisfied {
			return &InstallResult{
				Status:       StatusFailed,
				Message:      fmt.Sprintf("dependency not met: %s", dep.Name),
				Dependencies: depChecks,
			}, ErrDependencyNotMet
		}
	}

	// 检测冲突
	conflicts, err := s.DetectConflicts(ctx, req.TemplateID, req.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to detect conflicts: %w", err)
	}

	// 如果有不可解决的冲突，返回错误
	hasUnresolvableConflict := false
	for _, conflict := range conflicts {
		if !conflict.Resolvable {
			hasUnresolvableConflict = true
			break
		}
	}

	if hasUnresolvableConflict && !req.AutoResolve {
		return &InstallResult{
			Status:    StatusFailed,
			Message:   "conflicts detected, please resolve manually or enable auto_resolve",
			Conflicts: conflicts,
		}, ErrPortConflict
	}

	// 创建应用实例
	instance := &ApplicationInstance{
		Name:       req.InstanceName,
		TemplateID: req.TemplateID,
		Version:    template.Version,
		Status:     "installing",
		UserID:     req.UserID,
		TenantID:   req.TenantID,
	}

	// 序列化配置
	configJSON, err := json.Marshal(req.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	instance.Config = string(configJSON)

	// 创建实例记录
	if err := s.appStoreService.CreateInstance(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// 创建安装进度
	progress := s.progressStore.Create(instance.ID)

	// 异步执行安装
	go s.executeInstallation(context.Background(), instance, template, req.Parameters, progress)

	return &InstallResult{
		InstanceID:   instance.ID,
		ProgressID:   progress.ID,
		Status:       StatusPending,
		Message:      "installation started",
		Conflicts:    conflicts,
		Dependencies: depChecks,
	}, nil
}

// executeInstallation 执行安装过程
func (s *installerServiceImpl) executeInstallation(ctx context.Context, instance *ApplicationInstance, template *AppTemplate, params map[string]interface{}, progress *InstallationProgress) {
	// 更新进度：验证中
	s.progressStore.Update(progress.ID, StatusValidating, "Validating template and parameters", 1, 5)

	// 验证模板
	if err := s.appStoreService.ValidateTemplate(ctx, template); err != nil {
		s.handleInstallationError(ctx, instance, progress, "template validation", err)
		return
	}

	// 更新进度：渲染模板
	s.progressStore.Update(progress.ID, StatusInstalling, "Rendering template", 2, 5)

	// 渲染模板
	rendered, err := s.appStoreService.RenderTemplate(ctx, template.ID, params)
	if err != nil {
		s.handleInstallationError(ctx, instance, progress, "template rendering", err)
		return
	}

	// 更新进度：部署应用
	s.progressStore.Update(progress.ID, StatusInstalling, "Deploying application", 3, 5)

	// 这里应该调用实际的部署逻辑（Docker、Kubernetes等）
	// 为了演示，我们模拟部署过程
	if err := s.deployApplication(ctx, instance, rendered); err != nil {
		s.handleInstallationError(ctx, instance, progress, "application deployment", err)
		return
	}

	// 更新进度：验证部署
	s.progressStore.Update(progress.ID, StatusInstalling, "Verifying deployment", 4, 5)

	// 验证部署是否成功
	if err := s.verifyDeployment(ctx, instance); err != nil {
		s.handleInstallationError(ctx, instance, progress, "deployment verification", err)
		return
	}

	// 更新进度：完成
	s.progressStore.Update(progress.ID, StatusCompleted, "Installation completed successfully", 5, 5)

	// 更新实例状态
	instance.Status = "running"
	if err := s.appStoreService.UpdateInstance(ctx, instance); err != nil {
		// 记录错误但不回滚，因为应用已经部署成功
		s.progressStore.SetError(progress.ID, fmt.Sprintf("failed to update instance status: %v", err))
	}
}

// handleInstallationError 处理安装错误
func (s *installerServiceImpl) handleInstallationError(ctx context.Context, instance *ApplicationInstance, progress *InstallationProgress, step string, err error) {
	// 更新进度为失败
	s.progressStore.Update(progress.ID, StatusFailed, fmt.Sprintf("Failed at step: %s", step), progress.CompletedSteps, progress.TotalSteps)
	s.progressStore.SetError(progress.ID, err.Error())

	// 更新实例状态
	instance.Status = "error"
	if updateErr := s.appStoreService.UpdateInstance(ctx, instance); updateErr != nil {
		// 记录更新错误
		s.progressStore.SetError(progress.ID, fmt.Sprintf("failed to update instance status: %v", updateErr))
	}

	// 尝试回滚
	if rollbackErr := s.performRollback(ctx, instance, step); rollbackErr != nil {
		s.progressStore.SetError(progress.ID, fmt.Sprintf("rollback failed: %v", rollbackErr))
	}
}

// deployApplication 部署应用（模拟实现）
func (s *installerServiceImpl) deployApplication(ctx context.Context, instance *ApplicationInstance, rendered string) error {
	// 这里应该根据模板类型调用相应的部署逻辑
	// 例如：Docker Compose、Kubernetes 等
	// 为了演示，我们只是简单地返回成功
	
	// 模拟部署延迟
	time.Sleep(100 * time.Millisecond)
	
	return nil
}

// verifyDeployment 验证部署
func (s *installerServiceImpl) verifyDeployment(ctx context.Context, instance *ApplicationInstance) error {
	// 这里应该检查应用是否正常运行
	// 例如：检查容器状态、健康检查等
	
	// 模拟验证延迟
	time.Sleep(50 * time.Millisecond)
	
	return nil
}

// performRollback 执行回滚
func (s *installerServiceImpl) performRollback(ctx context.Context, instance *ApplicationInstance, failedStep string) error {
	// 这里应该实现实际的回滚逻辑
	// 例如：停止并删除已创建的容器、清理数据卷等
	
	return nil
}

// Uninstall 卸载应用
func (s *installerServiceImpl) Uninstall(ctx context.Context, req *UninstallRequest) error {
	if req == nil {
		return errors.New("uninstall request is nil")
	}

	// 获取实例
	instance, err := s.appStoreService.GetInstance(ctx, req.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// 如果不是强制卸载，检查是否有其他应用依赖此应用
	if !req.Force {
		// 这里应该检查依赖关系
		// 如果有其他应用依赖此应用，返回错误
	}

	// 停止应用
	if err := s.stopApplication(ctx, instance); err != nil {
		return fmt.Errorf("failed to stop application: %w", err)
	}

	// 清理资源
	if err := s.cleanupResources(ctx, instance); err != nil {
		return fmt.Errorf("failed to cleanup resources: %w", err)
	}

	// 删除实例记录
	if err := s.appStoreService.DeleteInstance(ctx, req.InstanceID); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	return nil
}

// stopApplication 停止应用
func (s *installerServiceImpl) stopApplication(ctx context.Context, instance *ApplicationInstance) error {
	// 这里应该实现实际的停止逻辑
	// 例如：停止容器、删除 Kubernetes 资源等
	
	return nil
}

// cleanupResources 清理资源
func (s *installerServiceImpl) cleanupResources(ctx context.Context, instance *ApplicationInstance) error {
	// 这里应该清理相关资源
	// 例如：删除数据卷、清理网络配置等
	
	return nil
}

// CheckDependencies 检查依赖
func (s *installerServiceImpl) CheckDependencies(ctx context.Context, templateID uint) ([]DependencyCheck, error) {
	return s.dependencyMgr.CheckDependencies(ctx, templateID)
}

// DetectConflicts 检测冲突
func (s *installerServiceImpl) DetectConflicts(ctx context.Context, templateID uint, params map[string]interface{}) ([]ConflictInfo, error) {
	return s.conflictChecker.DetectConflicts(ctx, templateID, params)
}

// GetProgress 获取安装进度
func (s *installerServiceImpl) GetProgress(ctx context.Context, progressID string) (*InstallationProgress, error) {
	progress := s.progressStore.Get(progressID)
	if progress == nil {
		return nil, errors.New("progress not found")
	}
	return progress, nil
}

// Rollback 回滚安装
func (s *installerServiceImpl) Rollback(ctx context.Context, instanceID uint) error {
	// 获取实例
	instance, err := s.appStoreService.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// 执行回滚
	if err := s.performRollback(ctx, instance, "manual rollback"); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	// 更新实例状态
	instance.Status = "rolled_back"
	if err := s.appStoreService.UpdateInstance(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	return nil
}
