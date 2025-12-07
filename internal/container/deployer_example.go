package container

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// ExampleDeployment 演示如何使用部署引擎
func ExampleDeployment(db *gorm.DB) {
	ctx := context.Background()
	
	// 创建服务
	composeService := NewComposeService(db)
	
	// 1. 创建一个 Compose 项目
	project := &ComposeProject{
		Name:        "my-web-app",
		DisplayName: "我的Web应用",
		Description: "一个简单的 Nginx + MySQL 应用",
		Content: `version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    depends_on:
      - db
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
      timeout: 10s
      retries: 3
  
  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: secret
      MYSQL_DATABASE: myapp
    volumes:
      - db_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  db_data:
`,
		Status:   ProjectStatusDraft,
		UserID:   1,
		TenantID: 1,
	}
	
	if err := composeService.CreateProject(ctx, project); err != nil {
		log.Fatalf("创建项目失败: %v", err)
	}
	
	fmt.Printf("✓ 项目创建成功，ID: %d\n", project.ID)
	
	// 2. 使用重建策略部署
	fmt.Println("\n=== 场景 1: 重建策略部署 ===")
	deployConfig := &DeploymentConfig{
		Strategy:           DeployStrategyRecreate,
		HealthCheckDelay:   10,
		HealthCheckRetries: 3,
		RollbackOnFailure:  true,
	}
	
	deployment, err := composeService.Deploy(ctx, project.ID, deployConfig)
	if err != nil {
		log.Fatalf("部署失败: %v", err)
	}
	
	fmt.Printf("✓ 部署已启动，ID: %d, 版本: %s\n", deployment.ID, deployment.Version)
	
	// 监控部署进度
	monitorDeployment(ctx, composeService, deployment.ID)
	
	// 3. 使用滚动更新策略更新
	fmt.Println("\n=== 场景 2: 滚动更新部署 ===")
	
	// 更新项目配置（例如修改镜像版本）
	project.Content = `version: '3.8'
services:
  web:
    image: nginx:1.21
    ports:
      - "8080:80"
    depends_on:
      - db
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
      timeout: 10s
      retries: 3
  
  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: secret
      MYSQL_DATABASE: myapp
    volumes:
      - db_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  db_data:
`
	
	if err := composeService.UpdateProject(ctx, project); err != nil {
		log.Fatalf("更新项目失败: %v", err)
	}
	
	rollingConfig := &DeploymentConfig{
		Strategy:           DeployStrategyRollingUpdate,
		MaxSurge:           1,
		MaxUnavailable:     0,
		HealthCheckDelay:   10,
		HealthCheckRetries: 3,
		RollbackOnFailure:  true,
	}
	
	deployment2, err := composeService.Deploy(ctx, project.ID, rollingConfig)
	if err != nil {
		log.Fatalf("滚动更新失败: %v", err)
	}
	
	fmt.Printf("✓ 滚动更新已启动，ID: %d, 版本: %s\n", deployment2.ID, deployment2.Version)
	monitorDeployment(ctx, composeService, deployment2.ID)
	
	// 4. 使用蓝绿部署策略
	fmt.Println("\n=== 场景 3: 蓝绿部署 ===")
	
	blueGreenConfig := &DeploymentConfig{
		Strategy:           DeployStrategyBlueGreen,
		HealthCheckDelay:   10,
		HealthCheckRetries: 3,
		RollbackOnFailure:  true,
		BlueGreenTimeout:   300,
	}
	
	deployment3, err := composeService.Deploy(ctx, project.ID, blueGreenConfig)
	if err != nil {
		log.Fatalf("蓝绿部署失败: %v", err)
	}
	
	fmt.Printf("✓ 蓝绿部署已启动，ID: %d, 版本: %s\n", deployment3.ID, deployment3.Version)
	monitorDeployment(ctx, composeService, deployment3.ID)
	
	// 5. 演示回滚
	fmt.Println("\n=== 场景 4: 回滚部署 ===")
	
	if err := composeService.RollbackDeployment(ctx, deployment3.ID); err != nil {
		log.Fatalf("回滚失败: %v", err)
	}
	
	fmt.Println("✓ 回滚已启动")
	monitorDeployment(ctx, composeService, deployment3.ID)
	
	// 6. 查看部署历史
	fmt.Println("\n=== 部署历史 ===")
	deployments, err := composeService.ListDeployments(ctx, project.ID)
	if err != nil {
		log.Fatalf("获取部署历史失败: %v", err)
	}
	
	for _, d := range deployments {
		fmt.Printf("部署 #%d: 版本=%s, 策略=%s, 状态=%s, 进度=%d%%\n",
			d.ID, d.Version, d.Strategy, d.Status, d.Progress)
	}
}

