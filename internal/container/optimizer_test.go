package container

import (
	"context"
	"testing"
)

func TestArchitectureOptimizer_AnalyzeArchitecture(t *testing.T) {
	optimizer := NewArchitectureOptimizer()
	ctx := context.Background()

	// 创建测试配置
	config := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image:   "nginx:latest",
				Ports:   []string{"80:80"},
				Restart: "unless-stopped",
			},
			"api": {
				Image:   "myapp:1.0.0",
				Ports:   []string{"8080:8080"},
				Restart: "on-failure",
				DependsOn: []interface{}{"db"},
				HealthCheck: &HealthCheck{
					Test:     []interface{}{"CMD", "curl", "-f", "http://localhost:8080/health"},
					Interval: "30s",
				},
				Deploy: &DeployConfig{
					Resources: &ResourcesConfig{
						Limits: &ResourceLimit{
							CPUs:   "1.0",
							Memory: "512M",
						},
					},
				},
			},
			"db": {
				Image:   "postgres:13",
				Restart: "unless-stopped",
			},
		},
		Networks: map[string]*Network{
			"default": {
				Driver: "bridge",
			},
		},
	}

	// 执行分析
	analysis, err := optimizer.AnalyzeArchitecture(ctx, config)
	if err != nil {
		t.Fatalf("AnalyzeArchitecture failed: %v", err)
	}

	// 验证结果
	if analysis == nil {
		t.Fatal("Analysis is nil")
	}

	if analysis.TotalServices != 3 {
		t.Errorf("Expected 3 services, got %d", analysis.TotalServices)
	}

	if analysis.Complexity != ComplexityLow {
		t.Errorf("Expected low complexity, got %s", analysis.Complexity)
	}

	if analysis.HealthScore < 0 || analysis.HealthScore > 100 {
		t.Errorf("Health score out of range: %d", analysis.HealthScore)
	}

	// 应该检测到一些问题（如缺少健康检查、资源限制等）
	if len(analysis.Issues) == 0 {
		t.Error("Expected some issues to be detected")
	}

	t.Logf("Analysis completed: %d services, %d issues, health score: %d", 
		analysis.TotalServices, len(analysis.Issues), analysis.HealthScore)
}

func TestArchitectureOptimizer_GenerateOptimizations(t *testing.T) {
	optimizer := NewArchitectureOptimizer()
	ctx := context.Background()

	config := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image: "nginx",
			},
		},
	}

	// 先分析
	analysis, err := optimizer.AnalyzeArchitecture(ctx, config)
	if err != nil {
		t.Fatalf("AnalyzeArchitecture failed: %v", err)
	}

	// 生成优化建议
	suggestions, err := optimizer.GenerateOptimizations(ctx, analysis)
	if err != nil {
		t.Fatalf("GenerateOptimizations failed: %v", err)
	}

	if suggestions == nil {
		t.Fatal("Suggestions is nil")
	}

	// 应该有一些优化建议
	if len(suggestions) == 0 {
		t.Error("Expected some optimization suggestions")
	}

	t.Logf("Generated %d optimization suggestions", len(suggestions))
}

func TestArchitectureOptimizer_GenerateSecurityRecommendations(t *testing.T) {
	optimizer := NewArchitectureOptimizer()
	ctx := context.Background()

	config := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image:      "nginx:latest",
				Privileged: true,
			},
		},
	}

	// 生成安全建议
	recommendations, err := optimizer.GenerateSecurityRecommendations(ctx, config)
	if err != nil {
		t.Fatalf("GenerateSecurityRecommendations failed: %v", err)
	}

	if recommendations == nil {
		t.Fatal("Recommendations is nil")
	}

	// 应该检测到特权模式和 latest 标签问题
	if len(recommendations) < 2 {
		t.Errorf("Expected at least 2 security recommendations, got %d", len(recommendations))
	}

	t.Logf("Generated %d security recommendations", len(recommendations))
}

