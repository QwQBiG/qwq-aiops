package aiagent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TaskExecutor AI 任务执行引擎
type TaskExecutor interface {
	// 执行任务
	ExecuteTask(ctx context.Context, task *ExecutionTask) (*TaskResult, error)
	
	// 生成配置文件
	GenerateConfig(ctx context.Context, configType string, parameters map[string]string) (*ConfigResult, error)
	
	// 验证执行结果
	ValidateResult(ctx context.Context, result *TaskResult) (*ValidationResult, error)
	
	// 回滚操作
	RollbackTask(ctx context.Context, taskID string) error
}

// ExecutionTask 执行任务
type ExecutionTask struct {
	ID          string            `json:"id"`
	Type        TaskType          `json:"type"`
	Command     string            `json:"command"`
	Parameters  map[string]string `json:"parameters"`
	WorkingDir  string            `json:"working_dir"`
	Timeout     time.Duration     `json:"timeout"`
	Reversible  bool              `json:"reversible"`
	DryRun      bool              `json:"dry_run"`
	CreatedAt   time.Time         `json:"created_at"`
}

// TaskType 任务类型
type TaskType string

const (
	TaskTypeShell      TaskType = "shell"
	TaskTypeDocker     TaskType = "docker"
	TaskTypeKubernetes TaskType = "kubernetes"
	TaskTypeConfig     TaskType = "config"
	TaskTypeFile       TaskType = "file"
	TaskTypeService    TaskType = "service"
)

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID      string            `json:"task_id"`
	Success     bool              `json:"success"`
	Output      string            `json:"output"`
	Error       string            `json:"error,omitempty"`
	ExitCode    int               `json:"exit_code"`
	Duration    time.Duration     `json:"duration"`
	Metadata    map[string]string `json:"metadata"`
	RollbackCmd string            `json:"rollback_cmd,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// ConfigResult 配置生成结果
type ConfigResult struct {
	ConfigType string            `json:"config_type"`
	Content    string            `json:"content"`
	FilePath   string            `json:"file_path"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  time.Time         `json:"created_at"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid       bool     `json:"valid"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
	Score       float64  `json:"score"`
}

// TaskExecutorImpl 任务执行引擎实现
type TaskExecutorImpl struct {
	workingDir    string
	allowedCmds   map[string]bool
	configTemplates map[string]string
	executionHistory []TaskResult
}

// NewTaskExecutor 创建新的任务执行引擎
func NewTaskExecutor(workingDir string) TaskExecutor {
	return &TaskExecutorImpl{
		workingDir:  workingDir,
		allowedCmds: initAllowedCommands(),
		configTemplates: initConfigTemplates(),
		executionHistory: make([]TaskResult, 0),
	}
}

// ExecuteTask 执行任务
func (e *TaskExecutorImpl) ExecuteTask(ctx context.Context, task *ExecutionTask) (*TaskResult, error) {
	startTime := time.Now()
	
	result := &TaskResult{
		TaskID:    task.ID,
		Metadata:  make(map[string]string),
		CreatedAt: startTime,
	}
	
	// 设置超时
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}
	
	// 根据任务类型执行
	switch task.Type {
	case TaskTypeShell:
		return e.executeShellTask(ctx, task, result)
	case TaskTypeDocker:
		return e.executeDockerTask(ctx, task, result)
	case TaskTypeKubernetes:
		return e.executeK8sTask(ctx, task, result)
	case TaskTypeConfig:
		return e.executeConfigTask(ctx, task, result)
	case TaskTypeFile:
		return e.executeFileTask(ctx, task, result)
	case TaskTypeService:
		return e.executeServiceTask(ctx, task, result)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("不支持的任务类型: %s", task.Type)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("不支持的任务类型: %s", task.Type)
	}
}

