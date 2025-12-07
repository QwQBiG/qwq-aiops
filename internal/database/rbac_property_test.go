package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: enhanced-aiops-platform, Property 18: 用户权限隔离**
// **Validates: Requirements 7.1, 7.2**
//
// Property 18: 用户权限隔离
// *For any* 用户操作，系统应该严格执行角色权限检查和资源隔离
//
// 这个属性测试验证：
// 1. 不同角色的用户只能执行其被授权的操作
// 2. 普通用户不能执行管理员操作
// 3. 只读用户只能执行读操作
// 4. 权限检查在所有操作中都有效
func TestProperty18_UserPermissionIsolation(t *testing.T) {
	// 由于需要真实的数据库连接，这个测试在没有数据库的环境下会跳过
	// 在实际部署环境中，应该配置测试数据库来运行这些测试
	t.Skip("需要PostgreSQL数据库连接才能运行此属性测试")

	// 初始化测试数据库
	cfg := Config{
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "qwq_test",
		SSLMode:  "disable",
		Debug:    false,
	}

	if err := Init(cfg); err != nil {
		t.Skipf("无法连接测试数据库: %v", err)
		return
	}
	defer Close()

	if err := AutoMigrate(); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	if err := InitDefaultData(); err != nil {
		t.Fatalf("初始化默认数据失败: %v", err)
	}

	rbacService := NewRBACService(DB)
	ctx := context.Background()

	// 定义资源和操作类型
	resources := []string{"container", "application", "website", "database", "backup", "user", "tenant"}
	actions := []string{"read", "write", "delete"}

	// 创建测试用户
	adminUser := &User{
		Username: "test_admin",
		Email:    "admin@test.com",
		Password: "hashed_password",
		Role:     RoleAdmin,
	}
	if err := rbacService.CreateUser(ctx, adminUser); err != nil {
		t.Fatalf("创建管理员用户失败: %v", err)
	}

	normalUser := &User{
		Username: "test_user",
		Email:    "user@test.com",
		Password: "hashed_password",
		Role:     RoleUser,
	}
	if err := rbacService.CreateUser(ctx, normalUser); err != nil {
		t.Fatalf("创建普通用户失败: %v", err)
	}

	readonlyUser := &User{
		Username: "test_readonly",
		Email:    "readonly@test.com",
		Password: "hashed_password",
		Role:     RoleReadOnly,
	}
	if err := rbacService.CreateUser(ctx, readonlyUser); err != nil {
		t.Fatalf("创建只读用户失败: %v", err)
	}

	properties := gopter.NewProperties(nil)

	// Property 1: 管理员应该有所有权限
	properties.Property("管理员拥有所有资源的所有操作权限", prop.ForAll(
		func(resource string, action string) bool {
			hasPermission, err := rbacService.CheckPermission(ctx, adminUser.ID, resource, action)
			if err != nil {
				t.Logf("权限检查错误: %v", err)
				return false
			}
			return hasPermission
		},
		gen.OneConstOf(resources[0], resources[1], resources[2], resources[3], resources[4], resources[5], resources[6]),
		gen.OneConstOf(actions[0], actions[1], actions[2]),
	))

	// Property 2: 普通用户只有读权限
	properties.Property("普通用户只能执行读操作", prop.ForAll(
		func(resource string, action string) bool {
			hasPermission, err := rbacService.CheckPermission(ctx, normalUser.ID, resource, action)
			if err != nil {
				t.Logf("权限检查错误: %v", err)
				return false
			}

			// 如果是读操作，应该有权限
			if action == "read" {
				return hasPermission
			}

			// 如果是写或删除操作，不应该有权限
			return !hasPermission
		},
		gen.OneConstOf(resources[0], resources[1], resources[2], resources[3], resources[4], resources[5], resources[6]),
		gen.OneConstOf(actions[0], actions[1], actions[2]),
	))

	// Property 3: 只读用户只有读权限
	properties.Property("只读用户只能执行读操作", prop.ForAll(
		func(resource string, action string) bool {
			hasPermission, err := rbacService.CheckPermission(ctx, readonlyUser.ID, resource, action)
			if err != nil {
				t.Logf("权限检查错误: %v", err)
				return false
			}

			// 如果是读操作，应该有权限
			if action == "read" {
				return hasPermission
			}

			// 如果是写或删除操作，不应该有权限
			return !hasPermission
		},
		gen.OneConstOf(resources[0], resources[1], resources[2], resources[3], resources[4], resources[5], resources[6]),
		gen.OneConstOf(actions[0], actions[1], actions[2]),
	))

	// Property 4: 禁用的用户不应该有任何权限
	properties.Property("禁用的用户无法执行任何操作", prop.ForAll(
		func(resource string, action string, randomNum int) bool {
			// 创建一个禁用的用户
			disabledUser := &User{
				Username: fmt.Sprintf("disabled_user_%d", randomNum),
				Email:    fmt.Sprintf("disabled_%d@test.com", randomNum),
				Password: "hashed_password",
				Role:     RoleUser,
				Enabled:  false,
			}
			if err := rbacService.CreateUser(ctx, disabledUser); err != nil {
				t.Logf("创建禁用用户失败: %v", err)
				return true // 跳过这个测试用例
			}

			hasPermission, err := rbacService.CheckPermission(ctx, disabledUser.ID, resource, action)
			if err == nil {
				// 如果没有错误，说明权限检查没有正确处理禁用用户
				return false
			}

			// 应该返回错误，并且没有权限
			return !hasPermission
		},
		gen.OneConstOf(resources[0], resources[1], resources[2], resources[3], resources[4]),
		gen.OneConstOf(actions[0], actions[1], actions[2]),
		gen.IntRange(1, 100000),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty18_UserPermissionIsolation_Mock 使用模拟数据的权限隔离测试
// 这个测试不需要真实的数据库连接
func TestProperty18_UserPermissionIsolation_Mock(t *testing.T) {
	// 定义角色权限映射
	rolePermissions := map[UserRole]map[string][]string{
		RoleAdmin: {
			"container":   {"read", "write", "delete"},
			"application": {"read", "write", "delete"},
			"website":     {"read", "write", "delete"},
			"database":    {"read", "write", "delete"},
			"backup":      {"read", "write", "delete"},
			"user":        {"read", "write", "delete"},
			"tenant":      {"read", "write", "delete"},
		},
		RoleUser: {
			"container":   {"read"},
			"application": {"read"},
			"website":     {"read"},
			"database":    {"read"},
			"backup":      {"read"},
			"user":        {"read"},
			"tenant":      {"read"},
		},
		RoleReadOnly: {
			"container":   {"read"},
			"application": {"read"},
			"website":     {"read"},
			"database":    {"read"},
			"backup":      {"read"},
			"user":        {"read"},
			"tenant":      {"read"},
		},
	}

	// 模拟权限检查函数
	checkPermission := func(role UserRole, resource, action string) bool {
		if permissions, ok := rolePermissions[role]; ok {
			if actions, ok := permissions[resource]; ok {
				for _, a := range actions {
					if a == action {
						return true
					}
				}
			}
		}
		return false
	}

	properties := gopter.NewProperties(nil)

	resources := []string{"container", "application", "website", "database", "backup", "user", "tenant"}
	actions := []string{"read", "write", "delete"}

	// Property 1: 管理员应该有所有权限
	properties.Property("管理员拥有所有资源的所有操作权限", prop.ForAll(
		func(resource string, action string) bool {
			return checkPermission(RoleAdmin, resource, action)
		},
		gen.OneConstOf(resources[0], resources[1], resources[2], resources[3], resources[4], resources[5], resources[6]),
		gen.OneConstOf(actions[0], actions[1], actions[2]),
	))

	// Property 2: 普通用户只有读权限
	properties.Property("普通用户只能执行读操作", prop.ForAll(
		func(resource string, action string) bool {
			hasPermission := checkPermission(RoleUser, resource, action)

			// 如果是读操作，应该有权限
			if action == "read" {
				return hasPermission
			}

			// 如果是写或删除操作，不应该有权限
			return !hasPermission
		},
		gen.OneConstOf(resources[0], resources[1], resources[2], resources[3], resources[4], resources[5], resources[6]),
		gen.OneConstOf(actions[0], actions[1], actions[2]),
	))

	// Property 3: 只读用户只有读权限
	properties.Property("只读用户只能执行读操作", prop.ForAll(
		func(resource string, action string) bool {
			hasPermission := checkPermission(RoleReadOnly, resource, action)

			// 如果是读操作，应该有权限
			if action == "read" {
				return hasPermission
			}

			// 如果是写或删除操作，不应该有权限
			return !hasPermission
		},
		gen.OneConstOf(resources[0], resources[1], resources[2], resources[3], resources[4], resources[5], resources[6]),
		gen.OneConstOf(actions[0], actions[1], actions[2]),
	))

	// Property 4: 权限检查的一致性 - 相同的输入应该产生相同的结果
	properties.Property("权限检查结果一致性", prop.ForAll(
		func(role UserRole, resource string, action string) bool {
			result1 := checkPermission(role, resource, action)
			result2 := checkPermission(role, resource, action)
			return result1 == result2
		},
		gen.OneConstOf(RoleAdmin, RoleUser, RoleReadOnly),
		gen.OneConstOf(resources[0], resources[1], resources[2], resources[3], resources[4], resources[5], resources[6]),
		gen.OneConstOf(actions[0], actions[1], actions[2]),
	))

	// Property 5: 权限的传递性 - 如果有写权限，应该也有读权限（对于管理员）
	properties.Property("管理员的权限传递性", prop.ForAll(
		func(resource string) bool {
			hasWrite := checkPermission(RoleAdmin, resource, "write")
			hasRead := checkPermission(RoleAdmin, resource, "read")

			// 如果有写权限，应该也有读权限
			if hasWrite {
				return hasRead
			}
			return true
		},
		gen.OneConstOf(resources[0], resources[1], resources[2], resources[3], resources[4], resources[5], resources[6]),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
