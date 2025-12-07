package aiagent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// TaskPlanner 任务规划器接口
// 负责将用户意图转换为可执行的任务序列，并进行优化和验证
type TaskPlanner interface {
	// PlanTasks 根据用户意图、实体和参数规划任务序列
	// 参数:
	//   - ctx: 上下文，用于控制超时和取消
	//   - intent: 用户意图（如部署、查询、重启等）
	//   - entities: 从用户输入中提取的实体列表（如服务名、端口等）
	//   - parameters: 已验证的参数映射
	// 返回:
	//   - []*ExecutionTask: 规划好的任务列表
	//   - error: 规划过程中的错误
	PlanTasks(ctx context.Context, intent Intent, entities []Entity, parameters map[string]string) ([]*ExecutionTask, error)
	
	// OptimizeTasks 优化任务序列，减少冗余操作，调整执行顺序
	// 参数:
	//   - ctx: 上下文
	//   - tasks: 待优化的任务列表
	// 返回:
	//   - []*ExecutionTask: 优化后的任务列表
	//   - error: 优化过程中的错误
	OptimizeTasks(ctx context.Context, tasks []*ExecutionTask) ([]*ExecutionTask, error)
	
	// ValidateTasks 验证任务序列的可行性，检查依赖关系和资源可用性
	// 参数:
	//   - ctx: 上下文
	//   - tasks: 待验证的任务列表
	// 返回:
	//   - *PlanValidation: 验证结果，包含问题、警告和建议
	//   - error: 验证过程中的错误
	ValidateTasks(ctx context.Context, tasks []*ExecutionTask) (*PlanValidation, error)
}

// PlanValidation 规划验证结果
// 包含任务序列的验证信息，帮助用户了解任务的可行性和潜在问题
type PlanValidation struct {
	Valid         bool          `json:"valid"`         // 任务序列是否有效
	Issues        []string      `json:"issues"`        // 阻止执行的严重问题列表
	Warnings      []string      `json:"warnings"`      // 不影响执行但需要注意的警告列表
	Suggestions   []string      `json:"suggestions"`   // 优化建议列表
	EstimatedTime time.Duration `json:"estimated_time"` // 预计执行时间
}

// TaskPlannerImpl 任务规划器实现
// 使用模板匹配和规则引擎来生成任务序列
type TaskPlannerImpl struct {
	executor  TaskExecutor              // 任务执行器，用于执行生成的任务
	templates map[Intent][]TaskTemplate // 意图到任务模板的映射，用于快速查找
}

// TaskTemplate 任务模板
// 定义了特定意图和服务的标准任务序列
type TaskTemplate struct {
	Intent       Intent            `json:"intent"`       // 适用的用户意图
	Service      string            `json:"service"`      // 目标服务名称（如 nginx、mysql）
	Tasks        []TaskDefinition  `json:"tasks"`        // 任务定义列表
	Dependencies []string          `json:"dependencies"` // 依赖的其他服务或资源
	Metadata     map[string]string `json:"metadata"`     // 额外的元数据信息
}

// TaskDefinition 任务定义
// 描述单个任务的详细信息和执行要求
type TaskDefinition struct {
	Type       TaskType          `json:"type"`       // 任务类型（如命令执行、配置生成等）
	Command    string            `json:"command"`    // 要执行的命令或操作
	Parameters map[string]string `json:"parameters"` // 任务参数
	Condition  string            `json:"condition"`  // 执行条件（可选）
	Timeout    time.Duration     `json:"timeout"`    // 超时时间
	Reversible bool              `json:"reversible"` // 是否可回滚
	Critical   bool              `json:"critical"`   // 是否为关键任务（失败时是否中止整个流程）
}