// executeShellTask 执行 Shell 任务
func (e *TaskExecutorImpl) executeShellTask(ctx context.Context, task *ExecutionTask, result *TaskResult) (*TaskResult, error) {
	startTime := time.Now()
	
	// 安全检查
	if !e.isCommandSafe(task.Command) {
		result.Success = false
		result.Error = "命令被安全策略阻止"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("命令被安全策略阻止: %s", task.Command)
	}
	
	// 如果是 DryRun，只返回模拟结果
	if task.DryRun {
		result.Success = true
		result.Output = fmt.Sprintf("[DRY RUN] 将执行命令: %s", task.Command)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	
	// 解析命令
	parts := strings.Fields(task.Command)
	if len(parts) == 0 {
		result.Success = false
		result.Error = "空命令"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("空命令")
	}
	
	// 创建命令
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	
	// 设置工作目录
	if task.WorkingDir != "" {
		cmd.Dir = task.WorkingDir
	} else if e.workingDir != "" {
		cmd.Dir = e.workingDir
	}
	
	// 执行命令
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	result.Duration = time.Since(startTime)
	
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		}
	} else {
		result.Success = true
		result.ExitCode = 0
	}
	
	// 生成回滚命令（如果可能）
	if task.Reversible {
		result.RollbackCmd = e.generateRollbackCommand(task.Command)
	}
	
	// 记录执行历史
	e.executionHistory = append(e.executionHistory, *result)
	
	return result, nil
}

// executeDockerTask 执行 Docker 任务
func (e *TaskExecutorImpl) executeDockerTask(ctx context.Context, task *ExecutionTask, result *TaskResult) (*TaskResult, error) {
	// Docker 命令前缀
	dockerCmd := "docker " + task.Command
	
	// 创建新的 Shell 任务
	dockerTask := &ExecutionTask{
		ID:         task.ID,
		Type:       TaskTypeShell,
		Command:    dockerCmd,
		Parameters: task.Parameters,
		WorkingDir: task.WorkingDir,
		Timeout:    task.Timeout,
		Reversible: task.Reversible,
		DryRun:     task.DryRun,
		CreatedAt:  task.CreatedAt,
	}
	
	// 执行 Docker 命令
	return e.executeShellTask(ctx, dockerTask, result)
}

// executeK8sTask 执行 Kubernetes 任务
func (e *TaskExecutorImpl) executeK8sTask(ctx context.Context, task *ExecutionTask, result *TaskResult) (*TaskResult, error) {
	// kubectl 命令前缀
	k8sCmd := "kubectl " + task.Command
	
	// 创建新的 Shell 任务
	k8sTask := &ExecutionTask{
		ID:         task.ID,
		Type:       TaskTypeShell,
		Command:    k8sCmd,
		Parameters: task.Parameters,
		WorkingDir: task.WorkingDir,
		Timeout:    task.Timeout,
		Reversible: task.Reversible,
		DryRun:     task.DryRun,
		CreatedAt:  task.CreatedAt,
	}
	
	// 执行 kubectl 命令
	return e.executeShellTask(ctx, k8sTask, result)
}

// executeConfigTask 执行配置生成任务
func (e *TaskExecutorImpl) executeConfigTask(ctx context.Context, task *ExecutionTask, result *TaskResult) (*TaskResult, error) {
	startTime := time.Now()
	
	configType := task.Parameters["type"]
	if configType == "" {
		result.Success = false
		result.Error = "缺少配置类型参数"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("缺少配置类型参数")
	}
	
	// 生成配置
	configResult, err := e.GenerateConfig(ctx, configType, task.Parameters)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}
	
	result.Success = true
	result.Output = fmt.Sprintf("配置文件已生成: %s", configResult.FilePath)
	result.Duration = time.Since(startTime)
	result.Metadata["config_type"] = configResult.ConfigType
	result.Metadata["file_path"] = configResult.FilePath
	
	return result, nil
}

// executeFileTask 执行文件操作任务
func (e *TaskExecutorImpl) executeFileTask(ctx context.Context, task *ExecutionTask, result *TaskResult) (*TaskResult, error) {
	startTime := time.Now()
	
	operation := task.Parameters["operation"]
	filePath := task.Parameters["path"]
	
	if operation == "" || filePath == "" {
		result.Success = false
		result.Error = "缺少操作类型或文件路径参数"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("缺少操作类型或文件路径参数")
	}
	
	switch operation {
	case "create":
		content := task.Parameters["content"]
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
			result.Output = fmt.Sprintf("文件已创建: %s", filePath)
		}
		
	case "delete":
		err := os.Remove(filePath)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
			result.Output = fmt.Sprintf("文件已删除: %s", filePath)
		}
		
	case "read":
		content, err := os.ReadFile(filePath)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
			result.Output = string(content)
		}
		
	default:
		result.Success = false
		result.Error = fmt.Sprintf("不支持的文件操作: %s", operation)
	}
	
	result.Duration = time.Since(startTime)
	return result, nil
}

