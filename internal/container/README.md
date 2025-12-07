# Container 模块

## 概述

Container 模块提供了 Docker Compose 文件的解析、验证和管理功能，是 qwq AIOps 平台容器编排管理的核心组件。

## 主要功能

### 1. Docker Compose 解析器

`ComposeParser` 提供了完整的 Docker Compose 文件解析和验证功能：

- **解析 Compose 文件**: 将 YAML 格式的 Compose 文件解析为结构化的配置对象
- **验证配置**: 检查 Compose 配置的有效性，包括：
  - 版本兼容性检查
  - 服务定义完整性验证
  - 端口映射格式验证
  - 网络和卷引用验证
  - 重启策略验证
  - 健康检查配置验证
- **渲染配置**: 将配置对象渲染为 YAML 格式
- **智能补全**: 提供上下文感知的自动补全建议

### 2. Compose 服务

`ComposeService` 提供了项目管理和操作功能：

- **项目管理**: 创建、读取、更新、删除 Compose 项目
- **文件操作**: 解析、验证、渲染 Compose 文件
- **可视化编辑**: 获取和更新项目结构，支持可视化编辑器
- **智能提示**: 提供自动补全和语法检查

## 数据模型

### ComposeProject

Compose 项目的数据库模型：

```go
type ComposeProject struct {
    ID          uint
    Name        string         // 项目名称（租户内唯一）
    DisplayName string         // 显示名称
    Description string         // 描述
    Content     string         // Compose 文件内容（YAML）
    Version     string         // Compose 文件版本
    Status      ProjectStatus  // 项目状态
    UserID      uint           // 用户ID
    TenantID    uint           // 租户ID
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### ComposeConfig

Compose 配置的结构化表示：

```go
type ComposeConfig struct {
    Version  string                 // Compose 文件版本
    Services map[string]*Service    // 服务定义
    Networks map[string]*Network    // 网络定义
    Volumes  map[string]*Volume     // 卷定义
    Secrets  map[string]*Secret     // 密钥定义
    Configs  map[string]*ConfigItem // 配置定义
}
```

## 使用示例

### 基本用法

```go
// 创建解析器
parser := NewComposeParser()

// 解析 Compose 文件
composeContent := `version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
`

config, err := parser.Parse(composeContent)
if err != nil {
    log.Fatal(err)
}

// 验证配置
result := parser.Validate(config)
if !result.Valid {
    for _, err := range result.Errors {
        fmt.Printf("错误: %s - %s\n", err.Field, err.Message)
    }
}

// 渲染配置
rendered, err := parser.Render(config)
if err != nil {
    log.Fatal(err)
}
```

### 使用服务管理项目

```go
// 创建服务
service := NewComposeService(db)

// 创建项目
project := &ComposeProject{
    Name:        "my-app",
    DisplayName: "我的应用",
    Content:     composeContent,
    UserID:      1,
    TenantID:    1,
}

err := service.CreateProject(ctx, project)
if err != nil {
    log.Fatal(err)
}

// 获取项目结构（用于可视化编辑）
config, err := service.GetProjectStructure(ctx, project.ID)
if err != nil {
    log.Fatal(err)
}

// 修改配置
config.Services["redis"] = &Service{
    Image:   "redis:7",
    Restart: "always",
}

// 更新项目
err = service.UpdateProjectStructure(ctx, project.ID, config)
if err != nil {
    log.Fatal(err)
}
```

### 获取自动补全建议

```go
// 获取补全建议
completions, err := service.GetCompletions(ctx, "services:\n  web:\n    ", 0)
if err != nil {
    log.Fatal(err)
}

