# AI 架构优化分析器

## 概述

AI 架构优化分析器是 qwq AIOps 平台的核心功能之一，用于分析 Docker Compose 架构并提供智能优化建议、安全加固提示和架构可视化。

## 功能特性

### 1. 架构分析 (Architecture Analysis)

全面分析 Docker Compose 配置，包括：

- **服务统计**: 服务、网络、卷的数量统计
- **复杂度评估**: 自动评估架构复杂度（低/中/高）
- **健康评分**: 0-100 分的健康评分
- **问题检测**: 自动检测配置问题和潜在风险
- **服务分析**: 每个服务的详细分析
- **网络拓扑**: 网络连接和隔离分析
- **资源估算**: CPU、内存、磁盘使用估算

### 2. 优化建议 (Optimization Suggestions)

基于分析结果生成智能优化建议：

- **性能优化**: 资源配置、缓存策略、网络优化
- **安全优化**: 权限控制、镜像安全、网络隔离
- **可靠性优化**: 健康检查、重启策略、故障恢复
- **成本优化**: 资源利用率、存储优化
- **可维护性优化**: 日志管理、监控告警、服务网格

每个建议包含：
- 优先级（关键/高/中/低）
- 预估影响（性能提升、安全改进、成本节省）
- 实施方法和代码示例

### 3. 安全建议 (Security Recommendations)

专门的安全分析和建议：

- **镜像安全**: 检测 latest 标签、未指定版本
- **权限安全**: 检测特权模式、root 用户
- **网络安全**: 检测端口暴露、网络隔离
- **数据安全**: 检测敏感信息管理、secrets 使用
- **最佳实践**: 只读文件系统、镜像扫描

每个建议包含：
- 严重程度（严重/高/中/低）
- 风险说明和缓解措施
- 参考资料和代码示例

### 4. 架构可视化 (Architecture Visualization)

生成可视化数据用于前端展示：

- **节点**: 服务、网络、卷的可视化节点
- **边**: 依赖关系、网络连接、卷挂载
- **样式**: 颜色、形状、大小、图标
- **布局**: 层次化布局
- **元数据**: 复杂度、统计信息

输出 JSON 格式，可直接用于前端图形库（如 vis.js、cytoscape.js）。

### 5. 依赖关系分析 (Dependency Analysis)

分析服务间的依赖关系：

- **依赖图**: 完整的服务依赖关系图
- **分层结构**: 自动分层，便于理解启动顺序
- **关键路径**: 识别关键服务链
- **循环检测**: 检测并报告循环依赖

### 6. 性能评估 (Performance Evaluation)

全面的性能评估：

- **总体评分**: 0-100 分的性能评分
- **性能指标**: 资源效率、可扩展性、可靠性
- **瓶颈识别**: 识别性能瓶颈和问题
- **性能建议**: 具体的性能优化建议
- **最佳实践对比**: 与行业最佳实践对比

## 使用方法

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "qwq/internal/container"
)