// executeServiceTask 执行服务管理任务
func (e *TaskExecutorImpl) executeServiceTask(ctx context.Context, task *ExecutionTask, result *TaskResult) (*TaskResult, error) {
	startTime := time.Now()
	
	service := task.Parameters["service"]
	action := task.Parameters["action"]
	
	if service == "" || action == "" {
		result.Success = false
		result.Error = "缺少服务名称或操作参数"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("缺少服务名称或操作参数")
	}
	
	// 构建 systemctl 命令
	systemctlCmd := fmt.Sprintf("systemctl %s %s", action, service)
	
	// 创建新的 Shell 任务
	serviceTask := &ExecutionTask{
		ID:         task.ID,
		Type:       TaskTypeShell,
		Command:    systemctlCmd,
		Parameters: task.Parameters,
		WorkingDir: task.WorkingDir,
		Timeout:    task.Timeout,
		Reversible: task.Reversible,
		DryRun:     task.DryRun,
		CreatedAt:  task.CreatedAt,
	}
	
	// 执行 systemctl 命令
	return e.executeShellTask(ctx, serviceTask, result)
}

// GenerateConfig 生成配置文件
func (e *TaskExecutorImpl) GenerateConfig(ctx context.Context, configType string, parameters map[string]string) (*ConfigResult, error) {
	template, exists := e.configTemplates[configType]
	if !exists {
		return nil, fmt.Errorf("不支持的配置类型: %s", configType)
	}
	
	// 替换模板中的参数
	content := template
	for key, value := range parameters {
		placeholder := fmt.Sprintf("{{%s}}", key)
		content = strings.ReplaceAll(content, placeholder, value)
	}
	
	// 生成文件路径
	fileName := fmt.Sprintf("%s.conf", configType)
	if customName, exists := parameters["filename"]; exists {
		fileName = customName
	}
	
	filePath := filepath.Join(e.workingDir, fileName)
	
	// 写入文件
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return nil, fmt.Errorf("写入配置文件失败: %v", err)
	}
	
	return &ConfigResult{
		ConfigType: configType,
		Content:    content,
		FilePath:   filePath,
		Metadata:   parameters,
		CreatedAt:  time.Now(),
	}, nil
}

// ValidateResult 验证执行结果
func (e *TaskExecutorImpl) ValidateResult(ctx context.Context, result *TaskResult) (*ValidationResult, error) {
	validation := &ValidationResult{
		Valid:       true,
		Issues:      []string{},
		Suggestions: []string{},
		Score:       1.0,
	}
	
	// 基本验证
	if !result.Success {
		validation.Valid = false
		validation.Issues = append(validation.Issues, "任务执行失败")
		validation.Score -= 0.5
	}
	
	if result.ExitCode != 0 {
		validation.Issues = append(validation.Issues, fmt.Sprintf("非零退出码: %d", result.ExitCode))
		validation.Score -= 0.2
	}
	
	if result.Duration > 30*time.Second {
		validation.Issues = append(validation.Issues, "执行时间过长")
		validation.Suggestions = append(validation.Suggestions, "考虑优化命令或增加超时时间")
		validation.Score -= 0.1
	}
	
	// 输出验证
	if strings.Contains(strings.ToLower(result.Output), "error") {
		validation.Issues = append(validation.Issues, "输出中包含错误信息")
		validation.Score -= 0.2
	}
	
	if strings.Contains(strings.ToLower(result.Output), "warning") {
		validation.Issues = append(validation.Issues, "输出中包含警告信息")
		validation.Score -= 0.1
	}
	
	// 设置最终验证状态
	if validation.Score < 0.5 {
		validation.Valid = false
	}
	
	return validation, nil
}

// RollbackTask 回滚任务
func (e *TaskExecutorImpl) RollbackTask(ctx context.Context, taskID string) error {
	// 查找任务执行历史
	var targetResult *TaskResult
	for _, result := range e.executionHistory {
		if result.TaskID == taskID {
			targetResult = &result
			break
		}
	}
	
	if targetResult == nil {
		return fmt.Errorf("未找到任务执行记录: %s", taskID)
	}
	
	if targetResult.RollbackCmd == "" {
		return fmt.Errorf("任务不支持回滚: %s", taskID)
	}
	
	// 创建回滚任务
	rollbackTask := &ExecutionTask{
		ID:         fmt.Sprintf("%s-rollback", taskID),
		Type:       TaskTypeShell,
		Command:    targetResult.RollbackCmd,
		Parameters: make(map[string]string),
		Timeout:    30 * time.Second,
		Reversible: false,
		DryRun:     false,
		CreatedAt:  time.Now(),
	}
	
	// 执行回滚
	_, err := e.ExecuteTask(ctx, rollbackTask)
	return err
}

