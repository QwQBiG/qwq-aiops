package container

import (
	"context"
	"fmt"
	"strings"
)

// ArchitectureOptimizer AI 架构优化分析器接口
type ArchitectureOptimizer interface {
	// 分析服务架构
	AnalyzeArchitecture(ctx context.Context, config *ComposeConfig) (*ArchitectureAnalysis, error)
	
	// 生成优化建议
	GenerateOptimizations(ctx context.Context, analysis *ArchitectureAnalysis) ([]*OptimizationSuggestion, error)
	
	// 生成安全加固建议
	GenerateSecurityRecommendations(ctx context.Context, config *ComposeConfig) ([]*SecurityRecommendation, error)
	
	// 生成架构可视化数据
	GenerateVisualization(ctx context.Context, config *ComposeConfig) (*ArchitectureVisualization, error)
	
	// 分析服务依赖关系
	AnalyzeDependencies(ctx context.Context, config *ComposeConfig) (*DependencyGraph, error)
	
	// 评估性能
	EvaluatePerformance(ctx context.Context, config *ComposeConfig) (*PerformanceEvaluation, error)
}

// ArchitectureAnalysis 架构分析结果
type ArchitectureAnalysis struct {
	TotalServices      int                        `json:"total_services"`       // 服务总数
	TotalNetworks      int                        `json:"total_networks"`       // 网络总数
	TotalVolumes       int                        `json:"total_volumes"`        // 卷总数
	Complexity         ComplexityLevel            `json:"complexity"`           // 复杂度等级
	HealthScore        int                        `json:"health_score"`         // 健康评分 (0-100)
	Issues             []*ArchitectureIssue       `json:"issues"`               // 发现的问题
	ServiceAnalysis    map[string]*ServiceAnalysis `json:"service_analysis"`    // 各服务分析
	NetworkTopology    *NetworkTopology           `json:"network_topology"`     // 网络拓扑
	ResourceUsage      *ResourceUsageEstimate     `json:"resource_usage"`       // 资源使用估算
}

// ComplexityLevel 复杂度等级
type ComplexityLevel string

const (
	ComplexityLow    ComplexityLevel = "low"    // 低复杂度 (1-3个服务)
	ComplexityMedium ComplexityLevel = "medium" // 中等复杂度 (4-10个服务)
	ComplexityHigh   ComplexityLevel = "high"   // 高复杂度 (11+个服务)
)


// ArchitectureIssue 架构问题
type ArchitectureIssue struct {
	Severity    IssueSeverity `json:"severity"`    // 严重程度
	Category    IssueCategory `json:"category"`    // 问题类别
	Service     string        `json:"service"`     // 相关服务
	Title       string        `json:"title"`       // 问题标题
	Description string        `json:"description"` // 问题描述
	Impact      string        `json:"impact"`      // 影响说明
	Suggestion  string        `json:"suggestion"`  // 修复建议
}

// IssueSeverity 问题严重程度
type IssueSeverity string

const (
	SeverityCritical IssueSeverity = "critical" // 严重
	SeverityHigh     IssueSeverity = "high"     // 高
	SeverityMedium   IssueSeverity = "medium"   // 中
	SeverityLow      IssueSeverity = "low"      // 低
	SeverityInfo     IssueSeverity = "info"     // 信息
)

// IssueCategory 问题类别
type IssueCategory string

const (
	CategorySecurity     IssueCategory = "security"     // 安全
	CategoryPerformance  IssueCategory = "performance"  // 性能
	CategoryReliability  IssueCategory = "reliability"  // 可靠性
	CategoryMaintenance  IssueCategory = "maintenance"  // 可维护性
	CategoryBestPractice IssueCategory = "best_practice" // 最佳实践
)

// ServiceAnalysis 服务分析
type ServiceAnalysis struct {
	ServiceName       string              `json:"service_name"`       // 服务名称
	Image             string              `json:"image"`              // 镜像
	HasHealthCheck    bool                `json:"has_health_check"`   // 是否有健康检查
	HasResourceLimits bool                `json:"has_resource_limits"` // 是否有资源限制
	RestartPolicy     string              `json:"restart_policy"`     // 重启策略
	ExposedPorts      []string            `json:"exposed_ports"`      // 暴露的端口
	Dependencies      []string            `json:"dependencies"`       // 依赖的服务
	Volumes           []string            `json:"volumes"`            // 使用的卷
	Networks          []string            `json:"networks"`           // 连接的网络
	SecurityIssues    []*SecurityIssue    `json:"security_issues"`    // 安全问题
	PerformanceHints  []*PerformanceHint  `json:"performance_hints"`  // 性能提示
}

// SecurityIssue 安全问题
type SecurityIssue struct {
	Type        string `json:"type"`        // 问题类型
	Description string `json:"description"` // 描述
	Severity    string `json:"severity"`    // 严重程度
	Fix         string `json:"fix"`         // 修复建议
}

// PerformanceHint 性能提示
type PerformanceHint struct {
	Type        string `json:"type"`        // 提示类型
	Description string `json:"description"` // 描述
	Impact      string `json:"impact"`      // 影响
	Suggestion  string `json:"suggestion"`  // 建议
}


// NetworkTopology 网络拓扑
type NetworkTopology struct {
	Networks         map[string]*NetworkInfo `json:"networks"`          // 网络信息
	ServiceConnections []*ServiceConnection  `json:"service_connections"` // 服务连接关系
	IsolationLevel   string                  `json:"isolation_level"`   // 隔离级别
}

// NetworkInfo 网络信息
type NetworkInfo struct {
	Name           string   `json:"name"`            // 网络名称
	Driver         string   `json:"driver"`          // 驱动类型
	ConnectedServices []string `json:"connected_services"` // 连接的服务
	IsExternal     bool     `json:"is_external"`     // 是否外部网络
}

// ServiceConnection 服务连接
type ServiceConnection struct {
	From    string `json:"from"`    // 源服务
	To      string `json:"to"`      // 目标服务
	Network string `json:"network"` // 通过的网络
	Type    string `json:"type"`    // 连接类型 (depends_on, network, volume)
}

// ResourceUsageEstimate 资源使用估算
type ResourceUsageEstimate struct {
	TotalCPU    string `json:"total_cpu"`    // 总CPU需求
	TotalMemory string `json:"total_memory"` // 总内存需求
	TotalDisk   string `json:"total_disk"`   // 总磁盘需求
	Services    map[string]*ServiceResourceEstimate `json:"services"` // 各服务资源估算
}

// ServiceResourceEstimate 服务资源估算
type ServiceResourceEstimate struct {
	CPULimit    string `json:"cpu_limit"`    // CPU限制
	MemoryLimit string `json:"memory_limit"` // 内存限制
	CPURequest  string `json:"cpu_request"`  // CPU请求
	MemoryRequest string `json:"memory_request"` // 内存请求
	HasLimits   bool   `json:"has_limits"`   // 是否设置了限制
}

// OptimizationSuggestion 优化建议
type OptimizationSuggestion struct {
	ID          string                 `json:"id"`          // 建议ID
	Category    OptimizationCategory   `json:"category"`    // 类别
	Priority    OptimizationPriority   `json:"priority"`    // 优先级
	Title       string                 `json:"title"`       // 标题
	Description string                 `json:"description"` // 描述
	Benefits    []string               `json:"benefits"`    // 收益
	Implementation string              `json:"implementation"` // 实施方法
	AffectedServices []string          `json:"affected_services"` // 影响的服务
	EstimatedImpact *ImpactEstimate    `json:"estimated_impact"` // 预估影响
	CodeExample     string             `json:"code_example,omitempty"` // 代码示例
}

