package database

import (
	"context"
	"testing"
	"time"
)



// TestDatabaseInit 测试数据库初始化
func TestDatabaseInit(t *testing.T) {
	t.Skip("跳过需要CGO的SQLite测试")
	
	// 使用内存SQLite数据库进行测试
	cfg := Config{
		Type:     "sqlite",
		FilePath: ":memory:",
		Debug:    false,
	}

	err := Init(cfg)
	if err != nil {
		t.Fatalf("数据库初始化失败: %v", err)
	}

	// 测试自动迁移
	err = AutoMigrate()
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 测试初始化默认数据
	err = InitDefaultData()
	if err != nil {
		t.Fatalf("初始化默认数据失败: %v", err)
	}

	// 验证默认租户是否创建
	var tenant Tenant
	err = DB.Where("code = ?", "default").First(&tenant).Error
	if err != nil {
		t.Fatalf("查询默认租户失败: %v", err)
	}

	if tenant.Name != "默认租户" {
		t.Errorf("默认租户名称不正确: got %s, want 默认租户", tenant.Name)
	}

	// 验证权限是否创建
	var permissions []Permission
	err = DB.Find(&permissions).Error
	if err != nil {
		t.Fatalf("查询权限失败: %v", err)
	}

	if len(permissions) == 0 {
		t.Error("没有创建默认权限")
	}

	// 验证角色权限关联是否创建
	var rolePermissions []RolePermission
	err = DB.Where("role = ?", RoleAdmin).Find(&rolePermissions).Error
	if err != nil {
		t.Fatalf("查询角色权限失败: %v", err)
	}

	if len(rolePermissions) == 0 {
		t.Error("没有为管理员角色分配权限")
	}
}

// TestRBACService 测试RBAC服务
func TestRBACService(t *testing.T) {
	t.Skip("跳过需要CGO的SQLite测试")
	
	// 初始化测试数据库
	cfg := Config{
		Type:     "sqlite",
		FilePath: ":memory:",
		Debug:    false,
	}

	if err := Init(cfg); err != nil {
		t.Fatalf("数据库初始化失败: %v", err)
	}

	if err := AutoMigrate(); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	if err := InitDefaultData(); err != nil {
		t.Fatalf("初始化默认数据失败: %v", err)
	}

	rbacService := NewRBACService(DB)
	ctx := context.Background()

	// 测试创建用户
	t.Run("CreateUser", func(t *testing.T) {
		user := &User{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "hashedpassword",
			Role:     RoleUser,
		}

		err := rbacService.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("创建用户失败: %v", err)
		}

		if user.ID == 0 {
			t.Error("用户ID未设置")
		}

		if user.TenantID == 0 {
			t.Error("用户租户ID未设置")
		}
	})

	// 测试权限检查
	t.Run("CheckPermission", func(t *testing.T) {
		// 创建管理员用户
		adminUser := &User{
			Username: "admin",
			Email:    "admin@example.com",
			Password: "hashedpassword",
			Role:     RoleAdmin,
		}
		err := rbacService.CreateUser(ctx, adminUser)
		if err != nil {
			t.Fatalf("创建管理员用户失败: %v", err)
		}

		// 管理员应该有所有权限
		hasPermission, err := rbacService.CheckPermission(ctx, adminUser.ID, "container", "read")
		if err != nil {
			t.Fatalf("检查权限失败: %v", err)
		}

		if !hasPermission {
			t.Error("管理员应该有容器读权限")
		}

		// 创建普通用户
		normalUser := &User{
			Username: "normaluser",
			Email:    "normal@example.com",
			Password: "hashedpassword",
			Role:     RoleUser,
		}
		err = rbacService.CreateUser(ctx, normalUser)
		if err != nil {
			t.Fatalf("创建普通用户失败: %v", err)
		}

		// 普通用户应该有读权限
		hasPermission, err = rbacService.CheckPermission(ctx, normalUser.ID, "container", "read")
		if err != nil {
			t.Fatalf("检查权限失败: %v", err)
		}

		if !hasPermission {
			t.Error("普通用户应该有容器读权限")
		}

		// 普通用户不应该有删除权限
		hasPermission, err = rbacService.CheckPermission(ctx, normalUser.ID, "container", "delete")
		if err != nil {
			t.Fatalf("检查权限失败: %v", err)
		}

		if hasPermission {
			t.Error("普通用户不应该有容器删除权限")
		}
	})

	// 测试租户访问控制
	t.Run("CheckTenantAccess", func(t *testing.T) {
		// 创建新租户
		tenant := &Tenant{
			Name:    "测试租户",
			Code:    "test-tenant",
			Enabled: true,
		}
		err := rbacService.CreateTenant(ctx, tenant)
		if err != nil {
			t.Fatalf("创建租户失败: %v", err)
		}

		// 创建属于该租户的用户
		user := &User{
			Username: "tenantuser",
			Email:    "tenant@example.com",
			Password: "hashedpassword",
			Role:     RoleUser,
			TenantID: tenant.ID,
		}
		err = rbacService.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("创建用户失败: %v", err)
		}

		// 用户应该可以访问自己的租户
		hasAccess, err := rbacService.CheckTenantAccess(ctx, user.ID, tenant.ID)
		if err != nil {
			t.Fatalf("检查租户访问失败: %v", err)
		}

		if !hasAccess {
			t.Error("用户应该可以访问自己的租户")
		}

		// 用户不应该可以访问其他租户
		var defaultTenant Tenant
		DB.Where("code = ?", "default").First(&defaultTenant)

		hasAccess, err = rbacService.CheckTenantAccess(ctx, user.ID, defaultTenant.ID)
		if err != nil {
			t.Fatalf("检查租户访问失败: %v", err)
		}

		if hasAccess {
			t.Error("用户不应该可以访问其他租户")
		}
	})

	// 测试获取用户权限
	t.Run("GetUserPermissions", func(t *testing.T) {
		var user User
		DB.Where("username = ?", "testuser").First(&user)

		permissions, err := rbacService.GetUserPermissions(ctx, user.ID)
		if err != nil {
			t.Fatalf("获取用户权限失败: %v", err)
		}

		if len(permissions) == 0 {
			t.Error("用户应该有权限")
		}

		// 验证所有权限都是读权限
		for _, perm := range permissions {
			if perm.Action != "read" {
				t.Errorf("普通用户应该只有读权限，但发现: %s", perm.Action)
			}
		}
	})
}

