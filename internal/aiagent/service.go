package aiagent

import (
	"context"
	"fmt"
	"time"
)

// AIAgentService AI Agent 服务主接口
type AIAgentService interface {
	// 自然语言理解
	NLUService
	
	// 处理用户输入
	ProcessUserInput(ctx context.Context, input string, userID string, sessionID string) (*AIResponse, error)
	
	// 执行AI推荐的操作
	ExecuteAction(ctx context.Context, action *AIAction, userID string) (*ExecutionResult, error)
	
	// 获取智能建议
	GetRecommendations(ctx context.Context, context *SystemContext) ([]*Recommendation, error)
	
	// 分析系统问题
	AnalyzeProblem(ctx context.Context, problem *ProblemContext) (*Analysis, error)
	
	// 健康检查
	HealthCheck(ctx context.Context) error
	
	// 清理资源
	Cleanup() error
}

// AIResponse AI响应
type AIResponse struct {
	Intent      Intent            `json:"intent"`
	Message     string            `json:"message"`
	Actions     []AIAction        `json:"actions"`
	Suggestions []string          `json:"suggestions"`
	Context     map[string]string `json:"context"`
	Confidence  float64           `json:"confidence"`
	Error       string            `json:"error,omitempty"`
}

// AIAction AI推荐的操作
type AIAction struct {
	Type        ActionType        `json:"type"`
	Command     string            `json:"command"`
	Parameters  map[string]string `json:"parameters"`
	Description string            `json:"description"`
	Risk        RiskLevel         `json:"risk"`
	Reversible  bool              `json:"reversible"`
}

// ActionType 操作类型
type ActionType string

const (
	ActionShellCommand   ActionType = "shell_command"
	ActionDockerCommand  ActionType = "docker_command"
	ActionK8sCommand     ActionType = "k8s_command"
	ActionConfigGenerate ActionType = "config_generate"
	ActionServiceManage  ActionType = "service_manage"
	ActionFileOperation  ActionType = "file_operation"
)

// RiskLevel 风险等级
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// ExecutionResult 执行结果
type ExecutionResult struct {
	Success    bool              `json:"success"`
	Output     string            `json:"output"`
	Error      string            `json:"error,omitempty"`
	Duration   time.Duration     `json:"duration"`
	Metadata   map[string]string `json:"metadata"`
}

// Recommendation 智能建议
type Recommendation struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Priority    int       `json:"priority"`
	Actions     []AIAction `json:"actions"`
}

// SystemContext 系统上下文
type SystemContext struct {
	SystemInfo   map[string]interface{} `json:"system_info"`
	Services     []ServiceStatus        `json:"services"`
	Resources    ResourceUsage          `json:"resources"`
	Alerts       []Alert                `json:"alerts"`
	Timestamp    time.Time              `json:"timestamp"`
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Health    string            `json:"health"`
	Uptime    time.Duration     `json:"uptime"`
	Metadata  map[string]string `json:"metadata"`
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
	Disk   float64 `json:"disk"`
	Network struct {
		InBytes  uint64 `json:"in_bytes"`
		OutBytes uint64 `json:"out_bytes"`
	} `json:"network"`
}