// OptimizationCategory 优化类别
type OptimizationCategory string

const (
	OptimizationPerformance  OptimizationCategory = "performance"  // 性能优化
	OptimizationSecurity     OptimizationCategory = "security"     // 安全优化
	OptimizationReliability  OptimizationCategory = "reliability"  // 可靠性优化
	OptimizationCost         OptimizationCategory = "cost"         // 成本优化
	OptimizationMaintainability OptimizationCategory = "maintainability" // 可维护性优化
)


// OptimizationPriority 优化优先级
type OptimizationPriority string

const (
	PriorityCritical OptimizationPriority = "critical" // 关键
	PriorityHigh     OptimizationPriority = "high"     // 高
	PriorityMedium   OptimizationPriority = "medium"   // 中
	PriorityLow      OptimizationPriority = "low"      // 低
)

// ImpactEstimate 影响估算
type ImpactEstimate struct {
	PerformanceGain string `json:"performance_gain"` // 性能提升
	CostSaving      string `json:"cost_saving"`      // 成本节省
	SecurityImprovement string `json:"security_improvement"` // 安全改进
	Effort          string `json:"effort"`           // 实施难度
}

// SecurityRecommendation 安全建议
type SecurityRecommendation struct {
	ID          string           `json:"id"`          // 建议ID
	Severity    IssueSeverity    `json:"severity"`    // 严重程度
	Title       string           `json:"title"`       // 标题
	Description string           `json:"description"` // 描述
	Risk        string           `json:"risk"`        // 风险说明
	Mitigation  string           `json:"mitigation"`  // 缓解措施
	Service     string           `json:"service"`     // 相关服务
	References  []string         `json:"references"`  // 参考资料
	CodeExample string           `json:"code_example,omitempty"` // 代码示例
}

// ArchitectureVisualization 架构可视化数据
type ArchitectureVisualization struct {
	Nodes []*VisualizationNode `json:"nodes"` // 节点（服务、网络、卷）
	Edges []*VisualizationEdge `json:"edges"` // 边（连接关系）
	Layout string              `json:"layout"` // 布局类型
	Metadata *VisualizationMetadata `json:"metadata"` // 元数据
}

// VisualizationNode 可视化节点
type VisualizationNode struct {
	ID       string                 `json:"id"`       // 节点ID
	Type     NodeType               `json:"type"`     // 节点类型
	Label    string                 `json:"label"`    // 标签
	Group    string                 `json:"group"`    // 分组
	Metadata map[string]interface{} `json:"metadata"` // 元数据
	Style    *NodeStyle             `json:"style"`    // 样式
}

// NodeType 节点类型
type NodeType string

const (
	NodeTypeService NodeType = "service" // 服务
	NodeTypeNetwork NodeType = "network" // 网络
	NodeTypeVolume  NodeType = "volume"  // 卷
	NodeTypeExternal NodeType = "external" // 外部依赖
)

// NodeStyle 节点样式
type NodeStyle struct {
	Color  string `json:"color"`  // 颜色
	Shape  string `json:"shape"`  // 形状
	Size   int    `json:"size"`   // 大小
	Icon   string `json:"icon"`   // 图标
}


// VisualizationEdge 可视化边
type VisualizationEdge struct {
	ID     string                 `json:"id"`     // 边ID
	From   string                 `json:"from"`   // 源节点ID
	To     string                 `json:"to"`     // 目标节点ID
	Type   EdgeType               `json:"type"`   // 边类型
	Label  string                 `json:"label"`  // 标签
	Style  *EdgeStyle             `json:"style"`  // 样式
	Metadata map[string]interface{} `json:"metadata"` // 元数据
}

// EdgeType 边类型
type EdgeType string

const (
	EdgeTypeDependency EdgeType = "dependency" // 依赖关系
	EdgeTypeNetwork    EdgeType = "network"    // 网络连接
	EdgeTypeVolume     EdgeType = "volume"     // 卷挂载
	EdgeTypePort       EdgeType = "port"       // 端口映射
)

// EdgeStyle 边样式
type EdgeStyle struct {
	Color     string `json:"color"`     // 颜色
	Width     int    `json:"width"`     // 宽度
	Dashed    bool   `json:"dashed"`    // 是否虚线
	Animated  bool   `json:"animated"`  // 是否动画
}

// VisualizationMetadata 可视化元数据
type VisualizationMetadata struct {
	Title       string `json:"title"`       // 标题
	Description string `json:"description"` // 描述
	Complexity  string `json:"complexity"`  // 复杂度
	ServiceCount int   `json:"service_count"` // 服务数量
	NetworkCount int   `json:"network_count"` // 网络数量
	VolumeCount  int   `json:"volume_count"`  // 卷数量
}

// DependencyGraph 依赖关系图
type DependencyGraph struct {
	Services     map[string]*ServiceDependency `json:"services"`      // 服务依赖
	Layers       [][]string                    `json:"layers"`        // 分层结构
	CriticalPath []string                      `json:"critical_path"` // 关键路径
	Cycles       [][]string                    `json:"cycles"`        // 循环依赖
}

// ServiceDependency 服务依赖
type ServiceDependency struct {
	ServiceName  string   `json:"service_name"`  // 服务名称
	Dependencies []string `json:"dependencies"`  // 依赖的服务
	Dependents   []string `json:"dependents"`    // 依赖此服务的服务
	Layer        int      `json:"layer"`         // 所在层级
	IsCritical   bool     `json:"is_critical"`   // 是否在关键路径上
}

// PerformanceEvaluation 性能评估
type PerformanceEvaluation struct {
	OverallScore    int                          `json:"overall_score"`    // 总体评分 (0-100)
	Metrics         *PerformanceMetrics          `json:"metrics"`          // 性能指标
	Bottlenecks     []*PerformanceBottleneck     `json:"bottlenecks"`      // 性能瓶颈
	Recommendations []*PerformanceRecommendation `json:"recommendations"`  // 性能建议
	Comparison      *PerformanceComparison       `json:"comparison"`       // 与最佳实践对比
}


// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	ResourceEfficiency  int `json:"resource_efficiency"`  // 资源效率 (0-100)
	ScalabilityScore    int `json:"scalability_score"`    // 可扩展性评分 (0-100)
	ReliabilityScore    int `json:"reliability_score"`    // 可靠性评分 (0-100)
	StartupTimeEstimate string `json:"startup_time_estimate"` // 启动时间估算
	MemoryFootprint     string `json:"memory_footprint"`     // 内存占用
	NetworkLatency      string `json:"network_latency"`      // 网络延迟估算
}

// PerformanceBottleneck 性能瓶颈
type PerformanceBottleneck struct {
	Type        string   `json:"type"`        // 瓶颈类型
	Service     string   `json:"service"`     // 相关服务
	Description string   `json:"description"` // 描述
	Impact      string   `json:"impact"`      // 影响
	Solutions   []string `json:"solutions"`   // 解决方案
}

