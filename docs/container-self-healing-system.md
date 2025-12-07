# 容器服务自愈系统实现文档

## 概述

容器服务自愈系统是 qwq AIOps 平台的核心功能之一，提供自动化的容器健康监控、异常检测和故障恢复能力。该系统能够在容器出现问题时自动重启，并记录详细的故障日志和发送告警通知。

## 核心功能

### 1. 容器健康检查和异常检测

- **定期健康检查**: 按配置的时间间隔检查容器状态
- **超时控制**: 健康检查支持超时设置，避免长时间阻塞
- **状态监控**: 监控容器的运行状态（running, stopped, exited, dead等）
- **连续失败计数**: 跟踪连续失败次数，达到阈值时触发自愈

### 2. 自动重启和故障恢复

- **智能重启策略**: 根据配置自动重启失败的容器
- **重启次数限制**: 在时间窗口内限制最大重启次数，防止无限重启
- **重启历史记录**: 记录每次重启的时间，用于分析和限制
- **故障恢复验证**: 重启后验证容器是否恢复正常

### 3. 详细的故障日志和告警通知

- **故障记录**: 记录每次故障的详细信息到数据库
- **告警通知**: 支持多种告警级别（info, warning, error, critical）
- **事件跟踪**: 记录故障检测、重启操作、恢复等完整事件链
- **故障分析**: 提供故障历史查询和分析功能

## 架构设计

### 核心组件

```
┌─────────────────────────────────────────────────────────────┐
│              Self-Healing Service                           │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │   Monitor    │  │   Healing    │  │   Notification   │  │
│  │   Loop       │──│   Engine     │──│   Service        │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
│         │                  │                    │           │
│         │                  │                    │           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  Container   │  │   Docker     │  │   Failure        │  │
│  │  Registry    │  │   Executor   │  │   Records DB     │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 数据模型

#### HealingConfig - 自愈配置
```go
type HealingConfig struct {
    CheckInterval    int  // 健康检查间隔（秒）
    CheckTimeout     int  // 健康检查超时（秒）
    FailureThreshold int  // 失败阈值（连续失败多少次触发自愈）
    MaxRestarts      int  // 最大重启次数（在时间窗口内）
    RestartWindow    int  // 重启时间窗口（秒）
    AutoRestart      bool // 是否启用自动重启
    SendAlert        bool // 是否发送告警通知
}
```

#### HealthStatus - 健康状态
```go
type HealthStatus struct {
    ContainerID         string    // 容器ID
    Status              string    // 健康状态: healthy, unhealthy, unknown
    LastCheckTime       time.Time // 最后检查时间
    ConsecutiveFailures int       // 连续失败次数
    TotalRestarts       int       // 总重启次数
    LastRestartTime     *time.Time // 最后重启时间
    Message             string    // 状态消息
}
```

#### FailureRecord - 故障记录
```go
type FailureRecord struct {
    ID           uint      // 记录ID
    ContainerID  string    // 容器ID
    ServiceName  string    // 服务名称
    ProjectName  string    // 项目名称
    FailureType  string    // 故障类型
    ErrorMessage string    // 错误消息
    Details      string    // 详细信息（JSON）
    Action       string    // 执行的操作: restart, alert, none
    ActionResult string    // 操作结果: success, failed, pending
    DetectedAt   time.Time // 检测时间
    ResolvedAt   *time.Time // 解决时间
    TenantID     uint      // 租户ID
}
```

## 使用方法

### 1. 创建和启动自愈服务

```go
// 创建依赖
db := // 数据库连接
dockerExecutor := NewDockerExecutor()
notifyService := NewSimpleNotificationService()

// 创建自愈服务
healingService := NewSelfHealingService(db, dockerExecutor, notifyService)

// 启动监控
ctx := context.Background()
if err := healingService.Start(ctx); err != nil {
    log.Fatalf("Failed to start healing service: %v", err)
}
defer healingService.Stop()
```

### 2. 注册容器到自愈服务

```go
// 使用默认配置
containerID := "my-container-id"
config := DefaultHealingConfig()

