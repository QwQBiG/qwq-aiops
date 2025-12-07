package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	
	_ "modernc.org/sqlite"
)

// **Feature: enhanced-aiops-platform, Property 22: 自动化任务执行可靠性**
// **Validates: Requirements 8.4, 8.5**
//
// Property 22: 自动化任务执行可靠性
// *For any* 自动化任务，系统应该提供详细的执行日志、错误处理和事务保证
//
// 这个属性测试验证：
// 1. Webhook 事件能够被正确触发和投递
// 2. 系统记录详细的执行日志（payload、状态码、响应、错误信息）
// 3. 失败的 Webhook 会自动重试
// 4. 重试次数被正确记录
// 5. 最终的成功/失败状态被正确记录
func TestProperty22_AutomatedTaskExecutionReliability(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20  // 减少测试次数以加快速度

	properties := gopter.NewProperties(parameters)

	properties.Property("Webhook 执行应该记录详细日志", prop.ForAll(
		func(eventType string, shouldSucceed bool) bool {
			// 设置测试环境
			db := setupWebhookTestDB(t)
			
			// 创建测试 HTTP 服务器
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				
				// 验证请求头
				if r.Header.Get("Content-Type") != "application/json" {
					t.Logf("Content-Type 不正确")
				}
				
				if r.Header.Get("User-Agent") != "QWQ-Webhook/1.0" {
					t.Logf("User-Agent 不正确")
				}
				
				// 根据测试参数决定成功或失败
				if shouldSucceed {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status":"success"}`))
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"status":"error"}`))
				}
			}))
			defer server.Close()

			service := NewWebhookService(db)
			ctx := context.Background()

			// 创建 Webhook 配置
			webhook := &Webhook{
				Name:       "测试 Webhook",
				URL:        server.URL,
				Events:     []EventType{EventType(eventType)},
				Enabled:    true,
				RetryCount: 2,
				Timeout:    10,
				UserID:     1,
				TenantID:   1,
			}

			if err := service.CreateWebhook(ctx, webhook); err != nil {
				t.Logf("创建 Webhook 失败: %v", err)
				return false
			}

			// 触发事件
			event := &Event{
				Type:      EventType(eventType),
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"test_key": "test_value",
					"count":    123,
				},
				UserID:   1,
				TenantID: 1,
			}

			if err := service.TriggerEvent(ctx, event); err != nil {
				t.Logf("触发事件失败: %v", err)
				return false
			}

			// 等待 Webhook 投递完成（异步操作）
			time.Sleep(3 * time.Second)

			// 查询事件日志
			logs, err := service.ListEvents(ctx, webhook.ID)
			if err != nil {
				t.Logf("查询事件日志失败: %v", err)
				return false
			}

			// 验证：应该有事件日志记录
			if len(logs) == 0 {
				t.Logf("没有找到事件日志")
				return false
			}

			log := logs[0]

			// 验证：日志应该包含 payload
			if log.Payload == "" {
				t.Logf("事件日志缺少 payload")
				return false
			}

			// 验证：payload 应该是有效的 JSON
			var payloadData Event
			if err := json.Unmarshal([]byte(log.Payload), &payloadData); err != nil {
				t.Logf("payload 不是有效的 JSON: %v", err)
				return false
			}

			// 验证：日志应该包含状态码
			if log.StatusCode == 0 {
				t.Logf("事件日志缺少状态码")
				return false
			}

			// 验证：日志应该包含响应
			if log.Response == "" {
				t.Logf("事件日志缺少响应")
				return false
			}

			// 验证：成功/失败状态应该正确
			if shouldSucceed {
				if !log.Success {
					t.Logf("期望成功但日志显示失败")
					return false
				}
				if log.StatusCode != http.StatusOK {
					t.Logf("期望状态码 200，实际 %d", log.StatusCode)
					return false
				}
			} else {
				if log.Success {
					t.Logf("期望失败但日志显示成功")
					return false
				}
				// 验证：失败时应该有错误信息或非 2xx 状态码
				if log.StatusCode >= 200 && log.StatusCode < 300 {
					t.Logf("失败的请求不应该返回 2xx 状态码")
					return false
				}
			}

			// 验证：重试次数应该被记录
			if !shouldSucceed {
				// 失败的请求应该重试
				if log.RetryCount < 1 {
					t.Logf("失败的请求应该有重试记录")
					return false
				}
				
				// 验证：实际调用次数应该等于 1 + 重试次数
				expectedCalls := 1 + webhook.RetryCount
				if callCount != expectedCalls {
					t.Logf("期望调用 %d 次，实际调用 %d 次", expectedCalls, callCount)
					return false
				}
			}

			return true
		},
		gen.OneConstOf(
			string(EventAppInstalled),
			string(EventContainerStarted),
			string(EventBackupCompleted),
			string(EventWebsiteCreated),
		),
		gen.Bool(), // 是否成功
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty22_WebhookSignatureVerification 测试 Webhook 签名验证
// **Feature: enhanced-aiops-platform, Property 22: 自动化任务执行可靠性**
// **Validates: Requirements 8.4, 8.5**
func TestProperty22_WebhookSignatureVerification(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Webhook 签名应该被正确生成和验证", prop.ForAll(
		func(secret string, payload string) bool {
			// 跳过空 secret 的情况
			if secret == "" {
				return true
			}

			// 设置测试环境
			db := setupWebhookTestDB(t)
			
			var receivedSignature string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 获取签名
				receivedSignature = r.Header.Get("X-Webhook-Signature")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			service := NewWebhookService(db).(*WebhookServiceImpl)
			ctx := context.Background()

			// 创建带 secret 的 Webhook
			webhook := &Webhook{
				Name:       "测试签名",
				URL:        server.URL,
				Secret:     secret,
				Events:     []EventType{EventAppInstalled},
				Enabled:    true,
				RetryCount: 0,
				Timeout:    10,
				UserID:     1,
				TenantID:   1,
			}

			if err := service.CreateWebhook(ctx, webhook); err != nil {
				t.Logf("创建 Webhook 失败: %v", err)
				return false
			}

			// 触发事件
			event := &Event{
				Type:      EventAppInstalled,
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"payload": payload},
				UserID:    1,
				TenantID:  1,
			}

			if err := service.TriggerEvent(ctx, event); err != nil {
				t.Logf("触发事件失败: %v", err)
				return false
			}

			// 等待投递完成
			time.Sleep(1 * time.Second)

			// 验证：应该收到签名
			if receivedSignature == "" {
				t.Logf("未收到签名")
				return false
			}

			// 验证：签名应该可以被验证
			eventJSON, _ := json.Marshal(event)
			if !VerifySignature(eventJSON, receivedSignature, secret) {
				t.Logf("签名验证失败")
				return false
			}

			return true
		},
		gen.AlphaString(),  // secret
		gen.AlphaString(),  // payload
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty22_WebhookEventFiltering 测试 Webhook 事件过滤
// **Feature: enhanced-aiops-platform, Property 22: 自动化任务执行可靠性**
// **Validates: Requirements 8.4, 8.5**
func TestProperty22_WebhookEventFiltering(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50

	properties := gopter.NewProperties(parameters)

	properties.Property("Webhook 应该只接收订阅的事件", prop.ForAll(
		func(subscribedEvent string, triggeredEvent string) bool {
			// 设置测试环境
			db := setupWebhookTestDB(t)
			
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			service := NewWebhookService(db)
			ctx := context.Background()

			// 创建只订阅特定事件的 Webhook
			webhook := &Webhook{
				Name:       "事件过滤测试",
				URL:        server.URL,
				Events:     []EventType{EventType(subscribedEvent)},
				Enabled:    true,
				RetryCount: 0,
				Timeout:    10,
				UserID:     1,
				TenantID:   1,
			}

			if err := service.CreateWebhook(ctx, webhook); err != nil {
				t.Logf("创建 Webhook 失败: %v", err)
				return false
			}

			// 触发事件
			event := &Event{
				Type:      EventType(triggeredEvent),
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"test": "data"},
				UserID:    1,
				TenantID:  1,
			}

			if err := service.TriggerEvent(ctx, event); err != nil {
				t.Logf("触发事件失败: %v", err)
				return false
			}

			// 等待投递完成
			time.Sleep(1 * time.Second)

			// 验证：只有订阅的事件才应该被投递
			shouldReceive := subscribedEvent == triggeredEvent
			
			if shouldReceive {
				if callCount == 0 {
					t.Logf("订阅的事件应该被投递")
					return false
				}
			} else {
				if callCount > 0 {
					t.Logf("未订阅的事件不应该被投递")
					return false
				}
			}

			return true
		},
		gen.OneConstOf(
			string(EventAppInstalled),
			string(EventContainerStarted),
			string(EventBackupCompleted),
		),
		gen.OneConstOf(
			string(EventAppInstalled),
			string(EventContainerStarted),
			string(EventBackupCompleted),
			string(EventWebsiteCreated),
		),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// setupWebhookTestDB 为 Webhook 测试设置数据库
func setupWebhookTestDB(t *testing.T) *gorm.DB {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("打开 SQL 数据库失败: %v", err)
	}

	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 GORM 数据库失败: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&Webhook{}, &EventLog{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}
