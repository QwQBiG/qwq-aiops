package container

import (
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// ExampleComposeParser 演示 Compose 解析器的使用
func ExampleComposeParser() {
	parser := NewComposeParser()

	// 示例 Compose 文件内容
	composeContent := `version: '3.8'

services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    volumes:
      - ./html:/usr/share/nginx/html
    networks:
      - frontend
    restart: always
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
      timeout: 10s
      retries: 3

  api:
    build:
      context: ./api
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production
      - DB_HOST=db
    depends_on:
      - db
    networks:
      - frontend
      - backend
    restart: unless-stopped

  db:
    image: postgres:15
    environment:
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=myapp
    volumes:
      - db_data:/var/lib/postgresql/data
    networks:
      - backend
    restart: always

networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge

volumes:
  db_data:
    driver: local
`

	// 1. 解析 Compose 文件
	fmt.Println("=== 解析 Compose 文件 ===")
	config, err := parser.Parse(composeContent)
	if err != nil {
		log.Fatalf("解析失败: %v", err)
	}
	fmt.Printf("版本: %s\n", config.Version)
	fmt.Printf("服务数量: %d\n", len(config.Services))
	fmt.Printf("网络数量: %d\n", len(config.Networks))
	fmt.Printf("卷数量: %d\n\n", len(config.Volumes))

	// 2. 验证配置
	fmt.Println("=== 验证配置 ===")
	validationResult := parser.Validate(config)
	if validationResult.Valid {
		fmt.Println("✓ 配置有效")
	} else {
		fmt.Printf("✗ 配置无效，发现 %d 个错误:\n", len(validationResult.Errors))
		for _, err := range validationResult.Errors {
			fmt.Printf("  - %s: %s\n", err.Field, err.Message)
		}
	}
	fmt.Println()

	// 3. 渲染配置
	fmt.Println("=== 渲染配置 ===")
	rendered, err := parser.Render(config)
	if err != nil {
		log.Fatalf("渲染失败: %v", err)
	}
	fmt.Println("渲染成功，YAML 长度:", len(rendered))
	fmt.Println()

	// 4. 获取自动补全建议
	fmt.Println("=== 自动补全建议 ===")
	completions := parser.GetCompletions("services:\n  web:\n    ", 0)
	fmt.Printf("找到 %d 个补全建议:\n", len(completions))
	for i, item := range completions {
		if i < 5 { // 只显示前5个
			fmt.Printf("  - %s (%s): %s\n", item.Label, item.Kind, item.Detail)
		}
	}
}

// ExampleComposeService 演示 Compose 服务的使用
func ExampleComposeService(db *gorm.DB) {
	ctx := context.Background()
	service := NewComposeService(db)

	// 1. 创建项目
	fmt.Println("=== 创建 Compose 项目 ===")
	project := &ComposeProject{
		Name:        "my-web-app",
		DisplayName: "我的 Web 应用",
		Description: "一个简单的 Web 应用示例",
		Content: `version: '3.8'

services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    restart: always

  redis:
    image: redis:7
    ports:
      - "6379:6379"
    restart: always
`,
		UserID:   1,
		TenantID: 1,
	}

	if err := service.CreateProject(ctx, project); err != nil {
		log.Fatalf("创建项目失败: %v", err)
	}
	fmt.Printf("✓ 项目创建成功，ID: %d\n\n", project.ID)

	// 2. 获取项目
	fmt.Println("=== 获取项目 ===")
	retrievedProject, err := service.GetProject(ctx, project.ID)
	if err != nil {
		log.Fatalf("获取项目失败: %v", err)
	}
	fmt.Printf("项目名称: %s\n", retrievedProject.Name)
	fmt.Printf("项目状态: %s\n", retrievedProject.Status)
	fmt.Printf("Compose 版本: %s\n\n", retrievedProject.Version)

	// 3. 验证 Compose 文件
	fmt.Println("=== 验证 Compose 文件 ===")
	validationResult, err := service.ValidateComposeFile(ctx, project.Content)
	if err != nil {
		log.Fatalf("验证失败: %v", err)
	}
	if validationResult.Valid {
		fmt.Println("✓ Compose 文件有效")
	} else {
		fmt.Printf("✗ Compose 文件无效，发现 %d 个错误\n", len(validationResult.Errors))
	}
	fmt.Println()

	// 4. 获取项目结构（用于可视化编辑）
	fmt.Println("=== 获取项目结构 ===")
	config, err := service.GetProjectStructure(ctx, project.ID)
	if err != nil {
		log.Fatalf("获取项目结构失败: %v", err)
	}
	fmt.Printf("服务列表:\n")
	for serviceName, svc := range config.Services {
		fmt.Printf("  - %s: %s\n", serviceName, svc.Image)
	}
	fmt.Println()

	// 5. 修改项目结构
	fmt.Println("=== 修改项目结构 ===")
	// 添加一个新服务
	config.Services["postgres"] = &Service{
		Image: "postgres:15",
		Environment: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_DB=myapp",
		},
		Restart: "always",
	}

	if err := service.UpdateProjectStructure(ctx, project.ID, config); err != nil {
		log.Fatalf("更新项目结构失败: %v", err)
	}
	fmt.Println("✓ 项目结构更新成功")
	fmt.Println()

	// 6. 列出所有项目
	fmt.Println("=== 列出所有项目 ===")
	projects, err := service.ListProjects(ctx, 1, 1)
	if err != nil {
		log.Fatalf("列出项目失败: %v", err)
	}
	fmt.Printf("找到 %d 个项目:\n", len(projects))
	for _, p := range projects {
		fmt.Printf("  - %s (%s)\n", p.Name, p.Status)
	}
	fmt.Println()

	// 7. 获取自动补全建议
	fmt.Println("=== 获取自动补全建议 ===")
	completions, err := service.GetCompletions(ctx, "services:\n  web:\n    ", 0)
	if err != nil {
		log.Fatalf("获取补全建议失败: %v", err)
	}
	fmt.Printf("找到 %d 个补全建议\n", len(completions))
}

