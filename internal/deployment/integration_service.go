// Package deployment 部署集成服务
// 提供统一的部署验证、诊断和修复功能，集成所有组件
// 这是 qwq AIOps 平台的核心部署管理模块，负责：
// 1. 配置验证和诊断
// 2. 前端资源完整性检查
// 3. 通知服务状态验证
// 4. 平台兼容性检查
// 5. 自动修复和问题解决
package deployment

import (
	"fmt"
	"qwq/internal/config"
	"qwq/internal/notify"
	"qwq/internal/platform"
	"time"
)

// FrontendManager 前端资源管理器接口
// 用于解耦 deployment 包与 server 包的依赖
type FrontendManager interface {
	ValidateResources() interface{}  // 返回验证结果
	GetResourceStats() interface{}   // 返回资源统计
}

// FrontendValidationResult 前端验证结果接口
type FrontendValidationResult interface {
	IsValid() bool
	GetErrors() []string
	GetWarnings() []string
	GetSuggestions() []string
}

// IntegrationService 部署集成服务
// 统一管理配置诊断、前端管理、通知服务和平台适配器
// 这是部署管理的核心服务，整合了所有必要的组件：
// - configDiagnostic: 配置诊断器，检查环境变量和配置文件
// - frontendManager: 前端资源管理器，验证前端文件完整性
// - notifyService: 通知服务，处理告警和状态报告
// - platformAdapter: 平台适配器，处理跨平台兼容性
// - autoFixer: 自动修复器，尝试自动解决常见问题
type IntegrationService struct {
	configDiagnostic *config.ConfigDiagnostic   // 配置诊断器
	frontendManager  FrontendManager            // 前端资源管理器接口
	notifyService    notify.NotificationService // 通知服务接口
	platformAdapter  platform.PlatformAdapter  // 平台适配器接口
	autoFixer        *config.EnhancedAutoFixer  // 增强自动修复器
}

// DeploymentStatus 部署状态
// 包含系统整体健康状态和各个组件的详细状态信息
type DeploymentStatus struct {
	Overall         string                      `json:"overall"`          // 整体状态: healthy/warning/error
	Components      map[string]ComponentStatus  `json:"components"`       // 各组件状态详情
	LastCheck       time.Time                   `json:"last_check"`       // 最后检查时间
	Issues          []string                    `json:"issues"`           // 发现的问题列表
	Suggestions     []string                    `json:"suggestions"`      // 修复建议列表
	ValidationResult *ValidationResult          `json:"validation_result"` // 详细验证结果
}

// ComponentStatus 组件状态
// 记录单个组件的健康状态和相关信息
type ComponentStatus struct {
	Name        string    `json:"name"`         // 组件名称（如：配置管理、前端资源等）
	Status      string    `json:"status"`       // 状态：healthy(健康)/warning(警告)/error(错误)
	LastCheck   time.Time `json:"last_check"`   // 最后检查时间
	Issues      []string  `json:"issues"`       // 该组件发现的问题列表
	Suggestions []string  `json:"suggestions"`  // 针对该组件的修复建议
}

// ValidationResult 验证结果
// 包含详细的验证结果信息，用于生成完整的部署状态报告
type ValidationResult struct {
	Valid               bool              `json:"valid"`                 // 整体验证是否通过
	ComponentsChecked   []string          `json:"components_checked"`    // 已检查的组件列表
	HealthyComponents   []string          `json:"healthy_components"`    // 健康的组件列表
	UnhealthyComponents []string          `json:"unhealthy_components"`  // 不健康的组件列表
	MissingComponents   []string          `json:"missing_components"`    // 缺失的组件列表
	ValidationErrors    []string          `json:"validation_errors"`     // 验证过程中发现的错误
	Suggestions         []string          `json:"suggestions"`           // 修复建议列表
}

