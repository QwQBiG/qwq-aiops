package container

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	// ErrProjectNotFound 项目未找到
	ErrProjectNotFound = errors.New("compose project not found")
	// ErrProjectAlreadyExists 项目已存在
	ErrProjectAlreadyExists = errors.New("compose project already exists")
	// ErrInvalidComposeFile 无效的 Compose 文件
	ErrInvalidComposeFile = errors.New("invalid compose file")
)

// ComposeService Docker Compose 服务接口
type ComposeService interface {
	// 项目管理
	CreateProject(ctx context.Context, project *ComposeProject) error
	GetProject(ctx context.Context, id uint) (*ComposeProject, error)
	GetProjectByName(ctx context.Context, name string, tenantID uint) (*ComposeProject, error)
	ListProjects(ctx context.Context, userID, tenantID uint) ([]*ComposeProject, error)
	UpdateProject(ctx context.Context, project *ComposeProject) error
	DeleteProject(ctx context.Context, id uint) error

	// Compose 文件操作
	ParseComposeFile(ctx context.Context, content string) (*ComposeConfig, error)
	ValidateComposeFile(ctx context.Context, content string) (*ValidationResult, error)
	RenderComposeConfig(ctx context.Context, config *ComposeConfig) (string, error)

	// 智能提示和补全
	GetCompletions(ctx context.Context, content string, position int) ([]*CompletionItem, error)

	// 可视化编辑
	GetProjectStructure(ctx context.Context, projectID uint) (*ComposeConfig, error)
	UpdateProjectStructure(ctx context.Context, projectID uint, config *ComposeConfig) error
	
	// 部署管理（集成部署服务）
	Deploy(ctx context.Context, projectID uint, config *DeploymentConfig) (*Deployment, error)
	GetDeployment(ctx context.Context, id uint) (*Deployment, error)
	ListDeployments(ctx context.Context, projectID uint) ([]*Deployment, error)
	RollbackDeployment(ctx context.Context, deploymentID uint) error
	GetDeploymentStatus(ctx context.Context, deploymentID uint) (*DeploymentStatus, error)
	
	// AI 架构优化分析
	AnalyzeProjectArchitecture(ctx context.Context, projectID uint) (*ArchitectureAnalysis, error)
	GetOptimizationSuggestions(ctx context.Context, projectID uint) ([]*OptimizationSuggestion, error)
	GetSecurityRecommendations(ctx context.Context, projectID uint) ([]*SecurityRecommendation, error)
	GetArchitectureVisualization(ctx context.Context, projectID uint) (*ArchitectureVisualization, error)
	GetDependencyGraph(ctx context.Context, projectID uint) (*DependencyGraph, error)
	EvaluateProjectPerformance(ctx context.Context, projectID uint) (*PerformanceEvaluation, error)
}

// composeServiceImpl Compose 服务实现
type composeServiceImpl struct {
	db                *gorm.DB
	parser            *ComposeParser
	deploymentService DeploymentService
	optimizer         ArchitectureOptimizer
}

// NewComposeService 创建 Compose 服务实例
func NewComposeService(db *gorm.DB) ComposeService {
	service := &composeServiceImpl{
		db:        db,
		parser:    NewComposeParser(),
		optimizer: NewArchitectureOptimizer(),
	}
	
	// 创建部署服务
	dockerExecutor := NewDockerExecutor()
	service.deploymentService = NewDeploymentService(db, service, dockerExecutor)
	
	return service
}

// CreateProject 创建项目
func (s *composeServiceImpl) CreateProject(ctx context.Context, project *ComposeProject) error {
	if project == nil {
		return errors.New("project is nil")
	}

	// 验证项目名称
	if project.Name == "" {
		return errors.New("project name is required")
	}

	// 检查项目名称是否已存在（同一租户下）
	var count int64
	if err := s.db.WithContext(ctx).Model(&ComposeProject{}).
		Where("name = ? AND tenant_id = ? AND deleted_at IS NULL", project.Name, project.TenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check project existence: %w", err)
	}

	if count > 0 {
		return ErrProjectAlreadyExists
	}

	// 验证 Compose 文件
	if project.Content != "" {
		config, err := s.parser.Parse(project.Content)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidComposeFile, err)
		}

		// 设置版本
		if project.Version == "" {
			project.Version = config.Version
		}

		// 验证配置
		validationResult := s.parser.Validate(config)
		if !validationResult.Valid {
			return fmt.Errorf("%w: validation failed with %d errors", ErrInvalidComposeFile, len(validationResult.Errors))
		}
	}

	// 设置默认状态
	if project.Status == "" {
		project.Status = ProjectStatusDraft
	}

	// 创建项目
	if err := s.db.WithContext(ctx).Create(project).Error; err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

