# 钉钉通知服务实现

## 概述

本模块实现了完整的钉钉通知服务集成，支持发送告警消息、状态报告，并提供了配置验证和连接测试功能。

## 功能特性

### 1. 统一通知服务接口

- `NotificationService` 接口提供统一的通知服务抽象
- 支持多种通知渠道（钉钉、Telegram 等）
- 自动配置检测和初始化

### 2. 钉钉通知服务

- `DingTalkNotificationService` 实现钉钉 Webhook 消息发送
- 支持 Markdown 和文本消息格式
- 包含连接测试和配置验证功能
- 完善的错误处理和日志记录

### 3. 容器告警集成

- `ContainerNotificationAdapter` 适配器支持容器自愈服务
- 格式化容器告警消息
- 避免循环导入问题

### 4. 向后兼容性

- 保持原有 `Send()` 函数的兼容性
- 渐进式升级到新的通知服务架构

## 使用方法

### 基本使用

```go
import "qwq/internal/notify"

// 初始化通知服务
notify.InitNotificationService()

// 发送告警消息
err := notify.GetNotificationService().SendAlert("告警标题", "告警内容")
if err != nil {
    log.Printf("发送失败: %v", err)
}

// 发送状态报告
err = notify.SendStatusReport("系统状态正常")
if err != nil {
    log.Printf("发送失败: %v", err)
}
```

### 容器告警使用

```go
// 创建容器通知适配器
adapter := notify.NewContainerNotificationAdapter()

// 创建告警信息
alert := &notify.ContainerAlert{
    Level:       "error",
    Title:       "容器异常",
    Message:     "容器无法启动",
    ContainerID: "container_123",
    ServiceName: "web-service",
    ProjectName: "my-project",
    Timestamp:   time.Now(),
}

// 发送告警
err := adapter.SendAlert(context.Background(), alert)
```

### 配置验证

```go
// 验证通知配置
err := notify.ValidateNotificationConfig()
if err != nil {
    log.Printf("配置无效: %v", err)
}

// 测试连接
err = notify.TestNotificationConnection()
if err != nil {
    log.Printf("连接测试失败: %v", err)
}
```

## 配置要求

### 环境变量

在 `.env` 文件或环境变量中配置：

```bash
# 钉钉 Webhook URL
DINGTALK_WEBHOOK=https://oapi.dingtalk.com/robot/send?access_token=your_token
```

### 配置文件

在 `config.json` 中配置：

```json
{
  "webhook": "https://oapi.dingtalk.com/robot/send?access_token=your_token"
}
```

## 错误处理

### 常见错误

1. **配置错误**
   - `未配置任何通知渠道`: 需要设置钉钉 Webhook URL
   - `钉钉 Webhook URL 格式不正确`: URL 格式验证失败

2. **网络错误**
   - `发送请求失败`: 网络连接问题
   - `钉钉服务器返回错误状态码`: 服务器响应异常

3. **消息格式错误**
   - `序列化消息失败`: JSON 序列化问题

### 错误恢复

- 自动重试机制（针对临时性错误）
- 详细的错误日志记录
- 优雅的降级处理

## 测试

### 属性基础测试

实现了两个核心属性测试：

1. **属性 5: 告警消息发送可靠性**
   - 验证有效配置下的消息发送成功
   - 验证无效配置下的发送失败
   - 验证消息格式的正确性

2. **属性 6: 配置验证准确性**
   - 验证有效 URL 格式的验证通过
   - 验证无效 URL 格式的验证失败
   - 验证空 URL 的处理

### 集成测试

- 完整的通知流程测试
- 向后兼容性测试
- 容器通知适配器测试

### 运行测试

```bash
# 运行所有通知相关测试
go test -v ./internal/notify

# 运行特定属性测试
go test -v ./internal/notify -run "TestProperty5_AlertMessageSendingReliability"
go test -v ./internal/notify -run "TestProperty6_ConfigValidationAccuracy"
```

## 架构设计

### 组件关系

```
┌─────────────────────┐
│   应用程序层        │
├─────────────────────┤
│ UnifiedNotification │ ← 统一通知服务
│ Service             │
├─────────────────────┤
│ DingTalkNotification│ ← 钉钉通知实现
│ Service             │
├─────────────────────┤
│ ContainerNotification│ ← 容器通知适配器
│ Adapter             │
└─────────────────────┘
```

### 接口设计

- `NotificationService`: 核心通知服务接口
- `DingTalkNotificationService`: 钉钉具体实现
- `ContainerNotificationAdapter`: 容器告警适配器

## 扩展性

### 添加新的通知渠道

1. 实现 `NotificationService` 接口
2. 在 `UnifiedNotificationService` 中集成
3. 添加相应的配置验证逻辑

### 自定义消息格式

1. 扩展消息结构体
2. 实现自定义格式化函数
3. 更新相应的测试用例

## 性能考虑

- 异步发送机制（使用 goroutine）
- 连接池和超时控制
- 合理的重试策略
- 内存使用优化

## 安全考虑

- Webhook URL 的安全存储
- 消息内容的敏感信息过滤
- 网络请求的安全配置
- 错误信息的安全处理