// NewTaskPlanner 创建新的任务规划器
// 参数:
//   - executor: 任务执行器实例
// 返回:
//   - TaskPlanner: 初始化完成的任务规划器
func NewTaskPlanner(executor TaskExecutor) TaskPlanner {
	planner := &TaskPlannerImpl{
		executor:  executor,
		templates: make(map[Intent][]TaskTemplate),
	}
	
	// 初始化任务模板，加载预定义的任务模板
	planner.initializeTaskTemplates()
	
	return planner
}

// PlanTasks 规划任务
// 这是任务规划的核心方法，将用户意图转换为具体的执行任务序列
func (p *TaskPlannerImpl) PlanTasks(ctx context.Context, intent Intent, entities []Entity, parameters map[string]string) ([]*ExecutionTask, error) {
	var tasks []*ExecutionTask
	
	// 1. 从实体列表中提取服务名称
	// 服务名称用于匹配特定的任务模板
	serviceName := ""
	for _, entity := range entities {
		if entity.Type == EntityService {
			serviceName = entity.Value
			break
		}
	}
	
	// 2. 查找匹配的任务模板
	// 根据意图和服务名称查找预定义的任务模板
	templates := p.findMatchingTemplates(intent, serviceName)
	if len(templates) == 0 {
		// 如果没有找到匹配的模板，生成通用任务
		return p.generateGenericTasks(intent, entities, parameters)
	}
	
	// 3. 根据模板生成具体的任务
	// 将模板中的占位符替换为实际参数
	for _, template := range templates {
		templateTasks, err := p.generateTasksFromTemplate(template, parameters)
		if err != nil {
			return nil, fmt.Errorf("从模板生成任务失败: %v", err)
		}
		tasks = append(tasks, templateTasks...)
	}
	
	// 4. 优化任务序列
	// 去除冗余操作，调整执行顺序以提高效率
	optimizedTasks, err := p.OptimizeTasks(ctx, tasks)
	if err != nil {
		// 优化失败时返回原始任务，不影响主流程
		return tasks, nil
	}
	
	return optimizedTasks, nil
}

// findMatchingTemplates 查找匹配的任务模板
// 根据意图和服务名称查找预定义的任务模板
func (p *TaskPlannerImpl) findMatchingTemplates(intent Intent, serviceName string) []TaskTemplate {
	var matchedTemplates []TaskTemplate
	
	// 获取该意图的所有模板
	templates, exists := p.templates[intent]
	if !exists {
		return matchedTemplates
	}
	
	// 如果没有指定服务名称，返回所有通用模板
	if serviceName == "" {
		for _, template := range templates {
			if template.Service == "" {
				matchedTemplates = append(matchedTemplates, template)
			}
		}
		return matchedTemplates
	}
	
	// 查找匹配服务名称的模板
	for _, template := range templates {
		if template.Service == serviceName || template.Service == "" {
			matchedTemplates = append(matchedTemplates, template)
		}
	}
	
	return matchedTemplates
}