// PerformanceRecommendation 性能建议
type PerformanceRecommendation struct {
	Title          string   `json:"title"`          // 标题
	Description    string   `json:"description"`    // 描述
	ExpectedGain   string   `json:"expected_gain"`  // 预期收益
	Implementation string   `json:"implementation"` // 实施方法
	Priority       string   `json:"priority"`       // 优先级
}

// PerformanceComparison 性能对比
type PerformanceComparison struct {
	BestPractices   []string `json:"best_practices"`   // 符合的最佳实践
	Deviations      []string `json:"deviations"`       // 偏离的地方
	IndustryAverage string   `json:"industry_average"` // 行业平均水平
	YourScore       string   `json:"your_score"`       // 当前评分
}

// architectureOptimizerImpl AI 架构优化分析器实现
type architectureOptimizerImpl struct {
	// 可以注入 AI 服务客户端
	// aiClient AIClient
}

// NewArchitectureOptimizer 创建架构优化分析器实例
func NewArchitectureOptimizer() ArchitectureOptimizer {
	return &architectureOptimizerImpl{}
}

// AnalyzeArchitecture 分析服务架构
func (o *architectureOptimizerImpl) AnalyzeArchitecture(ctx context.Context, config *ComposeConfig) (*ArchitectureAnalysis, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	analysis := &ArchitectureAnalysis{
		TotalServices:   len(config.Services),
		TotalNetworks:   len(config.Networks),
		TotalVolumes:    len(config.Volumes),
		Issues:          make([]*ArchitectureIssue, 0),
		ServiceAnalysis: make(map[string]*ServiceAnalysis),
	}

	// 计算复杂度
	analysis.Complexity = o.calculateComplexity(config)

	// 分析每个服务
	for serviceName, service := range config.Services {
		serviceAnalysis := o.analyzeService(serviceName, service, config)
		analysis.ServiceAnalysis[serviceName] = serviceAnalysis

		// 收集问题
		analysis.Issues = append(analysis.Issues, o.detectServiceIssues(serviceName, service)...)
	}

	// 分析网络拓扑
	analysis.NetworkTopology = o.analyzeNetworkTopology(config)

	// 估算资源使用
	analysis.ResourceUsage = o.estimateResourceUsage(config)

	// 计算健康评分
	analysis.HealthScore = o.calculateHealthScore(analysis)

	return analysis, nil
}


// calculateComplexity 计算复杂度
func (o *architectureOptimizerImpl) calculateComplexity(config *ComposeConfig) ComplexityLevel {
	serviceCount := len(config.Services)
	if serviceCount <= 3 {
		return ComplexityLow
	} else if serviceCount <= 10 {
		return ComplexityMedium
	}
	return ComplexityHigh
}

// analyzeService 分析单个服务
func (o *architectureOptimizerImpl) analyzeService(serviceName string, service *Service, config *ComposeConfig) *ServiceAnalysis {
	analysis := &ServiceAnalysis{
		ServiceName:       serviceName,
		Image:             service.Image,
		HasHealthCheck:    service.HealthCheck != nil,
		HasResourceLimits: service.Deploy != nil && service.Deploy.Resources != nil && service.Deploy.Resources.Limits != nil,
		RestartPolicy:     service.Restart,
		ExposedPorts:      service.Ports,
		Volumes:           service.Volumes,
		SecurityIssues:    make([]*SecurityIssue, 0),
		PerformanceHints:  make([]*PerformanceHint, 0),
	}

	// 提取依赖关系
	analysis.Dependencies = o.extractDependencies(service)

	// 提取网络连接
	analysis.Networks = o.extractNetworks(service)

	// 检测安全问题
	analysis.SecurityIssues = o.detectSecurityIssues(serviceName, service)

	// 生成性能提示
	analysis.PerformanceHints = o.generatePerformanceHints(serviceName, service)

	return analysis
}

// extractDependencies 提取服务依赖
func (o *architectureOptimizerImpl) extractDependencies(service *Service) []string {
	dependencies := make([]string, 0)

	if service.DependsOn != nil {
		switch deps := service.DependsOn.(type) {
		case []interface{}:
			for _, dep := range deps {
				if depStr, ok := dep.(string); ok {
					dependencies = append(dependencies, depStr)
				}
			}
		case map[string]interface{}:
			for depName := range deps {
				dependencies = append(dependencies, depName)
			}
		}
	}

	return dependencies
}

// extractNetworks 提取网络连接
func (o *architectureOptimizerImpl) extractNetworks(service *Service) []string {
	networks := make([]string, 0)

	if service.Networks != nil {
		switch nets := service.Networks.(type) {
		case []interface{}:
			for _, net := range nets {
				if netStr, ok := net.(string); ok {
					networks = append(networks, netStr)
				}
			}
		case map[string]interface{}:
			for netName := range nets {
				networks = append(networks, netName)
			}
		}
	}

	return networks
}


// detectServiceIssues 检测服务问题
func (o *architectureOptimizerImpl) detectServiceIssues(serviceName string, service *Service) []*ArchitectureIssue {
	issues := make([]*ArchitectureIssue, 0)

	// 检查健康检查
	if service.HealthCheck == nil {
		issues = append(issues, &ArchitectureIssue{
			Severity:    SeverityMedium,
			Category:    CategoryReliability,
			Service:     serviceName,
			Title:       "缺少健康检查配置",
			Description: fmt.Sprintf("服务 %s 没有配置健康检查", serviceName),
			Impact:      "无法自动检测服务健康状态，可能导致故障服务继续接收流量",
			Suggestion:  "添加 healthcheck 配置，定期检查服务健康状态",
		})
	}

	// 检查资源限制
	if service.Deploy == nil || service.Deploy.Resources == nil || service.Deploy.Resources.Limits == nil {
		issues = append(issues, &ArchitectureIssue{
			Severity:    SeverityMedium,
			Category:    CategoryPerformance,
			Service:     serviceName,
			Title:       "未设置资源限制",
			Description: fmt.Sprintf("服务 %s 没有设置资源限制", serviceName),
			Impact:      "服务可能消耗过多资源，影响其他服务运行",
			Suggestion:  "在 deploy.resources.limits 中设置 CPU 和内存限制",
		})
	}

	// 检查重启策略
	if service.Restart == "" || service.Restart == "no" {
		issues = append(issues, &ArchitectureIssue{
			Severity:    SeverityLow,
			Category:    CategoryReliability,
			Service:     serviceName,
			Title:       "未配置自动重启",
			Description: fmt.Sprintf("服务 %s 没有配置自动重启策略", serviceName),
			Impact:      "服务崩溃后不会自动恢复",
			Suggestion:  "设置 restart: unless-stopped 或 restart: on-failure",
		})
	}

	// 检查特权模式
	if service.Privileged {
		issues = append(issues, &ArchitectureIssue{
			Severity:    SeverityHigh,
			Category:    CategorySecurity,
			Service:     serviceName,
			Title:       "使用特权模式",
			Description: fmt.Sprintf("服务 %s 以特权模式运行", serviceName),
			Impact:      "容器拥有主机的完全权限，存在严重安全风险",
			Suggestion:  "除非绝对必要，否则移除 privileged: true 配置",
		})
	}

	return issues
}

