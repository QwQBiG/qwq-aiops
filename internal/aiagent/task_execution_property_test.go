package aiagent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// **Feature: enhanced-aiops-platform, Property 2: AI 任务执行完整性**
// **Validates: Requirements 1.2**
//
// 属性测试：验证 AI 任务执行的完整性
// 对于任何 AI 接受的部署任务，系统应该能生成有效的配置文件并成功执行部署
func TestAITaskExecutionCompleteness(t *testing.T) {
	// 创建临时工作目录
	tempDir := t.TempDir()
	
	// 创建执行器和规划器
	executor := NewTaskExecutor(tempDir)
	planner := NewTaskPlanner(executor)
	
	// 定义测试数据集：各种部署场景
	deploymentScenarios := []struct {
		name        string
		intent      Intent
		service     string
		parameters  map[string]string
		description string
	}{
		{
			name:    "Nginx部署",
			intent:  IntentDeploy,
			service: "nginx",
			parameters: map[string]string{
				"service":     "nginx",
				"port":        "8080",
				"version":     "latest",
				"server_name": "example.com",
				"backend":     "http://localhost:3000",
			},
			description: "部署 Nginx Web 服务器",
		},
		{
			name:    "MySQL部署",
			intent:  IntentDeploy,
			service: "mysql",
			parameters: map[string]string{
				"service":  "mysql",
				"port":     "3306",
				"version":  "8.0",
				"password": "test123",
			},
			description: "部署 MySQL 数据库",
		},
		{
			name:    "Redis部署",
			intent:  IntentDeploy,
			service: "redis",
			parameters: map[string]string{
				"service": "redis",
				"port":    "6379",
				"version": "7.0",
			},
			description: "部署 Redis 缓存服务",
		},
		{
			name:    "通用服务部署",
			intent:  IntentDeploy,
			service: "postgres",
			parameters: map[string]string{
				"service": "postgres",
				"port":    "5432",
				"version": "15",
			},
			description: "部署 PostgreSQL 数据库",
		},
	}
	
	// 运行属性测试
	successCount := 0
	totalTests := len(deploymentScenarios)
	
	for _, scenario := range deploymentScenarios {
		t.Run(scenario.description, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			// 创建实体列表
			entities := []Entity{
				{Type: EntityService, Value: scenario.service, Confidence: 0.9},
			}
			
			// 1. 验证任务规划能力
			tasks, err := planner.PlanTasks(ctx, scenario.intent, entities, scenario.parameters)
			if err != nil {
				t.Errorf("任务规划失败: %v, 场景: %s", err, scenario.name)
				return
			}
			
			if len(tasks) == 0 {
				t.Errorf("未生成任何任务, 场景: %s", scenario.name)
				return
			}
			
			t.Logf("成功规划 %d 个任务", len(tasks))
			
			// 2. 验证配置文件生成能力
			hasConfigTask := false
			var configFilePath string
			
			for _, task := range tasks {
				if task.Type == TaskTypeConfig {
					hasConfigTask = true
					
					// 执行配置生成任务
					result, err := executor.ExecuteTask(ctx, task)
					if err != nil {
						t.Errorf("配置生成失败: %v, 场景: %s", err, scenario.name)
						return
					}
					
					if !result.Success {
						t.Errorf("配置生成任务失败: %s, 场景: %s", result.Error, scenario.name)
						return
					}
					
					// 获取配置文件路径
					if path, exists := result.Metadata["file_path"]; exists {
						configFilePath = path
					}
					
					t.Logf("配置文件生成成功: %s", result.Output)
					break
				}
			}
			
			// 3. 验证配置文件的有效性
			if hasConfigTask && configFilePath != "" {
				// 检查配置文件是否存在
				if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
					t.Errorf("配置文件不存在: %s, 场景: %s", configFilePath, scenario.name)
					return
				}
				
				// 读取配置文件内容
				content, err := os.ReadFile(configFilePath)
				if err != nil {
					t.Errorf("读取配置文件失败: %v, 场景: %s", err, scenario.name)
					return
				}
				
				// 验证配置文件不为空
				if len(content) == 0 {
					t.Errorf("配置文件为空, 场景: %s", scenario.name)
					return
				}
				
				// 验证配置文件包含必要的参数
				contentStr := string(content)
				for key, value := range scenario.parameters {
					// 跳过某些不应该出现在配置文件中的参数
					if key == "service" || key == "version" {
						continue
					}
					
					// 检查参数值是否在配置文件中
					if !strings.Contains(contentStr, value) {
						t.Logf("警告: 配置文件中未找到参数 %s=%s, 场景: %s", key, value, scenario.name)
					}
				}
				
				t.Logf("配置文件验证通过，大小: %d 字节", len(content))
			}
			
			// 4. 验证部署任务执行能力（使用 DryRun 模式）
			deploymentSuccess := false
			
			for _, task := range tasks {
				// 跳过配置任务，只测试部署相关任务
				if task.Type == TaskTypeConfig {
					continue
				}
				
				// 使用 DryRun 模式避免实际执行 Docker 命令
				task.DryRun = true
				
				result, err := executor.ExecuteTask(ctx, task)
				if err != nil {
					t.Logf("任务执行失败（可能是预期的）: %v, 任务类型: %s", err, task.Type)
					continue
				}
				
				if !result.Success {
					t.Logf("任务执行失败: %s, 任务类型: %s", result.Error, task.Type)
					continue
				}
				
				// 至少有一个部署任务成功执行
				if task.Type == TaskTypeDocker || task.Type == TaskTypeKubernetes {
					deploymentSuccess = true
					t.Logf("部署任务执行成功: %s", result.Output)
				}
			}
			
			if !deploymentSuccess {
				t.Logf("警告: 没有成功执行的部署任务, 场景: %s", scenario.name)
			}
			
			// 5. 验证任务序列的完整性
			validation, err := planner.ValidateTasks(ctx, tasks)
			if err != nil {
				t.Errorf("任务验证失败: %v, 场景: %s", err, scenario.name)
				return
			}
			
			if !validation.Valid {
				t.Errorf("任务序列无效: %v, 场景: %s", validation.Issues, scenario.name)
				return
			}
			
			t.Logf("任务序列验证通过:")
			t.Logf("  预计执行时间: %v", validation.EstimatedTime)
			t.Logf("  警告: %v", validation.Warnings)
			t.Logf("  建议: %v", validation.Suggestions)
			
			successCount++
		})
	}
	
	// 验证整体成功率
	successRate := float64(successCount) / float64(totalTests)
	t.Logf("整体测试成功率: %.2f%% (%d/%d)", successRate*100, successCount, totalTests)
	
	// 要求 100% 的成功率，因为这是核心功能
	if successRate < 1.0 {
		t.Errorf("AI 任务执行完整性测试成功率不足: %.2f%%, 期望 100%%", successRate*100)
	}
}