// generateGenericTasks 生成通用任务
// 当没有找到匹配的模板时，根据意图生成基本的任务序列
func (p *TaskPlannerImpl) generateGenericTasks(intent Intent, entities []Entity, parameters map[string]string) ([]*ExecutionTask, error) {
	var tasks []*ExecutionTask
	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	
	// 根据意图生成基本任务
	switch intent {
	case IntentDeploy, IntentInstall:
		// 部署/安装任务
		serviceName := parameters["service"]
		if serviceName == "" {
			return nil, fmt.Errorf("缺少服务名称参数")
		}
		
		tasks = append(tasks, &ExecutionTask{
			ID:         taskID + "-1",
			Type:       TaskTypeDocker,
			Command:    fmt.Sprintf("pull %s", serviceName),
			Parameters: parameters,
			Timeout:    5 * time.Minute,
			Reversible: false,
			CreatedAt:  time.Now(),
		})
		
		tasks = append(tasks, &ExecutionTask{
			ID:         taskID + "-2",
			Type:       TaskTypeDocker,
			Command:    fmt.Sprintf("run -d --name %s %s", serviceName, serviceName),
			Parameters: parameters,
			Timeout:    2 * time.Minute,
			Reversible: true,
			CreatedAt:  time.Now(),
		})
		
	case IntentStart:
		// 启动服务任务
		serviceName := parameters["service"]
		if serviceName == "" {
			return nil, fmt.Errorf("缺少服务名称参数")
		}
		
		tasks = append(tasks, &ExecutionTask{
			ID:         taskID,
			Type:       TaskTypeDocker,
			Command:    fmt.Sprintf("start %s", serviceName),
			Parameters: parameters,
			Timeout:    1 * time.Minute,
			Reversible: true,
			CreatedAt:  time.Now(),
		})
		
	case IntentStop:
		// 停止服务任务
		serviceName := parameters["service"]
		if serviceName == "" {
			return nil, fmt.Errorf("缺少服务名称参数")
		}
		
		tasks = append(tasks, &ExecutionTask{
			ID:         taskID,
			Type:       TaskTypeDocker,
			Command:    fmt.Sprintf("stop %s", serviceName),
			Parameters: parameters,
			Timeout:    1 * time.Minute,
			Reversible: true,
			CreatedAt:  time.Now(),
		})
		
	case IntentRestart:
		// 重启服务任务
		serviceName := parameters["service"]
		if serviceName == "" {
			return nil, fmt.Errorf("缺少服务名称参数")
		}
		
		tasks = append(tasks, &ExecutionTask{
			ID:         taskID,
			Type:       TaskTypeDocker,
			Command:    fmt.Sprintf("restart %s", serviceName),
			Parameters: parameters,
			Timeout:    2 * time.Minute,
			Reversible: false,
			CreatedAt:  time.Now(),
		})
		
	case IntentQuery, IntentList:
		// 查询任务
		tasks = append(tasks, &ExecutionTask{
			ID:         taskID,
			Type:       TaskTypeDocker,
			Command:    "ps -a",
			Parameters: parameters,
			Timeout:    30 * time.Second,
			Reversible: false,
			CreatedAt:  time.Now(),
		})
		
	case IntentDelete:
		// 删除任务
		serviceName := parameters["service"]
		if serviceName == "" {
			return nil, fmt.Errorf("缺少服务名称参数")
		}
		
		tasks = append(tasks, &ExecutionTask{
			ID:         taskID + "-1",
			Type:       TaskTypeDocker,
			Command:    fmt.Sprintf("stop %s", serviceName),
			Parameters: parameters,
			Timeout:    1 * time.Minute,
			Reversible: false,
			CreatedAt:  time.Now(),
		})
		
		tasks = append(tasks, &ExecutionTask{
			ID:         taskID + "-2",
			Type:       TaskTypeDocker,
			Command:    fmt.Sprintf("rm %s", serviceName),
			Parameters: parameters,
			Timeout:    30 * time.Second,
			Reversible: false,
			CreatedAt:  time.Now(),
		})
		
	default:
		return nil, fmt.Errorf("不支持的意图类型: %s", intent)
	}
	
	return tasks, nil
}

// generateTasksFromTemplate 从模板生成具体任务
// 将模板中的占位符替换为实际参数值
func (p *TaskPlannerImpl) generateTasksFromTemplate(template TaskTemplate, parameters map[string]string) ([]*ExecutionTask, error) {
	var tasks []*ExecutionTask
	
	for i, taskDef := range template.Tasks {
		// 生成唯一的任务ID
		taskID := fmt.Sprintf("task-%s-%d-%d", template.Service, time.Now().UnixNano(), i)
		
		// 替换命令中的参数占位符
		command := taskDef.Command
		for key, value := range parameters {
			placeholder := fmt.Sprintf("{{%s}}", key)
			command = strings.ReplaceAll(command, placeholder, value)
		}
		
		// 合并任务参数
		taskParams := make(map[string]string)
		for k, v := range taskDef.Parameters {
			taskParams[k] = v
		}
		for k, v := range parameters {
			taskParams[k] = v
		}
		
		// 创建执行任务
		task := &ExecutionTask{
			ID:         taskID,
			Type:       taskDef.Type,
			Command:    command,
			Parameters: taskParams,
			Timeout:    taskDef.Timeout,
			Reversible: taskDef.Reversible,
			CreatedAt:  time.Now(),
		}
		
		// 设置默认超时时间
		if task.Timeout == 0 {
			task.Timeout = 2 * time.Minute
		}
		
		tasks = append(tasks, task)
	}
	
	return tasks, nil
}