// detectSecurityIssues 检测安全问题
func (o *architectureOptimizerImpl) detectSecurityIssues(serviceName string, service *Service) []*SecurityIssue {
	issues := make([]*SecurityIssue, 0)

	// 检查镜像标签
	if service.Image != "" && !strings.Contains(service.Image, ":") {
		issues = append(issues, &SecurityIssue{
			Type:        "image_tag",
			Description: "使用了 latest 标签或未指定标签",
			Severity:    "medium",
			Fix:         "使用具体的版本标签，如 nginx:1.21.0",
		})
	}

	// 检查特权模式
	if service.Privileged {
		issues = append(issues, &SecurityIssue{
			Type:        "privileged_mode",
			Description: "容器以特权模式运行",
			Severity:    "high",
			Fix:         "移除 privileged: true，使用更细粒度的权限控制",
		})
	}

	// 检查 root 用户
	if service.User == "" || service.User == "root" || service.User == "0" {
		issues = append(issues, &SecurityIssue{
			Type:        "root_user",
			Description: "容器以 root 用户运行",
			Severity:    "medium",
			Fix:         "使用非 root 用户运行容器，如 user: \"1000:1000\"",
		})
	}

	return issues
}


// generatePerformanceHints 生成性能提示
func (o *architectureOptimizerImpl) generatePerformanceHints(serviceName string, service *Service) []*PerformanceHint {
	hints := make([]*PerformanceHint, 0)

	// 检查资源限制
	if service.Deploy == nil || service.Deploy.Resources == nil {
		hints = append(hints, &PerformanceHint{
			Type:        "resource_limits",
			Description: "未设置资源限制",
			Impact:      "可能导致资源竞争和性能不稳定",
			Suggestion:  "设置合理的 CPU 和内存限制",
		})
	}

	// 检查日志驱动
	if service.Logging == nil || service.Logging.Driver == "" {
		hints = append(hints, &PerformanceHint{
			Type:        "logging",
			Description: "使用默认日志驱动",
			Impact:      "可能产生大量日志文件，占用磁盘空间",
			Suggestion:  "配置日志驱动和日志轮转策略",
		})
	}

	// 检查健康检查间隔
	if service.HealthCheck != nil && service.HealthCheck.Interval == "" {
		hints = append(hints, &PerformanceHint{
			Type:        "health_check",
			Description: "健康检查未设置间隔",
			Impact:      "可能使用默认值，不适合当前服务",
			Suggestion:  "根据服务特性设置合适的检查间隔",
		})
	}

	return hints
}

// analyzeNetworkTopology 分析网络拓扑
func (o *architectureOptimizerImpl) analyzeNetworkTopology(config *ComposeConfig) *NetworkTopology {
	topology := &NetworkTopology{
		Networks:           make(map[string]*NetworkInfo),
		ServiceConnections: make([]*ServiceConnection, 0),
		IsolationLevel:     "default",
	}

	// 分析网络
	for networkName, network := range config.Networks {
		info := &NetworkInfo{
			Name:              networkName,
			Driver:            network.Driver,
			ConnectedServices: make([]string, 0),
			IsExternal:        network.External,
		}

		// 查找连接到此网络的服务
		for serviceName, service := range config.Services {
			networks := o.extractNetworks(service)
			for _, net := range networks {
				if net == networkName {
					info.ConnectedServices = append(info.ConnectedServices, serviceName)
				}
			}
		}

		topology.Networks[networkName] = info
	}

	// 分析服务连接
	for serviceName, service := range config.Services {
		// 依赖关系连接
		dependencies := o.extractDependencies(service)
		for _, dep := range dependencies {
			topology.ServiceConnections = append(topology.ServiceConnections, &ServiceConnection{
				From:    serviceName,
				To:      dep,
				Network: "default",
				Type:    "depends_on",
			})
		}

		// 网络连接
		networks := o.extractNetworks(service)
		for _, network := range networks {
			for _, otherService := range topology.Networks[network].ConnectedServices {
				if otherService != serviceName {
					topology.ServiceConnections = append(topology.ServiceConnections, &ServiceConnection{
						From:    serviceName,
						To:      otherService,
						Network: network,
						Type:    "network",
					})
				}
			}
		}
	}

	// 评估隔离级别
	if len(topology.Networks) > 1 {
		topology.IsolationLevel = "multi-network"
	} else if len(topology.Networks) == 1 {
		topology.IsolationLevel = "single-network"
	}

	return topology
}


// estimateResourceUsage 估算资源使用
func (o *architectureOptimizerImpl) estimateResourceUsage(config *ComposeConfig) *ResourceUsageEstimate {
	estimate := &ResourceUsageEstimate{
		Services: make(map[string]*ServiceResourceEstimate),
	}

	totalCPU := 0.0
	totalMemory := 0

	for serviceName, service := range config.Services {
		serviceEstimate := &ServiceResourceEstimate{
			HasLimits: false,
		}

		if service.Deploy != nil && service.Deploy.Resources != nil {
			if service.Deploy.Resources.Limits != nil {
				serviceEstimate.HasLimits = true
				serviceEstimate.CPULimit = service.Deploy.Resources.Limits.CPUs
				serviceEstimate.MemoryLimit = service.Deploy.Resources.Limits.Memory

				// 简单解析 CPU 和内存（实际应该更严格）
				if serviceEstimate.CPULimit != "" {
					// 假设格式如 "0.5" 或 "1"
					var cpu float64
					fmt.Sscanf(serviceEstimate.CPULimit, "%f", &cpu)
					totalCPU += cpu
				}

				if serviceEstimate.MemoryLimit != "" {
					// 假设格式如 "512M" 或 "1G"
					var mem int
					var unit string
					fmt.Sscanf(serviceEstimate.MemoryLimit, "%d%s", &mem, &unit)
					if strings.HasPrefix(unit, "G") || strings.HasPrefix(unit, "g") {
						totalMemory += mem * 1024
					} else {
						totalMemory += mem
					}
				}
			}

			if service.Deploy.Resources.Reservations != nil {
				serviceEstimate.CPURequest = service.Deploy.Resources.Reservations.CPUs
				serviceEstimate.MemoryRequest = service.Deploy.Resources.Reservations.Memory
			}
		}

		// 如果没有设置限制，使用默认估算
		if !serviceEstimate.HasLimits {
			serviceEstimate.CPULimit = "未设置"
			serviceEstimate.MemoryLimit = "未设置"
			// 默认估算：每个服务 0.5 CPU 和 512MB 内存
			totalCPU += 0.5
			totalMemory += 512
		}

		estimate.Services[serviceName] = serviceEstimate
	}

	estimate.TotalCPU = fmt.Sprintf("%.1f cores", totalCPU)
	estimate.TotalMemory = fmt.Sprintf("%d MB", totalMemory)
	estimate.TotalDisk = "未估算"

	return estimate
}

// calculateHealthScore 计算健康评分
func (o *architectureOptimizerImpl) calculateHealthScore(analysis *ArchitectureAnalysis) int {
	score := 100

	// 根据问题严重程度扣分
	for _, issue := range analysis.Issues {
		switch issue.Severity {
		case SeverityCritical:
			score -= 20
		case SeverityHigh:
			score -= 10
		case SeverityMedium:
			score -= 5
		case SeverityLow:
			score -= 2
		}
	}

	// 确保分数在 0-100 之间
	if score < 0 {
		score = 0
	}

	return score
}


