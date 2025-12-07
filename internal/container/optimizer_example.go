package container

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// ExampleArchitectureOptimizer 演示如何使用架构优化分析器
func ExampleArchitectureOptimizer() {
	// 创建优化器实例
	optimizer := NewArchitectureOptimizer()

	// 示例 Compose 配置
	config := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image:   "nginx:latest",
				Ports:   []string{"80:80"},
				Restart: "unless-stopped",
				Networks: []string{"frontend"},
			},
			"api": {
				Image:   "myapp:1.0.0",
				Ports:   []string{"8080:8080"},
				Restart: "on-failure",
				DependsOn: []interface{}{"db"},
				Networks: []string{"frontend", "backend"},
				HealthCheck: &HealthCheck{
					Test:     []interface{}{"CMD", "curl", "-f", "http://localhost:8080/health"},
					Interval: "30s",
					Timeout:  "10s",
					Retries:  3,
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
				Networks: []string{"backend"},
				Volumes: []string{"db-data:/var/lib/postgresql/data"},
				Environment: map[string]string{
					"POSTGRES_PASSWORD": "secret",
				},
			},
		},
		Networks: map[string]*Network{
			"frontend": {
				Driver: "bridge",
			},
			"backend": {
				Driver: "bridge",
			},
		},
		Volumes: map[string]*Volume{
			"db-data": {
				Driver: "local",
			},
		},
	}

	ctx := context.Background()

	// 1. 分析架构
	fmt.Println("=== 架构分析 ===")
	analysis, err := optimizer.AnalyzeArchitecture(ctx, config)
	if err != nil {
		log.Fatalf("架构分析失败: %v", err)
	}

	fmt.Printf("服务总数: %d\n", analysis.TotalServices)
	fmt.Printf("网络总数: %d\n", analysis.TotalNetworks)
	fmt.Printf("卷总数: %d\n", analysis.TotalVolumes)
	fmt.Printf("复杂度: %s\n", analysis.Complexity)
	fmt.Printf("健康评分: %d/100\n", analysis.HealthScore)
	fmt.Printf("发现问题: %d 个\n\n", len(analysis.Issues))

	// 打印问题
	for i, issue := range analysis.Issues {
		fmt.Printf("问题 %d: [%s] %s\n", i+1, issue.Severity, issue.Title)
		fmt.Printf("  服务: %s\n", issue.Service)
		fmt.Printf("  影响: %s\n", issue.Impact)
		fmt.Printf("  建议: %s\n\n", issue.Suggestion)
	}

	// 2. 生成优化建议
	fmt.Println("=== 优化建议 ===")
	optimizations, err := optimizer.GenerateOptimizations(ctx, analysis)
	if err != nil {
		log.Fatalf("生成优化建议失败: %v", err)
	}

	for i, opt := range optimizations {
		fmt.Printf("建议 %d: [%s] %s\n", i+1, opt.Priority, opt.Title)
		fmt.Printf("  类别: %s\n", opt.Category)
		fmt.Printf("  描述: %s\n", opt.Description)
		fmt.Printf("  实施: %s\n", opt.Implementation)
		if opt.EstimatedImpact != nil {
			fmt.Printf("  预估影响:\n")
			fmt.Printf("    性能提升: %s\n", opt.EstimatedImpact.PerformanceGain)
			fmt.Printf("    安全改进: %s\n", opt.EstimatedImpact.SecurityImprovement)
			fmt.Printf("    实施难度: %s\n", opt.EstimatedImpact.Effort)
		}
		fmt.Println()
	}

	// 3. 生成安全建议
	fmt.Println("=== 安全建议 ===")
	securityRecs, err := optimizer.GenerateSecurityRecommendations(ctx, config)
	if err != nil {
		log.Fatalf("生成安全建议失败: %v", err)
	}

	for i, rec := range securityRecs {
		fmt.Printf("安全建议 %d: [%s] %s\n", i+1, rec.Severity, rec.Title)
		fmt.Printf("  服务: %s\n", rec.Service)
		fmt.Printf("  风险: %s\n", rec.Risk)
		fmt.Printf("  缓解: %s\n", rec.Mitigation)
		if rec.CodeExample != "" {
			fmt.Printf("  示例:\n%s\n", rec.CodeExample)
		}
		fmt.Println()
	}

	// 4. 分析依赖关系
	fmt.Println("=== 依赖关系分析 ===")
	depGraph, err := optimizer.AnalyzeDependencies(ctx, config)
	if err != nil {
		log.Fatalf("依赖关系分析失败: %v", err)
	}

	fmt.Printf("服务层级:\n")
	for i, layer := range depGraph.Layers {
		fmt.Printf("  层级 %d: %v\n", i, layer)
	}

	if len(depGraph.CriticalPath) > 0 {
		fmt.Printf("关键路径: %v\n", depGraph.CriticalPath)
	}

	if len(depGraph.Cycles) > 0 {
		fmt.Printf("检测到循环依赖: %v\n", depGraph.Cycles)
	}
	fmt.Println()

	// 5. 性能评估
	fmt.Println("=== 性能评估 ===")
	perfEval, err := optimizer.EvaluatePerformance(ctx, config)
	if err != nil {
		log.Fatalf("性能评估失败: %v", err)
	}

	fmt.Printf("总体评分: %d/100\n", perfEval.OverallScore)
	fmt.Printf("资源效率: %d/100\n", perfEval.Metrics.ResourceEfficiency)
	fmt.Printf("可扩展性: %d/100\n", perfEval.Metrics.ScalabilityScore)
	fmt.Printf("可靠性: %d/100\n", perfEval.Metrics.ReliabilityScore)
	fmt.Printf("启动时间估算: %s\n", perfEval.Metrics.StartupTimeEstimate)
	fmt.Printf("内存占用: %s\n\n", perfEval.Metrics.MemoryFootprint)

	fmt.Printf("性能瓶颈: %d 个\n", len(perfEval.Bottlenecks))
	for i, bottleneck := range perfEval.Bottlenecks {
		fmt.Printf("  %d. [%s] %s - %s\n", i+1, bottleneck.Service, bottleneck.Type, bottleneck.Description)
	}
	fmt.Println()

	// 6. 生成可视化数据
	fmt.Println("=== 架构可视化 ===")
	viz, err := optimizer.GenerateVisualization(ctx, config)
	if err != nil {
		log.Fatalf("生成可视化失败: %v", err)
	}

	fmt.Printf("节点数: %d\n", len(viz.Nodes))
	fmt.Printf("边数: %d\n", len(viz.Edges))
	fmt.Printf("布局: %s\n", viz.Layout)
	fmt.Printf("复杂度: %s\n\n", viz.Metadata.Complexity)

	// 输出 JSON 格式（用于前端可视化）
	vizJSON, _ := json.MarshalIndent(viz, "", "  ")
	fmt.Printf("可视化数据 (JSON):\n%s\n", string(vizJSON))
}
