package database

import (
	"time"

	"gorm.io/gorm"
)

// UserRole 用户角色类型
type UserRole string

const (
	RoleAdmin    UserRole = "admin"    // 管理员
	RoleUser     UserRole = "user"     // 普通用户
	RoleReadOnly UserRole = "readonly" // 只读用户
)

// User 用户模型
type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Username  string         `json:"username" gorm:"unique;not null;index"`
	Email     string         `json:"email" gorm:"unique;not null;index"`
	Password  string         `json:"-" gorm:"not null"` // 密码不返回给前端
	Role      UserRole       `json:"role" gorm:"default:user;index"`
	TenantID  uint           `json:"tenant_id" gorm:"index"`
	Tenant    *Tenant        `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	Enabled   bool           `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Tenant 租户模型
type Tenant struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"unique;not null;index"`
	Code      string         `json:"code" gorm:"unique;not null;index"` // 租户代码
	Enabled   bool           `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Permission 权限模型
type Permission struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"unique;not null;index"` // 权限名称，如 "container:read"
	Resource    string         `json:"resource" gorm:"not null;index"`    // 资源类型，如 "container"
	Action      string         `json:"action" gorm:"not null;index"`      // 操作类型，如 "read", "write", "delete"
	Description string         `json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// RolePermission 角色权限关联表
type RolePermission struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Role         UserRole       `json:"role" gorm:"not null;index"`
	PermissionID uint           `json:"permission_id" gorm:"not null;index"`
	Permission   *Permission    `json:"permission,omitempty" gorm:"foreignKey:PermissionID"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// AuditLog 审计日志模型
type AuditLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"index"`
	User       *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	TenantID   uint      `json:"tenant_id" gorm:"index"`
	Action     string    `json:"action" gorm:"not null;index"`      // 操作类型
	Resource   string    `json:"resource" gorm:"not null;index"`    // 资源类型
	ResourceID string    `json:"resource_id" gorm:"index"`          // 资源ID
	Details    string    `json:"details" gorm:"type:text"`          // 详细信息（JSON格式）
	IPAddress  string    `json:"ip_address"`                        // 操作IP
	UserAgent  string    `json:"user_agent"`                        // 用户代理
	Status     string    `json:"status" gorm:"index"`               // 操作状态：success, failed
	ErrorMsg   string    `json:"error_msg,omitempty" gorm:"type:text"` // 错误信息
	CreatedAt  time.Time `json:"created_at" gorm:"index"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

func (Tenant) TableName() string {
	return "tenants"
}

func (Permission) TableName() string {
	return "permissions"
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
