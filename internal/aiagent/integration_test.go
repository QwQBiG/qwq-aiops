package aiagent

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

// TestTaskExecutorBasicFunctionality 测试任务执行器的基本功能
func TestTaskExecutorBasicFunctionality(t *testing.T) {
	// 创建临时工作目录
	tempDir := t.TempDir()
	
	// 创建任务执行器
	executor := NewTaskExecutor(tempDir)
	
	t.Run("执行Shell命令", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		task := &ExecutionTask{
			ID:         "test-shell-1",
			Type:       TaskTypeShell,
			Command:    "whoami",
			Parameters: make(map[string]string),
			Timeout:    2 * time.Second,
			CreatedAt:  time.Now(),
		}
		
		result, err := executor.ExecuteTask(ctx, task)
		if err != nil {
			t.Fatalf("执行Shell任务失败: %v", err)
		}
		
		if !result.Success {
			t.Errorf("任务执行失败: %s", result.Error)
		}
		
		if result.ExitCode != 0 {
			t.Errorf("退出码错误: 期望 0，实际 %d", result.ExitCode)
		}
		
		t.Logf("Shell命令执行成功: %s", result.Output)
	})
	
	t.Run("生成配置文件", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		parameters := map[string]string{
			"port":        "8080",
			"server_name": "example.com",
			"backend":     "http://localhost:3000",
		}
		
		configResult, err := executor.GenerateConfig(ctx, "nginx", parameters)
		if err != nil {
			t.Fatalf("生成配置失败: %v", err)
		}
		
		if configResult.ConfigType != "nginx" {
			t.Errorf("配置类型错误: 期望 nginx，实际 %s", configResult.ConfigType)
		}
		
		// 验证配置文件是否创建
		if _, err := os.Stat(configResult.FilePath); os.IsNotExist(err) {
			t.Errorf("配置文件未创建: %s", configResult.FilePath)
		}
		
		t.Logf("配置文件生成成功: %s", configResult.FilePath)
	})
	
	t.Run("文件操作任务", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		testFile := filepath.Join(tempDir, "test.txt")
		testContent := "Hello, World!"
		
		// 创建文件
		createTask := &ExecutionTask{
			ID:   "test-file-create",
			Type: TaskTypeFile,
			Parameters: map[string]string{
				"operation": "create",
				"path":      testFile,
				"content":   testContent,
			},
			Timeout:   2 * time.Second,
			CreatedAt: time.Now(),
		}
		
		result, err := executor.ExecuteTask(ctx, createTask)
		if err != nil {
			t.Fatalf("创建文件失败: %v", err)
		}
		
		if !result.Success {
			t.Errorf("文件创建任务失败: %s", result.Error)
		}
		
		// 读取文件
		readTask := &ExecutionTask{
			ID:   "test-file-read",
			Type: TaskTypeFile,
			Parameters: map[string]string{
				"operation": "read",
				"path":      testFile,
			},
			Timeout:   2 * time.Second,
			CreatedAt: time.Now(),
		}
		
		readResult, err := executor.ExecuteTask(ctx, readTask)
		if err != nil {
			t.Fatalf("读取文件失败: %v", err)
		}
		
		if !readResult.Success {
			t.Errorf("文件读取任务失败: %s", readResult.Error)
		}
		
		if readResult.Output != testContent {
			t.Errorf("文件内容不匹配: 期望 %s，实际 %s", testContent, readResult.Output)
		}
		
		t.Logf("文件操作测试成功")
	})
	
	t.Run("DryRun模式", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		task := &ExecutionTask{
			ID:         "test-dryrun",
			Type:       TaskTypeShell,
			Command:    "whoami",
			Parameters: make(map[string]string),
			Timeout:    2 * time.Second,
			DryRun:     true,
			CreatedAt:  time.Now(),
		}
		
		result, err := executor.ExecuteTask(ctx, task)
		if err != nil {
			t.Fatalf("DryRun任务失败: %v", err)
		}
		
		if !result.Success {
			t.Errorf("DryRun任务应该成功")
		}
		
		if result.Output == "" {
			t.Errorf("DryRun应该返回模拟输出")
		}
		
		t.Logf("DryRun模式测试成功: %s", result.Output)
	})
}

