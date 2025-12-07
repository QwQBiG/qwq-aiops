package appstore

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupSimpleTestDB 设置测试数据库
func setupSimpleTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&AppTemplate{}, &ApplicationInstance{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

// createSimpleTestTemplate 创建测试模板
func createSimpleTestTemplate(t *testing.T, db *gorm.DB) *AppTemplate {
	template := &AppTemplate{
		Name:        "test-nginx",
		DisplayName: "Test Nginx",
		Description: "Test Nginx Server",
		Category:    CategoryWebServer,
		Type:        TemplateTypeDockerCompose,
		Version:     "1.0.0",
		Status:      TemplateStatusPublished,
		Content: `version: '3'
services:
  nginx:
    image: nginx:{{.Version}}
    ports:
      - "{{.Port}}:80"
    volumes:
      - {{.DataPath}}:/usr/share/nginx/html`,
		Parameters: `[
			{
				"name": "Version",
				"display_name": "Nginx Version",
				"type": "string",
				"default_value": "latest",
				"required": true
			},
			{
				"name": "Port",
				"display_name": "Host Port",
				"type": "int",
				"default_value": 8080,
				"required": true
			},
			{
				"name": "DataPath",
				"display_name": "Data Path",
				"type": "path",
				"default_value": "./data",
				"required": true
			}
		]`,
		Dependencies: `[]`,
	}

	if err := db.Create(template).Error; err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	return template
}

func TestSimpleInstallerService_Install(t *testing.T) {
	db := setupSimpleTestDB(t)
	appStoreService := NewAppStoreService(db)
	installerService := NewInstallerService(appStoreService)

	// 创建测试模板
	template := createSimpleTestTemplate(t, db)

	ctx := context.Background()

	t.Run("成功安装应用", func(t *testing.T) {
		req := &InstallRequest{
			TemplateID:   template.ID,
			InstanceName: "my-nginx",
			Parameters: map[string]interface{}{
				"Version":  "1.21",
				"Port":     8080,
				"DataPath": "/data/nginx",
			},
			UserID:      1,
			TenantID:    1,
			AutoResolve: false,
		}

		result, err := installerService.Install(ctx, req)
		if err != nil {
			t.Fatalf("安装失败: %v", err)
		}

		if result.InstanceID == 0 {
			t.Error("实例ID不应该为0")
		}

		if result.ProgressID == "" {
			t.Error("进度ID不应该为空")
		}

		if result.Status != StatusPending {
			t.Errorf("期望状态为 %s，实际为 %s", StatusPending, result.Status)
		}

		// 等待安装完成
		time.Sleep(200 * time.Millisecond)

		// 检查进度
		progress, err := installerService.GetProgress(ctx, result.ProgressID)
		if err != nil {
			t.Fatalf("获取进度失败: %v", err)
		}

		if progress.Status != StatusCompleted {
			t.Errorf("期望状态为 %s，实际为 %s (错误: %s)", StatusCompleted, progress.Status, progress.Error)
		}

		// 检查实例状态
		instance, err := appStoreService.GetInstance(ctx, result.InstanceID)
		if err != nil {
			t.Fatalf("获取实例失败: %v", err)
		}

		if instance.Status != "running" {
			t.Errorf("期望实例状态为 running，实际为 %s", instance.Status)
		}
	})
}

func TestSimpleProgressStore(t *testing.T) {
	store := NewProgressStore()

	t.Run("创建和获取进度", func(t *testing.T) {
		progress := store.Create(1)

		if progress.ID == "" {
			t.Error("进度ID不应该为空")
		}

		if progress.InstanceID != 1 {
			t.Errorf("期望实例ID为1，实际为 %d", progress.InstanceID)
		}

		if progress.Status != StatusPending {
			t.Errorf("期望状态为 %s，实际为 %s", StatusPending, progress.Status)
		}

		// 获取进度
		retrieved := store.Get(progress.ID)
		if retrieved == nil {
			t.Fatal("无法获取进度")
		}

		if retrieved.ID != progress.ID {
			t.Error("获取的进度ID不匹配")
		}
	})

	t.Run("更新进度", func(t *testing.T) {
		progress := store.Create(2)

		store.Update(progress.ID, StatusInstalling, "Installing application", 2, 5)

		updated := store.Get(progress.ID)
		if updated.Status != StatusInstalling {
			t.Errorf("期望状态为 %s，实际为 %s", StatusInstalling, updated.Status)
		}

		if updated.CompletedSteps != 2 {
			t.Errorf("期望完成步骤为2，实际为 %d", updated.CompletedSteps)
		}

		if updated.GetProgress() != 40.0 {
			t.Errorf("期望进度为40%%，实际为 %.1f%%", updated.GetProgress())
		}
	})
}

func TestSimpleConflictChecker(t *testing.T) {
	db := setupSimpleTestDB(t)
	appStoreService := NewAppStoreService(db)
	checker := NewConflictChecker(appStoreService)

	ctx := context.Background()

	// 创建第一个模板和实例（占用端口8080）
	template1 := createSimpleTestTemplate(t, db)
	instance1 := &ApplicationInstance{
		Name:       "existing-nginx",
		TemplateID: template1.ID,
		Version:    template1.Version,
		Status:     "running",
		UserID:     1,
		TenantID:   1,
	}

	config1 := map[string]interface{}{
		"Version":  "latest",
		"Port":     8080,
		"DataPath": "/data/nginx1",
	}
	configJSON, _ := json.Marshal(config1)
	instance1.Config = string(configJSON)

	if err := appStoreService.CreateInstance(ctx, instance1); err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	t.Run("检测端口冲突", func(t *testing.T) {
		// 尝试安装另一个使用相同端口的应用
		params := map[string]interface{}{
			"Version":  "latest",
			"Port":     8080, // 相同端口
			"DataPath": "/data/nginx2",
		}

		conflicts, err := checker.DetectConflicts(ctx, template1.ID, params)
		if err != nil {
			t.Fatalf("检测冲突失败: %v", err)
		}

		// 应该检测到端口冲突
		foundPortConflict := false
		for _, conflict := range conflicts {
			if conflict.Type == "port" && conflict.Resource == "8080" {
				foundPortConflict = true
				if !conflict.Resolvable {
					t.Error("端口冲突应该是可解决的")
				}
			}
		}

		if !foundPortConflict {
			t.Error("应该检测到端口冲突")
		}
	})
}

func TestSimpleDependencyManager(t *testing.T) {
	db := setupSimpleTestDB(t)
	appStoreService := NewAppStoreService(db)
	depMgr := NewDependencyManager(appStoreService)

	ctx := context.Background()

	// 创建有依赖的模板
	template := &AppTemplate{
		Name:        "test-app-with-deps",
		DisplayName: "Test App with Dependencies",
		Category:    CategoryWebServer,
		Type:        TemplateTypeDockerCompose,
		Version:     "1.0.0",
		Status:      TemplateStatusPublished,
		Content:     "version: '3'\nservices:\n  app:\n    image: test:latest",
		Dependencies: `[
			{
				"name": "mysql",
				"type": "application",
				"description": "MySQL Database",
				"optional": false
			},
			{
				"name": "redis",
				"type": "service",
				"description": "Redis Cache",
				"optional": true
			}
		]`,
	}

	if err := appStoreService.CreateTemplate(ctx, template); err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}

	t.Run("检查依赖", func(t *testing.T) {
		checks, err := depMgr.CheckDependencies(ctx, template.ID)
		if err != nil {
			t.Fatalf("检查依赖失败: %v", err)
		}

		if len(checks) != 2 {
			t.Errorf("期望2个依赖检查，实际为 %d", len(checks))
		}

		// 验证必需依赖
		foundRequired := false
		for _, check := range checks {
			if check.Name == "mysql" && check.Required {
				foundRequired = true
				if check.Satisfied {
					t.Error("MySQL依赖不应该被满足（未安装）")
				}
			}
		}

		if !foundRequired {
			t.Error("未找到必需的MySQL依赖")
		}
	})
}
