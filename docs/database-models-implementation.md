# 数据库模型实现说明

## 变动概述

新增了 `internal/database/models.go` 文件，实现了完整的数据库模型层，支持多租户架构和 RBAC 权限管理系统。

## 主要功能

### 1. 用户管理模型
- **User**: 用户基础信息，包含用户名、邮箱、密码、角色等
- **UserRole**: 三种角色类型（管理员、普通用户、只读用户）
- 支持软删除和时间戳自动管理

### 2. 多租户模型
- **Tenant**: 租户信息，通过租户代码实现资源隔离
- 用户通过 `TenantID` 关联到特定租户
- 支持租户启用/禁用状态管理

### 3. 权限管理模型
- **Permission**: 细粒度权限定义（资源+操作）
- **RolePermission**: 角色与权限的多对多关联
- 支持基于资源和操作的权限控制

### 4. 审计日志模型
- **AuditLog**: 记录所有用户操作
- 包含操作详情、IP 地址、用户代理等信息
- 支持操作状态和错误信息记录

## 修改原因

根据项目规划（阶段一任务 3.1），需要实现：
- 用户、角色、权限的 RBAC 模型
- 多租户资源隔离机制
- 权限检查中间件和审计日志

此文件是权限管理系统的数据层基础，为后续的 RBAC 功能实现提供数据模型支持。

## 影响范围

- **新增依赖**: `gorm.io/gorm` ORM 框架
- **关联模块**: 
  - `internal/database/rbac.go` - 权限检查逻辑
  - `internal/database/audit.go` - 审计日志记录
  - `internal/database/middleware.go` - 权限中间件
- **数据库表**: 将创建 5 张新表（users, tenants, permissions, role_permissions, audit_logs）

## 使用示例

```go
// 创建用户
user := &User{
    Username: "admin",
    Email:    "admin@example.com",
    Password: hashedPassword,
    Role:     RoleAdmin,
    TenantID: 1,
    Enabled:  true,
}

// 创建权限
permission := &Permission{
    Name:        "container:read",
    Resource:    "container",
    Action:      "read",
    Description: "读取容器信息",
}

// 记录审计日志
auditLog := &AuditLog{
    UserID:     user.ID,
    TenantID:   user.TenantID,
    Action:     "create",
    Resource:   "container",
    ResourceID: "container-123",
    Status:     "success",
    IPAddress:  "192.168.1.1",
}
```

## 下一步工作

- 实现数据库迁移脚本
- 完善 RBAC 权限检查逻辑
- 添加权限管理 API 接口
- 编写单元测试和属性测试
