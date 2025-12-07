package container

import (
	"time"

	"gorm.io/gorm"
)

// ComposeProject Docker Compose 项目
type ComposeProject struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null;uniqueIndex:idx_name_tenant"` // 项目名称
	DisplayName string         `json:"display_name"`                                      // 显示名称
	Description string         `json:"description" gorm:"type:text"`                      // 描述
	Content     string         `json:"content" gorm:"type:text;not null"`                 // Compose 文件内容（YAML）
	Version     string         `json:"version"`                                           // Compose 文件版本
	Status      ProjectStatus  `json:"status" gorm:"not null;index"`                      // 项目状态
	UserID      uint           `json:"user_id" gorm:"index"`                              // 用户ID
	TenantID    uint           `json:"tenant_id" gorm:"index;uniqueIndex:idx_name_tenant"` // 租户ID
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// ProjectStatus 项目状态
type ProjectStatus string

const (
	ProjectStatusDraft   ProjectStatus = "draft"   // 草稿
	ProjectStatusRunning ProjectStatus = "running" // 运行中
	ProjectStatusStopped ProjectStatus = "stopped" // 已停止
	ProjectStatusError   ProjectStatus = "error"   // 错误
	ProjectStatusUpdating ProjectStatus = "updating" // 更新中
)

// ComposeConfig Docker Compose 配置结构（简化版）
type ComposeConfig struct {
	Version  string                 `yaml:"version" json:"version"`   // Compose 文件版本
	Services map[string]*Service    `yaml:"services" json:"services"` // 服务定义
	Networks map[string]*Network    `yaml:"networks,omitempty" json:"networks,omitempty"` // 网络定义
	Volumes  map[string]*Volume     `yaml:"volumes,omitempty" json:"volumes,omitempty"`   // 卷定义
	Secrets  map[string]*Secret     `yaml:"secrets,omitempty" json:"secrets,omitempty"`   // 密钥定义
	Configs  map[string]*ConfigItem `yaml:"configs,omitempty" json:"configs,omitempty"`   // 配置定义
}

// Service 服务定义
type Service struct {
	Image         string              `yaml:"image,omitempty" json:"image,omitempty"`                   // 镜像
	Build         *BuildConfig        `yaml:"build,omitempty" json:"build,omitempty"`                   // 构建配置
	ContainerName string              `yaml:"container_name,omitempty" json:"container_name,omitempty"` // 容器名称
	Command       interface{}         `yaml:"command,omitempty" json:"command,omitempty"`               // 命令（字符串或数组）
	Entrypoint    interface{}         `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`         // 入口点
	Environment   interface{}         `yaml:"environment,omitempty" json:"environment,omitempty"`       // 环境变量（数组或映射）
	Ports         []string            `yaml:"ports,omitempty" json:"ports,omitempty"`                   // 端口映射
	Volumes       []string            `yaml:"volumes,omitempty" json:"volumes,omitempty"`               // 卷挂载
	Networks      interface{}         `yaml:"networks,omitempty" json:"networks,omitempty"`             // 网络（数组或映射）
	DependsOn     interface{}         `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`         // 依赖服务
	Restart       string              `yaml:"restart,omitempty" json:"restart,omitempty"`               // 重启策略
	HealthCheck   *HealthCheck        `yaml:"healthcheck,omitempty" json:"healthcheck,omitempty"`       // 健康检查
	Deploy        *DeployConfig       `yaml:"deploy,omitempty" json:"deploy,omitempty"`                 // 部署配置
	Labels        map[string]string   `yaml:"labels,omitempty" json:"labels,omitempty"`                 // 标签
	Logging       *LoggingConfig      `yaml:"logging,omitempty" json:"logging,omitempty"`               // 日志配置
	ExtraHosts    []string            `yaml:"extra_hosts,omitempty" json:"extra_hosts,omitempty"`       // 额外主机
	DNS           interface{}         `yaml:"dns,omitempty" json:"dns,omitempty"`                       // DNS服务器
	Privileged    bool                `yaml:"privileged,omitempty" json:"privileged,omitempty"`         // 特权模式
	User          string              `yaml:"user,omitempty" json:"user,omitempty"`                     // 用户
	WorkingDir    string              `yaml:"working_dir,omitempty" json:"working_dir,omitempty"`       // 工作目录
}

// BuildConfig 构建配置
type BuildConfig struct {
	Context    string            `yaml:"context,omitempty" json:"context,omitempty"`       // 构建上下文
	Dockerfile string            `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"` // Dockerfile 路径
	Args       map[string]string `yaml:"args,omitempty" json:"args,omitempty"`             // 构建参数
	Target     string            `yaml:"target,omitempty" json:"target,omitempty"`         // 构建目标
}