// OptimizeTasks 优化任务序列
// 去除冗余操作，调整执行顺序以提高效率
func (p *TaskPlannerImpl) OptimizeTasks(ctx context.Context, tasks []*ExecutionTask) ([]*ExecutionTask, error) {
	if len(tasks) == 0 {
		return tasks, nil
	}
	
	optimized := make([]*ExecutionTask, 0, len(tasks))
	taskMap := make(map[string]*ExecutionTask)
	
	// 1. 去除重复任务
	// 使用命令作为唯一标识，相同命令只保留一个
	for _, task := range tasks {
		key := fmt.Sprintf("%s:%s", task.Type, task.Command)
		if _, exists := taskMap[key]; !exists {
			taskMap[key] = task
			optimized = append(optimized, task)
		}
	}
	
	// 2. 调整任务顺序
	// 将查询类任务放在最后，配置类任务放在前面
	var configTasks []*ExecutionTask
	var execTasks []*ExecutionTask
	var queryTasks []*ExecutionTask
	
	for _, task := range optimized {
		switch task.Type {
		case TaskTypeConfig, TaskTypeFile:
			configTasks = append(configTasks, task)
		case TaskTypeShell, TaskTypeDocker, TaskTypeKubernetes, TaskTypeService:
			execTasks = append(execTasks, task)
		default:
			queryTasks = append(queryTasks, task)
		}
	}
	
	// 重新组合：配置 -> 执行 -> 查询
	result := make([]*ExecutionTask, 0, len(optimized))
	result = append(result, configTasks...)
	result = append(result, execTasks...)
	result = append(result, queryTasks...)
	
	return result, nil
}