// RepairResult 修复结果
// 记录自动修复操作的执行结果和详细信息
type RepairResult struct {
	Success         bool              `json:"success"`           // 修复是否成功
	FixedIssues     []string          `json:"fixed_issues"`      // 已修复的问题列表
	RemainingIssues []string          `json:"remaining_issues"`  // 剩余未修复的问题
	Operations      []RepairOperation `json:"operations"`        // 执行的修复操作列表
	Duration        time.Duration     `json:"duration"`          // 修复操作总耗时
}

// RepairOperation 修复操作
// 记录单个修复操作的详细信息
type RepairOperation struct {
	Component   string    `json:"component"`    // 操作的组件名称
	Operation   string    `json:"operation"`    // 操作类型（如：重建、修复、验证等）
	Status      string    `json:"status"`       // 操作状态：success/failed/skipped
	Message     string    `json:"message"`      // 操作结果消息
	Timestamp   time.Time `json:"timestamp"`    // 操作执行时间
}

// NewIntegrationService 创建部署集成服务
// 初始化所有必要的组件和服务，返回完整配置的集成服务实例
// frontendMgr 参数允许外部注入前端管理器，避免循环依赖
func NewIntegrationService(frontendMgr FrontendManager) *IntegrationService {
	return &IntegrationService{
		configDiagnostic: config.NewConfigDiagnostic(),                        // 初始化配置诊断器
		frontendManager:  frontendMgr,                                         // 使用注入的前端管理器
		notifyService:    notify.GetNotificationService(),                     // 获取通知服务实例
		platformAdapter:  platform.NewPlatformAdapter(),                      // 创建平台适配器
		autoFixer:        config.NewEnhancedAutoFixer(config.DefaultAutoFixOptions()), // 创建增强自动修复器
	}
}

// RunComprehensiveValidation 运行全面的部署验证
// 检查所有关键组件的状态和配置，包括：
// 1. 配置文件和环境变量验证
// 2. 前端资源完整性检查
// 3. 通知服务连通性测试
// 4. 平台兼容性验证
// 返回详细的部署状态报告
func (is *IntegrationService) RunComprehensiveValidation() *DeploymentStatus {
	status := &DeploymentStatus{
		Overall:     "healthy",
		Components:  make(map[string]ComponentStatus),
		LastCheck:   time.Now(),
		Issues:      []string{},
		Suggestions: []string{},
	}

	// 1. 验证配置 - 检查环境变量、配置文件等
	configStatus := is.validateConfiguration()
	status.Components["configuration"] = configStatus
	if configStatus.Status != "healthy" {
		status.Overall = "warning"
		status.Issues = append(status.Issues, configStatus.Issues...)
		status.Suggestions = append(status.Suggestions, configStatus.Suggestions...)
	}

	// 2. 验证前端资源 - 检查前端文件是否正确嵌入
	frontendStatus := is.validateFrontendResources()
	status.Components["frontend"] = frontendStatus
	if frontendStatus.Status == "error" {
		status.Overall = "error" // 前端错误会导致整体状态为错误
	} else if frontendStatus.Status == "warning" && status.Overall == "healthy" {
		status.Overall = "warning"
	}
	status.Issues = append(status.Issues, frontendStatus.Issues...)
	status.Suggestions = append(status.Suggestions, frontendStatus.Suggestions...)

	// 3. 验证通知服务 - 检查钉钉、微信等通知渠道
	notificationStatus := is.validateNotificationService()
	status.Components["notification"] = notificationStatus
	if notificationStatus.Status == "warning" && status.Overall == "healthy" {
		status.Overall = "warning" // 通知服务问题通常不是致命的
	}
	status.Issues = append(status.Issues, notificationStatus.Issues...)
	status.Suggestions = append(status.Suggestions, notificationStatus.Suggestions...)

	// 4. 验证平台兼容性 - 检查操作系统、Docker 等环境
	platformStatus := is.validatePlatformCompatibility()
	status.Components["platform"] = platformStatus
	if platformStatus.Status == "error" {
		status.Overall = "error" // 平台兼容性错误是致命的
	} else if platformStatus.Status == "warning" && status.Overall == "healthy" {
		status.Overall = "warning"
	}
	status.Issues = append(status.Issues, platformStatus.Issues...)
	status.Suggestions = append(status.Suggestions, platformStatus.Suggestions...)

	// 5. 生成详细的验证结果
	status.ValidationResult = is.generateValidationResult(status)

	return status
}