// GenerateOptimizations 生成优化建议
func (o *architectureOptimizerImpl) GenerateOptimizations(ctx context.Context, analysis *ArchitectureAnalysis) ([]*OptimizationSuggestion, error) {
	suggestions := make([]*OptimizationSuggestion, 0)

	// 基于分析结果生成优化建议
	for _, issue := range analysis.Issues {
		suggestion := o.issueToOptimization(issue)
		if suggestion != nil {
			suggestions = append(suggestions, suggestion)
		}
	}

	// 添加通用优化建议
	suggestions = append(suggestions, o.generateGeneralOptimizations(analysis)...)

	return suggestions, nil
}

// issueToOptimization 将问题转换为优化建议
func (o *architectureOptimizerImpl) issueToOptimization(issue *ArchitectureIssue) *OptimizationSuggestion {
	var category OptimizationCategory
	var priority OptimizationPriority

	// 映射类别
	switch issue.Category {
	case CategorySecurity:
		category = OptimizationSecurity
	case CategoryPerformance:
		category = OptimizationPerformance
	case CategoryReliability:
		category = OptimizationReliability
	default:
		category = OptimizationMaintainability
	}

	// 映射优先级
	switch issue.Severity {
	case SeverityCritical:
		priority = PriorityCritical
	case SeverityHigh:
		priority = PriorityHigh
	case SeverityMedium:
		priority = PriorityMedium
	default:
		priority = PriorityLow
	}

	return &OptimizationSuggestion{
		ID:                 fmt.Sprintf("opt-%s-%s", issue.Service, issue.Category),
		Category:           category,
		Priority:           priority,
		Title:              issue.Title,
		Description:        issue.Description,
		Benefits:           []string{fmt.Sprintf("解决: %s", issue.Impact)},
		Implementation:     issue.Suggestion,
		AffectedServices:   []string{issue.Service},
		EstimatedImpact:    o.estimateOptimizationImpact(issue),
	}
}

// estimateOptimizationImpact 估算优化影响
func (o *architectureOptimizerImpl) estimateOptimizationImpact(issue *ArchitectureIssue) *ImpactEstimate {
	impact := &ImpactEstimate{
		Effort: "低",
	}

	switch issue.Category {
	case CategorySecurity:
		impact.SecurityImprovement = "高"
		impact.PerformanceGain = "无"
		impact.CostSaving = "无"
	case CategoryPerformance:
		impact.SecurityImprovement = "无"
		impact.PerformanceGain = "中"
		impact.CostSaving = "低"
	case CategoryReliability:
		impact.SecurityImprovement = "低"
		impact.PerformanceGain = "低"
		impact.CostSaving = "无"
	}

	if issue.Severity == SeverityCritical || issue.Severity == SeverityHigh {
		impact.Effort = "中"
	}

	return impact
}


// generateGeneralOptimizations 生成通用优化建议
func (o *architectureOptimizerImpl) generateGeneralOptimizations(analysis *ArchitectureAnalysis) []*OptimizationSuggestion {
	suggestions := make([]*OptimizationSuggestion, 0)

	// 如果服务数量较多，建议使用服务网格
	if analysis.TotalServices > 10 {
		suggestions = append(suggestions, &OptimizationSuggestion{
			ID:          "opt-service-mesh",
			Category:    OptimizationMaintainability,
			Priority:    PriorityMedium,
			Title:       "考虑引入服务网格",
			Description: "当前架构包含较多服务，建议考虑使用服务网格（如 Istio）来管理服务间通信",
			Benefits: []string{
				"统一的流量管理",
				"增强的可观测性",
				"更好的安全性",
			},
			Implementation: "评估 Istio 或 Linkerd 等服务网格方案",
			EstimatedImpact: &ImpactEstimate{
				PerformanceGain:     "中",
				SecurityImprovement: "高",
				Effort:              "高",
			},
		})
	}

	// 如果没有使用网络隔离，建议添加
	if len(analysis.NetworkTopology.Networks) <= 1 {
		suggestions = append(suggestions, &OptimizationSuggestion{
			ID:          "opt-network-isolation",
			Category:    OptimizationSecurity,
			Priority:    PriorityHigh,
			Title:       "添加网络隔离",
			Description: "当前所有服务在同一网络中，建议根据功能划分不同的网络",
			Benefits: []string{
				"提高安全性",
				"减少攻击面",
				"更好的网络管理",
			},
			Implementation: "创建前端网络、后端网络和数据库网络，限制服务间访问",
			CodeExample: `networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
  database:
    driver: bridge
    internal: true  # 仅内部访问`,
			EstimatedImpact: &ImpactEstimate{
				SecurityImprovement: "高",
				PerformanceGain:     "无",
				Effort:              "中",
			},
		})
	}

	// 建议使用集中式日志
	suggestions = append(suggestions, &OptimizationSuggestion{
		ID:          "opt-centralized-logging",
		Category:    OptimizationMaintainability,
		Priority:    PriorityMedium,
		Title:       "实施集中式日志管理",
		Description: "建议使用 ELK Stack 或 Loki 等工具进行集中式日志管理",
		Benefits: []string{
			"统一的日志查看",
			"更好的问题排查",
			"日志分析和告警",
		},
		Implementation: "部署 Elasticsearch + Logstash + Kibana 或 Loki + Grafana",
		EstimatedImpact: &ImpactEstimate{
			PerformanceGain: "无",
			CostSaving:      "中",
			Effort:          "中",
		},
	})

	return suggestions
}

// GenerateSecurityRecommendations 生成安全建议
func (o *architectureOptimizerImpl) GenerateSecurityRecommendations(ctx context.Context, config *ComposeConfig) ([]*SecurityRecommendation, error) {
	recommendations := make([]*SecurityRecommendation, 0)

	for serviceName, service := range config.Services {
		// 检查镜像安全
		if service.Image != "" {
			if !strings.Contains(service.Image, ":") || strings.HasSuffix(service.Image, ":latest") {
				recommendations = append(recommendations, &SecurityRecommendation{
					ID:          fmt.Sprintf("sec-%s-image-tag", serviceName),
					Severity:    SeverityMedium,
					Title:       "使用具体的镜像版本",
					Description: fmt.Sprintf("服务 %s 使用了 latest 标签或未指定版本", serviceName),
					Risk:        "latest 标签可能导致不可预测的行为和安全漏洞",
					Mitigation:  "使用具体的版本标签，如 nginx:1.21.0-alpine",
					Service:     serviceName,
					References:  []string{"https://docs.docker.com/develop/dev-best-practices/"},
					CodeExample: fmt.Sprintf("image: %s:1.0.0  # 使用具体版本", strings.Split(service.Image, ":")[0]),
				})
			}
		}

		// 检查特权模式
		if service.Privileged {
			recommendations = append(recommendations, &SecurityRecommendation{
				ID:          fmt.Sprintf("sec-%s-privileged", serviceName),
				Severity:    SeverityCritical,
				Title:       "避免使用特权模式",
				Description: fmt.Sprintf("服务 %s 以特权模式运行", serviceName),
				Risk:        "特权容器可以访问主机的所有设备和内核功能，存在严重安全风险",
				Mitigation:  "移除 privileged 配置，使用 cap_add 添加必要的能力",
				Service:     serviceName,
				References:  []string{"https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities"},
				CodeExample: `# 不要使用
privileged: true

# 而是使用
cap_add:
  - NET_ADMIN  # 仅添加需要的能力`,
			})
		}

		// 检查 root 用户
		if service.User == "" || service.User == "root" || service.User == "0" {
			recommendations = append(recommendations, &SecurityRecommendation{
				ID:          fmt.Sprintf("sec-%s-root-user", serviceName),
				Severity:    SeverityMedium,
				Title:       "避免以 root 用户运行",
				Description: fmt.Sprintf("服务 %s 以 root 用户运行", serviceName),
				Risk:        "如果容器被攻破，攻击者将获得 root 权限",
				Mitigation:  "创建并使用非特权用户运行容器",
				Service:     serviceName,
				CodeExample: `user: "1000:1000"  # 使用非 root 用户`,
			})
		}

		// 检查敏感端口暴露
		for _, port := range service.Ports {
			if strings.Contains(port, "22:") || strings.Contains(port, ":22") {
				recommendations = append(recommendations, &SecurityRecommendation{
					ID:          fmt.Sprintf("sec-%s-ssh-port", serviceName),
					Severity:    SeverityHigh,
					Title:       "SSH 端口暴露",
					Description: fmt.Sprintf("服务 %s 暴露了 SSH 端口", serviceName),
					Risk:        "暴露 SSH 端口可能导致暴力破解攻击",
					Mitigation:  "避免暴露 SSH 端口，或使用 VPN/堡垒机访问",
					Service:     serviceName,
				})
			}
		}
	}

	// 添加通用安全建议
	recommendations = append(recommendations, o.generateGeneralSecurityRecommendations(config)...)

	return recommendations, nil
}