// monitorDeployment 监控部署进度
func monitorDeployment(ctx context.Context, service ComposeService, deploymentID uint) {
	fmt.Println("监控部署进度...")
	
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	timeout := time.After(5 * time.Minute)
	
	for {
		select {
		case <-timeout:
			fmt.Println("⚠ 监控超时")
			return
		case <-ticker.C:
			deployment, err := service.GetDeployment(ctx, deploymentID)
			if err != nil {
				fmt.Printf("⚠ 获取部署状态失败: %v\n", err)
				return
			}
			
			fmt.Printf("  状态: %s, 进度: %d%%, 消息: %s\n", 
				deployment.Status, deployment.Progress, deployment.Message)
			
			// 检查是否完成
			if deployment.Status == DeploymentStatusCompleted {
				fmt.Println("✓ 部署成功完成")
				
				// 显示服务实例
				instances, _ := service.(*composeServiceImpl).deploymentService.GetServiceInstances(ctx, deploymentID)
				if len(instances) > 0 {
					fmt.Println("\n服务实例:")
					for _, inst := range instances {
						fmt.Printf("  - %s: %s (状态: %s, 健康: %s)\n",
							inst.ServiceName, inst.ContainerName, inst.Status, inst.Health)
					}
				}
				return
			}
			
			if deployment.Status == DeploymentStatusFailed || 
			   deployment.Status == DeploymentStatusRolledBack {
				fmt.Printf("✗ 部署失败或已回滚: %s\n", deployment.Message)
				
				// 显示部署事件
				events, _ := service.(*composeServiceImpl).deploymentService.GetDeploymentEvents(ctx, deploymentID)
				if len(events) > 0 {
					fmt.Println("\n部署事件:")
					for _, event := range events {
						fmt.Printf("  [%s] %s: %s\n", 
							event.CreatedAt.Format("15:04:05"), event.EventType, event.Message)
					}
				}
				return
			}
		}
	}
}

// ExampleDeploymentStrategies 演示不同的部署策略
func ExampleDeploymentStrategies() {
	fmt.Println("=== 部署策略说明 ===")
	fmt.Println()
	
	fmt.Println("1. 重建策略 (Recreate)")
	fmt.Println("   - 先停止所有旧容器")
	fmt.Println("   - 然后启动所有新容器")
	fmt.Println("   - 优点: 简单直接，资源占用少")
	fmt.Println("   - 缺点: 有停机时间")
	fmt.Println("   - 适用场景: 开发环境、非关键服务")
	fmt.Println()
	
	fmt.Println("2. 滚动更新 (Rolling Update)")
	fmt.Println("   - 逐个服务进行更新")
	fmt.Println("   - 启动新容器，验证健康后停止旧容器")
	fmt.Println("   - 优点: 零停机，渐进式更新")
	fmt.Println("   - 缺点: 更新时间较长，可能出现新旧版本共存")
	fmt.Println("   - 适用场景: 生产环境、需要零停机的服务")
	fmt.Println()
	
	fmt.Println("3. 蓝绿部署 (Blue-Green)")
	fmt.Println("   - 部署完整的新环境（绿色）")
	fmt.Println("   - 验证新环境健康后切换流量")
	fmt.Println("   - 最后清理旧环境（蓝色）")
	fmt.Println("   - 优点: 快速切换，易于回滚")
	fmt.Println("   - 缺点: 需要双倍资源")
	fmt.Println("   - 适用场景: 关键业务、需要快速回滚能力")
}
