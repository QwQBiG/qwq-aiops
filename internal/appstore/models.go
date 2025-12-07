package appstore

import (
	"time"

	"gorm.io/gorm"
)

// TemplateType 模板类型
type TemplateType string

const (
	TemplateTypeDockerCompose TemplateType = "docker-compose" // Docker Compose 模板
	TemplateTypeHelmChart     TemplateType = "helm-chart"     // Helm Chart 模板
)

// TemplateStatus 模板状态
type TemplateStatus string

const (
	TemplateStatusDraft     TemplateStatus = "draft"     // 草稿
	TemplateStatusPublished TemplateStatus = "published" // 已发布
	TemplateStatusArchived  TemplateStatus = "archived"  // 已归档
)

// AppCategory 应用分类
type AppCategory string

const (
	CategoryWebServer    AppCategory = "web-server"    // Web 服务器
	CategoryDatabase     AppCategory = "database"      // 数据库
	CategoryDevTools     AppCategory = "dev-tools"     // 开发工具
	CategoryMonitoring   AppCategory = "monitoring"    // 监控工具
	CategoryMessageQueue AppCategory = "message-queue" // 消息队列
	CategoryStorage      AppCategory = "storage"       // 存储服务
	CategoryOther        AppCategory = "other"         // 其他
)

// ParameterType 参数类型
type ParameterType string

const (
	ParamTypeString   ParameterType = "string"   // 字符串
	ParamTypeInt      ParameterType = "int"      // 整数
	ParamTypeBool     ParameterType = "bool"     // 布尔值
	ParamTypeSelect   ParameterType = "select"   // 下拉选择
	ParamTypePassword ParameterType = "password" // 密码
	ParamTypePath     ParameterType = "path"     // 路径
)

// AppTemplate 应用模板
type AppTemplate struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null;index"`                // 模板名称
	DisplayName string         `json:"display_name" gorm:"not null"`              // 显示名称
	Description string         `json:"description" gorm:"type:text"`              // 描述
	Category    AppCategory    `json:"category" gorm:"not null;index"`            // 分类
	Type        TemplateType   `json:"type" gorm:"not null;index"`                // 模板类型
	Version     string         `json:"version" gorm:"not null"`                   // 版本号
	Icon        string         `json:"icon"`                                      // 图标URL
	Author      string         `json:"author"`                                    // 作者
	Status      TemplateStatus `json:"status" gorm:"default:draft;index"`         // 状态
	Tags        string         `json:"tags" gorm:"type:text"`                     // 标签（逗号分隔）
	Content     string         `json:"content" gorm:"type:text;not null"`         // 模板内容（YAML/JSON）
	Parameters  string         `json:"parameters" gorm:"type:text"`               // 参数定义（JSON）
	Dependencies string        `json:"dependencies" gorm:"type:text"`             // 依赖项（JSON）
	MinResources string        `json:"min_resources" gorm:"type:text"`            // 最小资源要求（JSON）
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TemplateParameter 模板参数定义
type TemplateParameter struct {
	Name         string        `json:"name"`                    // 参数名称
	DisplayName  string        `json:"display_name"`            // 显示名称
	Description  string        `json:"description"`             // 描述
	Type         ParameterType `json:"type"`                    // 参数类型
	DefaultValue interface{}   `json:"default_value,omitempty"` // 默认值
	Required     bool          `json:"required"`                // 是否必填
	Options      []string      `json:"options,omitempty"`       // 选项（用于 select 类型）
	Validation   string        `json:"validation,omitempty"`    // 验证规则（正则表达式）
	Placeholder  string        `json:"placeholder,omitempty"`   // 占位符
	Group        string        `json:"group,omitempty"`         // 参数分组
}

// TemplateDependency 模板依赖项
type TemplateDependency struct {
	Name        string `json:"name"`                  // 依赖名称
	Type        string `json:"type"`                  // 依赖类型（service, port, volume）
	Description string `json:"description,omitempty"` // 描述
	Optional    bool   `json:"optional"`              // 是否可选
}

// ResourceRequirements 资源要求
type ResourceRequirements struct {
	MinCPU    string `json:"min_cpu"`    // 最小CPU（如 "0.5"）
	MinMemory string `json:"min_memory"` // 最小内存（如 "512Mi"）
	MinDisk   string `json:"min_disk"`   // 最小磁盘（如 "1Gi"）
}

// ApplicationInstance 应用实例
type ApplicationInstance struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	Name       string         `json:"name" gorm:"not null;index"`           // 实例名称
	TemplateID uint           `json:"template_id" gorm:"not null;index"`    // 模板ID
	Template   *AppTemplate   `json:"template,omitempty" gorm:"foreignKey:TemplateID"`
	Version    string         `json:"version" gorm:"not null"`              // 使用的模板版本
	Status     string         `json:"status" gorm:"not null;index"`         // 状态（running, stopped, error, updating）
	Config     string         `json:"config" gorm:"type:text"`              // 实例配置（JSON）
	UserID     uint           `json:"user_id" gorm:"index"`                 // 用户ID
	TenantID   uint           `json:"tenant_id" gorm:"index"`               // 租户ID
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (AppTemplate) TableName() string {
	return "app_templates"
}

func (ApplicationInstance) TableName() string {
	return "application_instances"
}