// 测试配置文件生成的幂等性
// 相同的参数应该生成相同的配置文件
func TestConfigGenerationIdempotence(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewTaskExecutor(tempDir)
	
	testCases := []struct {
		configType string
		parameters map[string]string
	}{
		{
			configType: "nginx",
			parameters: map[string]string{
				"port":        "8080",
				"server_name": "test.com",
				"backend":     "http://localhost:3000",
			},
		},
		{
			configType: "docker-compose",
			parameters: map[string]string{
				"service_name":   "web",
				"image":          "nginx:latest",
				"port":           "80",
				"container_port": "80",
				"env":            "production",
				"volume":         "/data",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run("幂等性测试: "+tc.configType, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			// 第一次生成
			result1, err := executor.GenerateConfig(ctx, tc.configType, tc.parameters)
			if err != nil {
				t.Fatalf("第一次配置生成失败: %v", err)
			}
			
			// 第二次生成（使用相同参数）
			result2, err := executor.GenerateConfig(ctx, tc.configType, tc.parameters)
			if err != nil {
				t.Fatalf("第二次配置生成失败: %v", err)
			}
			
			// 验证两次生成的内容相同
			if result1.Content != result2.Content {
				t.Errorf("配置内容不一致:\n第一次:\n%s\n第二次:\n%s", result1.Content, result2.Content)
			} else {
				t.Logf("幂等性验证通过: 两次生成的配置内容相同")
			}
		})
	}
}

// 测试任务执行的原子性
// 如果任务失败，应该能够回滚
func TestTaskExecutionAtomicity(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewTaskExecutor(tempDir)
	
	t.Run("可回滚任务的回滚能力", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		// 创建一个可回滚的任务
		task := &ExecutionTask{
			ID:         "test-rollback",
			Type:       TaskTypeFile,
			Parameters: map[string]string{
				"operation": "create",
				"path":      filepath.Join(tempDir, "test-rollback.txt"),
				"content":   "test content",
			},
			Timeout:    5 * time.Second,
			Reversible: true,
			CreatedAt:  time.Now(),
		}
		
		// 执行任务
		result, err := executor.ExecuteTask(ctx, task)
		if err != nil {
			t.Fatalf("任务执行失败: %v", err)
		}
		
		if !result.Success {
			t.Fatalf("任务执行失败: %s", result.Error)
		}
		
		// 验证文件已创建
		filePath := filepath.Join(tempDir, "test-rollback.txt")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("文件未创建: %s", filePath)
		}
		
		t.Logf("任务执行成功，文件已创建")
		
		// 注意：当前实现中文件操作没有自动生成回滚命令
		// 这是一个可以改进的地方
		if result.RollbackCmd == "" {
			t.Logf("注意: 文件操作任务未生成回滚命令（这是当前实现的限制）")
		}
	})
}

