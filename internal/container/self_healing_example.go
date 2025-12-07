package container

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// ExampleSelfHealingService 演示如何使用容器自愈服务
func ExampleSelfHealingService() {
	// 假设已经有数据库连接
	var db *gorm.DB // 实际使用时需要初始化
	
	// 创建 Docker 执行器
	dockerExecutor := NewDockerExecutor()
	
	// 创建通知服务
	notifyService := NewSimpleNotificationService()
	
	// 创建自愈服务
	healingService := NewSelfHealingService(db, dockerExecutor, notifyService)
	
	// 启动自愈监控
	ctx := context.Background()
	if err := healingService.Start(ctx); err != nil {
		log.Fatalf("Failed to start healing service: %v", err)
	}
	defer healingService.Stop()
	
	// 注册需要监控的容器
	containerID := "example-container-id"
	config := &HealingConfig{
		CheckInterval:    30,  // 30秒检查一次
		CheckTimeout:     10,  // 10秒超时
		FailureThreshold: 3,   // 连续失败3次触发自愈
		MaxRestarts:      5,   // 5分钟内最多重启5次
		RestartWindow:    300, // 5分钟时间窗口
		AutoRestart:      true,
		SendAlert:        true,
	}
	
	if err := healingService.RegisterContainer(ctx, containerID, config); err != nil {
		log.Fatalf("Failed to register container: %v", err)
	}
	
	fmt.Printf("Container %s registered for self-healing monitoring\n", containerID)
	
	// 查询容器健康状态
	time.Sleep(1 * time.Minute)
	
	health, err := healingService.GetContainerHealth(ctx, containerID)
	if err != nil {
		log.Printf("Failed to get container health: %v", err)
	} else {
		fmt.Printf("Container Health Status:\n")
		fmt.Printf("  Status: %s\n", health.Status)
		fmt.Printf("  Last Check: %s\n", health.LastCheckTime.Format(time.RFC3339))
		fmt.Printf("  Consecutive Failures: %d\n", health.ConsecutiveFailures)
		fmt.Printf("  Total Restarts: %d\n", health.TotalRestarts)
		fmt.Printf("  Message: %s\n", health.Message)
	}
	
	// 查询故障历史
	failures, err := healingService.GetFailureHistory(ctx, containerID, 10)
	if err != nil {
		log.Printf("Failed to get failure history: %v", err)
	} else {
		fmt.Printf("\nFailure History (%d records):\n", len(failures))
		for _, failure := range failures {
			fmt.Printf("  [%s] %s: %s\n", 
				failure.DetectedAt.Format(time.RFC3339),
				failure.FailureType,
				failure.ErrorMessage)
		}
	}
	
	// 取消注册
	if err := healingService.UnregisterContainer(ctx, containerID); err != nil {
		log.Printf("Failed to unregister container: %v", err)
	}
}

// ExampleIntegratedDeployment 演示集成自愈功能的完整部署流程
func ExampleIntegratedDeployment() {
	// 假设已经有数据库连接
	var db *gorm.DB // 实际使用时需要初始化
	
	// 创建服务
	composeService := NewComposeService(db)
	dockerExecutor := NewDockerExecutor()
	notifyService := NewSimpleNotificationService()
	
	// 创建自愈服务并启动
	healingService := NewSelfHealingService(db, dockerExecutor, notifyService)
	ctx := context.Background()
	healingService.Start(ctx)
	defer healingService.Stop()
	
	// 创建部署服务
	deploymentService := NewDeploymentService(db, composeService, dockerExecutor)
	
	// 设置自愈服务（如果部署服务支持）
	if ds, ok := deploymentService.(*deploymentServiceImpl); ok {
		ds.SetHealingService(healingService)
	}
	
	// 创建项目
	project := &ComposeProject{
		Name:    "my-app",
		Content: `
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
    restart: always
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
      timeout: 10s
      retries: 3
  
  redis:
    image: redis:latest
    restart: unless-stopped
`,
		UserID:   1,
		TenantID: 1,
	}
	
	if err := composeService.CreateProject(ctx, project); err != nil {
		log.Fatalf("Failed to create project: %v", err)
	}
	
	// 部署项目
	deployConfig := &DeploymentConfig{
		Strategy:           DeployStrategyRecreate,
		HealthCheckDelay:   10,
		HealthCheckRetries: 3,
		RollbackOnFailure:  true,
	}
	
	deployment, err := deploymentService.Deploy(ctx, project.ID, deployConfig)
	if err != nil {
		log.Fatalf("Failed to deploy: %v", err)
	}
	
	fmt.Printf("Deployment started: ID=%d, Status=%s\n", deployment.ID, deployment.Status)
	
	// 等待部署完成
	time.Sleep(30 * time.Second)
	
	// 查询部署状态
	deployment, err = deploymentService.GetDeployment(ctx, deployment.ID)
	if err != nil {
		log.Fatalf("Failed to get deployment: %v", err)
	}
	
	fmt.Printf("Deployment completed: Status=%s, Progress=%d%%\n", 
		deployment.Status, deployment.Progress)
	
	// 查询服务实例
	instances, err := deploymentService.GetServiceInstances(ctx, deployment.ID)
	if err != nil {
		log.Fatalf("Failed to get service instances: %v", err)
	}
	
	fmt.Printf("\nService Instances (%d):\n", len(instances))
	for _, instance := range instances {
		fmt.Printf("  Service: %s, Container: %s, Status: %s, Health: %s\n",
			instance.ServiceName, instance.ContainerID, instance.Status, instance.Health)
		
		// 查询容器健康状态
		health, err := healingService.GetContainerHealth(ctx, instance.ContainerID)
		if err == nil {
			fmt.Printf("    Self-Healing Status: %s, Failures: %d, Restarts: %d\n",
				health.Status, health.ConsecutiveFailures, health.TotalRestarts)
		}
	}
}

