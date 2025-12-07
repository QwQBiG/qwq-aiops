package container

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	
	// 使用纯 Go 实现的 SQLite 驱动
	_ "modernc.org/sqlite"
)

// mockDockerExecutor 用于测试的 Mock Docker 执行器
type mockDockerExecutor struct {
	containerStatus map[string]string
	containerInfo   map[string]*ContainerInfo
	startCalls      []string
	stopCalls       []string
}

func newMockDockerExecutor() *mockDockerExecutor {
	return &mockDockerExecutor{
		containerStatus: make(map[string]string),
		containerInfo:   make(map[string]*ContainerInfo),
		startCalls:      make([]string, 0),
		stopCalls:       make([]string, 0),
	}
}

func (m *mockDockerExecutor) StartProject(ctx context.Context, projectName, composeContent string) error {
	return nil
}

func (m *mockDockerExecutor) StopProject(ctx context.Context, projectName string) error {
	return nil
}

func (m *mockDockerExecutor) RemoveProject(ctx context.Context, projectName string) error {
	return nil
}

func (m *mockDockerExecutor) StartService(ctx context.Context, projectName, serviceName string, service *Service) (string, error) {
	return "mock-container-id", nil
}

func (m *mockDockerExecutor) StopService(ctx context.Context, projectName, serviceName string) error {
	return nil
}

func (m *mockDockerExecutor) GetServiceContainers(ctx context.Context, projectName, serviceName string) ([]string, error) {
	return []string{}, nil
}

func (m *mockDockerExecutor) StartContainer(ctx context.Context, containerID string) error {
	m.startCalls = append(m.startCalls, containerID)
	m.containerStatus[containerID] = "running"
	return nil
}

func (m *mockDockerExecutor) StopContainer(ctx context.Context, containerID string) error {
	m.stopCalls = append(m.stopCalls, containerID)
	m.containerStatus[containerID] = "stopped"
	return nil
}

func (m *mockDockerExecutor) RemoveContainer(ctx context.Context, containerID string) error {
	delete(m.containerStatus, containerID)
	return nil
}

func (m *mockDockerExecutor) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	if status, ok := m.containerStatus[containerID]; ok {
		return status, nil
	}
	return "unknown", nil
}

func (m *mockDockerExecutor) GetContainerInfo(ctx context.Context, containerID string) (*ContainerInfo, error) {
	if info, ok := m.containerInfo[containerID]; ok {
		return info, nil
	}
	now := time.Now()
	return &ContainerInfo{
		ID:        containerID,
		Name:      "mock-container",
		Image:     "mock:latest",
		Status:    m.containerStatus[containerID],
		Health:    "healthy",
		StartedAt: &now,
	}, nil
}

