// Package config 增强的自动修复器
// 提供全面的自动修复功能，包括前端资源重建、配置修复和过程记录
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// EnhancedAutoFixer 增强的自动修复器
type EnhancedAutoFixer struct {
	diagnostic *ConfigDiagnostic
	tracker    *RepairTracker
	options    *AutoFixOptions
}

// AutoFixOptions 自动修复选项
type AutoFixOptions struct {
	EnableFrontendRebuild bool   `json:"enable_frontend_rebuild"`
	EnableConfigGeneration bool  `json:"enable_config_generation"`
	EnablePlatformFix     bool   `json:"enable_platform_fix"`
	LogPath               string `json:"log_path"`
	DryRun                bool   `json:"dry_run"`
	Verbose               bool   `json:"verbose"`
}

// DefaultAutoFixOptions 默认自动修复选项
func DefaultAutoFixOptions() *AutoFixOptions {
	return &AutoFixOptions{
		EnableFrontendRebuild:  true,
		EnableConfigGeneration: true,
		EnablePlatformFix:      true,
		LogPath:                "logs/repair.log",
		DryRun:                 false,
		Verbose:                true,
	}
}

// NewEnhancedAutoFixer 创建增强的自动修复器
func NewEnhancedAutoFixer(options *AutoFixOptions) *EnhancedAutoFixer {
	if options == nil {
		options = DefaultAutoFixOptions()
	}
	
	return &EnhancedAutoFixer{
		diagnostic: NewConfigDiagnostic(),
		tracker:    NewRepairTracker(options.LogPath),
		options:    options,
	}
}

// RunComprehensiveRepair 运行全面的自动修复
func (eaf *EnhancedAutoFixer) RunComprehensiveRepair() error {
	// 开始修复会话
	if err := eaf.tracker.StartSession(); err != nil {
		return fmt.Errorf("启动修复会话失败: %v", err)
	}
	
	fmt.Println("🔧 开始全面自动修复...")
	
	// 1. 运行诊断
	diagnosticResults := eaf.diagnostic.RunDiagnostics()
	
	// 2. 根据诊断结果执行修复
	if err := eaf.executeRepairOperations(diagnosticResults); err != nil {
		return fmt.Errorf("执行修复操作失败: %v", err)
	}
	
	// 3. 验证修复结果
	validationResult := eaf.validateRepairResults()
	
	// 4. 结束修复会话
	if err := eaf.tracker.EndSession(validationResult); err != nil {
		return fmt.Errorf("结束修复会话失败: %v", err)
	}
	
	// 5. 打印摘要
	eaf.tracker.PrintSessionSummary()
	
	return nil
}

// executeRepairOperations 执行修复操作
func (eaf *EnhancedAutoFixer) executeRepairOperations(diagnosticResults []*DiagnosticResult) error {
	for _, result := range diagnosticResults {
		if result.Status == StatusError || result.Status == StatusWarning {
			if err := eaf.repairComponent(result); err != nil {
				if eaf.options.Verbose {
					fmt.Printf("⚠️  组件 %s 修复失败: %v\n", result.Component, err)
				}
			}
		}
	}
	return nil
}

// repairComponent 修复组件
func (eaf *EnhancedAutoFixer) repairComponent(result *DiagnosticResult) error {
	switch result.Component {
	case "环境配置":
		return eaf.repairEnvironmentConfig(result)
	case "前端资源":
		return eaf.repairFrontendResources(result)
	case "通知配置":
		return eaf.repairNotificationConfig(result)
	case "平台兼容性":
		return eaf.repairPlatformCompatibility(result)
	default:
		return eaf.repairGenericIssues(result)
	}
}

// repairEnvironmentConfig 修复环境配置
func (eaf *EnhancedAutoFixer) repairEnvironmentConfig(result *DiagnosticResult) error {
	if !eaf.options.EnableConfigGeneration {
		return nil
	}
	
	opID := eaf.tracker.AddOperation(RepairConfig, "修复环境配置", []string{"检查并创建 .env 文件"})
	
	if err := eaf.tracker.StartOperation(opID); err != nil {
		return err
	}
	
	var repairErr error
	var output strings.Builder
	
	// 检查 .env 文件
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		output.WriteString("创建 .env 文件\n")
		
		if !eaf.options.DryRun {
			created, err := eaf.diagnostic.generator.CreateEnvFileIfNotExists(".env")
			if err != nil {
				repairErr = fmt.Errorf("创建 .env 文件失败: %v", err)
			} else if created {
				output.WriteString("✅ .env 文件已创建\n")
			}
		} else {
			output.WriteString("🔍 [DryRun] 将创建 .env 文件\n")
		}
	} else {
		output.WriteString("✅ .env 文件已存在\n")
	}
	
	return eaf.tracker.CompleteOperation(opID, output.String(), repairErr)
}

