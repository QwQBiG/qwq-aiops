// Package deployment reports the console's effective configuration and resources.
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
	ValidateResources() FrontendValidationResult
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
type IntegrationService struct {
	configDiagnostic *config.ConfigDiagnostic   // 配置诊断器
	frontendManager  FrontendManager            // 前端资源管理器接口
	notifyService    notify.NotificationService // 通知服务接口
	platformAdapter  platform.PlatformAdapter   // 平台适配器接口
}

// DeploymentStatus 部署状态
// 包含系统整体健康状态和各个组件的详细状态信息
type DeploymentStatus struct {
	Overall          string                     `json:"overall"`           // 整体状态: healthy/warning/error
	Components       map[string]ComponentStatus `json:"components"`        // 各组件状态详情
	LastCheck        time.Time                  `json:"last_check"`        // 最后检查时间
	Issues           []string                   `json:"issues"`            // 发现的问题列表
	Suggestions      []string                   `json:"suggestions"`       // 修复建议列表
	ValidationResult *ValidationResult          `json:"validation_result"` // 详细验证结果
}

// ComponentStatus 组件状态
// 记录单个组件的健康状态和相关信息
type ComponentStatus struct {
	Name        string    `json:"name"`        // 组件名称（如：配置管理、前端资源等）
	Status      string    `json:"status"`      // 状态：healthy(健康)/warning(警告)/error(错误)
	LastCheck   time.Time `json:"last_check"`  // 最后检查时间
	Issues      []string  `json:"issues"`      // 该组件发现的问题列表
	Suggestions []string  `json:"suggestions"` // 针对该组件的修复建议
}

// ValidationResult 验证结果
// 包含详细的验证结果信息，用于生成完整的部署状态报告
type ValidationResult struct {
	Valid               bool     `json:"valid"`                // 整体验证是否通过
	ComponentsChecked   []string `json:"components_checked"`   // 已检查的组件列表
	HealthyComponents   []string `json:"healthy_components"`   // 健康的组件列表
	UnhealthyComponents []string `json:"unhealthy_components"` // 不健康的组件列表
	MissingComponents   []string `json:"missing_components"`   // 缺失的组件列表
	ValidationErrors    []string `json:"validation_errors"`    // 验证过程中发现的错误
	Suggestions         []string `json:"suggestions"`          // 修复建议列表
}

// NewIntegrationService 创建部署集成服务
// 初始化所有必要的组件和服务，返回完整配置的集成服务实例
// frontendMgr 参数允许外部注入前端管理器，避免循环依赖
func NewIntegrationService(frontendMgr FrontendManager) *IntegrationService {
	return &IntegrationService{
		configDiagnostic: config.NewConfigDiagnostic(),    // 初始化配置诊断器
		frontendManager:  frontendMgr,                     // 使用注入的前端管理器
		notifyService:    notify.GetNotificationService(), // 获取通知服务实例
		platformAdapter:  platform.NewPlatformAdapter(),   // 创建平台适配器
	}
}

// RunComprehensiveValidation reports local deployment checks without changing
// configuration, rebuilding assets, or making external service requests.
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
		status.Overall = configStatus.Status
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

// validateConfiguration checks the effective model, access, and notification settings.
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
		for _, issue := range result.Issues {
			status.Issues = append(status.Issues, fmt.Sprintf("[%s] %s", result.Component, issue.Description))
		}
		if result.Status == config.StatusError {
			status.Status = "error" // 有错误时设置为错误状态
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

	if is.frontendManager == nil {
		status.Status = "error"
		status.Issues = append(status.Issues, "Frontend resource validator is unavailable")
		return status
	}
	validationResult := is.frontendManager.ValidateResources()
	if validationResult == nil {
		status.Status = "error"
		status.Issues = append(status.Issues, "Frontend resource validation returned no result")
		return status
	}
	if !validationResult.IsValid() {
		status.Status = "error"
		status.Issues = append(status.Issues, validationResult.GetErrors()...)
		status.Suggestions = append(status.Suggestions, validationResult.GetSuggestions()...)
	}
	if len(validationResult.GetWarnings()) > 0 {
		if status.Status == "healthy" {
			status.Status = "warning"
		}
		status.Issues = append(status.Issues, validationResult.GetWarnings()...)
	}

	return status
}

// validateNotificationService validates configured channels without sending a message.
func (is *IntegrationService) validateNotificationService() ComponentStatus {
	status := ComponentStatus{
		Name:        "通知服务",
		Status:      "healthy",
		LastCheck:   time.Now(),
		Issues:      []string{},
		Suggestions: []string{},
	}

	if config.GlobalConfig.DingTalkWebhook == "" && config.GlobalConfig.TelegramToken == "" && config.GlobalConfig.TelegramChatID == "" {
		status.Suggestions = append(status.Suggestions, "Notifications are optional and currently disabled")
		return status
	}
	// 验证通知配置的有效性
	if err := is.notifyService.ValidateConfig(); err != nil {
		status.Status = "warning" // 通知服务问题通常不是致命的
		status.Issues = append(status.Issues, fmt.Sprintf("通知配置验证失败: %v", err))
		status.Suggestions = append(status.Suggestions, "请配置有效的通知渠道（如钉钉 Webhook）")
	}

	return status
}

// validatePlatformCompatibility reports OS support and the Linux system-check boundary.
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
	if platformInfo.OS != "linux" && status.Status == "healthy" {
		status.Status = "warning"
		status.Issues = append(status.Issues, "Complete system checks require Bash and Linux tools")
	}
	status.Suggestions = append(status.Suggestions, "Docker and model availability must be verified through their respective operations")
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

// GetDeploymentWorkflow returns local diagnostics and manual deployment guidance.
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
				"title":       "检查生效配置",
				"description": "按诊断建议检查环境变量、配置文件和启动参数，然后重新启动程序",
				"command":     "manual",
				"automated":   false,
			},
			{
				"step":        2,
				"title":       "重新构建并部署",
				"description": "前端资源缺失时，从源码运行 node scripts/build.mjs，再重启生成的程序；镜像部署则重新构建并启动镜像",
				"command":     "manual",
				"automated":   false,
			},
			{
				"step":        3,
				"title":       "重新验证部署",
				"description": "重新请求 /api/deployment/status，按部署指南处理仍然存在的问题",
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

// GetHealthStatus 获取健康状态（用于健康检查接口）
// 返回简化的健康状态信息，用于系统健康检查接口
func (is *IntegrationService) GetHealthStatus() map[string]interface{} {
	status := is.RunComprehensiveValidation()

	return map[string]interface{}{
		"status":     status.Overall,              // 整体状态
		"timestamp":  status.LastCheck,            // 检查时间
		"components": len(status.Components),      // 组件数量
		"issues":     len(status.Issues),          // 问题数量
		"healthy":    status.Overall == "healthy", // 是否健康
	}
}
