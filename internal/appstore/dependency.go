package appstore

import (
	"context"
	"encoding/json"
	"fmt"
)

// DependencyManager 依赖管理器
type DependencyManager struct {
	appStoreService AppStoreService
}

// NewDependencyManager 创建依赖管理器实例
func NewDependencyManager(appStoreService AppStoreService) *DependencyManager {
	return &DependencyManager{
		appStoreService: appStoreService,
	}
}

// CheckDependencies 检查依赖
func (d *DependencyManager) CheckDependencies(ctx context.Context, templateID uint) ([]DependencyCheck, error) {
	// 获取模板
	template, err := d.appStoreService.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// 解析依赖项
	var dependencies []TemplateDependency
	if template.Dependencies != "" {
		if err := json.Unmarshal([]byte(template.Dependencies), &dependencies); err != nil {
			return nil, fmt.Errorf("failed to parse dependencies: %w", err)
		}
	}

	// 检查每个依赖项
	var checks []DependencyCheck
	for _, dep := range dependencies {
		check := DependencyCheck{
			Name:     dep.Name,
			Type:     dep.Type,
			Required: !dep.Optional,
		}

		// 根据依赖类型进行检查
		switch dep.Type {
		case "service":
			check.Satisfied = d.checkServiceDependency(ctx, dep.Name)
			if !check.Satisfied {
				check.Message = fmt.Sprintf("Required service '%s' is not running", dep.Name)
			}
		case "port":
			check.Satisfied = d.checkPortDependency(ctx, dep.Name)
			if !check.Satisfied {
				check.Message = fmt.Sprintf("Required port '%s' is not available", dep.Name)
			}
		case "volume":
			check.Satisfied = d.checkVolumeDependency(ctx, dep.Name)
			if !check.Satisfied {
				check.Message = fmt.Sprintf("Required volume '%s' does not exist", dep.Name)
			}
		case "application":
			check.Satisfied = d.checkApplicationDependency(ctx, dep.Name)
			if !check.Satisfied {
				check.Message = fmt.Sprintf("Required application '%s' is not installed", dep.Name)
			}
		default:
			check.Satisfied = false
			check.Message = fmt.Sprintf("Unknown dependency type: %s", dep.Type)
		}

		checks = append(checks, check)
	}

	return checks, nil
}

// checkServiceDependency 检查服务依赖
func (d *DependencyManager) checkServiceDependency(ctx context.Context, serviceName string) bool {
	// 检查指定的服务是否正在运行
	// 这里应该查询 Docker 或 Kubernetes 来确认服务状态
	
	// 获取所有实例
	instances, err := d.appStoreService.ListInstances(ctx, 0, 0)
	if err != nil {
		return false
	}

	// 查找匹配的服务
	for _, instance := range instances {
		if instance.Name == serviceName && instance.Status == "running" {
			return true
		}
	}

	return false
}

// checkPortDependency 检查端口依赖
func (d *DependencyManager) checkPortDependency(ctx context.Context, port string) bool {
	// 检查指定的端口是否可用
	// 这里应该实际检查端口是否被占用
	
	// 简化实现：假设端口总是可用
	return true
}

// checkVolumeDependency 检查数据卷依赖
func (d *DependencyManager) checkVolumeDependency(ctx context.Context, volumeName string) bool {
	// 检查指定的数据卷是否存在
	// 这里应该查询 Docker 或 Kubernetes 来确认数据卷存在
	
	// 简化实现：假设数据卷总是存在
	return true
}

// checkApplicationDependency 检查应用依赖
func (d *DependencyManager) checkApplicationDependency(ctx context.Context, appName string) bool {
	// 检查指定的应用是否已安装
	instances, err := d.appStoreService.ListInstances(ctx, 0, 0)
	if err != nil {
		return false
	}

	for _, instance := range instances {
		if instance.Template != nil && instance.Template.Name == appName {
			if instance.Status == "running" || instance.Status == "stopped" {
				return true
			}
		}
	}

	return false
}

// ResolveDependencies 解决依赖
func (d *DependencyManager) ResolveDependencies(ctx context.Context, templateID uint, autoInstall bool) error {
	// 检查依赖
	checks, err := d.CheckDependencies(ctx, templateID)
	if err != nil {
		return fmt.Errorf("failed to check dependencies: %w", err)
	}

	// 如果不自动安装，只检查是否满足
	if !autoInstall {
		for _, check := range checks {
			if check.Required && !check.Satisfied {
				return fmt.Errorf("dependency not met: %s - %s", check.Name, check.Message)
			}
		}
		return nil
	}

	// 自动安装缺失的依赖
	for _, check := range checks {
		if check.Required && !check.Satisfied {
			if err := d.installDependency(ctx, check); err != nil {
				return fmt.Errorf("failed to install dependency '%s': %w", check.Name, err)
			}
		}
	}

	return nil
}

