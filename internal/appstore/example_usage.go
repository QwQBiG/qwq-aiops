package appstore

import (
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// ExampleUsage 展示应用模板系统的使用方法
func ExampleUsage(db *gorm.DB) {
	ctx := context.Background()

	// 创建应用商店服务
	appStoreService := NewAppStoreService(db)

	// 1. 初始化内置模板
	fmt.Println("=== 初始化内置模板 ===")
	if err := appStoreService.InitBuiltinTemplates(ctx); err != nil {
		log.Printf("初始化内置模板失败: %v\n", err)
		return
	}
	fmt.Println("✓ 内置模板初始化成功")

	// 2. 列出所有已发布的模板
	fmt.Println("\n=== 列出所有已发布的模板 ===")
	templates, err := appStoreService.ListTemplates(ctx, "", TemplateStatusPublished)
	if err != nil {
		log.Printf("列出模板失败: %v\n", err)
		return
	}

	for _, template := range templates {
		fmt.Printf("- %s (%s) - %s\n", template.DisplayName, template.Name, template.Description)
	}

	// 3. 获取 Nginx 模板
	fmt.Println("\n=== 获取 Nginx 模板 ===")
	nginxTemplate, err := appStoreService.GetTemplateByName(ctx, "nginx")
	if err != nil {
		log.Printf("获取 Nginx 模板失败: %v\n", err)
		return
	}
	fmt.Printf("模板名称: %s\n", nginxTemplate.DisplayName)
	fmt.Printf("版本: %s\n", nginxTemplate.Version)
	fmt.Printf("分类: %s\n", nginxTemplate.Category)

	// 4. 解析模板参数
	fmt.Println("\n=== 解析模板参数 ===")
	params, err := ParseTemplateParameters(nginxTemplate.Parameters)
	if err != nil {
		log.Printf("解析参数失败: %v\n", err)
		return
	}

	for _, param := range params {
		fmt.Printf("- %s (%s): %s [默认值: %v]\n",
			param.DisplayName,
			param.Name,
			param.Description,
			param.DefaultValue,
		)
	}

	// 5. 使用自定义参数渲染模板
	fmt.Println("\n=== 渲染 Nginx 模板 ===")
	customParams := map[string]interface{}{
		"port":       8080,
		"https_port": 8443,
		"html_path":  "/var/www/html",
	}

	rendered, err := appStoreService.RenderTemplate(ctx, nginxTemplate.ID, customParams)
	if err != nil {
		log.Printf("渲染模板失败: %v\n", err)
		return
	}
	fmt.Println("渲染后的 Docker Compose 内容:")
	fmt.Println(rendered)

	// 6. 创建应用实例
	fmt.Println("\n=== 创建应用实例 ===")
	instance := &ApplicationInstance{
		Name:       "my-nginx-server",
		TemplateID: nginxTemplate.ID,
		Status:     "running",
		Config:     `{"port": 8080, "https_port": 8443, "html_path": "/var/www/html"}`,
		UserID:     1,
		TenantID:   1,
	}

	if err := appStoreService.CreateInstance(ctx, instance); err != nil {
		log.Printf("创建实例失败: %v\n", err)
		return
	}
	fmt.Printf("✓ 应用实例创建成功，ID: %d\n", instance.ID)

	// 7. 列出用户的所有应用实例
	fmt.Println("\n=== 列出用户的应用实例 ===")
	instances, err := appStoreService.ListInstances(ctx, 1, 1)
	if err != nil {
		log.Printf("列出实例失败: %v\n", err)
		return
	}

	for _, inst := range instances {
		fmt.Printf("- %s (模板: %s, 状态: %s)\n",
			inst.Name,
			inst.Template.DisplayName,
			inst.Status,
		)
	}

	// 8. 创建自定义模板
	fmt.Println("\n=== 创建自定义模板 ===")
	customTemplate := &AppTemplate{
		Name:        "custom-app",
		DisplayName: "自定义应用",
		Description: "这是一个自定义的应用模板",
		Category:    CategoryOther,
		Type:        TemplateTypeDockerCompose,
		Version:     "1.0.0",
		Author:      "user",
		Status:      TemplateStatusDraft,
		Content: `version: '3.8'
services:
  app:
    image: {{.image}}
    container_name: {{.container_name}}
    ports:
      - "{{.port}}:{{.port}}"
    environment:
      APP_ENV: {{.env}}
    restart: unless-stopped
`,
		Parameters: `[
			{
				"name": "image",
				"display_name": "镜像名称",
				"description": "Docker 镜像",
				"type": "string",
				"required": true
			},
			{
				"name": "container_name",
				"display_name": "容器名称",
				"description": "容器的名称",
				"type": "string",
				"required": true
			},
			{
				"name": "port",
				"display_name": "端口",
				"description": "应用端口",
				"type": "int",
				"default_value": 3000,
				"required": true
			},
			{
				"name": "env",
				"display_name": "环境",
				"description": "运行环境",
				"type": "select",
				"options": ["development", "production"],
				"default_value": "production",
				"required": true
			}
		]`,
	}

	if err := appStoreService.CreateTemplate(ctx, customTemplate); err != nil {
		log.Printf("创建自定义模板失败: %v\n", err)
		return
	}
	fmt.Printf("✓ 自定义模板创建成功，ID: %d\n", customTemplate.ID)

	// 9. 验证模板
	fmt.Println("\n=== 验证模板 ===")
	if err := appStoreService.ValidateTemplate(ctx, customTemplate); err != nil {
		log.Printf("模板验证失败: %v\n", err)
		return
	}
	fmt.Println("✓ 模板验证通过")

	// 10. 测试参数验证
	fmt.Println("\n=== 测试参数验证 ===")
	
	// 测试缺少必填参数
	invalidParams := map[string]interface{}{
		"port": 3000,
	}
	_, err = appStoreService.RenderTemplate(ctx, customTemplate.ID, invalidParams)
	if err != nil {
		fmt.Printf("✓ 正确检测到缺少必填参数: %v\n", err)
	}

	// 测试无效的参数类型
	invalidTypeParams := map[string]interface{}{
		"image":          "myapp:latest",
		"container_name": "myapp",
		"port":           "not-a-number", // 应该是整数
		"env":            "production",
	}
	_, err = appStoreService.RenderTemplate(ctx, customTemplate.ID, invalidTypeParams)
	if err != nil {
		fmt.Printf("✓ 正确检测到无效的参数类型: %v\n", err)
	}

	// 测试无效的选项值
	invalidOptionParams := map[string]interface{}{
		"image":          "myapp:latest",
		"container_name": "myapp",
		"port":           3000,
		"env":            "staging", // 不在允许的选项中
	}
	_, err = appStoreService.RenderTemplate(ctx, customTemplate.ID, invalidOptionParams)
	if err != nil {
		fmt.Printf("✓ 正确检测到无效的选项值: %v\n", err)
	}

	// 测试有效参数
	validParams := map[string]interface{}{
		"image":          "myapp:latest",
		"container_name": "myapp",
		"port":           3000,
		"env":            "production",
	}
	rendered, err = appStoreService.RenderTemplate(ctx, customTemplate.ID, validParams)
	if err != nil {
		log.Printf("渲染失败: %v\n", err)
		return
	}
	fmt.Println("\n✓ 使用有效参数渲染成功:")
	fmt.Println(rendered)

	fmt.Println("\n=== 示例完成 ===")
}
