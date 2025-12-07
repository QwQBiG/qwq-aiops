# 应用模板系统

应用模板系统是 qwq AIOps 平台的核心组件之一，提供标准化的应用部署模板管理功能。

## 功能特性

### 1. 标准化模板格式

支持两种主流的容器编排格式：

- **Docker Compose**: 适用于单机和小规模部署
- **Helm Chart**: 适用于 Kubernetes 集群部署

### 2. 模板解析和验证

- 自动解析 YAML/JSON 格式的模板内容
- 验证模板结构的完整性和正确性
- 检查必要字段和依赖关系
- 提供详细的错误信息

### 3. 参数化配置

支持灵活的参数定义：

- **参数类型**: string, int, bool, select, password, path
- **默认值**: 为参数提供合理的默认值
- **验证规则**: 支持正则表达式验证
- **选项约束**: 限制参数的可选值范围
- **参数分组**: 组织复杂的参数配置

### 4. 模板渲染

- 使用 `{{.ParameterName}}` 占位符语法
- 自动替换参数值
- 合并默认值和用户提供的值
- 生成可直接部署的配置文件

## 核心组件

### 数据模型

#### AppTemplate - 应用模板

```go
type AppTemplate struct {
    ID          uint           // 模板ID
    Name        string         // 模板名称（唯一标识）
    DisplayName string         // 显示名称
    Description string         // 描述
    Category    AppCategory    // 分类
    Type        TemplateType   // 模板类型（docker-compose/helm-chart）
    Version     string         // 版本号
    Content     string         // 模板内容（YAML/JSON）
    Parameters  string         // 参数定义（JSON）
    Status      TemplateStatus // 状态（draft/published/archived）
}
```

#### TemplateParameter - 模板参数

```go
type TemplateParameter struct {
    Name         string        // 参数名称
    DisplayName  string        // 显示名称
    Description  string        // 描述
    Type         ParameterType // 参数类型
    DefaultValue interface{}   // 默认值
    Required     bool          // 是否必填
    Options      []string      // 选项（用于 select 类型）
    Validation   string        // 验证规则（正则表达式）
}
```

#### ApplicationInstance - 应用实例

```go
type ApplicationInstance struct {
    ID         uint         // 实例ID
    Name       string       // 实例名称
    TemplateID uint         // 模板ID
    Version    string       // 使用的模板版本
    Status     string       // 状态（running/stopped/error）
    Config     string       // 实例配置（JSON）
    UserID     uint         // 用户ID
    TenantID   uint         // 租户ID
}
```

### 服务接口

#### AppStoreService

```go
type AppStoreService interface {
    // 模板管理
    CreateTemplate(ctx context.Context, template *AppTemplate) error
    GetTemplate(ctx context.Context, id uint) (*AppTemplate, error)
    ListTemplates(ctx context.Context, category AppCategory, status TemplateStatus) ([]*AppTemplate, error)
    UpdateTemplate(ctx context.Context, template *AppTemplate) error
    DeleteTemplate(ctx context.Context, id uint) error
    
    // 模板操作
    ValidateTemplate(ctx context.Context, template *AppTemplate) error
    RenderTemplate(ctx context.Context, templateID uint, params map[string]interface{}) (string, error)
    
    // 应用实例管理
    CreateInstance(ctx context.Context, instance *ApplicationInstance) error
    GetInstance(ctx context.Context, id uint) (*ApplicationInstance, error)
    ListInstances(ctx context.Context, userID, tenantID uint) ([]*ApplicationInstance, error)
    UpdateInstance(ctx context.Context, instance *ApplicationInstance) error
    DeleteInstance(ctx context.Context, id uint) error
    
    // 初始化内置模板
    InitBuiltinTemplates(ctx context.Context) error
}
```

#### TemplateService

```go
type TemplateService struct{}

// 核心方法
func (s *TemplateService) ParseTemplate(templateType TemplateType, content string) (map[string]interface{}, error)
func (s *TemplateService) ValidateTemplate(template *AppTemplate) error
func (s *TemplateService) RenderTemplate(template *AppTemplate, params map[string]interface{}) (string, error)
func (s *TemplateService) ValidateParameters(paramDefs []TemplateParameter, params map[string]interface{}) error
func (s *TemplateService) ExtractParameters(content string) []string
```

## 使用示例

### 1. 初始化服务

```go
import (
    "qwq/internal/appstore"
    "gorm.io/gorm"
)

// 创建应用商店服务
appStoreService := appstore.NewAppStoreService(db)

// 初始化内置模板
err := appStoreService.InitBuiltinTemplates(ctx)
```

### 2. 列出模板

```go
// 列出所有已发布的数据库模板
templates, err := appStoreService.ListTemplates(
    ctx,
    appstore.CategoryDatabase,
    appstore.TemplateStatusPublished,
)

for _, template := range templates {
    fmt.Printf("%s - %s\n", template.DisplayName, template.Description)
}
```

### 3. 获取模板详情

```go
// 根据名称获取模板
template, err := appStoreService.GetTemplateByName(ctx, "nginx")

// 解析参数定义
params, err := appstore.ParseTemplateParameters(template.Parameters)

for _, param := range params {
    fmt.Printf("%s (%s): %v\n", param.DisplayName, param.Type, param.DefaultValue)
}
```

### 4. 渲染模板