// ExampleCustomHealingConfig 演示自定义自愈配置
func ExampleCustomHealingConfig() {
	// 为关键服务配置更激进的自愈策略
	criticalServiceConfig := &HealingConfig{
		CheckInterval:    10,  // 10秒检查一次（更频繁）
		CheckTimeout:     5,   // 5秒超时
		FailureThreshold: 2,   // 连续失败2次就触发（更敏感）
		MaxRestarts:      10,  // 允许更多重启次数
		RestartWindow:    600, // 10分钟时间窗口
		AutoRestart:      true,
		SendAlert:        true,
	}
	
	// 为非关键服务配置更宽松的策略
	nonCriticalServiceConfig := &HealingConfig{
		CheckInterval:    60,  // 60秒检查一次
		CheckTimeout:     15,  // 15秒超时
		FailureThreshold: 5,   // 连续失败5次才触发
		MaxRestarts:      3,   // 限制重启次数
		RestartWindow:    300, // 5分钟时间窗口
		AutoRestart:      true,
		SendAlert:        false, // 不发送告警
	}
	
	// 为只读服务配置仅告警不重启的策略
	readOnlyServiceConfig := &HealingConfig{
		CheckInterval:    30,
		CheckTimeout:     10,
		FailureThreshold: 3,
		MaxRestarts:      0,
		RestartWindow:    300,
		AutoRestart:      false, // 不自动重启
		SendAlert:        true,  // 只发送告警
	}
	
	fmt.Printf("Critical Service Config: %+v\n", criticalServiceConfig)
	fmt.Printf("Non-Critical Service Config: %+v\n", nonCriticalServiceConfig)
	fmt.Printf("Read-Only Service Config: %+v\n", readOnlyServiceConfig)
}

// ExampleFailureAnalysis 演示故障分析
func ExampleFailureAnalysis() {
	var db *gorm.DB // 实际使用时需要初始化
	dockerExecutor := NewDockerExecutor()
	notifyService := NewSimpleNotificationService()
	
	healingService := NewSelfHealingService(db, dockerExecutor, notifyService)
	ctx := context.Background()
	
	containerID := "example-container-id"
	
	// 获取故障历史
	failures, err := healingService.GetFailureHistory(ctx, containerID, 50)
	if err != nil {
		log.Fatalf("Failed to get failure history: %v", err)
	}
	
	// 分析故障模式
	failureTypes := make(map[string]int)
	totalRestarts := 0
	totalFailures := len(failures)
	
	for _, failure := range failures {
		failureTypes[failure.FailureType]++
		if failure.Action == "restart" && failure.ActionResult == "success" {
			totalRestarts++
		}
	}
	
	fmt.Printf("Failure Analysis for Container %s:\n", containerID)
	fmt.Printf("  Total Failures: %d\n", totalFailures)
	fmt.Printf("  Successful Restarts: %d\n", totalRestarts)
	fmt.Printf("  Failure Types:\n")
	for failureType, count := range failureTypes {
		percentage := float64(count) / float64(totalFailures) * 100
		fmt.Printf("    %s: %d (%.1f%%)\n", failureType, count, percentage)
	}
	
	// 计算平均故障间隔时间（MTBF）
	if len(failures) > 1 {
		totalDuration := failures[0].DetectedAt.Sub(failures[len(failures)-1].DetectedAt)
		mtbf := totalDuration / time.Duration(len(failures)-1)
		fmt.Printf("  Mean Time Between Failures: %s\n", mtbf)
	}
	
	// 识别最近的故障趋势
	recentFailures := 0
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	for _, failure := range failures {
		if failure.DetectedAt.After(oneHourAgo) {
			recentFailures++
		}
	}
	
	fmt.Printf("  Recent Failures (last hour): %d\n", recentFailures)
	
	if recentFailures > 5 {
		fmt.Println("  ⚠️  WARNING: High failure rate detected!")
	}
}