func main() {
    // 创建优化器
    optimizer := container.NewArchitectureOptimizer()
    ctx := context.Background()

    // 准备 Compose 配置
    config := &container.ComposeConfig{
        Version: "3.8",
        Services: map[string]*container.Service{
            "web": {
                Image:   "nginx:1.21.0",
                Ports:   []string{"80:80"},
                Restart: "unless-stopped",
            },
            // ... 更多服务
        },
    }

    // 1. 分析架构
    analysis, err := optimizer.AnalyzeArchitecture(ctx, config)
    if err != nil {
        panic(err)
    }
    fmt.Printf("健康评分: %d/100\n", analysis.HealthScore)
    fmt.Printf("发现问题: %d 个\n", len(analysis.Issues))

    // 2. 获取优化建议
    suggestions, err := optimizer.GenerateOptimizations(ctx, analysis)
    if err != nil {
        panic(err)
    }
    for _, s := range suggestions {
        fmt.Printf("[%s] %s\n", s.Priority, s.Title)
    }

    // 3. 获取安全建议
    securityRecs, err := optimizer.GenerateSecurityRecommendations(ctx, config)
    if err != nil {
        panic(err)
    }
    for _, r := range securityRecs {
        fmt.Printf("[%s] %s\n", r.Severity, r.Title)
    }

    // 4. 生成可视化数据
    viz, err := optimizer.GenerateVisualization(ctx, config)
    if err != nil {
        panic(err)
    }
    fmt.Printf("节点数: %d, 边数: %d\n", len(viz.Nodes), len(viz.Edges))

    // 5. 分析依赖关系
    depGraph, err := optimizer.AnalyzeDependencies(ctx, config)
    if err != nil {
        panic(err)
    }
    fmt.Printf("服务层级: %d 层\n", len(depGraph.Layers))

    // 6. 评估性能
    perfEval, err := optimizer.EvaluatePerformance(ctx, config)
    if err != nil {
        panic(err)
    }
    fmt.Printf("性能评分: %d/100\n", perfEval.OverallScore)
}
```

### 集成到 Compose 服务

```go
// 通过 ComposeService 使用
composeService := container.NewComposeService(db)

// 分析项目架构
analysis, err := composeService.AnalyzeProjectArchitecture(ctx, projectID)

// 获取优化建议
suggestions, err := composeService.GetOptimizationSuggestions(ctx, projectID)

// 获取安全建议
securityRecs, err := composeService.GetSecurityRecommendations(ctx, projectID)

// 获取可视化数据
viz, err := composeService.GetArchitectureVisualization(ctx, projectID)

// 获取依赖关系图
depGraph, err := composeService.GetDependencyGraph(ctx, projectID)

// 评估性能
perfEval, err := composeService.EvaluateProjectPerformance(ctx, projectID)
```

## API 端点示例

```go
// GET /api/projects/:id/analysis
// 获取架构分析
func GetProjectAnalysis(c *gin.Context) {
    projectID := c.Param("id")
    analysis, err := composeService.AnalyzeProjectArchitecture(c, projectID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, analysis)
}

// GET /api/projects/:id/optimizations
// 获取优化建议
func GetOptimizations(c *gin.Context) {
    projectID := c.Param("id")
    suggestions, err := composeService.GetOptimizationSuggestions(c, projectID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, suggestions)
}

// GET /api/projects/:id/security
// 获取安全建议
func GetSecurityRecommendations(c *gin.Context) {
    projectID := c.Param("id")
    recommendations, err := composeService.GetSecurityRecommendations(c, projectID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, recommendations)
}

// GET /api/projects/:id/visualization
// 获取架构可视化
func GetVisualization(c *gin.Context) {
    projectID := c.Param("id")
    viz, err := composeService.GetArchitectureVisualization(c, projectID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, viz)
}

// GET /api/projects/:id/dependencies
// 获取依赖关系图
func GetDependencies(c *gin.Context) {
    projectID := c.Param("id")
    depGraph, err := composeService.GetDependencyGraph(c, projectID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, depGraph)
}