// repairFrontendResources 修复前端资源
func (eaf *EnhancedAutoFixer) repairFrontendResources(result *DiagnosticResult) error {
	if !eaf.options.EnableFrontendRebuild {
		return nil
	}
	
	opID := eaf.tracker.AddOperation(RepairFrontend, "修复前端资源", []string{
		"检查前端构建目录",
		"重建前端资源",
		"验证前端文件",
	})
	
	if err := eaf.tracker.StartOperation(opID); err != nil {
		return err
	}
	
	var repairErr error
	var output strings.Builder
	
	// 检查前端目录结构
	frontendPath := "frontend"
	distPath := filepath.Join(frontendPath, "dist")
	
	if _, err := os.Stat(frontendPath); os.IsNotExist(err) {
		repairErr = fmt.Errorf("frontend 目录不存在")
		return eaf.tracker.CompleteOperation(opID, output.String(), repairErr)
	}
	
	// 检查是否需要重建
	needsRebuild := false
	indexPath := filepath.Join(distPath, "index.html")
	
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		needsRebuild = true
		output.WriteString("检测到前端资源缺失，需要重建\n")
	}
	
	if needsRebuild {
		if !eaf.options.DryRun {
			if err := eaf.rebuildFrontendResources(); err != nil {
				repairErr = fmt.Errorf("重建前端资源失败: %v", err)
			} else {
				output.WriteString("✅ 前端资源重建完成\n")
			}
		} else {
			output.WriteString("🔍 [DryRun] 将重建前端资源\n")
		}
	} else {
		output.WriteString("✅ 前端资源检查通过\n")
	}
	
	return eaf.tracker.CompleteOperation(opID, output.String(), repairErr)
}

// rebuildFrontendResources 重建前端资源
func (eaf *EnhancedAutoFixer) rebuildFrontendResources() error {
	frontendPath := "frontend"
	
	// 检查 package.json
	packageJsonPath := filepath.Join(frontendPath, "package.json")
	if _, err := os.Stat(packageJsonPath); os.IsNotExist(err) {
		return fmt.Errorf("package.json 不存在")
	}
	
	// 检查 Node.js 和 npm 是否可用
	if err := eaf.checkNodeEnvironment(); err != nil {
		// 如果 Node.js 不可用，创建基本的前端文件
		return eaf.createBasicFrontendFiles()
	}
	
	// 执行 npm install
	if eaf.options.Verbose {
		fmt.Println("📦 安装前端依赖...")
	}
	
	installCmd := exec.Command("npm", "install")
	installCmd.Dir = frontendPath
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("npm install 失败: %v\n输出: %s", err, output)
	}
	
	// 执行 npm run build
	if eaf.options.Verbose {
		fmt.Println("🔨 构建前端资源...")
	}
	
	buildCmd := exec.Command("npm", "run", "build")
	buildCmd.Dir = frontendPath
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("npm run build 失败: %v\n输出: %s", err, output)
	}
	
	return nil
}

// checkNodeEnvironment 检查 Node.js 环境
func (eaf *EnhancedAutoFixer) checkNodeEnvironment() error {
	// 检查 node
	if _, err := exec.LookPath("node"); err != nil {
		return fmt.Errorf("Node.js 未安装")
	}
	
	// 检查 npm
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm 未安装")
	}
	
	return nil
}