// ValidateTasks 验证任务序列
// 检查任务的可行性、依赖关系和资源可用性
func (p *TaskPlannerImpl) ValidateTasks(ctx context.Context, tasks []*ExecutionTask) (*PlanValidation, error) {
	validation := &PlanValidation{
		Valid:         true,
		Issues:        []string{},
		Warnings:      []string{},
		Suggestions:   []string{},
		EstimatedTime: 0,
	}
	
	if len(tasks) == 0 {
		validation.Valid = false
		validation.Issues = append(validation.Issues, "任务列表为空")
		return validation, nil
	}
	
	// 1. 验证每个任务的基本信息
	for i, task := range tasks {
		// 检查任务ID
		if task.ID == "" {
			validation.Issues = append(validation.Issues, fmt.Sprintf("任务 %d 缺少ID", i))
			validation.Valid = false
		}
		
		// 检查任务类型
		if task.Type == "" {
			validation.Issues = append(validation.Issues, fmt.Sprintf("任务 %s 缺少类型", task.ID))
			validation.Valid = false
		}
		
		// 检查命令
		if task.Command == "" {
			validation.Issues = append(validation.Issues, fmt.Sprintf("任务 %s 缺少命令", task.ID))
			validation.Valid = false
		}
		
		// 检查超时设置
		if task.Timeout == 0 {
			validation.Warnings = append(validation.Warnings, fmt.Sprintf("任务 %s 未设置超时时间", task.ID))
		} else if task.Timeout > 10*time.Minute {
			validation.Warnings = append(validation.Warnings, fmt.Sprintf("任务 %s 超时时间过长: %v", task.ID, task.Timeout))
		}
		
		// 累计预计执行时间
		if task.Timeout > 0 {
			validation.EstimatedTime += task.Timeout
		} else {
			validation.EstimatedTime += 2 * time.Minute // 默认估计时间
		}
	}
	
	// 2. 检查任务依赖关系
	// 例如：部署任务应该在配置任务之后
	hasConfig := false
	hasDeploy := false
	configIndex := -1
	deployIndex := -1
	
	for i, task := range tasks {
		if task.Type == TaskTypeConfig {
			hasConfig = true
			configIndex = i
		}
		if task.Type == TaskTypeDocker && strings.Contains(task.Command, "run") {
			hasDeploy = true
			deployIndex = i
		}
	}
	
	if hasConfig && hasDeploy && configIndex > deployIndex {
		validation.Warnings = append(validation.Warnings, "配置任务应该在部署任务之前执行")
		validation.Suggestions = append(validation.Suggestions, "建议调整任务顺序，先生成配置再部署")
	}
	
	// 3. 检查资源冲突
	// 检查是否有多个任务操作同一个服务
	serviceOps := make(map[string][]string)
	for _, task := range tasks {
		if serviceName, exists := task.Parameters["service"]; exists {
			serviceOps[serviceName] = append(serviceOps[serviceName], task.Command)
		}
	}
	
	for service, ops := range serviceOps {
		if len(ops) > 3 {
			validation.Warnings = append(validation.Warnings, 
				fmt.Sprintf("服务 %s 有 %d 个操作，可能存在冗余", service, len(ops)))
		}
	}
	
	// 4. 提供优化建议
	if len(tasks) > 10 {
		validation.Suggestions = append(validation.Suggestions, "任务数量较多，建议分批执行")
	}
	
	if validation.EstimatedTime > 30*time.Minute {
		validation.Suggestions = append(validation.Suggestions, 
			fmt.Sprintf("预计执行时间较长 (%v)，建议优化任务或增加并行执行", validation.EstimatedTime))
	}
	
	return validation, nil
}