// HealthCheck 健康检查配置
type HealthCheck struct {
	Test        interface{} `yaml:"test,omitempty" json:"test,omitempty"`               // 测试命令
	Interval    string      `yaml:"interval,omitempty" json:"interval,omitempty"`       // 检查间隔
	Timeout     string      `yaml:"timeout,omitempty" json:"timeout,omitempty"`         // 超时时间
	Retries     int         `yaml:"retries,omitempty" json:"retries,omitempty"`         // 重试次数
	StartPeriod string      `yaml:"start_period,omitempty" json:"start_period,omitempty"` // 启动等待时间
}

// DeployConfig 部署配置
type DeployConfig struct {
	Replicas  int                    `yaml:"replicas,omitempty" json:"replicas,omitempty"`   // 副本数
	Resources *ResourcesConfig       `yaml:"resources,omitempty" json:"resources,omitempty"` // 资源限制
	RestartPolicy *RestartPolicy     `yaml:"restart_policy,omitempty" json:"restart_policy,omitempty"` // 重启策略
	Placement *PlacementConfig       `yaml:"placement,omitempty" json:"placement,omitempty"` // 放置约束
}

// ResourcesConfig 资源配置
type ResourcesConfig struct {
	Limits       *ResourceLimit `yaml:"limits,omitempty" json:"limits,omitempty"`             // 资源限制
	Reservations *ResourceLimit `yaml:"reservations,omitempty" json:"reservations,omitempty"` // 资源预留
}

// ResourceLimit 资源限制
type ResourceLimit struct {
	CPUs   string `yaml:"cpus,omitempty" json:"cpus,omitempty"`     // CPU限制
	Memory string `yaml:"memory,omitempty" json:"memory,omitempty"` // 内存限制
}

// RestartPolicy 重启策略
type RestartPolicy struct {
	Condition   string `yaml:"condition,omitempty" json:"condition,omitempty"`       // 重启条件
	Delay       string `yaml:"delay,omitempty" json:"delay,omitempty"`               // 延迟时间
	MaxAttempts int    `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty"` // 最大尝试次数
	Window      string `yaml:"window,omitempty" json:"window,omitempty"`             // 时间窗口
}

// PlacementConfig 放置配置
type PlacementConfig struct {
	Constraints []string `yaml:"constraints,omitempty" json:"constraints,omitempty"` // 约束条件
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Driver  string            `yaml:"driver,omitempty" json:"driver,omitempty"`   // 日志驱动
	Options map[string]string `yaml:"options,omitempty" json:"options,omitempty"` // 日志选项
}

// Network 网络定义
type Network struct {
	Driver     string            `yaml:"driver,omitempty" json:"driver,omitempty"`         // 网络驱动
	DriverOpts map[string]string `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty"` // 驱动选项
	External   bool              `yaml:"external,omitempty" json:"external,omitempty"`     // 是否外部网络
	Name       string            `yaml:"name,omitempty" json:"name,omitempty"`             // 网络名称
	Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`         // 标签
}

// Volume 卷定义
type Volume struct {
	Driver     string            `yaml:"driver,omitempty" json:"driver,omitempty"`         // 卷驱动
	DriverOpts map[string]string `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty"` // 驱动选项
	External   bool              `yaml:"external,omitempty" json:"external,omitempty"`     // 是否外部卷
	Name       string            `yaml:"name,omitempty" json:"name,omitempty"`             // 卷名称
	Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`         // 标签
}

// Secret 密钥定义
type Secret struct {
	File     string            `yaml:"file,omitempty" json:"file,omitempty"`         // 文件路径
	External bool              `yaml:"external,omitempty" json:"external,omitempty"` // 是否外部密钥
	Name     string            `yaml:"name,omitempty" json:"name,omitempty"`         // 密钥名称
	Labels   map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`     // 标签
}