// generateGeneralSecurityRecommendations 生成通用安全建议
func (o *architectureOptimizerImpl) generateGeneralSecurityRecommendations(config *ComposeConfig) []*SecurityRecommendation {
	recommendations := make([]*SecurityRecommendation, 0)

	// 建议使用 secrets 管理敏感信息
	if len(config.Secrets) == 0 {
		recommendations = append(recommendations, &SecurityRecommendation{
			ID:          "sec-use-secrets",
			Severity:    SeverityHigh,
			Title:       "使用 Docker Secrets 管理敏感信息",
			Description: "当前配置未使用 secrets，敏感信息可能以明文形式存储",
			Risk:        "密码、API密钥等敏感信息可能泄露",
			Mitigation:  "使用 Docker Secrets 或环境变量文件管理敏感信息",
			References:  []string{"https://docs.docker.com/engine/swarm/secrets/"},
			CodeExample: `secrets:
  db_password:
    file: ./secrets/db_password.txt

services:
  app:
    secrets:
      - db_password`,
		})
	}

	// 建议启用只读根文件系统
	recommendations = append(recommendations, &SecurityRecommendation{
		ID:          "sec-readonly-rootfs",
		Severity:    SeverityMedium,
		Title:       "考虑使用只读根文件系统",
		Description: "使用只读根文件系统可以防止容器内的恶意修改",
		Risk:        "容器文件系统可写可能导致恶意软件持久化",
		Mitigation:  "为不需要写入的服务启用只读根文件系统",
		CodeExample: `read_only: true
tmpfs:
  - /tmp
  - /var/run`,
	})

	// 建议定期扫描镜像漏洞
	recommendations = append(recommendations, &SecurityRecommendation{
		ID:          "sec-image-scanning",
		Severity:    SeverityHigh,
		Title:       "定期扫描镜像安全漏洞",
		Description: "建议使用工具定期扫描容器镜像的安全漏洞",
		Risk:        "使用存在已知漏洞的镜像可能导致安全问题",
		Mitigation:  "集成 Trivy、Clair 等镜像扫描工具到 CI/CD 流程",
		References:  []string{"https://github.com/aquasecurity/trivy"},
	})

	return recommendations
}

// GenerateVisualization 生成架构可视化数据
func (o *architectureOptimizerImpl) GenerateVisualization(ctx context.Context, config *ComposeConfig) (*ArchitectureVisualization, error) {
	viz := &ArchitectureVisualization{
		Nodes:  make([]*VisualizationNode, 0),
		Edges:  make([]*VisualizationEdge, 0),
		Layout: "hierarchical",
		Metadata: &VisualizationMetadata{
			Title:        "Docker Compose 架构图",
			Description:  "服务依赖和网络拓扑可视化",
			ServiceCount: len(config.Services),
			NetworkCount: len(config.Networks),
			VolumeCount:  len(config.Volumes),
		},
	}

	// 添加服务节点
	for serviceName, service := range config.Services {
		node := &VisualizationNode{
			ID:    fmt.Sprintf("service-%s", serviceName),
			Type:  NodeTypeService,
			Label: serviceName,
			Group: "services",
			Metadata: map[string]interface{}{
				"image":   service.Image,
				"ports":   service.Ports,
				"restart": service.Restart,
			},
			Style: &NodeStyle{
				Color: "#4A90E2",
				Shape: "box",
				Size:  50,
				Icon:  "docker",
			},
		}
		viz.Nodes = append(viz.Nodes, node)
	}

	// 添加网络节点
	for networkName := range config.Networks {
		node := &VisualizationNode{
			ID:    fmt.Sprintf("network-%s", networkName),
			Type:  NodeTypeNetwork,
			Label: networkName,
			Group: "networks",
			Style: &NodeStyle{
				Color: "#50C878",
				Shape: "ellipse",
				Size:  40,
				Icon:  "network",
			},
		}
		viz.Nodes = append(viz.Nodes, node)
	}

	// 添加卷节点
	for volumeName := range config.Volumes {
		node := &VisualizationNode{
			ID:    fmt.Sprintf("volume-%s", volumeName),
			Type:  NodeTypeVolume,
			Label: volumeName,
			Group: "volumes",
			Style: &NodeStyle{
				Color: "#FFB347",
				Shape: "cylinder",
				Size:  35,
				Icon:  "database",
			},
		}
		viz.Nodes = append(viz.Nodes, node)
	}

	// 添加依赖关系边
	edgeID := 0
	for serviceName, service := range config.Services {
		dependencies := o.extractDependencies(service)
		for _, dep := range dependencies {
			edge := &VisualizationEdge{
				ID:    fmt.Sprintf("edge-%d", edgeID),
				From:  fmt.Sprintf("service-%s", serviceName),
				To:    fmt.Sprintf("service-%s", dep),
				Type:  EdgeTypeDependency,
				Label: "depends_on",
				Style: &EdgeStyle{
					Color:  "#999",
					Width:  2,
					Dashed: false,
				},
			}
			viz.Edges = append(viz.Edges, edge)
			edgeID++
		}

		// 添加网络连接边
		networks := o.extractNetworks(service)
		for _, network := range networks {
			edge := &VisualizationEdge{
				ID:    fmt.Sprintf("edge-%d", edgeID),
				From:  fmt.Sprintf("service-%s", serviceName),
				To:    fmt.Sprintf("network-%s", network),
				Type:  EdgeTypeNetwork,
				Label: "connects",
				Style: &EdgeStyle{
					Color:  "#50C878",
					Width:  1,
					Dashed: true,
				},
			}
			viz.Edges = append(viz.Edges, edge)
			edgeID++
		}

		// 添加卷挂载边
		for _, volume := range service.Volumes {
			// 简单解析卷名称（实际应该更严格）
			parts := strings.Split(volume, ":")
			if len(parts) >= 2 {
				volumeName := parts[0]
				// 检查是否是命名卷
				if _, exists := config.Volumes[volumeName]; exists {
					edge := &VisualizationEdge{
						ID:    fmt.Sprintf("edge-%d", edgeID),
						From:  fmt.Sprintf("service-%s", serviceName),
						To:    fmt.Sprintf("volume-%s", volumeName),
						Type:  EdgeTypeVolume,
						Label: "mounts",
						Style: &EdgeStyle{
							Color:  "#FFB347",
							Width:  1,
							Dashed: true,
						},
					}
					viz.Edges = append(viz.Edges, edge)
					edgeID++
				}
			}
		}
	}

	// 设置复杂度
	if len(config.Services) <= 3 {
		viz.Metadata.Complexity = "low"
	} else if len(config.Services) <= 10 {
		viz.Metadata.Complexity = "medium"
	} else {
		viz.Metadata.Complexity = "high"
	}

	return viz, nil
}