```go
// 准备参数
params := map[string]interface{}{
    "port":       8080,
    "https_port": 8443,
    "html_path":  "/var/www/html",
}

// 渲染模板
rendered, err := appStoreService.RenderTemplate(ctx, template.ID, params)

// rendered 包含可直接使用的 Docker Compose 配置
fmt.Println(rendered)
```

### 5. 创建应用实例

```go
instance := &appstore.ApplicationInstance{
    Name:       "my-nginx-server",
    TemplateID: template.ID,
    Status:     "running",
    Config:     `{"port": 8080, "https_port": 8443}`,
    UserID:     1,
    TenantID:   1,
}

err := appStoreService.CreateInstance(ctx, instance)
```

### 6. 创建自定义模板

```go
customTemplate := &appstore.AppTemplate{
    Name:        "my-custom-app",
    DisplayName: "我的自定义应用",
    Description: "自定义应用模板",
    Category:    appstore.CategoryOther,
    Type:        appstore.TemplateTypeDockerCompose,
    Version:     "1.0.0",
    Status:      appstore.TemplateStatusDraft,
    Content: `version: '3.8'
services:
  app:
    image: {{.image}}
    ports:
      - "{{.port}}:{{.port}}"
    environment:
      APP_ENV: {{.env}}
`,
    Parameters: `[
        {
            "name": "image",
            "display_name": "镜像名称",
            "type": "string",
            "required": true
        },
        {
            "name": "port",
            "display_name": "端口",
            "type": "int",
            "default_value": 3000,
            "required": true
        },
        {
            "name": "env",
            "display_name": "环境",
            "type": "select",
            "options": ["development", "production"],
            "default_value": "production",
            "required": true
        }
    ]`,
}

err := appStoreService.CreateTemplate(ctx, customTemplate)
```

## 内置模板

系统提供以下内置模板：

### Web 服务器

- **Nginx**: 高性能的 HTTP 和反向代理服务器

### 数据库

- **MySQL**: 流行的开源关系型数据库
- **PostgreSQL**: 强大的开源对象关系型数据库
- **Redis**: 高性能的内存数据库和缓存

### 监控工具

- **Prometheus**: 开源的监控和告警工具

## 参数类型说明

### string - 字符串

普通文本输入，适用于名称、路径等。

```json
{
    "name": "app_name",
    "type": "string",
    "default_value": "myapp"
}
```

### int - 整数

数字输入，适用于端口、数量等。

```json
{
    "name": "port",
    "type": "int",
    "default_value": 8080
}
```

### bool - 布尔值

开关选项，true 或 false。

```json
{
    "name": "enable_ssl",
    "type": "bool",
    "default_value": true
}
```

### select - 下拉选择

从预定义的选项中选择。

```json
{
    "name": "environment",
    "type": "select",
    "options": ["dev", "staging", "production"],
    "default_value": "production"
}
```

### password - 密码

敏感信息输入，UI 会隐藏显示。

```json
{
    "name": "db_password",
    "type": "password",
    "required": true,
    "validation": "^.{8,}$"
}
```

### path - 路径

文件或目录路径。

```json
{
    "name": "data_path",
    "type": "path",
    "default_value": "./data"
}
```

## 验证规则

### 必填验证

```json
{
    "name": "required_field",
    "required": true
}
```

### 正则表达式验证

```json
{
    "name": "email",
    "type": "string",
    "validation": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
}
```

### 选项约束

```json
{
    "name": "size",
    "type": "select",
    "options": ["small", "medium", "large"]
}
```

## 错误处理

系统定义了以下错误类型：

- `ErrInvalidTemplateType`: 无效的模板类型
- `ErrInvalidTemplateContent`: 无效的模板内容
- `ErrMissingRequiredParameter`: 缺少必填参数
- `ErrInvalidParameterValue`: 无效的参数值
- `ErrParameterValidationFailed`: 参数验证失败
- `ErrTemplateNotFound`: 模板未找到
- `ErrTemplateAlreadyExists`: 模板已存在
- `ErrInstanceNotFound`: 实例未找到

## 最佳实践

### 1. 模板设计

- 使用清晰的参数命名
- 提供合理的默认值
- 添加详细的描述信息
- 使用参数分组组织复杂配置

### 2. 参数验证

- 为敏感参数添加验证规则
- 使用 select 类型限制可选值
- 标记必填参数

### 3. 版本管理

- 使用语义化版本号
- 记录版本变更
- 保持向后兼容性

### 4. 安全考虑

- 密码参数使用 password 类型
- 验证用户输入
- 限制资源访问权限

## 扩展开发

### 添加新的模板类型

1. 在 `TemplateType` 中添加新类型
2. 实现对应的验证逻辑
3. 更新 `ParseTemplate` 方法

### 添加新的参数类型

1. 在 `ParameterType` 中添加新类型
2. 实现类型验证逻辑
3. 更新前端 UI 组件

### 自定义验证规则

扩展 `ValidateParameters` 方法，添加自定义验证逻辑。

## 测试

运行示例代码：

```go
import "qwq/internal/appstore"

// 在测试或初始化代码中调用
appstore.ExampleUsage(db)
```

## 相关文档

- [Docker Compose 文档](https://docs.docker.com/compose/)
- [Helm Chart 文档](https://helm.sh/docs/)
- [YAML 语法](https://yaml.org/)

## 许可证

本项目采用 MIT 许可证。