// installDependency 安装依赖
func (d *DependencyManager) installDependency(ctx context.Context, check DependencyCheck) error {
	// 根据依赖类型执行相应的安装操作
	switch check.Type {
	case "application":
		// 查找并安装应用
		template, err := d.appStoreService.GetTemplateByName(ctx, check.Name)
		if err != nil {
			return fmt.Errorf("dependency application not found: %w", err)
		}

		// 创建实例（这里应该调用安装器服务）
		instance := &ApplicationInstance{
			Name:       check.Name,
			TemplateID: template.ID,
			Version:    template.Version,
			Status:     "installing",
			Config:     "{}",
		}

		if err := d.appStoreService.CreateInstance(ctx, instance); err != nil {
			return fmt.Errorf("failed to create dependency instance: %w", err)
		}

		return nil
	case "service":
		// 启动服务
		return fmt.Errorf("automatic service installation not implemented")
	case "volume":
		// 创建数据卷
		return fmt.Errorf("automatic volume creation not implemented")
	default:
		return fmt.Errorf("cannot install dependency of type: %s", check.Type)
	}
}

// GetDependencyTree 获取依赖树
func (d *DependencyManager) GetDependencyTree(ctx context.Context, templateID uint) (*DependencyTree, error) {
	tree := &DependencyTree{
		TemplateID: templateID,
		Children:   make([]*DependencyTree, 0),
	}

	// 获取模板
	template, err := d.appStoreService.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	tree.TemplateName = template.Name

	// 解析依赖项
	var dependencies []TemplateDependency
	if template.Dependencies != "" {
		if err := json.Unmarshal([]byte(template.Dependencies), &dependencies); err != nil {
			return nil, fmt.Errorf("failed to parse dependencies: %w", err)
		}
	}

	// 递归构建依赖树
	for _, dep := range dependencies {
		if dep.Type == "application" {
			// 查找依赖的应用模板
			depTemplate, err := d.appStoreService.GetTemplateByName(ctx, dep.Name)
			if err != nil {
				continue // 跳过找不到的依赖
			}

			// 递归获取子依赖树
			childTree, err := d.GetDependencyTree(ctx, depTemplate.ID)
			if err != nil {
				continue
			}

			tree.Children = append(tree.Children, childTree)
		}
	}

	return tree, nil
}

// DependencyTree 依赖树
type DependencyTree struct {
	TemplateID   uint              `json:"template_id"`
	TemplateName string            `json:"template_name"`
	Children     []*DependencyTree `json:"children"`
}

// ValidateDependencyTree 验证依赖树（检测循环依赖）
func (d *DependencyManager) ValidateDependencyTree(ctx context.Context, templateID uint) error {
	visited := make(map[uint]bool)
	return d.detectCircularDependency(ctx, templateID, visited, make(map[uint]bool))
}

// detectCircularDependency 检测循环依赖
func (d *DependencyManager) detectCircularDependency(ctx context.Context, templateID uint, visited, recursionStack map[uint]bool) error {
	// 标记当前节点为已访问
	visited[templateID] = true
	recursionStack[templateID] = true

	// 获取模板
	template, err := d.appStoreService.GetTemplate(ctx, templateID)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// 解析依赖项
	var dependencies []TemplateDependency
	if template.Dependencies != "" {
		if err := json.Unmarshal([]byte(template.Dependencies), &dependencies); err != nil {
			return fmt.Errorf("failed to parse dependencies: %w", err)
		}
	}

	// 检查每个依赖
	for _, dep := range dependencies {
		if dep.Type == "application" {
			depTemplate, err := d.appStoreService.GetTemplateByName(ctx, dep.Name)
			if err != nil {
				continue
			}

			// 如果依赖在递归栈中，说明存在循环依赖
			if recursionStack[depTemplate.ID] {
				return fmt.Errorf("circular dependency detected: %s -> %s", template.Name, dep.Name)
			}

			// 如果依赖未访问过，递归检查
			if !visited[depTemplate.ID] {
				if err := d.detectCircularDependency(ctx, depTemplate.ID, visited, recursionStack); err != nil {
					return err
				}
			}
		}
	}

	// 从递归栈中移除当前节点
	recursionStack[templateID] = false

	return nil
}

// GetInstallOrder 获取安装顺序（拓扑排序）
func (d *DependencyManager) GetInstallOrder(ctx context.Context, templateID uint) ([]uint, error) {
	// 验证依赖树
	if err := d.ValidateDependencyTree(ctx, templateID); err != nil {
		return nil, err
	}

	// 执行拓扑排序
	var order []uint
	visited := make(map[uint]bool)

	if err := d.topologicalSort(ctx, templateID, visited, &order); err != nil {
		return nil, err
	}

	return order, nil
}

// topologicalSort 拓扑排序
func (d *DependencyManager) topologicalSort(ctx context.Context, templateID uint, visited map[uint]bool, order *[]uint) error {
	visited[templateID] = true

	// 获取模板
	template, err := d.appStoreService.GetTemplate(ctx, templateID)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// 解析依赖项
	var dependencies []TemplateDependency
	if template.Dependencies != "" {
		if err := json.Unmarshal([]byte(template.Dependencies), &dependencies); err != nil {
			return fmt.Errorf("failed to parse dependencies: %w", err)
		}
	}

	// 先递归处理依赖
	for _, dep := range dependencies {
		if dep.Type == "application" {
			depTemplate, err := d.appStoreService.GetTemplateByName(ctx, dep.Name)
			if err != nil {
				continue
			}

			if !visited[depTemplate.ID] {
				if err := d.topologicalSort(ctx, depTemplate.ID, visited, order); err != nil {
					return err
				}
			}
		}
	}

	// 将当前模板添加到顺序中
	*order = append(*order, templateID)

	return nil
}