// 辅助方法

// isCommandSafe 检查命令是否安全
func (e *TaskExecutorImpl) isCommandSafe(command string) bool {
	// 危险命令列表
	dangerousCmds := []string{
		"rm -rf /",
		"dd if=",
		"mkfs",
		"fdisk",
		"format",
		"shutdown",
		"reboot",
		"halt",
		"init 0",
		"init 6",
	}
	
	lowerCmd := strings.ToLower(command)
	for _, dangerous := range dangerousCmds {
		if strings.Contains(lowerCmd, dangerous) {
			return false
		}
	}
	
	// 检查是否在允许的命令列表中
	parts := strings.Fields(command)
	if len(parts) > 0 {
		mainCmd := parts[0]
		return e.allowedCmds[mainCmd]
	}
	
	return false
}

// generateRollbackCommand 生成回滚命令
func (e *TaskExecutorImpl) generateRollbackCommand(originalCmd string) string {
	parts := strings.Fields(originalCmd)
	if len(parts) == 0 {
		return ""
	}
	
	mainCmd := parts[0]
	
	// 根据命令类型生成回滚命令
	switch mainCmd {
	case "docker":
		if len(parts) >= 2 {
			switch parts[1] {
			case "run":
				// docker run -> docker stop + docker rm
				if len(parts) >= 4 && parts[2] == "--name" {
					containerName := parts[3]
					return fmt.Sprintf("docker stop %s && docker rm %s", containerName, containerName)
				}
			case "start":
				if len(parts) >= 3 {
					containerName := parts[2]
					return fmt.Sprintf("docker stop %s", containerName)
				}
			}
		}
	case "systemctl":
		if len(parts) >= 3 {
			action := parts[1]
			service := parts[2]
			switch action {
			case "start":
				return fmt.Sprintf("systemctl stop %s", service)
			case "stop":
				return fmt.Sprintf("systemctl start %s", service)
			case "enable":
				return fmt.Sprintf("systemctl disable %s", service)
			case "disable":
				return fmt.Sprintf("systemctl enable %s", service)
			}
		}
	}
	
	return ""
}

// initAllowedCommands 初始化允许的命令列表
func initAllowedCommands() map[string]bool {
	return map[string]bool{
		// 基本命令
		"ls":     true,
		"pwd":    true,
		"cat":    true,
		"head":   true,
		"tail":   true,
		"grep":   true,
		"find":   true,
		"which":  true,
		"whoami": true,
		"id":     true,
		"date":   true,
		"uptime": true,
		"echo":   true,
		"sleep":  true,
		
		// 系统监控
		"ps":      true,
		"top":     true,
		"htop":    true,
		"free":    true,
		"df":      true,
		"du":      true,
		"netstat": true,
		"ss":      true,
		"lsof":    true,
		"iostat":  true,
		"vmstat":  true,
		
		// 网络工具
		"ping":    true,
		"curl":    true,
		"wget":    true,
		"telnet":  true,
		"nc":      true,
		"nslookup": true,
		"dig":     true,
		
		// Docker
		"docker": true,
		
		// Kubernetes
		"kubectl": true,
		
		// 系统服务
		"systemctl": true,
		"service":   true,
		
		// 日志
		"journalctl": true,
		
		// 文件操作（限制性）
		"mkdir": true,
		"touch": true,
		"cp":    true,
		"mv":    true,
		"chmod": true,
		"chown": true,
	}
}

// initConfigTemplates 初始化配置模板
func initConfigTemplates() map[string]string {
	return map[string]string{
		"nginx": `server {
    listen {{port}};
    server_name {{server_name}};
    
    location / {
        proxy_pass {{backend}};
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}`,
		
		"docker-compose": `version: '3.8'
services:
  {{service_name}}:
    image: {{image}}
    ports:
      - "{{port}}:{{container_port}}"
    environment:
      - ENV={{env}}
    volumes:
      - {{volume}}:/data
    restart: unless-stopped`,
		
		"systemd": `[Unit]
Description={{description}}
After=network.target

[Service]
Type={{type}}
User={{user}}
ExecStart={{exec_start}}
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target`,
		
		"prometheus": `global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: '{{job_name}}'
    static_configs:
      - targets: ['{{target}}']
    scrape_interval: {{interval}}s`,
	}
}