err := healingService.RegisterContainer(ctx, containerID, config)
if err != nil {
    log.Fatalf("Failed to register container: %v", err)
}
```

### 3. 自定义自愈配置

```go
// 为关键服务配置更激进的策略
criticalConfig := &HealingConfig{
    CheckInterval:    10,  // 10秒检查一次
    CheckTimeout:     5,   // 5秒超时
    FailureThreshold: 2,   // 失败2次就触发
    MaxRestarts:      10,  // 允许更多重启
    RestartWindow:    600, // 10分钟窗口
    AutoRestart:      true,
    SendAlert:        true,
}

err := healingService.RegisterContainer(ctx, containerID, criticalConfig)
```

### 4. 查询容器健康状态

```go
health, err := healingService.GetContainerHealth(ctx, containerID)
if err != nil {
    log.Printf("Failed to get health: %v", err)
} else {
    fmt.Printf("Status: %s\n", health.Status)
    fmt.Printf("Consecutive Failures: %d\n", health.ConsecutiveFailures)
    fmt.Printf("Total Restarts: %d\n", health.TotalRestarts)
}
```

### 5. 查询故障历史

```go
failures, err := healingService.GetFailureHistory(ctx, containerID, 50)
if err != nil {
    log.Printf("Failed to get failure history: %v", err)
} else {
    for _, failure := range failures {
        fmt.Printf("[%s] %s: %s\n", 
            failure.DetectedAt.Format(time.RFC3339),
            failure.FailureType,
            failure.ErrorMessage)
    }
}
```

## 集成到部署流程

自愈服务已经集成到容器部署流程中，在部署完成后会自动注册容器：

```go
// 部署服务会自动注册容器到自愈服务
deployment, err := deploymentService.Deploy(ctx, projectID, deployConfig)

// 部署完成后，所有容器都会被自动注册到自愈服务
// 自愈配置会根据 Docker Compose 文件中的配置自动生成
```

### 自动配置生成规则

系统会根据 Docker Compose 文件中的配置自动生成自愈配置：

1. **restart 策略映射**:
   - `no`: 禁用自动重启
   - `always`, `unless-stopped`: 启用自动重启，增加重启次数限制
   - `on-failure`: 启用自动重启，使用默认配置

2. **healthcheck 配置映射**:
   - `interval`: 映射到 CheckInterval
   - `retries`: 映射到 FailureThreshold

示例 Docker Compose 配置：
```yaml
services:
  web:
    image: nginx:latest
    restart: always
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
      timeout: 10s
      retries: 3
```

生成的自愈配置：
```go
&HealingConfig{
    CheckInterval:    30,  // 从 healthcheck.interval
    CheckTimeout:     10,  // 默认值
    FailureThreshold: 3,   // 从 healthcheck.retries
    MaxRestarts:      10,  // restart: always 增加限制
    RestartWindow:    300,
    AutoRestart:      true, // restart: always
    SendAlert:        true,
}
```

## 故障类型

系统识别以下故障类型：

1. **health_check_failed**: 健康检查失败（无法连接到容器）
2. **container_stopped**: 容器已停止
3. **container_error**: 容器处于错误状态
4. **restart_limit_exceeded**: 超过重启次数限制
5. **restart_failed**: 重启操作失败
6. **auto_restart**: 自动重启成功

## 告警级别

系统支持以下告警级别：

1. **info**: 信息性通知
2. **warning**: 警告（如容器已重启）
3. **error**: 错误（如重启失败）
4. **critical**: 严重问题（如超过重启限制、容器持续不健康）

## 通知服务

### 内置通知服务

1. **SimpleNotificationService**: 简单的日志输出通知
2. **MockNotificationService**: 用于测试的 Mock 服务
3. **WebhookNotificationService**: Webhook 通知服务（待实现）

### 自定义通知服务

可以实现 `NotificationService` 接口来自定义通知方式：

```go
type NotificationService interface {
    SendAlert(ctx context.Context, alert *Alert) error
}