func setupTestDB(t *testing.T) *gorm.DB {
	// 使用 modernc.org/sqlite 纯 Go 驱动
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open SQL database: %v", err)
	}

	// 使用 GORM 包装
	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open GORM database: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&FailureRecord{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestSelfHealingService_RegisterAndUnregister(t *testing.T) {
	db := setupTestDB(t)
	executor := newMockDockerExecutor()
	notifyService := NewMockNotificationService()

	service := NewSelfHealingService(db, executor, notifyService)
	ctx := context.Background()

	containerID := "test-container-1"
	config := DefaultHealingConfig()

	// 测试注册
	err := service.RegisterContainer(ctx, containerID, config)
	if err != nil {
		t.Errorf("Failed to register container: %v", err)
	}

	// 测试获取健康状态
	health, err := service.GetContainerHealth(ctx, containerID)
	if err != nil {
		t.Errorf("Failed to get container health: %v", err)
	}

	if health.ContainerID != containerID {
		t.Errorf("Expected container ID %s, got %s", containerID, health.ContainerID)
	}

	// 测试取消注册
	err = service.UnregisterContainer(ctx, containerID)
	if err != nil {
		t.Errorf("Failed to unregister container: %v", err)
	}

	// 取消注册后应该无法获取健康状态
	_, err = service.GetContainerHealth(ctx, containerID)
	if err == nil {
		t.Error("Expected error when getting health of unregistered container")
	}
}

func TestSelfHealingService_HealthyContainer(t *testing.T) {
	db := setupTestDB(t)
	executor := newMockDockerExecutor()
	notifyService := NewMockNotificationService()

	service := NewSelfHealingService(db, executor, notifyService)
	ctx := context.Background()

	// 启动监控
	if err := service.Start(ctx); err != nil {
		t.Fatalf("Failed to start healing service: %v", err)
	}
	defer service.Stop()

	containerID := "healthy-container"
	executor.containerStatus[containerID] = "running"

	config := &HealingConfig{
		CheckInterval:    1, // 1秒检查一次（测试用）
		CheckTimeout:     5,
		FailureThreshold: 3,
		MaxRestarts:      5,
		RestartWindow:    300,
		AutoRestart:      true,
		SendAlert:        true,
	}

	// 注册容器
	if err := service.RegisterContainer(ctx, containerID, config); err != nil {
		t.Fatalf("Failed to register container: %v", err)
	}

	// 等待几次健康检查
	time.Sleep(3 * time.Second)

	// 检查健康状态
	health, err := service.GetContainerHealth(ctx, containerID)
	if err != nil {
		t.Fatalf("Failed to get container health: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}

	if health.ConsecutiveFailures != 0 {
		t.Errorf("Expected 0 consecutive failures, got %d", health.ConsecutiveFailures)
	}

	// 不应该有重启
	if len(executor.startCalls) > 0 {
		t.Errorf("Expected no restart calls, got %d", len(executor.startCalls))
	}
}

func TestSelfHealingService_UnhealthyContainerAutoRestart(t *testing.T) {
	db := setupTestDB(t)
	executor := newMockDockerExecutor()
	notifyService := NewMockNotificationService()

	service := NewSelfHealingService(db, executor, notifyService)
	ctx := context.Background()

	// 启动监控
	if err := service.Start(ctx); err != nil {
		t.Fatalf("Failed to start healing service: %v", err)
	}
	defer service.Stop()

	containerID := "unhealthy-container"
	executor.containerStatus[containerID] = "stopped" // 容器停止

	config := &HealingConfig{
		CheckInterval:    1, // 1秒检查一次
		CheckTimeout:     5,
		FailureThreshold: 2, // 失败2次就触发
		MaxRestarts:      5,
		RestartWindow:    300,
		AutoRestart:      true,
		SendAlert:        true,
	}

	// 注册容器
	if err := service.RegisterContainer(ctx, containerID, config); err != nil {
		t.Fatalf("Failed to register container: %v", err)
	}

	// 等待自愈触发
	// 监控循环每10秒检查一次，FailureThreshold=2，所以需要等待至少2个周期
	time.Sleep(25 * time.Second)

	// 检查是否尝试重启
	if len(executor.startCalls) == 0 {
		t.Error("Expected container to be restarted, but no restart calls were made")
	}

	// 检查是否发送了告警
	alerts := notifyService.GetAlerts()
	if len(alerts) == 0 {
		t.Error("Expected alerts to be sent, but none were sent")
	}

	// 检查故障记录
	failures, err := service.GetFailureHistory(ctx, containerID, 10)
	if err != nil {
		t.Fatalf("Failed to get failure history: %v", err)
	}

	if len(failures) == 0 {
		t.Error("Expected failure records, but none were found")
	}
}

func TestSelfHealingService_RestartLimit(t *testing.T) {
	db := setupTestDB(t)
	executor := newMockDockerExecutor()
	notifyService := NewMockNotificationService()

	service := NewSelfHealingService(db, executor, notifyService)
	ctx := context.Background()

	// 启动监控
	if err := service.Start(ctx); err != nil {
		t.Fatalf("Failed to start healing service: %v", err)
	}
	defer service.Stop()

	containerID := "flapping-container"
	executor.containerStatus[containerID] = "stopped"

	config := &HealingConfig{
		CheckInterval:    1,
		CheckTimeout:     5,
		FailureThreshold: 1, // 失败1次就触发
		MaxRestarts:      2,  // 最多重启2次
		RestartWindow:    10, // 10秒窗口
		AutoRestart:      true,
		SendAlert:        true,
	}

	// 注册容器
	if err := service.RegisterContainer(ctx, containerID, config); err != nil {
		t.Fatalf("Failed to register container: %v", err)
	}

	// 等待多次重启尝试
	time.Sleep(8 * time.Second)

	// 检查重启次数不应超过限制
	restartCount := len(executor.startCalls)
	if restartCount > config.MaxRestarts {
		t.Errorf("Expected at most %d restarts, got %d", config.MaxRestarts, restartCount)
	}

	// 应该有告警通知重启限制已达到
	alerts := notifyService.GetAlerts()
	hasLimitAlert := false
	for _, alert := range alerts {
		if alert.Level == "critical" && alert.Title == "Container restart limit exceeded" {
			hasLimitAlert = true
			break
		}
	}

	if !hasLimitAlert {
		t.Error("Expected alert about restart limit exceeded")
	}
}

func TestSelfHealingService_NoAutoRestart(t *testing.T) {
	db := setupTestDB(t)
	executor := newMockDockerExecutor()
	notifyService := NewMockNotificationService()

	service := NewSelfHealingService(db, executor, notifyService)
	ctx := context.Background()

	// 启动监控
	if err := service.Start(ctx); err != nil {
		t.Fatalf("Failed to start healing service: %v", err)
	}
	defer service.Stop()

	containerID := "no-restart-container"
	executor.containerStatus[containerID] = "stopped"

	config := &HealingConfig{
		CheckInterval:    1,
		CheckTimeout:     5,
		FailureThreshold: 2,
		MaxRestarts:      5,
		RestartWindow:    300,
		AutoRestart:      false, // 禁用自动重启
		SendAlert:        true,
	}

	// 注册容器
	if err := service.RegisterContainer(ctx, containerID, config); err != nil {
		t.Fatalf("Failed to register container: %v", err)
	}

	// 等待检查
	time.Sleep(5 * time.Second)

	// 不应该有重启
	if len(executor.startCalls) > 0 {
		t.Errorf("Expected no restart calls when AutoRestart is false, got %d", len(executor.startCalls))
	}

	// 应该有告警
	alerts := notifyService.GetAlerts()
	if len(alerts) == 0 {
		t.Error("Expected alerts to be sent even when AutoRestart is false")
	}
}

func TestDefaultHealingConfig(t *testing.T) {
	config := DefaultHealingConfig()

	if config.CheckInterval != 30 {
		t.Errorf("Expected CheckInterval 30, got %d", config.CheckInterval)
	}

	if config.FailureThreshold != 3 {
		t.Errorf("Expected FailureThreshold 3, got %d", config.FailureThreshold)
	}

	if !config.AutoRestart {
		t.Error("Expected AutoRestart to be true")
	}

	if !config.SendAlert {
		t.Error("Expected SendAlert to be true")
	}
}

func TestFailureRecord_TableName(t *testing.T) {
	record := FailureRecord{}
	tableName := record.TableName()

	if tableName != "container_failure_records" {
		t.Errorf("Expected table name 'container_failure_records', got '%s'", tableName)
	}
}