// TestTaskPlannerBasicFunctionality 测试任务规划器的基本功能
func TestTaskPlannerBasicFunctionality(t *testing.T) {
	// 创建临时工作目录
	tempDir := t.TempDir()
	
	// 创建执行器和规划器
	executor := NewTaskExecutor(tempDir)
	planner := NewTaskPlanner(executor)
	
	t.Run("规划Nginx部署任务", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		intent := IntentDeploy
		entities := []Entity{
			{Type: EntityService, Value: "nginx", Confidence: 0.9},
			{Type: EntityPort, Value: "8080", Confidence: 0.8},
		}
		parameters := map[string]string{
			"service": "nginx",
			"port":    "8080",
			"version": "latest",
		}
		
		tasks, err := planner.PlanTasks(ctx, intent, entities, parameters)
		if err != nil {
			t.Fatalf("任务规划失败: %v", err)
		}
		
		if len(tasks) == 0 {
			t.Errorf("未生成任何任务")
		}
		
		t.Logf("成功规划 %d 个任务", len(tasks))
		for i, task := range tasks {
			t.Logf("  任务 %d: 类型=%s, 命令=%s", i+1, task.Type, task.Command)
		}
	})
	
	t.Run("规划通用启动任务", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		intent := IntentStart
		entities := []Entity{
			{Type: EntityService, Value: "redis", Confidence: 0.9},
		}
		parameters := map[string]string{
			"service": "redis",
		}
		
		tasks, err := planner.PlanTasks(ctx, intent, entities, parameters)
		if err != nil {
			t.Fatalf("任务规划失败: %v", err)
		}
		
		if len(tasks) == 0 {
			t.Errorf("未生成任何任务")
		}
		
		t.Logf("成功规划启动任务: %d 个", len(tasks))
	})
	
	t.Run("任务优化", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// 创建包含重复任务的列表
		tasks := []*ExecutionTask{
			{
				ID:      "task-1",
				Type:    TaskTypeDocker,
				Command: "ps -a",
				Timeout: 30 * time.Second,
			},
			{
				ID:      "task-2",
				Type:    TaskTypeConfig,
				Command: "generate",
				Timeout: 30 * time.Second,
			},
			{
				ID:      "task-3",
				Type:    TaskTypeDocker,
				Command: "ps -a", // 重复命令
				Timeout: 30 * time.Second,
			},
			{
				ID:      "task-4",
				Type:    TaskTypeShell,
				Command: "echo test",
				Timeout: 10 * time.Second,
			},
		}
		
		optimized, err := planner.OptimizeTasks(ctx, tasks)
		if err != nil {
			t.Fatalf("任务优化失败: %v", err)
		}
		
		if len(optimized) >= len(tasks) {
			t.Errorf("优化后任务数量未减少: 原始 %d，优化后 %d", len(tasks), len(optimized))
		}
		
		t.Logf("任务优化成功: %d -> %d", len(tasks), len(optimized))
	})
	
	t.Run("任务验证", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// 创建有效的任务列表
		tasks := []*ExecutionTask{
			{
				ID:      "task-1",
				Type:    TaskTypeDocker,
				Command: "pull nginx",
				Timeout: 5 * time.Minute,
			},
			{
				ID:      "task-2",
				Type:    TaskTypeDocker,
				Command: "run -d nginx",
				Timeout: 2 * time.Minute,
			},
		}
		
		validation, err := planner.ValidateTasks(ctx, tasks)
		if err != nil {
			t.Fatalf("任务验证失败: %v", err)
		}
		
		if !validation.Valid {
			t.Errorf("任务验证失败: %v", validation.Issues)
		}
		
		t.Logf("任务验证成功:")
		t.Logf("  有效: %v", validation.Valid)
		t.Logf("  问题: %v", validation.Issues)
		t.Logf("  警告: %v", validation.Warnings)
		t.Logf("  建议: %v", validation.Suggestions)
		t.Logf("  预计时间: %v", validation.EstimatedTime)
	})
	
	t.Run("验证无效任务", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// 创建包含无效任务的列表
		tasks := []*ExecutionTask{
			{
				ID:      "", // 缺少ID
				Type:    TaskTypeDocker,
				Command: "pull nginx",
			},
			{
				ID:      "task-2",
				Type:    "", // 缺少类型
				Command: "run nginx",
			},
		}
		
		validation, err := planner.ValidateTasks(ctx, tasks)
		if err != nil {
			t.Fatalf("任务验证失败: %v", err)
		}
		
		if validation.Valid {
			t.Errorf("应该检测到无效任务")
		}
		
		if len(validation.Issues) == 0 {
			t.Errorf("应该报告问题")
		}
		
		t.Logf("成功检测到无效任务: %v", validation.Issues)
	})
}

