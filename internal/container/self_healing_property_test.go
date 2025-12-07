package container

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	
	// 使用纯 Go 实现的 SQLite 驱动
	_ "modernc.org/sqlite"
)

// **Feature: enhanced-aiops-platform, Property 8: 容器服务自愈能力**
// **Validates: Requirements 3.4**
//
// Property 8: 容器服务自愈能力
// *For any* 出现异常的容器服务，系统应该能自动重启并记录详细的故障日志
//
// 这个属性测试验证：
// 1. 当容器状态变为异常（stopped, error等）时，自愈系统能够检测到
// 2. 系统会自动尝试重启容器（如果配置了 AutoRestart）
// 3. 系统会记录详细的故障日志到数据库
// 4. 故障记录包含必要的信息：容器ID、故障类型、错误消息、操作结果等
func TestProperty_ContainerSelfHealingCapability(t *testing.T) {
	// 由于监控循环每10秒检查一次，属性测试会非常耗时
	// 这里使用较少的测试次数来验证核心属性
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 5 // 减少到5次测试
	parameters.MaxSize = 5

	properties := gopter.NewProperties(parameters)

	// Property: 对于任何异常容器，系统应该能自动重启并记录故障日志
	properties.Property("unhealthy containers should be auto-restarted and failures logged", 
		prop.ForAll(
			func(containerNum int) bool {
				// 设置测试环境
				db := setupPropertyTestDB(t)
				executor := newMockDockerExecutor()
				notifyService := NewMockNotificationService()

				service := NewSelfHealingService(db, executor, notifyService)
				// 使用足够长的超时时间
				ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
				defer cancel()

				// 启动自愈服务
				if err := service.Start(ctx); err != nil {
					t.Logf("Failed to start healing service: %v", err)
					return false
				}
				defer service.Stop()

				// 生成唯一的容器ID
				containerID := fmt.Sprintf("test-container-%d", containerNum)
				
				// 设置容器为异常状态（stopped）
				executor.containerStatus[containerID] = "stopped"

				// 配置自愈参数（使用固定值以减少变化）
				config := &HealingConfig{
					CheckInterval:    1,
					CheckTimeout:     5,
					FailureThreshold: 2, // 失败2次触发
					MaxRestarts:      10,
					RestartWindow:    300,
					AutoRestart:      true,
					SendAlert:        true,
				}

				// 注册容器
				if err := service.RegisterContainer(ctx, containerID, config); err != nil {
					t.Logf("Failed to register container: %v", err)
					return false
				}

				// 等待足够的时间让自愈系统检测并处理
				// 监控循环每10秒检查一次，FailureThreshold=2，所以需要等待至少2个周期
				time.Sleep(25 * time.Second)

				// 验证1: 检查是否尝试重启容器
				restartAttempted := len(executor.startCalls) > 0
				if !restartAttempted {
					t.Logf("Expected container %s to be restarted, but no restart calls were made", containerID)
					return false
				}

				// 验证2: 检查故障日志是否被记录
				failures, err := service.GetFailureHistory(ctx, containerID, 50)
				if err != nil {
					t.Logf("Failed to get failure history: %v", err)
					return false
				}

				if len(failures) == 0 {
					t.Logf("Expected failure records for container %s, but none were found", containerID)
					return false
				}

				// 验证3: 检查故障记录的完整性
				hasValidFailureRecord := false
				for _, failure := range failures {
					// 故障记录必须包含容器ID
					if failure.ContainerID != containerID {
						continue
					}

					// 故障记录必须有故障类型
					if failure.FailureType == "" {
						t.Logf("Failure record missing failure type")
						continue
					}

					// 故障记录必须有错误消息
					if failure.ErrorMessage == "" {
						t.Logf("Failure record missing error message")
						continue
					}

					// 故障记录必须有操作类型（restart, alert, none）
					if failure.Action == "" {
						t.Logf("Failure record missing action")
						continue
					}

					// 故障记录必须有检测时间
					if failure.DetectedAt.IsZero() {
						t.Logf("Failure record missing detected time")
						continue
					}

					hasValidFailureRecord = true
					break
				}

				if !hasValidFailureRecord {
					t.Logf("No valid failure record found for container %s", containerID)
					return false
				}

				// 验证4: 检查是否发送了告警
				alerts := notifyService.GetAlerts()
				if len(alerts) == 0 {
					t.Logf("Expected alerts to be sent for container %s, but none were sent", containerID)
					return false
				}

				return true
			},
			// 生成随机容器编号
			gen.IntRange(1, 10000),
		))

	properties.TestingRun(t)
}

// TestProperty_HealthyContainersNotRestarted 测试健康容器不会被重启
// **Feature: enhanced-aiops-platform, Property 8: 容器服务自愈能力**
// **Validates: Requirements 3.4**
//
// 这个属性验证：健康运行的容器不应该被自愈系统重启
func TestProperty_HealthyContainersNotRestarted(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 5 // 减少测试次数

	properties := gopter.NewProperties(parameters)

	properties.Property("healthy containers should not be restarted", 
		prop.ForAll(
			func(containerNum int) bool {
				// 设置测试环境
				db := setupPropertyTestDB(t)
				executor := newMockDockerExecutor()
				notifyService := NewMockNotificationService()

				service := NewSelfHealingService(db, executor, notifyService)
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()

				// 启动自愈服务
				if err := service.Start(ctx); err != nil {
					t.Logf("Failed to start healing service: %v", err)
					return false
				}
				defer service.Stop()

				// 生成唯一的容器ID
				containerID := fmt.Sprintf("healthy-%d", containerNum)
				
				// 设置容器为健康状态
				executor.containerStatus[containerID] = "running"

				// 配置自愈参数
				config := &HealingConfig{
					CheckInterval:    1,
					CheckTimeout:     5,
					FailureThreshold: 2,
					MaxRestarts:      5,
					RestartWindow:    300,
					AutoRestart:      true,
					SendAlert:        true,
				}

				// 注册容器
				if err := service.RegisterContainer(ctx, containerID, config); err != nil {
					t.Logf("Failed to register container: %v", err)
					return false
				}

				// 等待多次检查周期（监控循环每10秒一次，等待3个周期）
				time.Sleep(35 * time.Second)

				// 验证：健康容器不应该被重启
				if len(executor.startCalls) > 0 {
					t.Logf("Healthy container %s should not be restarted, but got %d restart calls", 
						containerID, len(executor.startCalls))
					return false
				}

				// 验证：健康状态应该正确
				health, err := service.GetContainerHealth(ctx, containerID)
				if err != nil {
					t.Logf("Failed to get container health: %v", err)
					return false
				}

				if health.Status != "healthy" {
					t.Logf("Expected status 'healthy' for container %s, got '%s'", containerID, health.Status)
					return false
				}

				if health.ConsecutiveFailures != 0 {
					t.Logf("Expected 0 consecutive failures for healthy container %s, got %d", 
						containerID, health.ConsecutiveFailures)
					return false
				}

				return true
			},
			// 生成随机容器编号
			gen.IntRange(1, 10000),
		))

	properties.TestingRun(t)
}

// setupPropertyTestDB 为属性测试设置数据库
func setupPropertyTestDB(t *testing.T) *gorm.DB {
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
