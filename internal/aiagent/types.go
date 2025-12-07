package aiagent

import (
	"context"
	"time"
)

// Intent 意图类型
type Intent string

const (
	// 部署相关意图
	IntentDeploy     Intent = "deploy"      // 部署应用
	IntentInstall    Intent = "install"     // 安装服务
	IntentCreate     Intent = "create"      // 创建资源
	
	// 查询相关意图
	IntentQuery      Intent = "query"       // 查询状态
	IntentList       Intent = "list"        // 列出资源
	IntentShow       Intent = "show"        // 显示详情
	
	// 管理相关意图
	IntentStart      Intent = "start"       // 启动服务
	IntentStop       Intent = "stop"        // 停止服务
	IntentRestart    Intent = "restart"     // 重启服务
	IntentUpdate     Intent = "update"      // 更新配置
	IntentDelete     Intent = "delete"      // 删除资源
	
	// 诊断相关意图
	IntentDiagnose   Intent = "diagnose"    // 诊断问题
	IntentAnalyze    Intent = "analyze"     // 分析性能
	IntentTroubleshoot Intent = "troubleshoot" // 故障排除
	
	// 配置相关意图
	IntentConfigure  Intent = "configure"   // 配置服务
	IntentGenerate   Intent = "generate"    // 生成配置
	
	// 未知意图
	IntentUnknown    Intent = "unknown"
)

// Entity 实体类型
type Entity struct {
	Type       EntityType `json:"type"`        // 实体类型
	Value      string     `json:"value"`       // 实体值
	Confidence float64    `json:"confidence"`  // 置信度
	StartPos   int        `json:"start_pos"`   // 开始位置
	EndPos     int        `json:"end_pos"`     // 结束位置
}

// EntityType 实体类型
type EntityType string

const (
	EntityService    EntityType = "service"     // 服务名称 (nginx, mysql, redis)
	EntityContainer  EntityType = "container"   // 容器名称
	EntityApplication EntityType = "application" // 应用名称
	EntityResource   EntityType = "resource"    // 资源类型 (cpu, memory, disk)
	EntityPort       EntityType = "port"        // 端口号
	EntityPath       EntityType = "path"        // 文件路径
	EntityDomain     EntityType = "domain"      // 域名
	EntityDatabase   EntityType = "database"    // 数据库名称
	EntityUser       EntityType = "user"        // 用户名
	EntityNumber     EntityType = "number"      // 数字
	EntityTime       EntityType = "time"        // 时间
)

// NLURequest 自然语言理解请求
type NLURequest struct {
	Text      string            `json:"text"`       // 用户输入文本
	Context   *ConversationContext `json:"context"`    // 对话上下文
	UserID    string            `json:"user_id"`    // 用户ID
	SessionID string            `json:"session_id"` // 会话ID
	Language  string            `json:"language"`   // 语言 (zh, en)
}

// NLUResponse 自然语言理解响应
type NLUResponse struct {
	Intent     Intent            `json:"intent"`     // 识别的意图
	Entities   []Entity          `json:"entities"`   // 提取的实体
	Confidence float64           `json:"confidence"` // 整体置信度
	Parameters map[string]string `json:"parameters"` // 解析的参数
	Suggestions []string         `json:"suggestions"` // 建议的操作
	Error      string            `json:"error,omitempty"` // 错误信息
}

// ConversationContext 对话上下文
type ConversationContext struct {
	SessionID     string                 `json:"session_id"`
	UserID        string                 `json:"user_id"`
	LastIntent    Intent                 `json:"last_intent"`
	LastEntities  []Entity               `json:"last_entities"`
	History       []ConversationTurn     `json:"history"`
	Variables     map[string]interface{} `json:"variables"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ConversationTurn 对话轮次
type ConversationTurn struct {
	UserInput    string    `json:"user_input"`
	Intent       Intent    `json:"intent"`
	Entities     []Entity  `json:"entities"`
	Response     string    `json:"response"`
	Timestamp    time.Time `json:"timestamp"`
}

// NLUService 自然语言理解服务接口
type NLUService interface {
	// 理解用户输入
	Understand(ctx context.Context, req *NLURequest) (*NLUResponse, error)
	
	// 意图识别
	RecognizeIntent(ctx context.Context, text string, context *ConversationContext) (Intent, float64, error)
	
	// 实体提取
	ExtractEntities(ctx context.Context, text string, intent Intent) ([]Entity, error)
	
	// 参数验证
	ValidateParameters(ctx context.Context, intent Intent, entities []Entity) (map[string]string, error)
	
	// 上下文管理
	UpdateContext(ctx context.Context, sessionID string, turn *ConversationTurn) error
	GetContext(ctx context.Context, sessionID string) (*ConversationContext, error)
	
	// 多语言支持
	DetectLanguage(ctx context.Context, text string) (string, error)
	TranslateIntent(ctx context.Context, intent Intent, language string) (string, error)
}

// CommandTemplate 命令模板
type CommandTemplate struct {
	Intent      Intent            `json:"intent"`
	Pattern     string            `json:"pattern"`     // 正则表达式模式
	Parameters  []string          `json:"parameters"`  // 参数列表
	Examples    []string          `json:"examples"`    // 示例
	Description string            `json:"description"` // 描述
}

// ServiceDefinition 服务定义
type ServiceDefinition struct {
	Name        string   `json:"name"`         // 服务名称
	Aliases     []string `json:"aliases"`      // 别名
	Category    string   `json:"category"`     // 分类
	Description string   `json:"description"`  // 描述
	Ports       []int    `json:"ports"`        // 默认端口
	Commands    []string `json:"commands"`     // 相关命令
}