// ExampleInvalidComposeFile 演示无效 Compose 文件的处理
func ExampleInvalidComposeFile() {
	parser := NewComposeParser()

	// 无效的 Compose 文件示例
	invalidContent := `version: '3.8'

services:
  web:
    # 缺少 image 或 build
    ports:
      - "invalid-port"  # 无效的端口格式
    restart: invalid-policy  # 无效的重启策略
    networks:
      - undefined-network  # 未定义的网络
`

	fmt.Println("=== 解析无效的 Compose 文件 ===")
	config, err := parser.Parse(invalidContent)
	if err != nil {
		fmt.Printf("✗ 解析失败: %v\n\n", err)
		return
	}

	fmt.Println("=== 验证配置 ===")
	validationResult := parser.Validate(config)
	if !validationResult.Valid {
		fmt.Printf("✗ 发现 %d 个验证错误:\n", len(validationResult.Errors))
		for _, err := range validationResult.Errors {
			fmt.Printf("  - [%s] %s\n", err.Field, err.Message)
		}
	}
}

// ExampleComposeWithAdvancedFeatures 演示高级功能的 Compose 文件
func ExampleComposeWithAdvancedFeatures() {
	parser := NewComposeParser()

	advancedContent := `version: '3.8'

services:
  web:
    image: nginx:latest
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
        window: 120s
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    networks:
      frontend:
        aliases:
          - web-service
    labels:
      com.example.description: "Web服务"
      com.example.version: "1.0"

networks:
  frontend:
    driver: bridge
    driver_opts:
      com.docker.network.bridge.name: br-frontend
    labels:
      com.example.network: "frontend"
`

	fmt.Println("=== 解析高级功能 Compose 文件 ===")
	config, err := parser.Parse(advancedContent)
	if err != nil {
		log.Fatalf("解析失败: %v", err)
	}

	fmt.Printf("版本: %s\n", config.Version)
	
	// 检查部署配置
	if webService, ok := config.Services["web"]; ok {
		if webService.Deploy != nil {
			fmt.Printf("副本数: %d\n", webService.Deploy.Replicas)
			if webService.Deploy.Resources != nil && webService.Deploy.Resources.Limits != nil {
				fmt.Printf("CPU 限制: %s\n", webService.Deploy.Resources.Limits.CPUs)
				fmt.Printf("内存限制: %s\n", webService.Deploy.Resources.Limits.Memory)
			}
		}
		
		// 检查健康检查
		if webService.HealthCheck != nil {
			fmt.Printf("健康检查间隔: %s\n", webService.HealthCheck.Interval)
			fmt.Printf("健康检查超时: %s\n", webService.HealthCheck.Timeout)
		}
	}

	// 验证配置
	validationResult := parser.Validate(config)
	if validationResult.Valid {
		fmt.Println("✓ 高级配置验证通过")
	} else {
		fmt.Printf("✗ 发现 %d 个验证错误\n", len(validationResult.Errors))
	}
}
