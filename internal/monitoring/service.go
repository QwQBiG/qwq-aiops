package monitoring

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrAlertRuleNotFound 告警规则不存在
	ErrAlertRuleNotFound = errors.New("告警规则不存在")
	
	// ErrMetricNotFound 指标不存在
	ErrMetricNotFound = errors.New("指标不存在")
)

// MonitoringService 监控服务接口，提供指标收集、告警管理和 AI 分析功能
type MonitoringService interface {
	// 指标管理
	RecordMetric(ctx context.Context, metric *Metric) error                                 // 记录指标数据
	QueryMetrics(ctx context.Context, query *MetricQuery) ([]*MetricData, error)            // 查询指标数据
	ListMetrics(ctx context.Context, userID, tenantID uint) ([]*MetricDefinition, error)    // 列出指标定义
	
	// 告警规则管理
	CreateAlertRule(ctx context.Context, rule *AlertRule) error                             // 创建告警规则
	UpdateAlertRule(ctx context.Context, id uint, rule *AlertRule) error                    // 更新告警规则
	DeleteAlertRule(ctx context.Context, id uint) error                                     // 删除告警规则
	GetAlertRule(ctx context.Context, id uint) (*AlertRule, error)                          // 获取告警规则
	ListAlertRules(ctx context.Context, userID, tenantID uint) ([]*AlertRule, error)        // 列出告警规则
	
	// 告警管理
	ListAlerts(ctx context.Context, filters *AlertFilters) ([]*Alert, error)                // 列出告警（支持过滤）
	AcknowledgeAlert(ctx context.Context, alertID uint, userID uint) error                  // 确认告警
	ResolveAlert(ctx context.Context, alertID uint, userID uint) error                      // 解决告警
	
	// AI 分析
	PredictIssues(ctx context.Context, resourceType, resourceID string) (*PredictionResult, error) // AI 预测潜在问题
	AnalyzeCapacity(ctx context.Context, resourceType string) (*CapacityAnalysis, error)           // AI 容量分析
}

// MetricType 指标类型
type MetricType string

const (
	MetricTypeGauge     MetricType = "gauge"     // 瞬时值
	MetricTypeCounter   MetricType = "counter"   // 计数器
	MetricTypeHistogram MetricType = "histogram" // 直方图
	MetricTypeSummary   MetricType = "summary"   // 摘要
)

// AlertSeverity 告警严重程度
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical" // 严重
	SeverityWarning  AlertSeverity = "warning"  // 警告
	SeverityInfo     AlertSeverity = "info"     // 信息
)

// AlertStatus 告警状态
type AlertStatus string

const (
	AlertStatusFiring       AlertStatus = "firing"       // 触发中
	AlertStatusAcknowledged AlertStatus = "acknowledged" // 已确认
	AlertStatusResolved     AlertStatus = "resolved"     // 已解决
)

// Metric 指标数据点，用于记录实时监控数据
type Metric struct {
	Name      string                 `json:"name"`      // 指标名称
	Type      MetricType             `json:"type"`      // 指标类型
	Value     float64                `json:"value"`     // 指标值
	Labels    map[string]string      `json:"labels"`    // 标签（用于多维度查询）
	Timestamp time.Time              `json:"timestamp"` // 时间戳
	UserID    uint                   `json:"user_id"`   // 用户 ID
	TenantID  uint                   `json:"tenant_id"` // 租户 ID
}

// MetricDefinition 指标定义，描述指标的元数据信息
type MetricDefinition struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"uniqueIndex;not null"` // 指标名称（唯一）
	Type        MetricType `json:"type" gorm:"not null"`             // 指标类型
	Description string     `json:"description"`                      // 指标描述
	Unit        string     `json:"unit"`                             // 单位（如 bytes, ms, %）
	UserID      uint       `json:"user_id" gorm:"index"`
	TenantID    uint       `json:"tenant_id" gorm:"index"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// MetricData 指标数据