// GetDependencyTree 获取依赖树
func (d *DependencyManager) GetDependencyTree(ctx context.Context, templateID uint) (*DependencyTree, error) {
	tree := &DependencyTree{
		TemplateID: templateID,
		Children:   make([]*DependencyTree, 0),
	}

	// 获取模板
	template, err := d.appStoreService.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	tree.TemplateName = template.Name

	// 解析依赖项
	var dependencies []TemplateDependency
	if template.Dependencies != "" {
		if err := json.Unmarshal([]byte(template.Dependencies), &dependencies); err != nil {
			return nil, fmt.Errorf("failed to parse dependencies: %w", err)
		}
	}

	// 递归构建依赖树
	for _, dep := range dependencies {
		if dep.Type == "application" {
			// 查找依赖的应用模板
			depTemplate, err := d.appStoreService.GetTemplateByName(ctx, dep.Name)
			if err != nil {
				continue // 跳过找不到的依赖
			}

			// 递归获取子依赖树
			childTree, err := d.GetDependencyTree(ctx, depTemplate.ID)
			if err != nil {
				continue
			}

			tree.Children = append(tree.Children, childTree)
		}
	}

	return tree, nil
}

// DependencyTree 依赖树
type DependencyTree struct {
	TemplateID   uint              `json:"template_id"`
	TemplateName string            `json:"template_name"`
	Children     []*DependencyTree `json:"children"`
}

// ValidateDependencyTree 验证依赖树（检测循环依赖）
func (d *DependencyManager) ValidateDependencyTree(ctx context.Context, templateID uint) error {
	visited := make(map[uint]bool)
	return d.detectCircularDependency(ctx, templateID, visited, make(map[uint]bool))
}

// detectCircularDependency 检测循环依赖
func (d *DependencyManager) detectCircularDependency(ctx context.Context, templateID uint, visited, recursionStack map[uint]bool) error {
	// 标记当前节点为已访问
	visited[templateID] = true
	recursionStack[templateID] = true

	// 获取模板
	template, err := d.appStoreService.GetTemplate(ctx, templateID)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// 解析依赖项
	var dependencies []TemplateDependency
	if template.Dependencies != "" {
		if err := json.Unmarshal([]byte(template.Dependencies), &dependencies); err != nil {
			return fmt.Errorf("failed to parse dependencies: %w", err)
		}
	}

	// 检查每个依赖
	for _, dep := range dependencies {
		if dep.Type == "application" {
			depTemplate, err := d.appStoreService.GetTemplateByName(ctx, dep.Name)
			if err != nil {
				continue
			}

			// 如果依赖在递归栈中，说明存在循环依赖
			if recursionStack[depTemplate.ID] {
				return fmt.Errorf("circular dependency detected: %s -> %s", template.Name, dep.Name)
			}

			// 如果依赖未访问过，递归检查
			if !visited[depTemplate.ID] {
				if err := d.detectCircularDependency(ctx, depTemplate.ID, visited, recursionStack); err != nil {
					return err
				}
			}
		}
	}

	// 从递归栈中移除当前节点
	recursionStack[templateID] = false

	return nil
}

// GetInstallOrder 获取安装顺序（拓扑排序）
func (d *DependencyManager) GetInstallOrder(ctx context.Context, templateID uint) ([]uint, error) {
	// 验证依赖树
	if err := d.ValidateDependencyTree(ctx, templateID); err != nil {
		return nil, err
	}

	// 执行拓扑排序
	var order []uint
	visited := make(map[uint]bool)

	if err := d.topologicalSort(ctx, templateID, visited, &order); err != nil {
		return nil, err
	}

	return order, nil
}

// topologicalSort 拓扑排序
func (d *DependencyManager) topologicalSort(ctx context.Context, templateID uint, visited map[uint]bool, order *[]uint) error {
	visited[templateID] = true

	// 获取模板
	template, err := d.appStoreService.GetTemplate(ctx, templateID)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// 解析依赖项
	var dependencies []TemplateDependency
	if template.Dependencies != "" {
		if err := json.Unmarshal([]byte(template.Dependencies), &dependencies); err != nil {
			return fmt.Errorf("failed to parse dependencies: %w", err)
		}
	}

	// 先递归处理依赖
	for _, dep := range dependencies {
		if dep.Type == "application" {
			depTemplate, err := d.appStoreService.GetTemplateByName(ctx, dep.Name)
			if err != nil {
				continue
			}

			if !visited[depTemplate.ID] {
				if err := d.topologicalSort(ctx, depTemplate.ID, visited, order); err != nil {
					return err
				}
			}
		}
	}

	// 将当前模板添加到顺序中
	*order = append(*order, templateID)

	return nil
}