// TestAuditService 测试审计服务
func TestAuditService(t *testing.T) {
	t.Skip("跳过需要CGO的SQLite测试")
	
	// 初始化测试数据库
	cfg := Config{
		Type:     "sqlite",
		FilePath: ":memory:",
		Debug:    false,
	}

	if err := Init(cfg); err != nil {
		t.Fatalf("数据库初始化失败: %v", err)
	}

	if err := AutoMigrate(); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	if err := InitDefaultData(); err != nil {
		t.Fatalf("初始化默认数据失败: %v", err)
	}

	auditService := NewAuditService(DB)
	rbacService := NewRBACService(DB)
	ctx := context.Background()

	// 创建测试用户
	user := &User{
		Username: "audituser",
		Email:    "audit@example.com",
		Password: "hashedpassword",
		Role:     RoleUser,
	}
	err := rbacService.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	// 测试记录成功操作
	t.Run("LogSuccess", func(t *testing.T) {
		err := auditService.LogSuccess(
			ctx,
			user.ID,
			user.TenantID,
			"read",
			"container",
			"container-123",
			map[string]string{"name": "test-container"},
			"192.168.1.1",
			"Mozilla/5.0",
		)

		if err != nil {
			t.Fatalf("记录成功操作失败: %v", err)
		}
	})

	// 测试记录失败操作
	t.Run("LogFailure", func(t *testing.T) {
		err := auditService.LogFailure(
			ctx,
			user.ID,
			user.TenantID,
			"delete",
			"container",
			"container-456",
			map[string]string{"name": "test-container"},
			"权限不足",
			"192.168.1.1",
			"Mozilla/5.0",
		)

		if err != nil {
			t.Fatalf("记录失败操作失败: %v", err)
		}
	})

	// 测试查询日志
	t.Run("QueryLogs", func(t *testing.T) {
		filter := AuditLogFilter{
			UserID:   user.ID,
			Page:     1,
			PageSize: 10,
		}

		logs, total, err := auditService.QueryLogs(ctx, filter)
		if err != nil {
			t.Fatalf("查询日志失败: %v", err)
		}

		if total != 2 {
			t.Errorf("日志总数不正确: got %d, want 2", total)
		}

		if len(logs) != 2 {
			t.Errorf("返回的日志数量不正确: got %d, want 2", len(logs))
		}
	})

	// 测试获取最近日志
	t.Run("GetRecentLogs", func(t *testing.T) {
		logs, err := auditService.GetRecentLogs(ctx, 10)
		if err != nil {
			t.Fatalf("获取最近日志失败: %v", err)
		}

		if len(logs) == 0 {
			t.Error("应该有日志记录")
		}
	})

	// 测试删除旧日志
	t.Run("DeleteOldLogs", func(t *testing.T) {
		// 删除未来的日志（应该删除0条）
		futureDate := time.Now().Add(24 * time.Hour)
		count, err := auditService.DeleteOldLogs(ctx, futureDate)
		if err != nil {
			t.Fatalf("删除旧日志失败: %v", err)
		}

		if count != 2 {
			t.Errorf("删除的日志数量不正确: got %d, want 2", count)
		}
	})
}