// GetProject 获取项目
func (s *composeServiceImpl) GetProject(ctx context.Context, id uint) (*ComposeProject, error) {
	var project ComposeProject
	if err := s.db.WithContext(ctx).First(&project, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}

// GetProjectByName 根据名称获取项目
func (s *composeServiceImpl) GetProjectByName(ctx context.Context, name string, tenantID uint) (*ComposeProject, error) {
	var project ComposeProject
	if err := s.db.WithContext(ctx).
		Where("name = ? AND tenant_id = ?", name, tenantID).
		First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project by name: %w", err)
	}

	return &project, nil
}

// ListProjects 列出项目
func (s *composeServiceImpl) ListProjects(ctx context.Context, userID, tenantID uint) ([]*ComposeProject, error) {
	var projects []*ComposeProject
	query := s.db.WithContext(ctx).Model(&ComposeProject{})

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.Order("created_at DESC").Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return projects, nil
}

// UpdateProject 更新项目
func (s *composeServiceImpl) UpdateProject(ctx context.Context, project *ComposeProject) error {
	if project == nil {
		return errors.New("project is nil")
	}

	// 验证 Compose 文件
	if project.Content != "" {
		config, err := s.parser.Parse(project.Content)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidComposeFile, err)
		}

		// 更新版本
		project.Version = config.Version

		// 验证配置
		validationResult := s.parser.Validate(config)
		if !validationResult.Valid {
			return fmt.Errorf("%w: validation failed with %d errors", ErrInvalidComposeFile, len(validationResult.Errors))
		}
	}

	// 更新项目
	if err := s.db.WithContext(ctx).Save(project).Error; err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	return nil
}

// DeleteProject 删除项目（软删除）
func (s *composeServiceImpl) DeleteProject(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&ComposeProject{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete project: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}

	return nil
}

// ParseComposeFile 解析 Compose 文件
func (s *composeServiceImpl) ParseComposeFile(ctx context.Context, content string) (*ComposeConfig, error) {
	config, err := s.parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	return config, nil
}

// ValidateComposeFile 验证 Compose 文件
func (s *composeServiceImpl) ValidateComposeFile(ctx context.Context, content string) (*ValidationResult, error) {
	// 先解析
	config, err := s.parser.Parse(content)
	if err != nil {
		return &ValidationResult{
			Valid: false,
			Errors: []*ValidationError{
				{
					Field:   "content",
					Message: err.Error(),
				},
			},
		}, nil
	}

	// 再验证
	result := s.parser.Validate(config)
	return result, nil
}

// RenderComposeConfig 渲染 Compose 配置
func (s *composeServiceImpl) RenderComposeConfig(ctx context.Context, config *ComposeConfig) (string, error) {
	content, err := s.parser.Render(config)
	if err != nil {
		return "", fmt.Errorf("failed to render compose config: %w", err)
	}

	return content, nil
}

// GetCompletions 获取自动补全建议
func (s *composeServiceImpl) GetCompletions(ctx context.Context, content string, position int) ([]*CompletionItem, error) {
	completions := s.parser.GetCompletions(content, position)
	return completions, nil
}

// GetProjectStructure 获取项目结构（用于可视化编辑）
func (s *composeServiceImpl) GetProjectStructure(ctx context.Context, projectID uint) (*ComposeConfig, error) {
	// 获取项目
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 解析 Compose 文件
	config, err := s.parser.Parse(project.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse project content: %w", err)
	}

	return config, nil
}