for _, item := range completions {
    fmt.Printf("%s (%s): %s\n", item.Label, item.Kind, item.Detail)
}
```

## 支持的功能

### Compose 文件版本

支持 Docker Compose 文件版本 3.x：
- 3.0, 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8, 3.9

### 服务配置

支持的服务配置选项：
- `image`: Docker 镜像
- `build`: 构建配置
- `ports`: 端口映射
- `volumes`: 卷挂载
- `environment`: 环境变量
- `networks`: 网络配置
- `depends_on`: 服务依赖
- `restart`: 重启策略
- `healthcheck`: 健康检查
- `deploy`: 部署配置（资源限制、副本数等）
- `logging`: 日志配置
- `labels`: 标签

### 验证规则

解析器会验证以下内容：

1. **版本验证**: 检查 Compose 文件版本是否支持
2. **服务验证**: 
   - 至少需要一个服务
   - 每个服务必须有 `image` 或 `build`
   - 端口映射格式正确
   - 重启策略有效
   - 健康检查配置完整
3. **引用验证**:
   - 网络引用必须已定义
   - 命名卷引用必须已定义
4. **格式验证**:
   - 端口映射格式: `port`, `host:container`, `ip:host:container`
   - 支持端口范围: `8080-8090:80-90`
   - 支持协议后缀: `/tcp`, `/udp`

### 自动补全

提供以下类型的自动补全：

1. **服务级别属性**: image, build, ports, volumes, environment, depends_on, restart, networks, healthcheck
2. **常用镜像**: nginx, mysql, postgres, redis, mongo
3. **重启策略**: no, always, on-failure, unless-stopped
4. **网络驱动**: bridge, host, overlay

## 错误处理

模块定义了以下错误类型：

- `ErrProjectNotFound`: 项目未找到
- `ErrProjectAlreadyExists`: 项目已存在（同一租户下名称重复）
- `ErrInvalidComposeFile`: 无效的 Compose 文件

验证错误通过 `ValidationResult` 返回：

```go
type ValidationResult struct {
    Valid  bool               // 是否有效
    Errors []*ValidationError // 错误列表
}

type ValidationError struct {
    Field   string // 字段名
    Message string // 错误消息
    Line    int    // 行号（如果适用）
}
```

## 测试

模块包含完整的单元测试，覆盖以下场景：

- 基本解析和验证
- 复杂配置解析
- 错误处理
- 往返测试（解析 -> 渲染 -> 解析）
- 网络和卷引用验证
- 健康检查验证
- 自动补全功能

运行测试：

```bash
go test ./internal/container -v
```

## 容器编排部署引擎

### 3. 部署服务

`DeploymentService` 提供了容器编排的部署、更新和回滚功能：

- **多种部署策略**: 重建、滚动更新、蓝绿部署
- **部署状态监控**: 实时跟踪部署进度和状态
- **健康检查**: 自动验证服务健康状态
- **自动回滚**: 部署失败时自动回滚到上一个版本
- **部署历史**: 完整的部署记录和事件追踪

### 部署策略详解

#### 1. 重建策略 (Recreate)

最简单的部署策略，先停止所有旧容器，然后启动新容器。

**特点:**
- ✓ 实现简单，资源占用少
- ✗ 有停机时间

**适用场景:**
- 开发环境
- 非关键服务
- 资源受限的环境

**使用示例:**
```go
deployConfig := &DeploymentConfig{
    Strategy:           DeployStrategyRecreate,
    HealthCheckDelay:   10,
    HealthCheckRetries: 3,
    RollbackOnFailure:  true,
}

deployment, err := service.Deploy(ctx, projectID, deployConfig)
```

#### 2. 滚动更新 (Rolling Update)

逐个服务进行更新，确保始终有服务在运行。

**特点:**
- ✓ 零停机部署
- ✓ 渐进式更新，风险可控
- ✗ 更新时间较长
- ✗ 可能出现新旧版本共存

**适用场景:**
- 生产环境
- 需要零停机的服务
- 有状态服务

**使用示例:**
```go
deployConfig := &DeploymentConfig{
    Strategy:           DeployStrategyRollingUpdate,
    MaxSurge:           1,      // 最多超出1个实例
    MaxUnavailable:     0,      // 不允许不可用实例
    HealthCheckDelay:   10,
    HealthCheckRetries: 3,
    RollbackOnFailure:  true,
}

deployment, err := service.Deploy(ctx, projectID, deployConfig)
```

#### 3. 蓝绿部署 (Blue-Green)

部署完整的新环境，验证后切换流量。

**特点:**
- ✓ 快速切换
- ✓ 易于回滚
- ✓ 新旧版本完全隔离
- ✗ 需要双倍资源

**适用场景:**
- 关键业务系统
- 需要快速回滚能力
- 资源充足的环境

**使用示例:**
```go
deployConfig := &DeploymentConfig{
    Strategy:           DeployStrategyBlueGreen,
    HealthCheckDelay:   10,
    HealthCheckRetries: 3,
    RollbackOnFailure:  true,
    BlueGreenTimeout:   300,    // 切换超时时间（秒）
}