// TestIntegrationNLUToPlannerToExecutor 测试从NLU到规划器到执行器的完整流程
func TestIntegrationNLUToPlannerToExecutor(t *testing.T) {
	// 创建临时工作目录
	tempDir := t.TempDir()
	
	// 创建服务实例
	nluService := &NLUServiceImpl{
		contexts:       make(map[string]*ConversationContext),
		intentPatterns: make(map[Intent][]regexp.Regexp),
	}
	nluService.initializeTemplates()
	nluService.initializeServices()
	nluService.compilePatterns()
	
	executor := NewTaskExecutor(tempDir)
	planner := NewTaskPlanner(executor)
	
	t.Run("完整流程: 部署Nginx", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		// 1. 用户输入
		userInput := "部署nginx"
		
		// 2. NLU理解
		nluReq := &NLURequest{
			Text:      userInput,
			UserID:    "test-user",
			SessionID: "test-session",
			Language:  "zh",
		}
		
		nluResp, err := nluService.Understand(ctx, nluReq)
		if err != nil {
			t.Fatalf("NLU理解失败: %v", err)
		}
		
		t.Logf("NLU结果: 意图=%s, 置信度=%.2f", nluResp.Intent, nluResp.Confidence)
		
		// 3. 任务规划
		tasks, err := planner.PlanTasks(ctx, nluResp.Intent, nluResp.Entities, nluResp.Parameters)
		if err != nil {
			t.Fatalf("任务规划失败: %v", err)
		}
		
		t.Logf("规划了 %d 个任务", len(tasks))
		
		// 4. 任务验证
		validation, err := planner.ValidateTasks(ctx, tasks)
		if err != nil {
			t.Fatalf("任务验证失败: %v", err)
		}
		
		if !validation.Valid {
			t.Logf("任务验证警告: %v", validation.Issues)
		}
		
		// 5. 执行任务（DryRun模式）
		for i, task := range tasks {
			task.DryRun = true // 使用DryRun模式避免实际执行Docker命令
			
			result, err := executor.ExecuteTask(ctx, task)
			if err != nil {
				t.Logf("任务 %d 执行失败: %v", i+1, err)
				continue
			}
			
			t.Logf("任务 %d 执行结果: 成功=%v, 输出=%s", i+1, result.Success, result.Output)
		}
		
		t.Logf("完整流程测试成功")
	})
}

// TestTaskExecutorSafety 测试任务执行器的安全机制
func TestTaskExecutorSafety(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewTaskExecutor(tempDir)
	
	t.Run("阻止危险命令", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		dangerousCommands := []string{
			"rm -rf /",
			"dd if=/dev/zero of=/dev/sda",
			"mkfs.ext4 /dev/sda",
			"shutdown -h now",
		}
		
		for _, cmd := range dangerousCommands {
			task := &ExecutionTask{
				ID:         "test-dangerous",
				Type:       TaskTypeShell,
				Command:    cmd,
				Parameters: make(map[string]string),
				Timeout:    2 * time.Second,
				CreatedAt:  time.Now(),
			}
			
			result, err := executor.ExecuteTask(ctx, task)
			if err == nil || result.Success {
				t.Errorf("危险命令应该被阻止: %s", cmd)
			} else {
				t.Logf("成功阻止危险命令: %s", cmd)
			}
		}
	})
	
	t.Run("允许安全命令", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		safeCommands := []string{
			"whoami",
		}
		
		for _, cmd := range safeCommands {
			task := &ExecutionTask{
				ID:         "test-safe",
				Type:       TaskTypeShell,
				Command:    cmd,
				Parameters: make(map[string]string),
				Timeout:    2 * time.Second,
				CreatedAt:  time.Now(),
			}
			
			result, err := executor.ExecuteTask(ctx, task)
			if err != nil || !result.Success {
				t.Errorf("安全命令应该被允许: %s, 错误: %v", cmd, err)
			} else {
				t.Logf("成功执行安全命令: %s", cmd)
			}
		}
	})
}
