# qwq AIOps 平台 - 增强自动修复功能

## 概述

本模块实现了 qwq AIOps 平台的增强自动修复功能，提供全面的部署问题诊断和自动修复能力。

## 主要功能

### 1. 部署验证全面性 (Property 17)
- **验证需求**: Requirements 5.1
- **功能**: 自动检查所有关键组件的状态
- **组件**: 前端、后端、数据库、配置、通知服务
- **验证内容**:
  - 组件存在性检查
  - 组件健康状态验证
  - 依赖关系验证
  - 失败时提供修复建议

### 2. 自动修复有效性 (Property 18)
- **验证需求**: Requirements 5.2
- **功能**: 检测并自动修复前端资源问题
- **修复类型**:
  - 前端资源重建
  - 配置文件生成
  - 平台兼容性修复
  - 通知服务配置

## 核心组件

### ConfigDiagnostic (增强版)
```go
// 扩展的自动修复功能
func (d *ConfigDiagnostic) AutoFix() error
func (d *ConfigDiagnostic) fixEnvFile() error
func (d *ConfigDiagnostic) fixFrontendResources() error
func (d *ConfigDiagnostic) fixPlatformCompatibility() error
```

### EnhancedAutoFixer
```go
// 全面的自动修复器
type EnhancedAutoFixer struct {
    diagnostic *ConfigDiagnostic
    tracker    *RepairTracker
    options    *AutoFixOptions
}

// 主要方法
func (eaf *EnhancedAutoFixer) RunComprehensiveRepair() error
func (eaf *EnhancedAutoFixer) rebuildFrontendResources() error
```

### RepairTracker
```go
// 修复过程跟踪和记录
type RepairTracker struct {
    logPath string
    session *RepairSession
}

// 会话管理
func (rt *RepairTracker) StartSession() error
func (rt *RepairTracker) AddOperation(opType RepairType, description string, commands []string) string
func (rt *RepairTracker) CompleteOperation(operationID string, output string, err error) error
func (rt *RepairTracker) EndSession(validationResult *DeploymentValidationResult) error
```

## 使用示例

### 基础自动修复
```go
diagnostic := NewConfigDiagnostic()
if err := diagnostic.AutoFix(); err != nil {
    log.Printf("自动修复失败: %v", err)
}
```

### 增强自动修复
```go
options := DefaultAutoFixOptions()
options.Verbose = true
autoFixer := NewEnhancedAutoFixer(options)

if err := autoFixer.RunComprehensiveRepair(); err != nil {
    log.Printf("全面修复失败: %v", err)
}
```

### 预览模式
```go
options := DefaultAutoFixOptions()
options.DryRun = true  // 只预览，不实际执行
autoFixer := NewEnhancedAutoFixer(options)
autoFixer.RunComprehensiveRepair()
```

## 修复类型

### 1. 前端资源修复 (RepairFrontend)
- 检查前端构建目录
- 自动执行 npm install 和 npm run build
- 创建基础前端文件（当 Node.js 不可用时）
- 验证关键文件存在性

### 2. 配置修复 (RepairConfig)
- 创建 .env 配置文件
- 生成安全密钥
- 验证配置项完整性
- 修复无效配置值

### 3. 平台兼容性修复 (RepairPlatform)
- Windows 环境 Docker 配置
- Linux 环境权限问题
- 跨平台路径处理

### 4. 通知服务修复 (RepairNotification)
- 钉钉 Webhook 配置验证
- 通知渠道连通性测试

## 修复会话记录

每次修复操作都会创建详细的会话记录：

```json
{
  "id": "repair_1703123456",
  "start_time": "2023-12-21T10:30:56Z",
  "end_time": "2023-12-21T10:31:23Z",
  "status": "completed",
  "operations": [
    {
      "id": "op_1",
      "type": "config",
      "description": "修复环境配置",
      "status": "completed",
      "commands": ["检查并创建 .env 文件"],
      "output": "✅ .env 文件已创建"
    }
  ],
  "summary": {
    "total_operations": 3,
    "completed_operations": 2,
    "failed_operations": 1,
    "duration": "27s",
    "recommendations": ["重启服务以确保更改生效"]
  }
}
```

## 属性测试

### Property 17: 部署验证全面性
- 测试文件: `deployment_validation_property_test.go`
- 验证所有关键组件都被检查
- 验证组件状态正确识别
- 验证依赖关系检查
- 验证失败时提供建议

### Property 18: 自动修复有效性
- 测试文件: `auto_repair_property_test.go`
- 验证支持的问题类型能被修复
- 验证前端重建功能
- 验证修复操作记录完整性
- 验证修复失败时提供建议

## 配置选项

```go
type AutoFixOptions struct {
    EnableFrontendRebuild  bool   // 启用前端重建
    EnableConfigGeneration bool   // 启用配置生成
    EnablePlatformFix     bool   // 启用平台修复
    LogPath               string // 日志路径
    DryRun                bool   // 预览模式
    Verbose               bool   // 详细输出
}
```

## 错误处理

- 所有修复操作都有详细的错误记录
- 失败的操作不会阻止其他操作继续执行
- 提供具体的修复建议和命令
- 支持部分修复成功的场景

## 日志和监控

- 修复会话自动保存到 JSON 文件
- 支持查看历史修复记录
- 提供修复摘要和统计信息
- 集成到系统诊断报告中

## 扩展性

- 支持添加新的修复类型
- 可自定义修复策略
- 支持插件式修复器
- 可配置的修复优先级

## 最佳实践

1. **使用预览模式**: 在生产环境中先使用 DryRun 模式预览修复操作
2. **定期备份**: 在执行修复前备份重要配置文件
3. **监控日志**: 定期检查修复会话日志
4. **渐进修复**: 对于复杂问题，分步骤执行修复
5. **验证结果**: 修复后运行完整的系统验证

## 故障排除

### 常见问题

1. **前端重建失败**
   - 检查 Node.js 和 npm 是否安装
   - 确认 frontend/package.json 存在
   - 检查网络连接（npm install）

2. **配置文件创建失败**
   - 检查文件系统权限
   - 确认磁盘空间充足
   - 验证 .env.example 文件存在

3. **平台兼容性问题**
   - Windows: 检查 Docker Desktop 状态
   - Linux: 验证 Docker 权限配置
   - 确认环境变量设置正确

### 调试技巧

- 启用 Verbose 模式查看详细输出
- 使用 DryRun 模式预览操作
- 检查修复会话日志文件
- 运行单独的诊断命令验证问题