deployment, err := service.Deploy(ctx, projectID, deployConfig)
```

### 部署监控

#### 监控部署进度

```go
// 获取部署状态
deployment, err := service.GetDeployment(ctx, deploymentID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("状态: %s, 进度: %d%%\n", deployment.Status, deployment.Progress)

// 获取部署事件
events, err := service.GetDeploymentEvents(ctx, deploymentID)
for _, event := range events {
    fmt.Printf("[%s] %s: %s\n", 
        event.CreatedAt.Format("15:04:05"), 
        event.EventType, 
        event.Message)
}

// 获取服务实例
instances, err := service.GetServiceInstances(ctx, deploymentID)
for _, inst := range instances {
    fmt.Printf("服务: %s, 容器: %s, 状态: %s\n",
        inst.ServiceName, inst.ContainerName, inst.Status)
}
```

### 回滚部署

```go
// 手动回滚到上一个成功的部署
err := service.RollbackDeployment(ctx, deploymentID)
if err != nil {
    log.Fatal(err)
}

// 自动回滚（在部署配置中启用）
deployConfig := &DeploymentConfig{
    Strategy:          DeployStrategyRollingUpdate,
    RollbackOnFailure: true,  // 失败时自动回滚
}
```

### 部署数据模型

#### Deployment

部署记录：

```go
type Deployment struct {
    ID              uint
    ProjectID       uint
    Version         string           // 部署版本
    Strategy        DeployStrategy   // 部署策略
    Status          DeploymentStatus // 部署状态
    Progress        int              // 部署进度（0-100）
    Message         string           // 状态消息
    StartedAt       *time.Time
    CompletedAt     *time.Time
    RollbackVersion string
    UserID          uint
    TenantID        uint
}
```

#### ServiceInstance

服务实例（运行中的容器）：

```go
type ServiceInstance struct {
    ID            uint
    DeploymentID  uint
    ServiceName   string
    ContainerID   string
    ContainerName string
    Image         string
    Status        string
    Health        string
    StartedAt     *time.Time
}
```

#### DeploymentEvent

部署事件：

```go
type DeploymentEvent struct {
    ID           uint
    DeploymentID uint
    EventType    string  // 事件类型
    ServiceName  string  // 服务名称
    Message      string  // 事件消息
    Details      string  // 详细信息（JSON）
    CreatedAt    time.Time
}
```

### 完整部署流程示例

```go
// 1. 创建项目
project := &ComposeProject{
    Name:    "my-web-app",
    Content: composeYAML,
    UserID:  1,
    TenantID: 1,
}
service.CreateProject(ctx, project)

// 2. 部署项目
deployConfig := &DeploymentConfig{
    Strategy:           DeployStrategyRollingUpdate,
    MaxSurge:           1,
    MaxUnavailable:     0,
    HealthCheckDelay:   10,
    HealthCheckRetries: 3,
    RollbackOnFailure:  true,
}

deployment, err := service.Deploy(ctx, project.ID, deployConfig)

// 3. 监控部署
ticker := time.NewTicker(2 * time.Second)
for range ticker.C {
    d, _ := service.GetDeployment(ctx, deployment.ID)
    fmt.Printf("进度: %d%%, 状态: %s\n", d.Progress, d.Status)
    
    if d.Status == DeploymentStatusCompleted {
        fmt.Println("部署成功!")
        break
    }
    if d.Status == DeploymentStatusFailed {
        fmt.Println("部署失败!")
        break
    }
}

// 4. 查看部署历史
deployments, _ := service.ListDeployments(ctx, project.ID)
for _, d := range deployments {
    fmt.Printf("版本: %s, 策略: %s, 状态: %s\n",
        d.Version, d.Strategy, d.Status)
}
```

## 容器服务自愈系统

### 4. 自愈服务

`SelfHealingService` 提供了自动化的容器健康监控、异常检测和故障恢复能力：

- **健康检查**: 定期检查容器状态，支持自定义检查间隔和超时
- **异常检测**: 监控容器运行状态，跟踪连续失败次数
- **自动重启**: 智能重启策略，支持重启次数限制和时间窗口
- **故障记录**: 详细记录每次故障的信息到数据库
- **告警通知**: 支持多种告警级别和通知方式

#### 核心功能

**1. 容器注册和监控**

```go
// 创建自愈服务
healingService := NewSelfHealingService(db, dockerExecutor, notifyService)

// 启动监控
ctx := context.Background()
healingService.Start(ctx)
defer healingService.Stop()

// 注册容器
config := &HealingConfig{
    CheckInterval:    30,  // 30秒检查一次
    CheckTimeout:     10,  // 10秒超时
    FailureThreshold: 3,   // 连续失败3次触发自愈
    MaxRestarts:      5,   // 5分钟内最多重启5次
    RestartWindow:    300, // 5分钟时间窗口
    AutoRestart:      true,
    SendAlert:        true,
}

healingService.RegisterContainer(ctx, containerID, config)
```

**2. 健康状态查询**

```go
// 查询容器健康状态
health, err := healingService.GetContainerHealth(ctx, containerID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("状态: %s\n", health.Status)
fmt.Printf("连续失败: %d\n", health.ConsecutiveFailures)
fmt.Printf("总重启次数: %d\n", health.TotalRestarts)
fmt.Printf("最后检查: %s\n", health.LastCheckTime)
```

**3. 故障历史分析**

```go
// 获取故障历史
failures, err := healingService.GetFailureHistory(ctx, containerID, 50)
if err != nil {
    log.Fatal(err)
}

for _, failure := range failures {
    fmt.Printf("[%s] %s: %s\n",
        failure.DetectedAt.Format(time.RFC3339),
        failure.FailureType,
        failure.ErrorMessage)
}
```

#### 自愈配置策略

**关键服务**（数据库、API网关）:
```go
&HealingConfig{
    CheckInterval:    10,  // 更频繁的检查
    FailureThreshold: 2,   // 更敏感的触发
    MaxRestarts:      10,  // 允许更多重启
    AutoRestart:      true,
    SendAlert:        true,
}
```

**非关键服务**（缓存、队列）:
```go
&HealingConfig{
    CheckInterval:    60,  // 较少的检查
    FailureThreshold: 5,   // 较高的阈值
    MaxRestarts:      3,   // 限制重启次数
    AutoRestart:      true,
    SendAlert:        false,
}
```

**只读服务**（监控、日志）:
```go
&HealingConfig{
    CheckInterval:    30,
    FailureThreshold: 3,
    AutoRestart:      false, // 不自动重启
    SendAlert:        true,  // 只发送告警
}
```

#### 自动集成

自愈服务已集成到部署流程中，部署完成后会自动注册容器：

```go
// 部署时自动注册容器到自愈服务
deployment, err := deploymentService.Deploy(ctx, projectID, deployConfig)

// 系统会根据 Docker Compose 配置自动生成自愈配置
// restart: always -> AutoRestart: true, MaxRestarts: 10
// healthcheck.interval -> CheckInterval
// healthcheck.retries -> FailureThreshold
```

#### 故障类型

系统识别以下故障类型：

- `health_check_failed`: 健康检查失败
- `container_stopped`: 容器已停止
- `container_error`: 容器处于错误状态
- `restart_limit_exceeded`: 超过重启次数限制
- `restart_failed`: 重启操作失败
- `auto_restart`: 自动重启成功

#### 告警级别

- `info`: 信息性通知
- `warning`: 警告（如容器已重启）
- `error`: 错误（如重启失败）
- `critical`: 严重问题（如超过重启限制）

#### 通知服务

支持多种通知方式：

```go
// 简单日志通知
notifyService := NewSimpleNotificationService()

// Webhook 通知
notifyService := NewWebhookNotificationService("https://your-webhook-url")

// 自定义通知服务
type MyNotificationService struct {}

func (s *MyNotificationService) SendAlert(ctx context.Context, alert *Alert) error {
    // 实现自定义通知逻辑
    // - 发送邮件
    // - 发送短信
    // - 推送到 Slack/钉钉/企业微信
    return nil
}
```

详细文档请参考: [容器服务自愈系统](../../docs/container-self-healing-system.md)

## 未来扩展

计划支持的功能：

1. **Kubernetes 支持**: 将 Compose 文件转换为 Kubernetes 资源
2. **模板变量**: 支持环境变量替换和模板参数
3. **依赖分析**: 分析服务依赖关系图
4. **安全扫描**: 检查配置中的安全问题
5. **性能优化建议**: 基于 AI 的配置优化建议
6. **版本迁移**: 自动升级旧版本 Compose 文件
7. **金丝雀发布**: 支持金丝雀部署策略
8. **流量管理**: 集成负载均衡器进行流量切换
9. **部署审批**: 支持部署前的审批流程
10. **智能自愈**: 基于机器学习的故障预测和自愈策略优化

## 相关模块

- `internal/appstore`: 应用商店模块，使用 Compose 模板
- `internal/executor`: 执行器模块，负责实际部署 Compose 项目
- `internal/monitor`: 监控模块，监控 Compose 项目状态

## 参考文档

- [Docker Compose 文件规范](https://docs.docker.com/compose/compose-file/)
- [Docker Compose 版本兼容性](https://docs.docker.com/compose/compose-file/compose-versioning/)