// UpdateProjectStructure 更新项目结构（从可视化编辑器）
func (s *composeServiceImpl) UpdateProjectStructure(ctx context.Context, projectID uint, config *ComposeConfig) error {
	// 获取项目
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	// 验证配置
	validationResult := s.parser.Validate(config)
	if !validationResult.Valid {
		return fmt.Errorf("invalid compose config: %d validation errors", len(validationResult.Errors))
	}

	// 渲染为 YAML
	content, err := s.parser.Render(config)
	if err != nil {
		return fmt.Errorf("failed to render config: %w", err)
	}

	// 更新项目内容
	project.Content = content
	project.Version = config.Version

	// 保存
	if err := s.UpdateProject(ctx, project); err != nil {
		return err
	}

	return nil
}

// Deploy 部署项目
func (s *composeServiceImpl) Deploy(ctx context.Context, projectID uint, config *DeploymentConfig) (*Deployment, error) {
	return s.deploymentService.Deploy(ctx, projectID, config)
}

// GetDeployment 获取部署
func (s *composeServiceImpl) GetDeployment(ctx context.Context, id uint) (*Deployment, error) {
	return s.deploymentService.GetDeployment(ctx, id)
}

// ListDeployments 列出部署
func (s *composeServiceImpl) ListDeployments(ctx context.Context, projectID uint) ([]*Deployment, error) {
	return s.deploymentService.ListDeployments(ctx, projectID)
}

// RollbackDeployment 回滚部署
func (s *composeServiceImpl) RollbackDeployment(ctx context.Context, deploymentID uint) error {
	return s.deploymentService.RollbackDeployment(ctx, deploymentID)
}

// GetDeploymentStatus 获取部署状态
func (s *composeServiceImpl) GetDeploymentStatus(ctx context.Context, deploymentID uint) (*DeploymentStatus, error) {
	return s.deploymentService.GetDeploymentStatus(ctx, deploymentID)
}

// AnalyzeProjectArchitecture 分析项目架构
func (s *composeServiceImpl) AnalyzeProjectArchitecture(ctx context.Context, projectID uint) (*ArchitectureAnalysis, error) {
	// 获取项目
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 解析 Compose 配置
	config, err := s.parser.Parse(project.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// 执行架构分析
	analysis, err := s.optimizer.AnalyzeArchitecture(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze architecture: %w", err)
	}

	return analysis, nil
}

// GetOptimizationSuggestions 获取优化建议
func (s *composeServiceImpl) GetOptimizationSuggestions(ctx context.Context, projectID uint) ([]*OptimizationSuggestion, error) {
	// 先进行架构分析
	analysis, err := s.AnalyzeProjectArchitecture(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 生成优化建议
	suggestions, err := s.optimizer.GenerateOptimizations(ctx, analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to generate optimizations: %w", err)
	}

	return suggestions, nil
}

// GetSecurityRecommendations 获取安全建议
func (s *composeServiceImpl) GetSecurityRecommendations(ctx context.Context, projectID uint) ([]*SecurityRecommendation, error) {
	// 获取项目
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 解析 Compose 配置
	config, err := s.parser.Parse(project.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// 生成安全建议
	recommendations, err := s.optimizer.GenerateSecurityRecommendations(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate security recommendations: %w", err)
	}

	return recommendations, nil
}

// GetArchitectureVisualization 获取架构可视化数据
func (s *composeServiceImpl) GetArchitectureVisualization(ctx context.Context, projectID uint) (*ArchitectureVisualization, error) {
	// 获取项目
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 解析 Compose 配置
	config, err := s.parser.Parse(project.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// 生成可视化数据
	visualization, err := s.optimizer.GenerateVisualization(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate visualization: %w", err)
	}

	return visualization, nil
}

// GetDependencyGraph 获取依赖关系图
func (s *composeServiceImpl) GetDependencyGraph(ctx context.Context, projectID uint) (*DependencyGraph, error) {
	// 获取项目
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 解析 Compose 配置
	config, err := s.parser.Parse(project.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// 分析依赖关系
	depGraph, err := s.optimizer.AnalyzeDependencies(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	return depGraph, nil
}

// EvaluateProjectPerformance 评估项目性能
func (s *composeServiceImpl) EvaluateProjectPerformance(ctx context.Context, projectID uint) (*PerformanceEvaluation, error) {
	// 获取项目
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 解析 Compose 配置
	config, err := s.parser.Parse(project.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// 评估性能
	evaluation, err := s.optimizer.EvaluatePerformance(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate performance: %w", err)
	}

	return evaluation, nil
}