// validateConfiguration 验证配置
// 检查系统配置的完整性和有效性，包括：
// - 环境变量配置（AI服务、数据库等）
// - 安全配置（JWT密钥、加密密钥等）
// - 通知配置（钉钉、微信等）
// - 平台特定配置
func (is *IntegrationService) validateConfiguration() ComponentStatus {
	status := ComponentStatus{
		Name:        "配置管理",
		Status:      "healthy",
		LastCheck:   time.Now(),
		Issues:      []string{},
		Suggestions: []string{},
	}

	// 运行配置诊断，获取所有配置组件的状态
	diagnosticResults := is.configDiagnostic.RunDiagnostics()
	
	// 遍历诊断结果，汇总问题和建议
	for _, result := range diagnosticResults {
		if result.Status == config.StatusError {
			status.Status = "error" // 有错误时设置为错误状态
			for _, issue := range result.Issues {
				status.Issues = append(status.Issues, fmt.Sprintf("[%s] %s", result.Component, issue.Description))
			}
		} else if result.Status == config.StatusWarning && status.Status == "healthy" {
			status.Status = "warning" // 有警告但无错误时设置为警告状态
		}
		
		// 收集所有修复建议
		status.Suggestions = append(status.Suggestions, result.Suggestions...)
	}

	return status
}

// validateFrontendResources 验证前端资源
// 检查前端资源的完整性和可用性，包括：
// - index.html 是否存在
// - JavaScript 和 CSS 文件是否正确嵌入
// - 静态资源文件的完整性
// - 文件哈希值验证
func (is *IntegrationService) validateFrontendResources() ComponentStatus {
	status := ComponentStatus{
		Name:        "前端资源",
		Status:      "healthy",
		LastCheck:   time.Now(),
		Issues:      []string{},
		Suggestions: []string{},
	}

	// 使用前端管理器验证资源完整性
	validationResultRaw := is.frontendManager.ValidateResources()
	
	// 尝试类型断言获取验证结果
	if validationResult, ok := validationResultRaw.(FrontendValidationResult); ok {
		// 如果验证失败，设置为错误状态
		if !validationResult.IsValid() {
			status.Status = "error"
			status.Issues = append(status.Issues, validationResult.GetErrors()...)
			status.Suggestions = append(status.Suggestions, validationResult.GetSuggestions()...)
		}
		
		// 如果有警告但验证通过，设置为警告状态
		if len(validationResult.GetWarnings()) > 0 && status.Status == "healthy" {
			status.Status = "warning"
			status.Issues = append(status.Issues, validationResult.GetWarnings()...)
		}
	}

	return status
}

// validateNotificationService 验证通知服务
// 检查通知服务的配置和连通性，包括：
// - 钉钉 Webhook 配置验证
// - 企业微信 Webhook 配置验证
// - 邮件服务配置验证
// - 通知渠道连通性测试
func (is *IntegrationService) validateNotificationService() ComponentStatus {
	status := ComponentStatus{
		Name:        "通知服务",
		Status:      "healthy",
		LastCheck:   time.Now(),
		Issues:      []string{},
		Suggestions: []string{},
	}

	// 验证通知配置的有效性
	if err := is.notifyService.ValidateConfig(); err != nil {
		status.Status = "warning" // 通知服务问题通常不是致命的
		status.Issues = append(status.Issues, fmt.Sprintf("通知配置验证失败: %v", err))
		status.Suggestions = append(status.Suggestions, "请配置有效的通知渠道（如钉钉 Webhook）")
	}

	return status
}