// Alert 告警信息
type Alert struct {
	ID          string    `json:"id"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Source      string    `json:"source"`
	Timestamp   time.Time `json:"timestamp"`
	Resolved    bool      `json:"resolved"`
}

// ProblemContext 问题上下文
type ProblemContext struct {
	Description string                 `json:"description"`
	Symptoms    []string               `json:"symptoms"`
	Logs        []LogEntry             `json:"logs"`
	Metrics     map[string]interface{} `json:"metrics"`
	Environment map[string]string      `json:"environment"`
}

// Analysis 分析结果
type Analysis struct {
	Summary     string         `json:"summary"`
	RootCause   string         `json:"root_cause"`
	Solutions   []Solution     `json:"solutions"`
	Prevention  []string       `json:"prevention"`
	Confidence  float64        `json:"confidence"`
}

// Solution 解决方案
type Solution struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Steps       []string   `json:"steps"`
	Actions     []AIAction `json:"actions"`
	Risk        RiskLevel  `json:"risk"`
	Success     float64    `json:"success_rate"`
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
}

// AIAgentServiceImpl AI Agent 服务实现
type AIAgentServiceImpl struct {
	*NLUServiceImpl
	persistPath string
}

// NewAIAgentService 创建新的AI Agent服务
func NewAIAgentService() (AIAgentService, error) {
	// 创建NLU服务
	nluService := NewNLUService()
	nluService.persistPath = "./data/sessions" // 可配置的持久化路径
	
	service := &AIAgentServiceImpl{
		NLUServiceImpl: nluService,
		persistPath:    "./data/sessions",
	}
	
	// 启动清理任务
	go service.startCleanupTask()
	
	return service, nil
}

// ProcessUserInput 处理用户输入
func (s *AIAgentServiceImpl) ProcessUserInput(ctx context.Context, input string, userID string, sessionID string) (*AIResponse, error) {
	// 构建NLU请求
	nluReq := &NLURequest{
		Text:      input,
		UserID:    userID,
		SessionID: sessionID,
	}
	
	// 进行自然语言理解
	nluResp, err := s.Understand(ctx, nluReq)
	if err != nil {
		return &AIResponse{
			Intent:     IntentUnknown,
			Message:    "抱歉，我无法理解您的请求",
			Error:      err.Error(),
			Confidence: 0.0,
		}, err
	}
	
	// 根据意图生成响应
	response := &AIResponse{
		Intent:      nluResp.Intent,
		Confidence:  nluResp.Confidence,
		Suggestions: nluResp.Suggestions,
		Context:     make(map[string]string),
	}
	
	// 生成操作建议
	actions, message := s.generateActionsForIntent(nluResp.Intent, nluResp.Parameters)
	response.Actions = actions
	response.Message = message
	
	// 填充上下文信息
	for key, value := range nluResp.Parameters {
		response.Context[key] = value
	}
	
	return response, nil
}

// generateActionsForIntent 根据意图生成操作
func (s *AIAgentServiceImpl) generateActionsForIntent(intent Intent, parameters map[string]string) ([]AIAction, string) {
	var actions []AIAction
	var message string
	
	switch intent {
	case IntentDeploy:
		service := parameters["service"]
		if service == "" {
			message = "请指定要部署的服务名称"
		} else {
			message = fmt.Sprintf("准备部署 %s 服务", service)
			actions = append(actions, AIAction{
				Type:        ActionDockerCommand,
				Command:     fmt.Sprintf("docker run -d --name %s %s", service, service),
				Parameters:  parameters,
				Description: fmt.Sprintf("部署 %s 容器", service),
				Risk:        RiskMedium,
				Reversible:  true,
			})
		}
		
	case IntentQuery:
		target := parameters["service"]
		if target == "" {
			message = "显示系统整体状态"
			actions = append(actions, AIAction{
				Type:        ActionShellCommand,
				Command:     "docker ps -a",
				Description: "查看所有容器状态",
				Risk:        RiskLow,
				Reversible:  true,
			})
		} else {
			message = fmt.Sprintf("查看 %s 的状态", target)
			actions = append(actions, AIAction{
				Type:        ActionShellCommand,
				Command:     fmt.Sprintf("docker ps | grep %s", target),
				Parameters:  parameters,
				Description: fmt.Sprintf("查看 %s 容器状态", target),
				Risk:        RiskLow,
				Reversible:  true,
			})
		}
		
	case IntentStart:
		service := parameters["service"]
		if service == "" {
			message = "请指定要启动的服务名称"
		} else {
			message = fmt.Sprintf("启动 %s 服务", service)
			actions = append(actions, AIAction{
				Type:        ActionDockerCommand,
				Command:     fmt.Sprintf("docker start %s", service),
				Parameters:  parameters,
				Description: fmt.Sprintf("启动 %s 容器", service),
				Risk:        RiskLow,
				Reversible:  true,
			})
		}
		
	case IntentStop:
		service := parameters["service"]
		if service == "" {
			message = "请指定要停止的服务名称"
		} else {
			message = fmt.Sprintf("停止 %s 服务", service)
			actions = append(actions, AIAction{
				Type:        ActionDockerCommand,
				Command:     fmt.Sprintf("docker stop %s", service),
				Parameters:  parameters,
				Description: fmt.Sprintf("停止 %s 容器", service),
				Risk:        RiskMedium,
				Reversible:  true,
			})
		}
		
	case IntentRestart:
		service := parameters["service"]
		if service == "" {
			message = "请指定要重启的服务名称"
		} else {
			message = fmt.Sprintf("重启 %s 服务", service)
			actions = append(actions, AIAction{
				Type:        ActionDockerCommand,
				Command:     fmt.Sprintf("docker restart %s", service),
				Parameters:  parameters,
				Description: fmt.Sprintf("重启 %s 容器", service),
				Risk:        RiskMedium,
				Reversible:  true,
			})
		}
		
	case IntentGenerate:
		target := parameters["target"]
		if target == "" {
			message = "请指定要生成的配置类型"
		} else {
			message = fmt.Sprintf("生成 %s 配置", target)
			actions = append(actions, AIAction{
				Type:        ActionConfigGenerate,
				Command:     fmt.Sprintf("generate_%s_config", target),
				Parameters:  parameters,
				Description: fmt.Sprintf("生成 %s 配置文件", target),
				Risk:        RiskLow,
				Reversible:  true,
			})
		}
		
	default:
		message = "我理解了您的请求，但需要更多信息来执行操作"
	}
	
	return actions, message
}

// ExecuteAction 执行AI推荐的操作
func (s *AIAgentServiceImpl) ExecuteAction(ctx context.Context, action *AIAction, userID string) (*ExecutionResult, error) {
	startTime := time.Now()
	
	result := &ExecutionResult{
		Metadata: make(map[string]string),
	}
	
	// 这里应该集成实际的执行引擎
	// 暂时返回模拟结果
	switch action.Type {
	case ActionShellCommand, ActionDockerCommand:
		result.Success = true
		result.Output = fmt.Sprintf("模拟执行命令: %s", action.Command)
		result.Metadata["command"] = action.Command
		result.Metadata["user"] = userID
		
	case ActionConfigGenerate:
		result.Success = true
		result.Output = "配置文件生成成功"
		result.Metadata["type"] = "config_generation"
		
	default:
		result.Success = false
		result.Error = fmt.Sprintf("不支持的操作类型: %s", action.Type)
	}
	
	result.Duration = time.Since(startTime)
	return result, nil
}

// GetRecommendations 获取智能建议
func (s *AIAgentServiceImpl) GetRecommendations(ctx context.Context, sysCtx *SystemContext) ([]*Recommendation, error) {
	var recommendations []*Recommendation
	
	// 基于系统状态生成建议
	if sysCtx.Resources.CPU > 80 {
		recommendations = append(recommendations, &Recommendation{
			Title:       "CPU使用率过高",
			Description: "系统CPU使用率超过80%，建议检查高负载进程",
			Category:    "performance",
			Priority:    1,
			Actions: []AIAction{
				{
					Type:        ActionShellCommand,
					Command:     "top -b -n 1 | head -20",
					Description: "查看CPU使用率最高的进程",
					Risk:        RiskLow,
				},
			},
		})
	}
	
	if sysCtx.Resources.Memory > 90 {
		recommendations = append(recommendations, &Recommendation{
			Title:       "内存使用率过高",
			Description: "系统内存使用率超过90%，建议释放内存或扩容",
			Category:    "performance",
			Priority:    1,
			Actions: []AIAction{
				{
					Type:        ActionShellCommand,
					Command:     "free -h",
					Description: "查看内存使用详情",
					Risk:        RiskLow,
				},
			},
		})
	}
	
	return recommendations, nil
}

// AnalyzeProblem 分析系统问题
func (s *AIAgentServiceImpl) AnalyzeProblem(ctx context.Context, problem *ProblemContext) (*Analysis, error) {
	analysis := &Analysis{
		Summary:    fmt.Sprintf("问题分析: %s", problem.Description),
		Solutions:  []Solution{},
		Prevention: []string{},
		Confidence: 0.7,
	}
	
	// 基于症状分析根本原因
	if len(problem.Symptoms) > 0 {
		analysis.RootCause = "基于提供的症状，可能的根本原因包括资源不足、配置错误或服务依赖问题"
		
		// 生成解决方案
		analysis.Solutions = append(analysis.Solutions, Solution{
			Title:       "检查系统资源",
			Description: "检查CPU、内存、磁盘使用情况",
			Steps:       []string{"查看系统负载", "检查内存使用", "检查磁盘空间"},
			Actions: []AIAction{
				{
					Type:        ActionShellCommand,
					Command:     "top -b -n 1",
					Description: "查看系统负载",
					Risk:        RiskLow,
				},
			},
			Risk:    RiskLow,
			Success: 0.8,
		})
	}
	
	return analysis, nil
}

// HealthCheck 健康检查
func (s *AIAgentServiceImpl) HealthCheck(ctx context.Context) error {
	// 检查上下文存储
	if s.contexts == nil {
		return fmt.Errorf("上下文存储未初始化")
	}
	
	// 检查模板和服务定义
	if len(s.templates) == 0 {
		return fmt.Errorf("命令模板未初始化")
	}
	
	if len(s.services) == 0 {
		return fmt.Errorf("服务定义未初始化")
	}
	
	return nil
}

// Cleanup 清理资源
func (s *AIAgentServiceImpl) Cleanup() error {
	// 清理过期会话
	s.CleanupExpiredSessions()
	return nil
}

// startCleanupTask 启动清理任务
func (s *AIAgentServiceImpl) startCleanupTask() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		s.CleanupExpiredSessions()
	}
}