func TestArchitectureOptimizer_GenerateVisualization(t *testing.T) {
	optimizer := NewArchitectureOptimizer()
	ctx := context.Background()

	config := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image:     "nginx",
				DependsOn: []interface{}{"api"},
			},
			"api": {
				Image:     "myapp",
				DependsOn: []interface{}{"db"},
			},
			"db": {
				Image: "postgres",
			},
		},
		Networks: map[string]*Network{
			"default": {
				Driver: "bridge",
			},
		},
		Volumes: map[string]*Volume{
			"db-data": {
				Driver: "local",
			},
		},
	}

	// 生成可视化
	viz, err := optimizer.GenerateVisualization(ctx, config)
	if err != nil {
		t.Fatalf("GenerateVisualization failed: %v", err)
	}

	if viz == nil {
		t.Fatal("Visualization is nil")
	}

	// 应该有 3 个服务节点 + 1 个网络节点 + 1 个卷节点
	expectedNodes := 5
	if len(viz.Nodes) != expectedNodes {
		t.Errorf("Expected %d nodes, got %d", expectedNodes, len(viz.Nodes))
	}

	// 应该有依赖关系边
	if len(viz.Edges) == 0 {
		t.Error("Expected some edges")
	}

	t.Logf("Generated visualization: %d nodes, %d edges", len(viz.Nodes), len(viz.Edges))
}

func TestArchitectureOptimizer_AnalyzeDependencies(t *testing.T) {
	optimizer := NewArchitectureOptimizer()
	ctx := context.Background()

	config := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image:     "nginx",
				DependsOn: []interface{}{"api"},
			},
			"api": {
				Image:     "myapp",
				DependsOn: []interface{}{"db", "cache"},
			},
			"db": {
				Image: "postgres",
			},
			"cache": {
				Image: "redis",
			},
		},
	}

	// 分析依赖
	depGraph, err := optimizer.AnalyzeDependencies(ctx, config)
	if err != nil {
		t.Fatalf("AnalyzeDependencies failed: %v", err)
	}

	if depGraph == nil {
		t.Fatal("Dependency graph is nil")
	}

	// 应该有 4 个服务
	if len(depGraph.Services) != 4 {
		t.Errorf("Expected 4 services, got %d", len(depGraph.Services))
	}

	// 应该有分层结构
	if len(depGraph.Layers) == 0 {
		t.Error("Expected some layers")
	}

	// db 和 cache 应该在第一层（没有依赖）
	firstLayer := depGraph.Layers[0]
	if len(firstLayer) != 2 {
		t.Errorf("Expected 2 services in first layer, got %d", len(firstLayer))
	}

	t.Logf("Dependency analysis: %d layers, critical path: %v", 
		len(depGraph.Layers), depGraph.CriticalPath)
}

func TestArchitectureOptimizer_EvaluatePerformance(t *testing.T) {
	optimizer := NewArchitectureOptimizer()
	ctx := context.Background()

	config := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image:   "nginx",
				Restart: "unless-stopped",
				HealthCheck: &HealthCheck{
					Test:     []interface{}{"CMD", "curl", "-f", "http://localhost"},
					Interval: "30s",
				},
				Deploy: &DeployConfig{
					Resources: &ResourcesConfig{
						Limits: &ResourceLimit{
							CPUs:   "0.5",
							Memory: "256M",
						},
					},
				},
			},
		},
	}

	// 评估性能
	evaluation, err := optimizer.EvaluatePerformance(ctx, config)
	if err != nil {
		t.Fatalf("EvaluatePerformance failed: %v", err)
	}

	if evaluation == nil {
		t.Fatal("Evaluation is nil")
	}

	if evaluation.OverallScore < 0 || evaluation.OverallScore > 100 {
		t.Errorf("Overall score out of range: %d", evaluation.OverallScore)
	}

	if evaluation.Metrics == nil {
		t.Fatal("Metrics is nil")
	}

	t.Logf("Performance evaluation: overall score %d, resource efficiency %d, scalability %d, reliability %d",
		evaluation.OverallScore, evaluation.Metrics.ResourceEfficiency, 
		evaluation.Metrics.ScalabilityScore, evaluation.Metrics.ReliabilityScore)
}