// GET /api/projects/:id/performance
// 获取性能评估
func GetPerformanceEvaluation(c *gin.Context) {
    projectID := c.Param("id")
    perfEval, err := composeService.EvaluateProjectPerformance(c, projectID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, perfEval)
}
```

## 前端集成示例

### 显示架构分析

```javascript
// 获取架构分析
async function getArchitectureAnalysis(projectId) {
    const response = await fetch(`/api/projects/${projectId}/analysis`);
    const analysis = await response.json();
    
    // 显示健康评分
    document.getElementById('health-score').textContent = `${analysis.health_score}/100`;
    
    // 显示问题列表
    const issuesList = document.getElementById('issues-list');
    analysis.issues.forEach(issue => {
        const item = document.createElement('div');
        item.className = `issue-item severity-${issue.severity}`;
        item.innerHTML = `
            <h4>${issue.title}</h4>
            <p>${issue.description}</p>
            <p class="suggestion">${issue.suggestion}</p>
        `;
        issuesList.appendChild(item);
    });
}
```

### 显示架构可视化

```javascript
// 使用 vis.js 显示架构图
async function showArchitectureVisualization(projectId) {
    const response = await fetch(`/api/projects/${projectId}/visualization`);
    const viz = await response.json();
    
    // 转换为 vis.js 格式
    const nodes = new vis.DataSet(viz.nodes.map(node => ({
        id: node.id,
        label: node.label,
        shape: node.style.shape,
        color: node.style.color,
        group: node.group
    })));
    
    const edges = new vis.DataSet(viz.edges.map(edge => ({
        from: edge.from,
        to: edge.to,
        label: edge.label,
        color: edge.style.color,
        dashes: edge.style.dashed
    })));
    
    // 创建网络图
    const container = document.getElementById('architecture-graph');
    const data = { nodes, edges };
    const options = {
        layout: {
            hierarchical: {
                direction: 'UD',
                sortMethod: 'directed'
            }
        }
    };
    
    new vis.Network(container, data, options);
}
```

## 扩展和定制

### 添加自定义检测规则

```go
// 在 detectServiceIssues 方法中添加自定义规则
func (o *architectureOptimizerImpl) detectServiceIssues(serviceName string, service *Service) []*ArchitectureIssue {
    issues := make([]*ArchitectureIssue, 0)
    
    // 自定义规则：检查是否使用了推荐的镜像
    recommendedImages := map[string]bool{
        "nginx": true,
        "postgres": true,
        "redis": true,
    }
    
    imageName := strings.Split(service.Image, ":")[0]
    if !recommendedImages[imageName] {
        issues = append(issues, &ArchitectureIssue{
            Severity:    SeverityInfo,
            Category:    CategoryBestPractice,
            Service:     serviceName,
            Title:       "使用了非推荐镜像",
            Description: fmt.Sprintf("服务 %s 使用的镜像不在推荐列表中", serviceName),
            Impact:      "可能缺少社区支持和最佳实践",
            Suggestion:  "考虑使用官方推荐的镜像",
        })
    }
    
    return issues
}
```

### 集成 AI 服务

```go
// 可以注入 AI 客户端来增强分析能力
type architectureOptimizerImpl struct {
    aiClient AIClient // AI 服务客户端
}

// 使用 AI 生成更智能的建议
func (o *architectureOptimizerImpl) generateAIEnhancedSuggestions(ctx context.Context, config *ComposeConfig) ([]*OptimizationSuggestion, error) {
    // 将配置发送给 AI 服务
    prompt := fmt.Sprintf("分析以下 Docker Compose 配置并提供优化建议:\n%v", config)
    aiResponse, err := o.aiClient.Complete(ctx, prompt)
    if err != nil {
        return nil, err
    }
    
    // 解析 AI 响应并转换为建议
    // ...
}
```

## 性能考虑

- 架构分析是计算密集型操作，建议异步执行
- 可以缓存分析结果，避免重复计算
- 对于大型配置（>50个服务），考虑分批处理
- 可视化数据可能较大，考虑压缩传输

## 未来改进

- [ ] 集成真实的 AI 模型进行更智能的分析
- [ ] 支持 Kubernetes 配置分析
- [ ] 添加历史趋势分析
- [ ] 支持自定义规则引擎
- [ ] 添加性能基准测试
- [ ] 支持多语言建议
- [ ] 集成漏洞扫描数据库

## 相关文档

- [Docker Compose 最佳实践](https://docs.docker.com/compose/production/)
- [容器安全指南](https://docs.docker.com/engine/security/)
- [性能优化指南](https://docs.docker.com/config/containers/resource_constraints/)