// AnalyzeDependencies 分析服务依赖关系
func (o *architectureOptimizerImpl) AnalyzeDependencies(ctx context.Context, config *ComposeConfig) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		Services:     make(map[string]*ServiceDependency),
		Layers:       make([][]string, 0),
		CriticalPath: make([]string, 0),
		Cycles:       make([][]string, 0),
	}

	// 构建依赖图
	for serviceName, service := range config.Services {
		dependencies := o.extractDependencies(service)
		graph.Services[serviceName] = &ServiceDependency{
			ServiceName:  serviceName,
			Dependencies: dependencies,
			Dependents:   make([]string, 0),
			Layer:        -1,
		}
	}

	// 计算反向依赖
	for serviceName, dep := range graph.Services {
		for _, depName := range dep.Dependencies {
			if targetDep, exists := graph.Services[depName]; exists {
				targetDep.Dependents = append(targetDep.Dependents, serviceName)
			}
		}
	}

	// 拓扑排序和分层
	graph.Layers = o.topologicalSort(graph.Services)

	// 更新层级信息
	for layerIndex, layer := range graph.Layers {
		for _, serviceName := range layer {
			if dep, exists := graph.Services[serviceName]; exists {
				dep.Layer = layerIndex
			}
		}
	}

	// 检测循环依赖
	graph.Cycles = o.detectCycles(graph.Services)

	// 计算关键路径
	graph.CriticalPath = o.findCriticalPath(graph.Services)

	// 标记关键路径上的服务
	criticalSet := make(map[string]bool)
	for _, serviceName := range graph.CriticalPath {
		criticalSet[serviceName] = true
	}
	for serviceName := range graph.Services {
		if criticalSet[serviceName] {
			graph.Services[serviceName].IsCritical = true
		}
	}

	return graph, nil
}

// topologicalSort 拓扑排序
func (o *architectureOptimizerImpl) topologicalSort(services map[string]*ServiceDependency) [][]string {
	layers := make([][]string, 0)
	visited := make(map[string]bool)
	inDegree := make(map[string]int)

	// 计算入度
	for serviceName, dep := range services {
		inDegree[serviceName] = len(dep.Dependencies)
	}

	// 分层处理
	for len(visited) < len(services) {
		currentLayer := make([]string, 0)

		// 找出入度为 0 的节点
		for serviceName, degree := range inDegree {
			if !visited[serviceName] && degree == 0 {
				currentLayer = append(currentLayer, serviceName)
			}
		}

		if len(currentLayer) == 0 {
			// 如果没有入度为 0 的节点，说明有循环依赖
			// 选择一个未访问的节点
			for serviceName := range services {
				if !visited[serviceName] {
					currentLayer = append(currentLayer, serviceName)
					break
				}
			}
		}

		// 标记为已访问
		for _, serviceName := range currentLayer {
			visited[serviceName] = true

			// 减少依赖此服务的其他服务的入度
			for _, dependent := range services[serviceName].Dependents {
				inDegree[dependent]--
			}
		}

		layers = append(layers, currentLayer)
	}

	return layers
}

// detectCycles 检测循环依赖
func (o *architectureOptimizerImpl) detectCycles(services map[string]*ServiceDependency) [][]string {
	cycles := make([][]string, 0)
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(string, []string) bool
	dfs = func(serviceName string, path []string) bool {
		visited[serviceName] = true
		recStack[serviceName] = true
		path = append(path, serviceName)

		for _, dep := range services[serviceName].Dependencies {
			if !visited[dep] {
				if dfs(dep, path) {
					return true
				}
			} else if recStack[dep] {
				// 找到循环
				cycleStart := -1
				for i, name := range path {
					if name == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := append([]string{}, path[cycleStart:]...)
					cycle = append(cycle, dep)
					cycles = append(cycles, cycle)
				}
				return true
			}
		}

		recStack[serviceName] = false
		return false
	}

	for serviceName := range services {
		if !visited[serviceName] {
			dfs(serviceName, []string{})
		}
	}

	return cycles
}

// findCriticalPath 找到关键路径
func (o *architectureOptimizerImpl) findCriticalPath(services map[string]*ServiceDependency) []string {
	// 简化实现：找到依赖链最长的路径
	var longestPath []string
	visited := make(map[string]bool)

	var dfs func(string, []string)
	dfs = func(serviceName string, path []string) {
		visited[serviceName] = true
		path = append(path, serviceName)

		if len(path) > len(longestPath) {
			longestPath = append([]string{}, path...)
		}

		for _, dep := range services[serviceName].Dependencies {
			if !visited[dep] {
				dfs(dep, path)
			}
		}

		visited[serviceName] = false
	}

	for serviceName := range services {
		dfs(serviceName, []string{})
	}

	return longestPath
}


// EvaluatePerformance 评估性能
func (o *architectureOptimizerImpl) EvaluatePerformance(ctx context.Context, config *ComposeConfig) (*PerformanceEvaluation, error) {
	evaluation := &PerformanceEvaluation{
		Metrics:         &PerformanceMetrics{},
		Bottlenecks:     make([]*PerformanceBottleneck, 0),
		Recommendations: make([]*PerformanceRecommendation, 0),
		Comparison:      &PerformanceComparison{},
	}

	// 计算各项指标
	evaluation.Metrics = o.calculatePerformanceMetrics(config)

	// 识别性能瓶颈
	evaluation.Bottlenecks = o.identifyBottlenecks(config)

	// 生成性能建议
	evaluation.Recommendations = o.generatePerformanceRecommendations(config, evaluation.Bottlenecks)

	// 与最佳实践对比
	evaluation.Comparison = o.compareWithBestPractices(config)

	// 计算总体评分
	evaluation.OverallScore = o.calculateOverallPerformanceScore(evaluation.Metrics)

	return evaluation, nil
}