// 实现自定义通知服务
type MyNotificationService struct {
    // 添加依赖（邮件服务、短信服务等）
}

func (s *MyNotificationService) SendAlert(ctx context.Context, alert *Alert) error {
    // 实现通知逻辑
    // - 发送邮件
    // - 发送短信
    // - 调用 Webhook
    // - 推送到 Slack/钉钉/企业微信
    return nil
}
```

## 性能考虑

### 监控开销

- 监控循环每10秒运行一次，检查所有注册的容器
- 每个容器的实际检查间隔由 `CheckInterval` 配置控制
- 健康检查使用超时控制，避免阻塞

### 数据库写入

- 故障记录异步写入数据库
- 写入失败不影响主流程
- 建议定期清理历史记录

### 并发安全

- 使用读写锁保护容器注册表
- 每个容器的健康状态独立加锁
- 支持并发检查多个容器

## 最佳实践

### 1. 配置建议

**关键服务**（数据库、API网关等）:
```go
&HealingConfig{
    CheckInterval:    10,  // 更频繁的检查
    FailureThreshold: 2,   // 更敏感的触发
    MaxRestarts:      10,  // 允许更多重启
    AutoRestart:      true,
    SendAlert:        true,
}
```

**非关键服务**（缓存、队列等）:
```go
&HealingConfig{
    CheckInterval:    60,  // 较少的检查
    FailureThreshold: 5,   // 较高的阈值
    MaxRestarts:      3,   // 限制重启次数
    AutoRestart:      true,
    SendAlert:        false,
}
```

**只读服务**（监控、日志等）:
```go
&HealingConfig{
    CheckInterval:    30,
    FailureThreshold: 3,
    AutoRestart:      false, // 不自动重启
    SendAlert:        true,  // 只发送告警
}
```

### 2. 故障分析

定期分析故障历史，识别问题模式：

```go
failures, _ := healingService.GetFailureHistory(ctx, containerID, 100)

// 统计故障类型
failureTypes := make(map[string]int)
for _, failure := range failures {
    failureTypes[failure.FailureType]++
}

// 计算平均故障间隔时间（MTBF）
if len(failures) > 1 {
    totalDuration := failures[0].DetectedAt.Sub(failures[len(failures)-1].DetectedAt)
    mtbf := totalDuration / time.Duration(len(failures)-1)
    fmt.Printf("MTBF: %s\n", mtbf)
}
```

### 3. 告警降噪

- 使用合适的 `FailureThreshold` 避免误报
- 为非关键服务禁用告警
- 实现告警聚合和去重逻辑

### 4. 监控指标

建议监控以下指标：

- 容器健康状态分布
- 平均故障间隔时间（MTBF）
- 重启成功率
- 告警数量和级别分布
- 自愈响应时间

## 未来改进

1. **智能阈值调整**: 根据历史数据自动调整失败阈值
2. **故障预测**: 使用机器学习预测容器故障
3. **依赖感知**: 考虑服务依赖关系的自愈策略
4. **多级恢复**: 支持重启、重新部署、切换备用等多级恢复策略
5. **告警聚合**: 实现智能告警聚合和降噪
6. **性能优化**: 优化大规模容器监控的性能

## 相关文档

- [容器编排部署引擎](./container-deployment-engine.md)
- [Docker Compose 解析器](./compose-parser-implementation.md)
- [数据库模型实现](./database-models-implementation.md)

## 验证需求

该实现满足以下需求：

- **Requirements 3.4**: 容器服务自愈能力
  - ✅ 创建容器健康检查和异常检测
  - ✅ 实现自动重启和故障恢复逻辑
  - ✅ 添加详细的故障日志和告警通知

- **Property 8**: 容器服务自愈能力
  - *For any* 出现异常的容器服务，系统应该能自动重启并记录详细的故障日志