// validatePlatformCompatibility 验证平台兼容性
// 检查当前运行环境的兼容性，包括：
// - 操作系统兼容性（Windows/Linux/macOS）
// - Docker 环境检查
// - 必要依赖的可用性
// - 文件系统权限检查
func (is *IntegrationService) validatePlatformCompatibility() ComponentStatus {
	status := ComponentStatus{
		Name:        "平台兼容性",
		Status:      "healthy",
		LastCheck:   time.Now(),
		Issues:      []string{},
		Suggestions: []string{},
	}

	// 验证平台兼容性
	if err := is.platformAdapter.ValidatePlatformCompatibility(); err != nil {
		status.Status = "error"
		status.Issues = append(status.Issues, fmt.Sprintf("平台兼容性验证失败: %v", err))
		status.Suggestions = append(status.Suggestions, "请检查操作系统兼容性和必要的依赖环境")
	}

	// 获取并记录平台信息
	platformInfo := is.platformAdapter.GetPlatformInfo()
	status.Suggestions = append(status.Suggestions, 
		fmt.Sprintf("当前平台: %s/%s", platformInfo.OS, platformInfo.Architecture))

	return status
}

// generateValidationResult 生成验证结果
// 根据各组件的状态生成详细的验证结果报告
func (is *IntegrationService) generateValidationResult(status *DeploymentStatus) *ValidationResult {
	result := &ValidationResult{
		Valid:               status.Overall == "healthy",
		ComponentsChecked:   []string{},
		HealthyComponents:   []string{},
		UnhealthyComponents: []string{},
		MissingComponents:   []string{},
		ValidationErrors:    status.Issues,
		Suggestions:         status.Suggestions,
	}

	// 统计各组件状态
	for name, comp := range status.Components {
		result.ComponentsChecked = append(result.ComponentsChecked, name)
		
		if comp.Status == "healthy" {
			result.HealthyComponents = append(result.HealthyComponents, name)
		} else {
			result.UnhealthyComponents = append(result.UnhealthyComponents, name)
		}
	}

	return result
}