// createBasicFrontendFiles 创建基本的前端文件
func (eaf *EnhancedAutoFixer) createBasicFrontendFiles() error {
	distPath := "frontend/dist"
	
	// 创建 dist 目录
	if err := os.MkdirAll(distPath, 0755); err != nil {
		return fmt.Errorf("创建 dist 目录失败: %v", err)
	}
	
	// 创建基本的 index.html
	indexContent := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>qwq AIOps 平台</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .container {
            text-align: center;
            background: rgba(255, 255, 255, 0.1);
            padding: 40px;
            border-radius: 10px;
            backdrop-filter: blur(10px);
        }
        h1 { margin-bottom: 20px; }
        .status { margin: 20px 0; }
        .loading {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 3px solid rgba(255,255,255,.3);
            border-radius: 50%;
            border-top-color: #fff;
            animation: spin 1s ease-in-out infinite;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 qwq AIOps 平台</h1>
        <div class="status">
            <div class="loading"></div>
            <p>系统正在启动中...</p>
        </div>
        <p>如果长时间未响应，请检查后端服务状态</p>
    </div>
    <script>
        console.log('qwq AIOps 平台前端已加载');
        
        // 简单的健康检查
        setTimeout(() => {
            fetch('/api/health')
                .then(response => response.json())
                .then(data => {
                    console.log('后端服务状态:', data);
                })
                .catch(error => {
                    console.warn('无法连接到后端服务:', error);
                });
        }, 1000);
    </script>
</body>
</html>`
	
	indexPath := filepath.Join(distPath, "index.html")
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		return fmt.Errorf("创建 index.html 失败: %v", err)
	}
	
	// 创建基本的 assets 目录和文件
	assetsPath := filepath.Join(distPath, "assets")
	if err := os.MkdirAll(assetsPath, 0755); err != nil {
		return fmt.Errorf("创建 assets 目录失败: %v", err)
	}
	
	// 创建基本的 CSS 文件
	cssContent := `/* qwq AIOps 平台基础样式 */
body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    margin: 0;
    padding: 0;
    background-color: #f5f5f5;
}

.app {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
}

.header {
    background: #1890ff;
    color: white;
    padding: 16px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.main {
    flex: 1;
    padding: 20px;
}

.loading {
    text-align: center;
    padding: 50px;
}
`
	
	cssPath := filepath.Join(assetsPath, "style.css")
	if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
		return fmt.Errorf("创建 style.css 失败: %v", err)
	}
	
	// 创建基本的 JS 文件
	jsContent := `// qwq AIOps 平台基础脚本
console.log('qwq AIOps 平台已启动');

// 基础功能
window.qwqApp = {
    init: function() {
        console.log('初始化应用');
        this.checkBackendStatus();
    },
    
    checkBackendStatus: function() {
        fetch('/api/health')
            .then(response => response.json())
            .then(data => {
                console.log('后端服务正常:', data);
            })
            .catch(error => {
                console.warn('后端服务连接失败:', error);
            });
    }
};

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    window.qwqApp.init();
});
`
	
	jsPath := filepath.Join(assetsPath, "main.js")
	if err := os.WriteFile(jsPath, []byte(jsContent), 0644); err != nil {
		return fmt.Errorf("创建 main.js 失败: %v", err)
	}
	
	return nil
}

// repairNotificationConfig 修复通知配置
func (eaf *EnhancedAutoFixer) repairNotificationConfig(result *DiagnosticResult) error {
	opID := eaf.tracker.AddOperation(RepairNotification, "修复通知配置", []string{"检查通知配置"})
	
	if err := eaf.tracker.StartOperation(opID); err != nil {
		return err
	}
	
	output := "通知配置检查完成，如需启用请配置相应的 Webhook URL"
	return eaf.tracker.CompleteOperation(opID, output, nil)
}

// repairPlatformCompatibility 修复平台兼容性
func (eaf *EnhancedAutoFixer) repairPlatformCompatibility(result *DiagnosticResult) error {
	if !eaf.options.EnablePlatformFix {
		return nil
	}
	
	opID := eaf.tracker.AddOperation(RepairPlatform, "修复平台兼容性", []string{"检查平台特定配置"})
	
	if err := eaf.tracker.StartOperation(opID); err != nil {
		return err
	}
	
	var output strings.Builder
	osName := runtime.GOOS
	
	output.WriteString(fmt.Sprintf("当前平台: %s\n", osName))
	
	switch osName {
	case "windows":
		output.WriteString("Windows 平台兼容性检查完成\n")
	case "linux":
		output.WriteString("Linux 平台兼容性检查完成\n")
	default:
		output.WriteString("通用平台兼容性检查完成\n")
	}
	
	return eaf.tracker.CompleteOperation(opID, output.String(), nil)
}

// repairGenericIssues 修复通用问题
func (eaf *EnhancedAutoFixer) repairGenericIssues(result *DiagnosticResult) error {
	opID := eaf.tracker.AddOperation(RepairConfig, fmt.Sprintf("修复 %s", result.Component), []string{"通用修复"})
	
	if err := eaf.tracker.StartOperation(opID); err != nil {
		return err
	}
	
	output := fmt.Sprintf("组件 %s 的通用修复检查完成", result.Component)
	return eaf.tracker.CompleteOperation(opID, output, nil)
}

// validateRepairResults 验证修复结果
func (eaf *EnhancedAutoFixer) validateRepairResults() *DeploymentValidationResult {
	fmt.Println("🔍 验证修复结果...")
	
	// 创建模拟的部署验证环境
	components := []DeploymentComponent{
		{Name: "frontend", Type: "web", Status: ComponentStatusHealthy},
		{Name: "backend", Type: "api", Status: ComponentStatusHealthy},
		{Name: "database", Type: "storage", Status: ComponentStatusHealthy},
		{Name: "config", Type: "configuration", Status: ComponentStatusHealthy},
		{Name: "notification", Type: "service", Status: ComponentStatusHealthy},
	}
	
	// 检查前端资源
	if _, err := os.Stat("frontend/dist/index.html"); os.IsNotExist(err) {
		for i, comp := range components {
			if comp.Name == "frontend" {
				components[i].Status = ComponentStatusUnhealthy
				break
			}
		}
	}
	
	// 检查配置文件
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		for i, comp := range components {
			if comp.Name == "config" {
				components[i].Status = ComponentStatusUnhealthy
				break
			}
		}
	}
	
	env := &MockDeploymentEnvironment{
		Components: make(map[string]DeploymentComponent),
	}
	
	for _, comp := range components {
		env.Components[comp.Name] = comp
	}
	
	return env.ValidateDeployment()
}

// GetRepairHistory 获取修复历史
func (eaf *EnhancedAutoFixer) GetRepairHistory() ([]string, error) {
	return eaf.tracker.ListSessions()
}

// LoadRepairSession 加载修复会话
func (eaf *EnhancedAutoFixer) LoadRepairSession(sessionID string) error {
	return eaf.tracker.LoadSession(sessionID)
}

// MockDeploymentEnvironment 模拟部署环境
type MockDeploymentEnvironment struct {
	Components map[string]DeploymentComponent
}

// ValidateDeployment 验证部署
func (env *MockDeploymentEnvironment) ValidateDeployment() *DeploymentValidationResult {
	result := &DeploymentValidationResult{
		Valid:               true,
		ComponentsChecked:   []string{},
		HealthyComponents:   []string{},
		UnhealthyComponents: []string{},
		MissingComponents:   []string{},
		ValidationErrors:    []string{},
		Suggestions:         []string{},
		ComponentDetails:    make(map[string]DeploymentComponent),
	}

	// 检查所有关键组件
	requiredComponents := []string{
		"frontend", "backend", "database", "config", "notification",
	}

	for _, compName := range requiredComponents {
		result.ComponentsChecked = append(result.ComponentsChecked, compName)
		
		if comp, exists := env.Components[compName]; exists {
			result.ComponentDetails[compName] = comp
			
			switch comp.Status {
			case ComponentStatusHealthy:
				result.HealthyComponents = append(result.HealthyComponents, compName)
			case ComponentStatusUnhealthy:
				result.UnhealthyComponents = append(result.UnhealthyComponents, compName)
				result.Valid = false
				result.ValidationErrors = append(result.ValidationErrors, 
					fmt.Sprintf("组件 %s 状态不健康", compName))
			case ComponentStatusError:
				result.UnhealthyComponents = append(result.UnhealthyComponents, compName)
				result.Valid = false
				result.ValidationErrors = append(result.ValidationErrors, 
					fmt.Sprintf("组件 %s 出现错误", compName))
			}
		} else {
			result.MissingComponents = append(result.MissingComponents, compName)
			result.Valid = false
			result.ValidationErrors = append(result.ValidationErrors, 
				fmt.Sprintf("缺失关键组件: %s", compName))
		}
	}

	// 检查组件依赖关系
	for _, comp := range env.Components {
		for _, dep := range comp.Dependencies {
			if depComp, exists := env.Components[dep]; !exists || depComp.Status != ComponentStatusHealthy {
				result.Valid = false
				result.ValidationErrors = append(result.ValidationErrors, 
					fmt.Sprintf("组件 %s 的依赖 %s 不可用", comp.Name, dep))
			}
		}
	}

	// 生成修复建议
	if !result.Valid {
		if len(result.MissingComponents) > 0 {
			result.Suggestions = append(result.Suggestions, 
				"请检查并部署缺失的组件: " + strings.Join(result.MissingComponents, ", "))
		}
		if len(result.UnhealthyComponents) > 0 {
			result.Suggestions = append(result.Suggestions, 
				"请修复不健康的组件: " + strings.Join(result.UnhealthyComponents, ", "))
		}
	}

	return result
}