type MetricData struct {
	Timestamp time.Time         `json:"timestamp"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
}

// MetricQuery 指标查询条件，支持时间范围和标签过滤
type MetricQuery struct {
	MetricName  string            `json:"metric_name"`  // 指标名称
	Labels      map[string]string `json:"labels"`       // 标签过滤条件
	StartTime   time.Time         `json:"start_time"`   // 开始时间
	EndTime     time.Time         `json:"end_time"`     // 结束时间
	Step        time.Duration     `json:"step"`         // 采样间隔
	Aggregation string            `json:"aggregation"`  // 聚合方式：avg, sum, min, max
}

// AlertRule 告警规则，定义触发告警的条件和通知方式
type AlertRule struct {
	ID          uint          `json:"id" gorm:"primaryKey"`
	Name        string        `json:"name" gorm:"not null"`        // 规则名称
	Description string        `json:"description"`                 // 规则描述
	Enabled     bool          `json:"enabled" gorm:"default:true"` // 是否启用
	Severity    AlertSeverity `json:"severity" gorm:"not null"`    // 严重程度
	
	// 规则条件
	MetricName  string  `json:"metric_name" gorm:"not null"`      // 监控指标名称
	Operator    string  `json:"operator" gorm:"not null"`         // 比较操作符：>, <, >=, <=, ==, !=
	Threshold   float64 `json:"threshold" gorm:"not null"`        // 阈值
	Duration    int     `json:"duration"`                         // 持续时间（秒），超过此时间才触发
	Labels      string  `json:"labels" gorm:"type:jsonb"`         // 标签过滤条件
	
	// 告警配置
	Cooldown    int    `json:"cooldown" gorm:"default:300"`       // 冷却时间（秒），避免重复告警
	Aggregation string `json:"aggregation" gorm:"default:avg"`    // 聚合方式：avg, sum, min, max
	
	// 通知配置
	NotifyChannels string `json:"notify_channels" gorm:"type:jsonb"` // 通知渠道（邮件、短信、Webhook 等）
	
	UserID    uint      `json:"user_id" gorm:"index"`
	TenantID  uint      `json:"tenant_id" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Alert 告警实例，记录实际触发的告警事件
type Alert struct {
	ID          uint          `json:"id" gorm:"primaryKey"`
	RuleID      uint          `json:"rule_id" gorm:"index"`                    // 关联的告警规则 ID
	Rule        *AlertRule    `json:"rule,omitempty" gorm:"foreignKey:RuleID"` // 关联的告警规则
	Status      AlertStatus   `json:"status" gorm:"index"`                     // 告警状态
	Severity    AlertSeverity `json:"severity"`                                // 严重程度
	Message     string        `json:"message" gorm:"type:text"`                // 告警消息
	Value       float64       `json:"value"`                                   // 触发时的指标值
	Labels      string        `json:"labels" gorm:"type:jsonb"`                // 标签信息
	
	FiredAt        time.Time  `json:"fired_at"`                      // 触发时间
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`     // 确认时间
	AcknowledgedBy uint       `json:"acknowledged_by,omitempty"`     // 确认人 ID
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`         // 解决时间
	ResolvedBy     uint       `json:"resolved_by,omitempty"`         // 解决人 ID
	
	UserID    uint      `json:"user_id" gorm:"index"`
	TenantID  uint      `json:"tenant_id" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AlertFilters 告警过滤器
type AlertFilters struct {
	Status    AlertStatus
	Severity  AlertSeverity
	RuleID    uint
	StartTime time.Time
	EndTime   time.Time
	UserID    uint
	TenantID  uint
}

// PredictionResult AI 预测结果，基于历史数据预测潜在问题
type PredictionResult struct {
	ResourceType string       `json:"resource_type"` // 资源类型（如 container, database）
	ResourceID   string       `json:"resource_id"`   // 资源 ID
	Predictions  []Prediction `json:"predictions"`   // 预测列表
	Confidence   float64      `json:"confidence"`    // 预测置信度（0-1）
	PredictedAt  time.Time    `json:"predicted_at"`  // 预测时间
}

// Prediction 单个预测项，描述可能发生的问题
type Prediction struct {
	Issue           string        `json:"issue"`            // 问题描述
	Probability     float64       `json:"probability"`      // 发生概率（0-1）
	TimeToIssue     int           `json:"time_to_issue"`    // 预计多少秒后发生
	Severity        AlertSeverity `json:"severity"`         // 严重程度
	Recommendations []string      `json:"recommendations"`  // AI 建议的解决方案
}

// CapacityAnalysis 容量分析结果，用于容量规划和资源优化
type CapacityAnalysis struct {
	ResourceType    string    `json:"resource_type"`    // 资源类型（如 disk, memory, cpu）
	CurrentUsage    float64   `json:"current_usage"`    // 当前使用量
	Capacity        float64   `json:"capacity"`         // 总容量
	UsagePercent    float64   `json:"usage_percent"`    // 使用率百分比
	Trend           string    `json:"trend"`            // 趋势：increasing（增长）, stable（稳定）, decreasing（下降）
	TimeToFull      int       `json:"time_to_full"`     // 预计多少天后满（基于当前趋势）
	Recommendations []string  `json:"recommendations"`  // AI 优化建议
	AnalyzedAt      time.Time `json:"analyzed_at"`      // 分析时间
}

// TableName 指定表名
func (MetricDefinition) TableName() string {
	return "metric_definitions"
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

func (Alert) TableName() string {
	return "alerts"
}