// initializeTaskTemplates 初始化任务模板
// 加载预定义的常用服务部署模板
func (p *TaskPlannerImpl) initializeTaskTemplates() {
	// Nginx 部署模板
	p.templates[IntentDeploy] = append(p.templates[IntentDeploy], TaskTemplate{
		Intent:  IntentDeploy,
		Service: "nginx",
		Tasks: []TaskDefinition{
			{
				Type:       TaskTypeConfig,
				Command:    "generate",
				Parameters: map[string]string{"type": "nginx"},
				Timeout:    30 * time.Second,
				Reversible: false,
				Critical:   true,
			},
			{
				Type:       TaskTypeDocker,
				Command:    "pull nginx:{{version}}",
				Parameters: map[string]string{},
				Timeout:    5 * time.Minute,
				Reversible: false,
				Critical:   true,
			},
			{
				Type:       TaskTypeDocker,
				Command:    "run -d --name {{service}} -p {{port}}:80 -v {{config_path}}:/etc/nginx/nginx.conf:ro nginx:{{version}}",
				Parameters: map[string]string{},
				Timeout:    2 * time.Minute,
				Reversible: true,
				Critical:   true,
			},
			{
				Type:       TaskTypeDocker,
				Command:    "ps | grep {{service}}",
				Parameters: map[string]string{},
				Timeout:    10 * time.Second,
				Reversible: false,
				Critical:   false,
			},
		},
		Dependencies: []string{},
		Metadata: map[string]string{
			"category":    "web-server",
			"description": "部署 Nginx Web 服务器",
		},
	})
	
	// MySQL 部署模板
	p.templates[IntentDeploy] = append(p.templates[IntentDeploy], TaskTemplate{
		Intent:  IntentDeploy,
		Service: "mysql",
		Tasks: []TaskDefinition{
			{
				Type:       TaskTypeDocker,
				Command:    "pull mysql:{{version}}",
				Parameters: map[string]string{},
				Timeout:    5 * time.Minute,
				Reversible: false,
				Critical:   true,
			},
			{
				Type:       TaskTypeDocker,
				Command:    "run -d --name {{service}} -p {{port}}:3306 -e MYSQL_ROOT_PASSWORD={{password}} mysql:{{version}}",
				Parameters: map[string]string{},
				Timeout:    2 * time.Minute,
				Reversible: true,
				Critical:   true,
			},
			{
				Type:       TaskTypeShell,
				Command:    "sleep 10",
				Parameters: map[string]string{},
				Timeout:    15 * time.Second,
				Reversible: false,
				Critical:   false,
			},
			{
				Type:       TaskTypeDocker,
				Command:    "exec {{service}} mysql -uroot -p{{password}} -e 'SELECT 1'",
				Parameters: map[string]string{},
				Timeout:    30 * time.Second,
				Reversible: false,
				Critical:   false,
			},
		},
		Dependencies: []string{},
		Metadata: map[string]string{
			"category":    "database",
			"description": "部署 MySQL 数据库",
		},
	})
	
	// Redis 部署模板
	p.templates[IntentDeploy] = append(p.templates[IntentDeploy], TaskTemplate{
		Intent:  IntentDeploy,
		Service: "redis",
		Tasks: []TaskDefinition{
			{
				Type:       TaskTypeDocker,
				Command:    "pull redis:{{version}}",
				Parameters: map[string]string{},
				Timeout:    3 * time.Minute,
				Reversible: false,
				Critical:   true,
			},
			{
				Type:       TaskTypeDocker,
				Command:    "run -d --name {{service}} -p {{port}}:6379 redis:{{version}}",
				Parameters: map[string]string{},
				Timeout:    1 * time.Minute,
				Reversible: true,
				Critical:   true,
			},
			{
				Type:       TaskTypeDocker,
				Command:    "exec {{service}} redis-cli ping",
				Parameters: map[string]string{},
				Timeout:    10 * time.Second,
				Reversible: false,
				Critical:   false,
			},
		},
		Dependencies: []string{},
		Metadata: map[string]string{
			"category":    "cache",
			"description": "部署 Redis 缓存服务",
		},
	})
	
	// 通用启动模板
	p.templates[IntentStart] = append(p.templates[IntentStart], TaskTemplate{
		Intent:  IntentStart,
		Service: "",
		Tasks: []TaskDefinition{
			{
				Type:       TaskTypeDocker,
				Command:    "start {{service}}",
				Parameters: map[string]string{},
				Timeout:    1 * time.Minute,
				Reversible: true,
				Critical:   true,
			},
		},
		Dependencies: []string{},
		Metadata: map[string]string{
			"category":    "management",
			"description": "启动容器服务",
		},
	})
	
	// 通用停止模板
	p.templates[IntentStop] = append(p.templates[IntentStop], TaskTemplate{
		Intent:  IntentStop,
		Service: "",
		Tasks: []TaskDefinition{
			{
				Type:       TaskTypeDocker,
				Command:    "stop {{service}}",
				Parameters: map[string]string{},
				Timeout:    1 * time.Minute,
				Reversible: true,
				Critical:   true,
			},
		},
		Dependencies: []string{},
		Metadata: map[string]string{
			"category":    "management",
			"description": "停止容器服务",
		},
	})
	
	// 通用重启模板
	p.templates[IntentRestart] = append(p.templates[IntentRestart], TaskTemplate{
		Intent:  IntentRestart,
		Service: "",
		Tasks: []TaskDefinition{
			{
				Type:       TaskTypeDocker,
				Command:    "restart {{service}}",
				Parameters: map[string]string{},
				Timeout:    2 * time.Minute,
				Reversible: false,
				Critical:   true,
			},
		},
		Dependencies: []string{},
		Metadata: map[string]string{
			"category":    "management",
			"description": "重启容器服务",
		},
	})
}