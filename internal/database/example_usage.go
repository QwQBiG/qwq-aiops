package database

import (
	"context"
	"fmt"
	"log"
)

// ExampleUsage 演示如何使用数据库和RBAC系统
func ExampleUsage() {
	// 1. 初始化数据库
	cfg := Config{
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "your_password",
		DBName:   "qwq",
		SSLMode:  "disable",
		Debug:    true,
	}

	if err := Init(cfg); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer Close()

	// 2. 自动迁移数据库表结构
	if err := AutoMigrate(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 3. 初始化默认数据（租户、权限、角色权限）
	if err := InitDefaultData(); err != nil {
		log.Fatalf("初始化默认数据失败: %v", err)
	}

	// 4. 创建服务实例
	rbacService := NewRBACService(DB)
	auditService := NewAuditService(DB)
	ctx := context.Background()

	// 5. 创建租户
	tenant := &Tenant{
		Name:    "示例公司",
		Code:    "example-corp",
		Enabled: true,
	}
	if err := rbacService.CreateTenant(ctx, tenant); err != nil {
		log.Printf("创建租户失败: %v", err)
	} else {
		log.Printf("✅ 创建租户成功: %s (ID: %d)", tenant.Name, tenant.ID)
	}

	// 6. 创建用户
	adminUser := &User{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "hashed_password_here", // 实际应用中应该使用bcrypt等加密
		Role:     RoleAdmin,
		TenantID: tenant.ID,
		Enabled:  true,
	}
	if err := rbacService.CreateUser(ctx, adminUser); err != nil {
		log.Printf("创建管理员用户失败: %v", err)
	} else {
		log.Printf("✅ 创建管理员用户成功: %s (ID: %d)", adminUser.Username, adminUser.ID)
	}

	normalUser := &User{
		Username: "user1",
		Email:    "user1@example.com",
		Password: "hashed_password_here",
		Role:     RoleUser,
		TenantID: tenant.ID,
		Enabled:  true,
	}
	if err := rbacService.CreateUser(ctx, normalUser); err != nil {
		log.Printf("创建普通用户失败: %v", err)
	} else {
		log.Printf("✅ 创建普通用户成功: %s (ID: %d)", normalUser.Username, normalUser.ID)
	}

	// 7. 检查权限
	hasPermission, err := rbacService.CheckPermission(ctx, adminUser.ID, "container", "write")
	if err != nil {
		log.Printf("权限检查失败: %v", err)
	} else {
		log.Printf("✅ 管理员用户 container:write 权限: %v", hasPermission)
	}

	hasPermission, err = rbacService.CheckPermission(ctx, normalUser.ID, "container", "write")
	if err != nil {
		log.Printf("权限检查失败: %v", err)
	} else {
		log.Printf("✅ 普通用户 container:write 权限: %v", hasPermission)
	}

	hasPermission, err = rbacService.CheckPermission(ctx, normalUser.ID, "container", "read")
	if err != nil {
		log.Printf("权限检查失败: %v", err)
	} else {
		log.Printf("✅ 普通用户 container:read 权限: %v", hasPermission)
	}

	// 8. 检查租户访问权限
	hasAccess, err := rbacService.CheckTenantAccess(ctx, normalUser.ID, tenant.ID)
	if err != nil {
		log.Printf("租户访问检查失败: %v", err)
	} else {
		log.Printf("✅ 用户访问自己租户的权限: %v", hasAccess)
	}

	// 9. 获取用户的所有权限
	permissions, err := rbacService.GetUserPermissions(ctx, normalUser.ID)
	if err != nil {
		log.Printf("获取用户权限失败: %v", err)
	} else {
		log.Printf("✅ 普通用户拥有 %d 个权限:", len(permissions))
		for _, perm := range permissions {
			log.Printf("   - %s (%s)", perm.Name, perm.Description)
		}
	}

	// 10. 记录审计日志
	err = auditService.LogSuccess(
		ctx,
		normalUser.ID,
		tenant.ID,
		"read",
		"container",
		"container-123",
		map[string]string{
			"container_name": "nginx",
			"action":         "list",
		},
		"192.168.1.100",
		"Mozilla/5.0",
	)
	if err != nil {
		log.Printf("记录审计日志失败: %v", err)
	} else {
		log.Printf("✅ 成功记录审计日志")
	}

	// 11. 查询审计日志
	logs, total, err := auditService.GetLogsByUser(ctx, normalUser.ID, 1, 10)
	if err != nil {
		log.Printf("查询审计日志失败: %v", err)
	} else {
		log.Printf("✅ 查询到 %d 条审计日志 (总共 %d 条):", len(logs), total)
		for _, log := range logs {
			fmt.Printf("   - [%s] %s %s:%s (状态: %s)\n",
				log.CreatedAt.Format("2006-01-02 15:04:05"),
				log.Action,
				log.Resource,
				log.ResourceID,
				log.Status,
			)
		}
	}

	// 12. 列出所有用户
	users, total, err := rbacService.ListUsers(ctx, tenant.ID, 1, 10)
	if err != nil {
		log.Printf("列出用户失败: %v", err)
	} else {
		log.Printf("✅ 租户 %s 有 %d 个用户:", tenant.Name, total)
		for _, user := range users {
			fmt.Printf("   - %s (%s) - 角色: %s\n", user.Username, user.Email, user.Role)
		}
	}

	log.Println("\n✅ 数据库和RBAC系统演示完成！")
}
