package database

import (
	"testing"
	"time"
)

// TestUserRole 测试用户角色常量
func TestUserRole(t *testing.T) {
	tests := []struct {
		role     UserRole
		expected string
	}{
		{RoleAdmin, "admin"},
		{RoleUser, "user"},
		{RoleReadOnly, "readonly"},
	}

	for _, tt := range tests {
		if string(tt.role) != tt.expected {
			t.Errorf("角色值不正确: got %s, want %s", tt.role, tt.expected)
		}
	}
}

// TestUserModel 测试用户模型
func TestUserModel(t *testing.T) {
	user := User{
		ID:        1,
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "hashedpassword",
		Role:      RoleUser,
		TenantID:  1,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if user.Username != "testuser" {
		t.Errorf("用户名不正确: got %s, want testuser", user.Username)
	}

	if user.Role != RoleUser {
		t.Errorf("用户角色不正确: got %s, want %s", user.Role, RoleUser)
	}

	if !user.Enabled {
		t.Error("用户应该是启用状态")
	}

	// 测试表名
	tableName := user.TableName()
	if tableName != "users" {
		t.Errorf("表名不正确: got %s, want users", tableName)
	}
}

// TestTenantModel 测试租户模型
func TestTenantModel(t *testing.T) {
	tenant := Tenant{
		ID:        1,
		Name:      "测试租户",
		Code:      "test-tenant",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if tenant.Name != "测试租户" {
		t.Errorf("租户名称不正确: got %s, want 测试租户", tenant.Name)
	}

	if tenant.Code != "test-tenant" {
		t.Errorf("租户代码不正确: got %s, want test-tenant", tenant.Code)
	}

	// 测试表名
	tableName := tenant.TableName()
	if tableName != "tenants" {
		t.Errorf("表名不正确: got %s, want tenants", tableName)
	}
}

// TestPermissionModel 测试权限模型
func TestPermissionModel(t *testing.T) {
	permission := Permission{
		ID:          1,
		Name:        "container:read",
		Resource:    "container",
		Action:      "read",
		Description: "查看容器",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if permission.Name != "container:read" {
		t.Errorf("权限名称不正确: got %s, want container:read", permission.Name)
	}

	if permission.Resource != "container" {
		t.Errorf("资源类型不正确: got %s, want container", permission.Resource)
	}

	if permission.Action != "read" {
		t.Errorf("操作类型不正确: got %s, want read", permission.Action)
	}

	// 测试表名
	tableName := permission.TableName()
	if tableName != "permissions" {
		t.Errorf("表名不正确: got %s, want permissions", tableName)
	}
}

// TestRolePermissionModel 测试角色权限模型
func TestRolePermissionModel(t *testing.T) {
	rolePermission := RolePermission{
		ID:           1,
		Role:         RoleAdmin,
		PermissionID: 1,
		CreatedAt:    time.Now(),
	}

	if rolePermission.Role != RoleAdmin {
		t.Errorf("角色不正确: got %s, want %s", rolePermission.Role, RoleAdmin)
	}

	if rolePermission.PermissionID != 1 {
		t.Errorf("权限ID不正确: got %d, want 1", rolePermission.PermissionID)
	}

	// 测试表名
	tableName := rolePermission.TableName()
	if tableName != "role_permissions" {
		t.Errorf("表名不正确: got %s, want role_permissions", tableName)
	}
}

// TestAuditLogModel 测试审计日志模型
func TestAuditLogModel(t *testing.T) {
	auditLog := AuditLog{
		ID:         1,
		UserID:     1,
		TenantID:   1,
		Action:     "read",
		Resource:   "container",
		ResourceID: "container-123",
		Details:    `{"name": "test"}`,
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Status:     "success",
		CreatedAt:  time.Now(),
	}

	if auditLog.Action != "read" {
		t.Errorf("操作类型不正确: got %s, want read", auditLog.Action)
	}

	if auditLog.Resource != "container" {
		t.Errorf("资源类型不正确: got %s, want container", auditLog.Resource)
	}

	if auditLog.Status != "success" {
		t.Errorf("状态不正确: got %s, want success", auditLog.Status)
	}

	// 测试表名
	tableName := auditLog.TableName()
	if tableName != "audit_logs" {
		t.Errorf("表名不正确: got %s, want audit_logs", tableName)
	}
}

// TestDatabaseConfig 测试数据库配置
func TestDatabaseConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "SQLite配置",
			config: Config{
				Type:     "sqlite",
				FilePath: "./test.db",
				Debug:    false,
			},
		},
		{
			name: "PostgreSQL配置",
			config: Config{
				Type:     "postgres",
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "password",
				DBName:   "testdb",
				SSLMode:  "disable",
				Debug:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Type == "" {
				t.Error("数据库类型不能为空")
			}

			if tt.config.Type == "sqlite" && tt.config.FilePath == "" {
				t.Error("SQLite文件路径不能为空")
			}

			if tt.config.Type == "postgres" {
				if tt.config.Host == "" {
					t.Error("PostgreSQL主机不能为空")
				}
				if tt.config.Port == 0 {
					t.Error("PostgreSQL端口不能为0")
				}
			}
		})
	}
}