// ConfigItem 配置定义
type ConfigItem struct {
	File     string            `yaml:"file,omitempty" json:"file,omitempty"`         // 文件路径
	External bool              `yaml:"external,omitempty" json:"external,omitempty"` // 是否外部配置
	Name     string            `yaml:"name,omitempty" json:"name,omitempty"`         // 配置名称
	Labels   map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`     // 标签
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string `json:"field"`   // 字段名
	Message string `json:"message"` // 错误消息
	Line    int    `json:"line"`    // 行号（如果适用）
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid  bool               `json:"valid"`  // 是否有效
	Errors []*ValidationError `json:"errors"` // 错误列表
}

// CompletionItem 自动补全项
type CompletionItem struct {
	Label         string `json:"label"`          // 标签
	Kind          string `json:"kind"`           // 类型（keyword, property, value）
	Detail        string `json:"detail"`         // 详细信息
	Documentation string `json:"documentation"`  // 文档说明
	InsertText    string `json:"insert_text"`    // 插入文本
}

// Deployment 部署记录
type Deployment struct {
	ID              uint             `json:"id" gorm:"primaryKey"`
	ProjectID       uint             `json:"project_id" gorm:"not null;index"`                // 项目ID
	Project         *ComposeProject  `json:"project,omitempty" gorm:"foreignKey:ProjectID"`   // 关联项目
	Version         string           `json:"version" gorm:"not null"`                         // 部署版本
	Strategy        DeployStrategy   `json:"strategy" gorm:"not null"`                        // 部署策略
	Status          DeploymentStatus `json:"status" gorm:"not null;index"`                    // 部署状态
	Progress        int              `json:"progress" gorm:"default:0"`                       // 部署进度（0-100）
	Message         string           `json:"message" gorm:"type:text"`                        // 状态消息
	StartedAt       *time.Time       `json:"started_at"`                                      // 开始时间
	CompletedAt     *time.Time       `json:"completed_at"`                                    // 完成时间
	RollbackVersion string           `json:"rollback_version"`                                // 回滚版本
	UserID          uint             `json:"user_id" gorm:"index"`                            // 用户ID
	TenantID        uint             `json:"tenant_id" gorm:"index"`                          // 租户ID
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	DeletedAt       gorm.DeletedAt   `json:"-" gorm:"index"`
}

// DeployStrategy 部署策略
type DeployStrategy string

const (
	DeployStrategyRecreate      DeployStrategy = "recreate"       // 重建策略（停止所有，然后启动新的）
	DeployStrategyRollingUpdate DeployStrategy = "rolling_update" // 滚动更新
	DeployStrategyBlueGreen     DeployStrategy = "blue_green"     // 蓝绿部署
)

// DeploymentStatus 部署状态
type DeploymentStatus string

const (
	DeploymentStatusPending    DeploymentStatus = "pending"     // 等待中
	DeploymentStatusInProgress DeploymentStatus = "in_progress" // 进行中
	DeploymentStatusCompleted  DeploymentStatus = "completed"   // 已完成
	DeploymentStatusFailed     DeploymentStatus = "failed"      // 失败
	DeploymentStatusRollingBack DeploymentStatus = "rolling_back" // 回滚中
	DeploymentStatusRolledBack DeploymentStatus = "rolled_back" // 已回滚
)

// ServiceInstance 服务实例（运行中的容器）
type ServiceInstance struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	DeploymentID  uint           `json:"deployment_id" gorm:"not null;index"`              // 部署ID
	Deployment    *Deployment    `json:"deployment,omitempty" gorm:"foreignKey:DeploymentID"` // 关联部署
	ServiceName   string         `json:"service_name" gorm:"not null;index"`               // 服务名称
	ContainerID   string         `json:"container_id" gorm:"uniqueIndex"`                  // 容器ID
	ContainerName string         `json:"container_name"`                                   // 容器名称
	Image         string         `json:"image"`                                            // 镜像
	Status        string         `json:"status" gorm:"index"`                              // 容器状态
	Health        string         `json:"health"`                                           // 健康状态
	StartedAt     *time.Time     `json:"started_at"`                                       // 启动时间
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// DeploymentEvent 部署事件
type DeploymentEvent struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	DeploymentID uint           `json:"deployment_id" gorm:"not null;index"`            // 部署ID
	Deployment   *Deployment    `json:"deployment,omitempty" gorm:"foreignKey:DeploymentID"` // 关联部署
	EventType    string         `json:"event_type" gorm:"not null"`                     // 事件类型
	ServiceName  string         `json:"service_name"`                                   // 服务名称
	Message      string         `json:"message" gorm:"type:text"`                       // 事件消息
	Details      string         `json:"details" gorm:"type:text"`                       // 详细信息（JSON）
	CreatedAt    time.Time      `json:"created_at"`
}

// DeploymentConfig 部署配置
type DeploymentConfig struct {
	Strategy         DeployStrategy `json:"strategy"`                    // 部署策略
	MaxSurge         int            `json:"max_surge"`                   // 滚动更新时最多超出的实例数
	MaxUnavailable   int            `json:"max_unavailable"`             // 滚动更新时最多不可用的实例数
	HealthCheckDelay int            `json:"health_check_delay"`          // 健康检查延迟（秒）
	HealthCheckRetries int          `json:"health_check_retries"`        // 健康检查重试次数
	RollbackOnFailure bool          `json:"rollback_on_failure"`         // 失败时自动回滚
	BlueGreenTimeout  int            `json:"blue_green_timeout"`          // 蓝绿部署切换超时（秒）
}

// TableName 指定表名
func (ComposeProject) TableName() string {
	return "compose_projects"
}

// TableName 指定表名
func (Deployment) TableName() string {
	return "deployments"
}

// TableName 指定表名
func (ServiceInstance) TableName() string {
	return "service_instances"
}

// TableName 指定表名
func (DeploymentEvent) TableName() string {
	return "deployment_events"
}
