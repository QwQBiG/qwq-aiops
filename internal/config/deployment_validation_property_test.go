// Package config 部署验证属性测试
// 使用 gopter 库进行属性基础测试，验证部署验证功能的全面性和正确性
// 主要测试部署环境中各组件的状态检查、依赖关系验证和错误处理
package config

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// ============================================
// 测试辅助函数 - 模拟部署环境
// ============================================

// 这些类型定义已移动到 repair_tracker.go 文件中

// createMockDeploymentEnvironment 创建模拟部署环境
// 用于测试的辅助函数，将组件列表转换为可验证的部署环境
func createMockDeploymentEnvironment(components []DeploymentComponent) *MockDeploymentEnvironment {
	env := &MockDeploymentEnvironment{
		Components: make(map[string]DeploymentComponent),
	}
	
	// 将组件列表转换为以名称为键的映射，便于快速查找
	for _, comp := range components {
		env.Components[comp.Name] = comp
	}
	
	return env
}

// 使用 enhanced_auto_fixer.go 中的 MockDeploymentEnvironment 和 ValidateDeployment 方法

// ============================================
// 属性测试 17: 部署验证全面性
// ============================================

// **Feature: deployment-ai-config-fix, Property 17: 部署验证全面性**
// **Validates: Requirements 5.1**
//
// 验证内容：
// 1. 所有关键组件都被检查
// 2. 组件状态被正确识别
// 3. 依赖关系被正确验证
// 4. 验证失败时提供有用的建议
func TestProperty17_DeploymentValidationComprehensiveness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// 属性 1: 所有关键组件都被检查
	// 验证部署验证器能够检查所有必需的系统组件
	properties.Property("所有关键组件都被检查", prop.ForAll(
		func(extraComponents int) bool {
			// 创建包含所有关键组件的完整部署环境
			components := []DeploymentComponent{
				{Name: "frontend", Type: "web", Status: ComponentStatusHealthy},        // 前端服务
				{Name: "backend", Type: "api", Status: ComponentStatusHealthy},         // 后端API
				{Name: "database", Type: "storage", Status: ComponentStatusHealthy},    // 数据库
				{Name: "config", Type: "configuration", Status: ComponentStatusHealthy}, // 配置服务
				{Name: "notification", Type: "service", Status: ComponentStatusHealthy}, // 通知服务
			}

			// 添加额外组件
			for i := 0; i < extraComponents; i++ {
				components = append(components, DeploymentComponent{
					Name:   fmt.Sprintf("extra%d", i),
					Type:   "extra",
					Status: ComponentStatusHealthy,
				})
			}

			env := createMockDeploymentEnvironment(components)
			result := env.ValidateDeployment()

			// 验证所有关键组件都被检查
			requiredComponents := []string{"frontend", "backend", "database", "config", "notification"}
			for _, comp := range requiredComponents {
				found := false
				for _, checked := range result.ComponentsChecked {
					if checked == comp {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
			return true
		},
		gen.IntRange(0, 5),
	))

	// 属性 2: 缺失关键组件时验证失败
	// 验证当系统缺少必需组件时，验证器能正确识别并报告失败
	properties.Property("缺失关键组件时验证失败", prop.ForAll(
		func(missingComponent string) bool {
			// 创建缺少某个关键组件的不完整部署环境
			components := []DeploymentComponent{
				{Name: "frontend", Type: "web", Status: ComponentStatusHealthy},
				{Name: "backend", Type: "api", Status: ComponentStatusHealthy},
				{Name: "database", Type: "storage", Status: ComponentStatusHealthy},
				{Name: "config", Type: "configuration", Status: ComponentStatusHealthy},
				{Name: "notification", Type: "service", Status: ComponentStatusHealthy},
			}

			// 移除指定的组件
			filteredComponents := []DeploymentComponent{}
			for _, comp := range components {
				if comp.Name != missingComponent {
					filteredComponents = append(filteredComponents, comp)
				}
			}

			env := createMockDeploymentEnvironment(filteredComponents)
			result := env.ValidateDeployment()

			// 验证应该失败，并且缺失的组件应该被记录
			if result.Valid {
				return false
			}

			for _, missing := range result.MissingComponents {
				if missing == missingComponent {
					return true
				}
			}
			return false
		},
		gen.OneConstOf("frontend", "backend", "database", "config", "notification"),
	))

	// 属性 3: 不健康组件时验证失败
	properties.Property("不健康组件时验证失败", prop.ForAll(
		func(unhealthyComponent string, status ComponentStatus) bool {
			if status == ComponentStatusHealthy {
				return true // 跳过健康状态
			}

			components := []DeploymentComponent{
				{Name: "frontend", Type: "web", Status: ComponentStatusHealthy},
				{Name: "backend", Type: "api", Status: ComponentStatusHealthy},
				{Name: "database", Type: "storage", Status: ComponentStatusHealthy},
				{Name: "config", Type: "configuration", Status: ComponentStatusHealthy},
				{Name: "notification", Type: "service", Status: ComponentStatusHealthy},
			}

			// 设置指定组件为不健康状态
			for i, comp := range components {
				if comp.Name == unhealthyComponent {
					components[i].Status = status
					break
				}
			}

			env := createMockDeploymentEnvironment(components)
			result := env.ValidateDeployment()

			// 验证应该失败，并且不健康的组件应该被记录
			if result.Valid {
				return false
			}

			for _, unhealthy := range result.UnhealthyComponents {
				if unhealthy == unhealthyComponent {
					return true
				}
			}
			return false
		},
		gen.OneConstOf("frontend", "backend", "database", "config", "notification"),
		gen.OneConstOf(ComponentStatusUnhealthy, ComponentStatusError),
	))

	// 属性 4: 依赖关系验证
	properties.Property("依赖关系被正确验证", prop.ForAll(
		func(dependentComp, dependency string) bool {
			if dependentComp == dependency {
				return true // 跳过自依赖
			}

			components := []DeploymentComponent{
				{Name: "frontend", Type: "web", Status: ComponentStatusHealthy},
				{Name: "backend", Type: "api", Status: ComponentStatusHealthy},
				{Name: "database", Type: "storage", Status: ComponentStatusHealthy},
				{Name: "config", Type: "configuration", Status: ComponentStatusHealthy},
				{Name: "notification", Type: "service", Status: ComponentStatusHealthy},
			}

			// 添加依赖关系
			for i, comp := range components {
				if comp.Name == dependentComp {
					components[i].Dependencies = []string{dependency}
					break
				}
			}

			// 设置依赖组件为不健康状态
			for i, comp := range components {
				if comp.Name == dependency {
					components[i].Status = ComponentStatusUnhealthy
					break
				}
			}

			env := createMockDeploymentEnvironment(components)
			result := env.ValidateDeployment()

			// 验证应该失败，因为依赖不可用
			if result.Valid {
				return false
			}

			// 检查是否有依赖相关的错误
			for _, err := range result.ValidationErrors {
				if strings.Contains(err, dependentComp) && strings.Contains(err, dependency) {
					return true
				}
			}
			return false
		},
		gen.OneConstOf("frontend", "backend", "notification"),
		gen.OneConstOf("database", "config", "backend"),
	))

	// 属性 5: 验证失败时提供修复建议
	properties.Property("验证失败时提供修复建议", prop.ForAll(
		func(missingCount, unhealthyCount int) bool {
			if missingCount == 0 && unhealthyCount == 0 {
				return true // 跳过没有问题的情况
			}

			components := []DeploymentComponent{
				{Name: "frontend", Type: "web", Status: ComponentStatusHealthy},
				{Name: "backend", Type: "api", Status: ComponentStatusHealthy},
				{Name: "database", Type: "storage", Status: ComponentStatusHealthy},
				{Name: "config", Type: "configuration", Status: ComponentStatusHealthy},
				{Name: "notification", Type: "service", Status: ComponentStatusHealthy},
			}

			// 移除一些组件（模拟缺失）
			if missingCount > 0 {
				components = components[:len(components)-missingCount]
			}

			// 设置一些组件为不健康状态
			for i := 0; i < unhealthyCount && i < len(components); i++ {
				components[i].Status = ComponentStatusUnhealthy
			}

			env := createMockDeploymentEnvironment(components)
			result := env.ValidateDeployment()

			// 验证失败时应该有修复建议
			if !result.Valid {
				return len(result.Suggestions) > 0
			}
			return true
		},
		gen.IntRange(0, 2),
		gen.IntRange(0, 2),
	))

	// 属性 6: 所有组件都健康时验证通过
	properties.Property("所有组件健康时验证通过", prop.ForAll(
		func(extraHealthyComponents int) bool {
			components := []DeploymentComponent{
				{Name: "frontend", Type: "web", Status: ComponentStatusHealthy},
				{Name: "backend", Type: "api", Status: ComponentStatusHealthy},
				{Name: "database", Type: "storage", Status: ComponentStatusHealthy},
				{Name: "config", Type: "configuration", Status: ComponentStatusHealthy},
				{Name: "notification", Type: "service", Status: ComponentStatusHealthy},
			}

			// 添加额外的健康组件
			for i := 0; i < extraHealthyComponents; i++ {
				components = append(components, DeploymentComponent{
					Name:   fmt.Sprintf("extra%d", i),
					Type:   "extra",
					Status: ComponentStatusHealthy,
				})
			}

			env := createMockDeploymentEnvironment(components)
			result := env.ValidateDeployment()

			return result.Valid && len(result.ValidationErrors) == 0
		},
		gen.IntRange(0, 5),
	))

	// 属性 7: 组件详情被正确记录
	properties.Property("组件详情被正确记录", prop.ForAll(
		func(componentType string) bool {
			// 使用关键组件名称，因为验证只检查关键组件
			components := []DeploymentComponent{
				{Name: "frontend", Type: componentType, Status: ComponentStatusHealthy},
				{Name: "backend", Type: "api", Status: ComponentStatusHealthy},
				{Name: "database", Type: "storage", Status: ComponentStatusHealthy},
				{Name: "config", Type: "configuration", Status: ComponentStatusHealthy},
				{Name: "notification", Type: "service", Status: ComponentStatusHealthy},
			}

			env := createMockDeploymentEnvironment(components)
			result := env.ValidateDeployment()

			// 检查前端组件详情是否被记录
			if detail, exists := result.ComponentDetails["frontend"]; exists {
				return detail.Type == componentType && detail.Status == ComponentStatusHealthy
			}
			return false
		},
		gen.OneConstOf("web", "api", "storage", "configuration", "service", "cache", "queue"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}