// calculatePerformanceMetrics 计算性能指标
func (o *architectureOptimizerImpl) calculatePerformanceMetrics(config *ComposeConfig) *PerformanceMetrics {
	metrics := &PerformanceMetrics{}

	// 资源效率评分
	servicesWithLimits := 0
	for _, service := range config.Services {
		if service.Deploy != nil && service.Deploy.Resources != nil && service.Deploy.Resources.Limits != nil {
			servicesWithLimits++
		}
	}
	if len(config.Services) > 0 {
		metrics.ResourceEfficiency = (servicesWithLimits * 100) / len(config.Services)
	}

	// 可扩展性评分
	servicesWithReplicas := 0
	for _, service := range config.Services {
		if service.Deploy != nil && service.Deploy.Replicas > 1 {
			servicesWithReplicas++
		}
	}
	metrics.ScalabilityScore = 50 // 基础分
	if servicesWithReplicas > 0 {
		metrics.ScalabilityScore += 30
	}
	if len(config.Networks) > 1 {
		metrics.ScalabilityScore += 20
	}

	// 可靠性评分
	servicesWithHealthCheck := 0
	servicesWithRestart := 0
	for _, service := range config.Services {
		if service.HealthCheck != nil {
			servicesWithHealthCheck++
		}
		if service.Restart != "" && service.Restart != "no" {
			servicesWithRestart++
		}
	}
	if len(config.Services) > 0 {
		metrics.ReliabilityScore = ((servicesWithHealthCheck + servicesWithRestart) * 50) / len(config.Services)
	}

	// 启动时间估算
	metrics.StartupTimeEstimate = fmt.Sprintf("%d-% d秒", len(config.Services)*2, len(config.Services)*5)

	// 内存占用估算
	totalMemory := len(config.Services) * 512 // 每个服务默认 512MB
	metrics.MemoryFootprint = fmt.Sprintf("约 %d MB", totalMemory)

	// 网络延迟估算
	if len(config.Networks) > 1 {
		metrics.NetworkLatency = "低 (多网络隔离)"
	} else {
		metrics.NetworkLatency = "极低 (单网络)"
	}

	return metrics
}

// identifyBottlenecks 识别性能瓶颈
func (o *architectureOptimizerImpl) identifyBottlenecks(config *ComposeConfig) []*PerformanceBottleneck {
	bottlenecks := make([]*PerformanceBottleneck, 0)

	for serviceName, service := range config.Services {
		// 检查资源限制
		if service.Deploy == nil || service.Deploy.Resources == nil || service.Deploy.Resources.Limits == nil {
			bottlenecks = append(bottlenecks, &PerformanceBottleneck{
				Type:        "resource_limits",
				Service:     serviceName,
				Description: "未设置资源限制",
				Impact:      "可能导致资源竞争，影响整体性能",
				Solutions: []string{
					"设置合理的 CPU 和内存限制",
					"根据实际负载调整资源配置",
				},
			})
		}

		// 检查健康检查
		if service.HealthCheck == nil {
			bottlenecks = append(bottlenecks, &PerformanceBottleneck{
				Type:        "health_check",
				Service:     serviceName,
				Description: "缺少健康检查",
				Impact:      "无法及时发现服务异常，可能影响可用性",
				Solutions: []string{
					"添加健康检查配置",
					"设置合理的检查间隔和超时",
				},
			})
		}

		// 检查日志配置
		if service.Logging == nil {
			bottlenecks = append(bottlenecks, &PerformanceBottleneck{
				Type:        "logging",
				Service:     serviceName,
				Description: "未配置日志管理",
				Impact:      "可能产生大量日志，占用磁盘空间",
				Solutions: []string{
					"配置日志驱动和日志轮转",
					"限制日志文件大小和数量",
				},
			})
		}
	}

	return bottlenecks
}


// generatePerformanceRecommendations 生成性能建议
func (o *architectureOptimizerImpl) generatePerformanceRecommendations(config *ComposeConfig, bottlenecks []*PerformanceBottleneck) []*PerformanceRecommendation {
	recommendations := make([]*PerformanceRecommendation, 0)

	// 基于瓶颈生成建议
	if len(bottlenecks) > 0 {
		recommendations = append(recommendations, &PerformanceRecommendation{
			Title:          "优化资源配置",
			Description:    "为所有服务设置合理的资源限制和预留",
			ExpectedGain:   "提升 20-30% 的资源利用率",
			Implementation: "在 deploy.resources 中配置 limits 和 reservations",
			Priority:       "high",
		})
	}

	// 通用性能建议
	recommendations = append(recommendations, &PerformanceRecommendation{
		Title:          "启用容器缓存",
		Description:    "使用 BuildKit 和多阶段构建优化镜像构建速度",
		ExpectedGain:   "减少 50% 的构建时间",
		Implementation: "启用 DOCKER_BUILDKIT=1 并优化 Dockerfile",
		Priority:       "medium",
	})

	recommendations = append(recommendations, &PerformanceRecommendation{
		Title:          "优化网络配置",
		Description:    "根据服务通信模式优化网络拓扑",
		ExpectedGain:   "降低 10-20% 的网络延迟",
		Implementation: "使用自定义网络并启用 DNS 缓存",
		Priority:       "medium",
	})

	recommendations = append(recommendations, &PerformanceRecommendation{
		Title:          "实施监控和告警",
		Description:    "部署 Prometheus 和 Grafana 进行性能监控",
		ExpectedGain:   "及时发现性能问题，提升可观测性",
		Implementation: "添加 Prometheus exporter 并配置告警规则",
		Priority:       "high",
	})

	return recommendations
}

// compareWithBestPractices 与最佳实践对比
func (o *architectureOptimizerImpl) compareWithBestPractices(config *ComposeConfig) *PerformanceComparison {
	comparison := &PerformanceComparison{
		BestPractices: make([]string, 0),
		Deviations:    make([]string, 0),
	}

	// 检查最佳实践
	hasHealthChecks := false
	hasResourceLimits := false
	hasRestartPolicy := false
	hasNetworkIsolation := len(config.Networks) > 1

	for _, service := range config.Services {
		if service.HealthCheck != nil {
			hasHealthChecks = true
		}
		if service.Deploy != nil && service.Deploy.Resources != nil && service.Deploy.Resources.Limits != nil {
			hasResourceLimits = true
		}
		if service.Restart != "" && service.Restart != "no" {
			hasRestartPolicy = true
		}
	}

	// 符合的最佳实践
	if hasHealthChecks {
		comparison.BestPractices = append(comparison.BestPractices, "✓ 配置了健康检查")
	} else {
		comparison.Deviations = append(comparison.Deviations, "✗ 缺少健康检查配置")
	}

	if hasResourceLimits {
		comparison.BestPractices = append(comparison.BestPractices, "✓ 设置了资源限制")
	} else {
		comparison.Deviations = append(comparison.Deviations, "✗ 未设置资源限制")
	}

	if hasRestartPolicy {
		comparison.BestPractices = append(comparison.BestPractices, "✓ 配置了重启策略")
	} else {
		comparison.Deviations = append(comparison.Deviations, "✗ 缺少重启策略")
	}

	if hasNetworkIsolation {
		comparison.BestPractices = append(comparison.BestPractices, "✓ 使用了网络隔离")
	} else {
		comparison.Deviations = append(comparison.Deviations, "✗ 未使用网络隔离")
	}

	// 计算评分
	score := len(comparison.BestPractices) * 25
	comparison.YourScore = fmt.Sprintf("%d/100", score)
	comparison.IndustryAverage = "70/100"

	return comparison
}

// calculateOverallPerformanceScore 计算总体性能评分
func (o *architectureOptimizerImpl) calculateOverallPerformanceScore(metrics *PerformanceMetrics) int {
	// 加权平均
	score := (metrics.ResourceEfficiency*30 + metrics.ScalabilityScore*30 + metrics.ReliabilityScore*40) / 100
	return score
}