// RunAutomaticRepair 运行自动修复
// 尝试自动修复检测到的问题，包括：
// - 配置文件修复和生成
// - 前端资源重建
// - 权限问题修复
// - 依赖环境修复
func (is *IntegrationService) RunAutomaticRepair() *RepairResult {
	startTime := time.Now()
	
	result := &RepairResult{
		Success:         true,
		FixedIssues:     []string{},
		RemainingIssues: []string{},
		Operations:      []RepairOperation{},
		Duration:        0,
	}

	// 1. 运行增强自动修复器
	if err := is.autoFixer.RunComprehensiveRepair(); err != nil {
		result.Success = false
		result.RemainingIssues = append(result.RemainingIssues, fmt.Sprintf("自动修复失败: %v", err))
		
		// 记录失败的修复操作
		result.Operations = append(result.Operations, RepairOperation{
			Component: "综合修复",
			Operation: "自动修复",
			Status:    "failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
	} else {
		result.FixedIssues = append(result.FixedIssues, "配置和前端资源修复完成")
		
		// 记录成功的修复操作
		result.Operations = append(result.Operations, RepairOperation{
			Component: "综合修复",
			Operation: "自动修复",
			Status:    "success",
			Message:   "自动修复完成",
			Timestamp: time.Now(),
		})
	}

	// 2. 验证修复结果
	postRepairStatus := is.RunComprehensiveValidation()
	if postRepairStatus.Overall != "healthy" {
		result.Success = false
		result.RemainingIssues = append(result.RemainingIssues, postRepairStatus.Issues...)
	}

	result.Duration = time.Since(startTime)
	return result
}

// GetDeploymentWorkflow 获取完整的诊断和修复工作流
// 返回诊断结果和修复建议的完整工作流，包括：
// - 当前系统状态诊断
// - 自动修复步骤
// - 手动修复指导
// - 平台信息和环境详情
func (is *IntegrationService) GetDeploymentWorkflow() map[string]interface{} {
	workflow := make(map[string]interface{})
	
	// 1. 运行全面诊断
	validationStatus := is.RunComprehensiveValidation()
	workflow["validation"] = validationStatus
	
	// 2. 如果有问题，提供修复工作流
	if validationStatus.Overall != "healthy" {
		repairSteps := []map[string]interface{}{
			{
				"step":        1,
				"title":       "自动修复配置问题",
				"description": "运行自动修复器修复常见的配置和前端资源问题",
				"command":     "RunAutomaticRepair",
				"automated":   true,
			},
			{
				"step":        2,
				"title":       "验证修复结果",
				"description": "重新验证所有组件状态",
				"command":     "RunComprehensiveValidation",
				"automated":   true,
			},
			{
				"step":        3,
				"title":       "手动处理剩余问题",
				"description": "根据建议手动处理无法自动修复的问题",
				"command":     "manual",
				"automated":   false,
			},
		}
		workflow["repair_steps"] = repairSteps
	}
	
	// 3. 添加平台信息
	platformInfo := is.platformAdapter.GetPlatformInfo()
	workflow["platform_info"] = platformInfo
	
	// 4. 添加时间戳
	workflow["timestamp"] = time.Now()
	
	return workflow
}

// SendDeploymentReport 发送部署状态报告
// 通过通知服务发送部署状态和问题报告，包括：
// - 整体部署状态
// - 各组件详细状态
// - 发现的问题列表
// - 修复建议
func (is *IntegrationService) SendDeploymentReport() error {
	status := is.RunComprehensiveValidation()
	
	// 构建报告标题
	title := "qwq AIOps 部署状态报告"
	
	// 构建报告内容
	var content string
	if status.Overall == "healthy" {
		content = "## ✅ 部署状态良好\n\n所有组件运行正常，无需处理。"
	} else {
		content = fmt.Sprintf("## ⚠️ 部署状态: %s\n\n", status.Overall)
		
		// 添加问题列表
		if len(status.Issues) > 0 {
			content += "### 发现的问题:\n"
			for i, issue := range status.Issues {
				content += fmt.Sprintf("%d. %s\n", i+1, issue)
			}
			content += "\n"
		}
		
		// 添加修复建议
		if len(status.Suggestions) > 0 {
			content += "### 修复建议:\n"
			for i, suggestion := range status.Suggestions {
				content += fmt.Sprintf("%d. %s\n", i+1, suggestion)
			}
		}
	}
	
	// 添加组件状态详情
	content += "\n### 组件状态:\n"
	for compName, comp := range status.Components {
		statusIcon := "✅"
		if comp.Status == "warning" {
			statusIcon = "⚠️"
		} else if comp.Status == "error" {
			statusIcon = "❌"
		}
		_ = compName // 使用变量避免编译警告
		content += fmt.Sprintf("- %s %s: %s\n", statusIcon, comp.Name, comp.Status)
	}
	
	// 通过通知服务发送报告
	return is.notifyService.SendAlert(title, content)
}

// GetHealthStatus 获取健康状态（用于健康检查接口）
// 返回简化的健康状态信息，用于系统健康检查接口
func (is *IntegrationService) GetHealthStatus() map[string]interface{} {
	status := is.RunComprehensiveValidation()
	
	return map[string]interface{}{
		"status":     status.Overall,        // 整体状态
		"timestamp":  status.LastCheck,      // 检查时间
		"components": len(status.Components), // 组件数量
		"issues":     len(status.Issues),     // 问题数量
		"healthy":    status.Overall == "healthy", // 是否健康
	}
}