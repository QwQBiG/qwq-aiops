package container

import (
	"context"
	"fmt"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: enhanced-aiops-platform, Property 9: AI 架构优化建议质量**
// **Validates: Requirements 3.3**

// TestProperty9_OptimizationSuggestionQuality_AllServicesAnalyzed 测试所有服务都被分析
func TestProperty9_OptimizationSuggestionQuality_AllServicesAnalyzed(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("对于任何服务架构配置，AI 分析应该覆盖所有服务", prop.ForAll(
		func(serviceCount int) bool {
			optimizer := NewArchitectureOptimizer()
			ctx := context.Background()

			// 生成测试配置
			config := generateTestConfig(serviceCount)

			// 执行分析
			analysis, err := optimizer.AnalyzeArchitecture(ctx, config)
			if err != nil {
				t.Logf("分析失败: %v", err)
				return false
			}

			// 验证所有服务都被分析
			if len(analysis.ServiceAnalysis) != serviceCount {
				t.Logf("期望分析 %d 个服务，实际分析了 %d 个", serviceCount, len(analysis.ServiceAnalysis))
				return false
			}

			// 验证每个服务都有分析结果
			for serviceName := range config.Services {
				if _, exists := analysis.ServiceAnalysis[serviceName]; !exists {
					t.Logf("服务 %s 未被分析", serviceName)
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty9_OptimizationSuggestionQuality_IssuesDetected 测试问题检测
func TestProperty9_OptimizationSuggestionQuality_IssuesDetected(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("对于有明显问题的配置，AI 应该能检测到问题", prop.ForAll(
		func(hasHealthCheck bool, hasResourceLimits bool, hasRestartPolicy bool) bool {
			optimizer := NewArchitectureOptimizer()
			ctx := context.Background()

			// 创建有问题的配置
			config := &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"test-service": {
						Image:   "nginx:latest",
						Restart: "",
					},
				},
				Networks: map[string]*Network{
					"default": {Driver: "bridge"},
				},
			}

			// 根据参数设置配置
			if hasHealthCheck {
				config.Services["test-service"].HealthCheck = &HealthCheck{
					Test:     []interface{}{"CMD", "curl", "-f", "http://localhost"},
					Interval: "30s",
				}
			}

			if hasResourceLimits {
				config.Services["test-service"].Deploy = &DeployConfig{
					Resources: &ResourcesConfig{
						Limits: &ResourceLimit{
							CPUs:   "1.0",
							Memory: "512M",
						},
					},
				}
			}

			if hasRestartPolicy {
				config.Services["test-service"].Restart = "unless-stopped"
			}

			// 执行分析
			analysis, err := optimizer.AnalyzeArchitecture(ctx, config)
			if err != nil {
				t.Logf("分析失败: %v", err)
				return false
			}

			// 如果配置不完善，应该检测到问题
			expectedIssues := 0
			if !hasHealthCheck {
				expectedIssues++
			}
			if !hasResourceLimits {
				expectedIssues++
			}
			if !hasRestartPolicy {
				expectedIssues++
			}

			if expectedIssues > 0 && len(analysis.Issues) == 0 {
				t.Logf("期望检测到 %d 个问题，但未检测到任何问题", expectedIssues)
				return false
			}

			return true
		},
		gen.Bool(),
		gen.Bool(),
		gen.Bool(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty9_OptimizationSuggestionQuality_SuggestionsRelevant 测试优化建议相关性
func TestProperty9_OptimizationSuggestionQuality_SuggestionsRelevant(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("对于任何架构分析结果，生成的优化建议应该与检测到的问题相关", prop.ForAll(
		func(serviceCount int) bool {
			optimizer := NewArchitectureOptimizer()
			ctx := context.Background()

			// 生成测试配置
			config := generateTestConfig(serviceCount)

			// 执行分析
			analysis, err := optimizer.AnalyzeArchitecture(ctx, config)
			if err != nil {
				t.Logf("分析失败: %v", err)
				return false
			}

			// 生成优化建议
			suggestions, err := optimizer.GenerateOptimizations(ctx, analysis)
			if err != nil {
				t.Logf("生成优化建议失败: %v", err)
				return false
			}

			// 验证建议不为空（如果有问题的话）
			if len(analysis.Issues) > 0 && len(suggestions) == 0 {
				t.Logf("检测到 %d 个问题，但未生成任何优化建议", len(analysis.Issues))
				return false
			}

			// 验证每个建议都有必要的字段
			for _, suggestion := range suggestions {
				if suggestion.Title == "" {
					t.Logf("优化建议缺少标题")
					return false
				}
				if suggestion.Description == "" {
					t.Logf("优化建议缺少描述")
					return false
				}
				if suggestion.Implementation == "" {
					t.Logf("优化建议缺少实施方法")
					return false
				}
				if len(suggestion.Benefits) == 0 {
					t.Logf("优化建议缺少收益说明")
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty9_OptimizationSuggestionQuality_SecurityRecommendations 测试安全建议
func TestProperty9_OptimizationSuggestionQuality_SecurityRecommendations(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("对于有安全问题的配置，AI 应该提供安全加固建议", prop.ForAll(
		func(useLatestTag bool, usePrivileged bool, useRootUser bool) bool {
			optimizer := NewArchitectureOptimizer()
			ctx := context.Background()

			// 创建有安全问题的配置
			config := &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"test-service": {
						Image:      "nginx",
						Privileged: usePrivileged,
						User:       "",
					},
				},
			}

			if useLatestTag {
				config.Services["test-service"].Image = "nginx:latest"
			} else {
				config.Services["test-service"].Image = "nginx:1.21.0"
			}

			if !useRootUser {
				config.Services["test-service"].User = "1000:1000"
			}

			// 生成安全建议
			recommendations, err := optimizer.GenerateSecurityRecommendations(ctx, config)
			if err != nil {
				t.Logf("生成安全建议失败: %v", err)
				return false
			}

			// 计算预期的安全问题数量
			expectedIssues := 0
			if useLatestTag {
				expectedIssues++
			}
			if usePrivileged {
				expectedIssues++
			}
			if useRootUser {
				expectedIssues++
			}

			// 如果有安全问题，应该生成相应的建议
			if expectedIssues > 0 {
				serviceSpecificRecs := 0
				for _, rec := range recommendations {
					if rec.Service == "test-service" {
						serviceSpecificRecs++
					}
				}

				if serviceSpecificRecs == 0 {
					t.Logf("期望至少有 1 个针对服务的安全建议，但实际为 0")
					return false
				}
			}

			// 验证每个建议都有必要的字段
			for _, rec := range recommendations {
				if rec.Title == "" {
					t.Logf("安全建议缺少标题")
					return false
				}
				if rec.Description == "" {
					t.Logf("安全建议缺少描述")
					return false
				}
				if rec.Risk == "" {
					t.Logf("安全建议缺少风险说明")
					return false
				}
				if rec.Mitigation == "" {
					t.Logf("安全建议缺少缓解措施")
					return false
				}
			}

			return true
		},
		gen.Bool(),
		gen.Bool(),
		gen.Bool(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty9_OptimizationSuggestionQuality_HealthScoreRange 测试健康评分范围
func TestProperty9_OptimizationSuggestionQuality_HealthScoreRange(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("对于任何服务架构配置，健康评分应该在 0-100 范围内", prop.ForAll(
		func(serviceCount int) bool {
			optimizer := NewArchitectureOptimizer()
			ctx := context.Background()

			// 生成测试配置
			config := generateTestConfig(serviceCount)

			// 执行分析
			analysis, err := optimizer.AnalyzeArchitecture(ctx, config)
			if err != nil {
				t.Logf("分析失败: %v", err)
				return false
			}

			// 验证健康评分在有效范围内
			if analysis.HealthScore < 0 || analysis.HealthScore > 100 {
				t.Logf("健康评分 %d 超出有效范围 [0, 100]", analysis.HealthScore)
				return false
			}

			return true
		},
		gen.IntRange(1, 15),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty9_OptimizationSuggestionQuality_ComplexityClassification 测试复杂度分类
func TestProperty9_OptimizationSuggestionQuality_ComplexityClassification(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("对于任何服务架构配置，复杂度分类应该准确", prop.ForAll(
		func(serviceCount int) bool {
			optimizer := NewArchitectureOptimizer()
			ctx := context.Background()

			// 生成测试配置
			config := generateTestConfig(serviceCount)

			// 执行分析
			analysis, err := optimizer.AnalyzeArchitecture(ctx, config)
			if err != nil {
				t.Logf("分析失败: %v", err)
				return false
			}

			// 验证复杂度分类
			var expectedComplexity ComplexityLevel
			if serviceCount <= 3 {
				expectedComplexity = ComplexityLow
			} else if serviceCount <= 10 {
				expectedComplexity = ComplexityMedium
			} else {
				expectedComplexity = ComplexityHigh
			}

			if analysis.Complexity != expectedComplexity {
				t.Logf("服务数量 %d，期望复杂度 %s，实际复杂度 %s", 
					serviceCount, expectedComplexity, analysis.Complexity)
				return false
			}

			return true
		},
		gen.IntRange(1, 20),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty9_OptimizationSuggestionQuality_PerformanceEvaluation 测试性能评估
func TestProperty9_OptimizationSuggestionQuality_PerformanceEvaluation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("对于任何服务架构配置，性能评估应该提供有价值的指标", prop.ForAll(
		func(serviceCount int) bool {
			optimizer := NewArchitectureOptimizer()
			ctx := context.Background()

			// 生成测试配置
			config := generateTestConfig(serviceCount)

			// 执行性能评估
			evaluation, err := optimizer.EvaluatePerformance(ctx, config)
			if err != nil {
				t.Logf("性能评估失败: %v", err)
				return false
			}

			// 验证总体评分在有效范围内
			if evaluation.OverallScore < 0 || evaluation.OverallScore > 100 {
				t.Logf("总体评分 %d 超出有效范围 [0, 100]", evaluation.OverallScore)
				return false
			}

			// 验证指标存在
			if evaluation.Metrics == nil {
				t.Logf("性能指标为空")
				return false
			}

			// 验证各项指标在有效范围内
			if evaluation.Metrics.ResourceEfficiency < 0 || evaluation.Metrics.ResourceEfficiency > 100 {
				t.Logf("资源效率评分 %d 超出有效范围", evaluation.Metrics.ResourceEfficiency)
				return false
			}

			if evaluation.Metrics.ScalabilityScore < 0 || evaluation.Metrics.ScalabilityScore > 100 {
				t.Logf("可扩展性评分 %d 超出有效范围", evaluation.Metrics.ScalabilityScore)
				return false
			}

			if evaluation.Metrics.ReliabilityScore < 0 || evaluation.Metrics.ReliabilityScore > 100 {
				t.Logf("可靠性评分 %d 超出有效范围", evaluation.Metrics.ReliabilityScore)
				return false
			}

			// 验证对比数据存在
			if evaluation.Comparison == nil {
				t.Logf("性能对比数据为空")
				return false
			}

			return true
		},
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty9_OptimizationSuggestionQuality_DependencyAnalysis 测试依赖分析
func TestProperty9_OptimizationSuggestionQuality_DependencyAnalysis(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("对于任何服务架构配置，依赖分析应该正确识别依赖关系", prop.ForAll(
		func(serviceCount int) bool {
			if serviceCount < 2 {
				return true // 至少需要 2 个服务才能有依赖关系
			}

			optimizer := NewArchitectureOptimizer()
			ctx := context.Background()

			// 生成有依赖关系的配置
			config := generateConfigWithDependencies(serviceCount)

			// 执行依赖分析
			depGraph, err := optimizer.AnalyzeDependencies(ctx, config)
			if err != nil {
				t.Logf("依赖分析失败: %v", err)
				return false
			}

			// 验证所有服务都在依赖图中
			if len(depGraph.Services) != serviceCount {
				t.Logf("期望 %d 个服务，依赖图中有 %d 个", serviceCount, len(depGraph.Services))
				return false
			}

			// 验证分层结构
			if len(depGraph.Layers) == 0 {
				t.Logf("依赖图应该有分层结构")
				return false
			}

			// 验证所有服务都被分配到某一层
			totalServicesInLayers := 0
			for _, layer := range depGraph.Layers {
				totalServicesInLayers += len(layer)
			}

			if totalServicesInLayers != serviceCount {
				t.Logf("分层中的服务总数 %d 与实际服务数 %d 不匹配", 
					totalServicesInLayers, serviceCount)
				return false
			}

			return true
		},
		gen.IntRange(2, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty9_OptimizationSuggestionQuality_VisualizationCompleteness 测试可视化完整性
func TestProperty9_OptimizationSuggestionQuality_VisualizationCompleteness(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("对于任何服务架构配置，可视化数据应该包含所有组件", prop.ForAll(
		func(serviceCount int, networkCount int, volumeCount int) bool {
			optimizer := NewArchitectureOptimizer()
			ctx := context.Background()

			// 生成测试配置
			config := generateComplexConfig(serviceCount, networkCount, volumeCount)

			// 生成可视化
			viz, err := optimizer.GenerateVisualization(ctx, config)
			if err != nil {
				t.Logf("生成可视化失败: %v", err)
				return false
			}

			// 验证节点数量（服务 + 网络 + 卷）
			expectedNodes := serviceCount + networkCount + volumeCount
			if len(viz.Nodes) != expectedNodes {
				t.Logf("期望 %d 个节点，实际有 %d 个", expectedNodes, len(viz.Nodes))
				return false
			}

			// 验证元数据
			if viz.Metadata == nil {
				t.Logf("可视化元数据为空")
				return false
			}

			if viz.Metadata.ServiceCount != serviceCount {
				t.Logf("元数据中的服务数量 %d 与实际 %d 不匹配", 
					viz.Metadata.ServiceCount, serviceCount)
				return false
			}

			if viz.Metadata.NetworkCount != networkCount {
				t.Logf("元数据中的网络数量 %d 与实际 %d 不匹配", 
					viz.Metadata.NetworkCount, networkCount)
				return false
			}

			if viz.Metadata.VolumeCount != volumeCount {
				t.Logf("元数据中的卷数量 %d 与实际 %d 不匹配", 
					viz.Metadata.VolumeCount, volumeCount)
				return false
			}

			return true
		},
		gen.IntRange(1, 5),
		gen.IntRange(1, 3),
		gen.IntRange(0, 3),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// 辅助函数：生成测试配置
func generateTestConfig(serviceCount int) *ComposeConfig {
	config := &ComposeConfig{
		Version:  "3.8",
		Services: make(map[string]*Service),
		Networks: map[string]*Network{
			"default": {Driver: "bridge"},
		},
	}

	for i := 0; i < serviceCount; i++ {
		serviceName := fmt.Sprintf("service-%d", i)
		config.Services[serviceName] = &Service{
			Image:   fmt.Sprintf("app:v%d", i),
			Restart: "no",
		}
	}

	return config
}

// 辅助函数：生成有依赖关系的配置
func generateConfigWithDependencies(serviceCount int) *ComposeConfig {
	config := &ComposeConfig{
		Version:  "3.8",
		Services: make(map[string]*Service),
		Networks: map[string]*Network{
			"default": {Driver: "bridge"},
		},
	}

	// 创建链式依赖：service-0 <- service-1 <- service-2 ...
	for i := 0; i < serviceCount; i++ {
		serviceName := fmt.Sprintf("service-%d", i)
		service := &Service{
			Image:   fmt.Sprintf("app:v%d", i),
			Restart: "unless-stopped",
		}

		// 添加依赖（除了第一个服务）
		if i > 0 {
			service.DependsOn = []interface{}{fmt.Sprintf("service-%d", i-1)}
		}

		config.Services[serviceName] = service
	}

	return config
}

// 辅助函数：生成复杂配置
func generateComplexConfig(serviceCount int, networkCount int, volumeCount int) *ComposeConfig {
	config := &ComposeConfig{
		Version:  "3.8",
		Services: make(map[string]*Service),
		Networks: make(map[string]*Network),
		Volumes:  make(map[string]*Volume),
	}

	// 创建服务
	for i := 0; i < serviceCount; i++ {
		serviceName := fmt.Sprintf("service-%d", i)
		config.Services[serviceName] = &Service{
			Image:   fmt.Sprintf("app:v%d", i),
			Restart: "unless-stopped",
		}
	}

	// 创建网络
	for i := 0; i < networkCount; i++ {
		networkName := fmt.Sprintf("network-%d", i)
		config.Networks[networkName] = &Network{
			Driver: "bridge",
		}
	}

	// 创建卷
	for i := 0; i < volumeCount; i++ {
		volumeName := fmt.Sprintf("volume-%d", i)
		config.Volumes[volumeName] = &Volume{
			Driver: "local",
		}
	}

	return config
}