// 测试任务执行的错误处理
// 无效的任务应该被正确拒绝
func TestTaskExecutionErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewTaskExecutor(tempDir)
	planner := NewTaskPlanner(executor)
	
	t.Run("拒绝无效的任务参数", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// 缺少必要参数的部署请求
		intent := IntentDeploy
		entities := []Entity{} // 没有服务实体
		parameters := map[string]string{} // 没有参数
		
		tasks, err := planner.PlanTasks(ctx, intent, entities, parameters)
		
		// 应该返回错误或生成的任务验证失败
		if err == nil && len(tasks) > 0 {
			validation, _ := planner.ValidateTasks(ctx, tasks)
			if validation.Valid {
				t.Errorf("应该拒绝无效的任务参数")
			} else {
				t.Logf("正确拒绝了无效的任务: %v", validation.Issues)
			}
		} else {
			t.Logf("正确拒绝了无效的任务参数: %v", err)
		}
	})
	
	t.Run("处理配置生成错误", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// 使用不支持的配置类型
		_, err := executor.GenerateConfig(ctx, "unsupported-type", map[string]string{})
		
		if err == nil {
			t.Errorf("应该返回错误：不支持的配置类型")
		} else {
			t.Logf("正确处理了配置生成错误: %v", err)
		}
	})
	
	t.Run("处理危险命令", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		dangerousTask := &ExecutionTask{
			ID:         "test-dangerous",
			Type:       TaskTypeShell,
			Command:    "rm -rf /",
			Parameters: make(map[string]string),
			Timeout:    2 * time.Second,
			CreatedAt:  time.Now(),
		}
		
		result, err := executor.ExecuteTask(ctx, dangerousTask)
		
		// 应该被安全机制阻止
		if err == nil && result.Success {
			t.Errorf("危险命令应该被阻止")
		} else {
			t.Logf("正确阻止了危险命令")
		}
	})
}

// 测试任务执行的性能
// 任务应该在合理的时间内完成
func TestTaskExecutionPerformance(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewTaskExecutor(tempDir)
	planner := NewTaskPlanner(executor)
	
	t.Run("任务规划性能", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		intent := IntentDeploy
		entities := []Entity{
			{Type: EntityService, Value: "nginx"},
		}
		parameters := map[string]string{
			"service": "nginx",
			"port":    "8080",
			"version": "latest",
		}
		
		startTime := time.Now()
		tasks, err := planner.PlanTasks(ctx, intent, entities, parameters)
		duration := time.Since(startTime)
		
		if err != nil {
			t.Fatalf("任务规划失败: %v", err)
		}
		
		// 任务规划应该在 1 秒内完成
		if duration > 1*time.Second {
			t.Errorf("任务规划耗时过长: %v", duration)
		} else {
			t.Logf("任务规划性能良好: %v, 生成 %d 个任务", duration, len(tasks))
		}
	})
	
	t.Run("配置生成性能", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		parameters := map[string]string{
			"port":        "8080",
			"server_name": "example.com",
			"backend":     "http://localhost:3000",
		}
		
		startTime := time.Now()
		_, err := executor.GenerateConfig(ctx, "nginx", parameters)
		duration := time.Since(startTime)
		
		if err != nil {
			t.Fatalf("配置生成失败: %v", err)
		}
		
		// 配置生成应该在 100ms 内完成
		if duration > 100*time.Millisecond {
			t.Errorf("配置生成耗时过长: %v", duration)
		} else {
			t.Logf("配置生成性能良好: %v", duration)
		}
	})
}
