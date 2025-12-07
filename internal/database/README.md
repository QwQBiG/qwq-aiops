# 数据库和权限管理系统

这个包实现了 qwq AIOps 平台的数据库架构、RBAC权限管理和审计日志系统。

## 功能特性

### 1. 数据库管理
- ✅ 支持 PostgreSQL 数据库
- ✅ 自动数据库迁移
- ✅ 连接池管理
- ✅ 查询优化

### 2. RBAC 权限管理
- ✅ 基于角色的访问控制（Role-Based Access Control）
- ✅ 三种预定义角色：管理员、普通用户、只读用户
- ✅ 细粒度的资源权限控制
- ✅ 多租户资源隔离

### 3. 审计日志
- ✅ 完整的操作审计记录
- ✅ 支持按用户、租户、时间等多维度查询
- ✅ 记录操作详情、IP地址、用户代理等信息
- ✅ 自动化日志清理

### 4. 中间件支持
- ✅ HTTP 权限检查中间件
- ✅ 租户访问控制中间件
- ✅ 自动审计日志记录中间件

## 数据模型

### User (用户)
```go
type User struct {
    ID        uint
    Username  string
    Email     string
    Password  string  // 加密存储
    Role      UserRole // admin, user, readonly
    TenantID  uint
    Enabled   bool
}
```

### Tenant (租户)
```go
type Tenant struct {
    ID      uint
    Name    string
    Code    string  // 租户代码
    Enabled bool
}
```

### Permission (权限)
```go
type Permission struct {
    ID          uint
    Name        string  // 如 "container:read"
    Resource    string  // 资源类型
    Action      string  // 操作类型：read, write, delete
    Description string
}
```

### AuditLog (审计日志)
```go
type AuditLog struct {
    ID         uint
    UserID     uint
    TenantID   uint
    Action     string
    Resource   string
    ResourceID string
    Details    string  // JSON格式
    IPAddress  string
    UserAgent  string
    Status     string  // success, failed
}
```

## 快速开始

### 1. 初始化数据库

```go
import "qwq/internal/database"

// 配置数据库连接
cfg := database.Config{
    Type:     "postgres",
    Host:     "localhost",
    Port:     5432,
    User:     "postgres",
    Password: "your_password",
    DBName:   "qwq",
    SSLMode:  "disable",
    Debug:    false,
}

// 初始化数据库
if err := database.Init(cfg); err != nil {
    log.Fatal(err)
}
defer database.Close()

// 自动迁移表结构
if err := database.AutoMigrate(); err != nil {
    log.Fatal(err)
}

// 初始化默认数据（租户、权限、角色）
if err := database.InitDefaultData(); err != nil {
    log.Fatal(err)
}
```

### 2. 使用 RBAC 服务

```go
rbacService := database.NewRBACService(database.DB)
ctx := context.Background()

// 创建用户
user := &database.User{
    Username: "john",
    Email:    "john@example.com",
    Password: "hashed_password",
    Role:     database.RoleUser,
}
err := rbacService.CreateUser(ctx, user)

// 检查权限
hasPermission, err := rbacService.CheckPermission(ctx, user.ID, "container", "read")
if hasPermission {
    // 用户有权限执行操作
}

// 检查租户访问
hasAccess, err := rbacService.CheckTenantAccess(ctx, user.ID, tenantID)

// 获取用户的所有权限
permissions, err := rbacService.GetUserPermissions(ctx, user.ID)
```

### 3. 使用审计日志

```go
auditService := database.NewAuditService(database.DB)

// 记录成功操作
err := auditService.LogSuccess(
    ctx,
    userID,
    tenantID,
    "read",
    "container",
    "container-123",
    map[string]string{"action": "list"},
    "192.168.1.1",
    "Mozilla/5.0",
)

// 记录失败操作
err := auditService.LogFailure(
    ctx,
    userID,
    tenantID,
    "delete",
    "container",
    "container-456",
    nil,
    "权限不足",
    "192.168.1.1",
    "Mozilla/5.0",
)

// 查询日志
filter := database.AuditLogFilter{
    UserID:   userID,
    Page:     1,
    PageSize: 10,
}
logs, total, err := auditService.QueryLogs(ctx, filter)
```

### 4. 使用 HTTP 中间件

```go
import "net/http"

rbacService := database.NewRBACService(database.DB)
auditService := database.NewAuditService(database.DB)
middleware := database.NewPermissionMiddleware(rbacService, auditService)

// 需要容器读权限的路由
http.Handle("/api/containers", 
    middleware.RequirePermission("container", "read")(
        http.HandlerFunc(listContainersHandler),
    ),
)

// 需要租户访问权限的路由
http.Handle("/api/tenant/resources",
    middleware.RequireTenantAccess()(
        http.HandlerFunc(listTenantResourcesHandler),
    ),
)

// 自动记录审计日志的路由
http.Handle("/api/containers/create",
    middleware.AuditLog("container")(
        http.HandlerFunc(createContainerHandler),
    ),
)
```

## 权限系统

### 预定义角色

| 角色 | 说明 | 权限 |
|------|------|------|
| `admin` | 管理员 | 所有资源的所有操作权限 |
| `user` | 普通用户 | 所有资源的读权限 |
| `readonly` | 只读用户 | 所有资源的读权限 |

### 资源类型

- `container` - 容器管理
- `application` - 应用管理
- `website` - 网站管理
- `database` - 数据库管理
- `backup` - 备份管理
- `user` - 用户管理
- `tenant` - 租户管理

### 操作类型

- `read` - 读取/查看
- `write` - 创建/修改
- `delete` - 删除

### 权限命名规范

权限名称格式：`{resource}:{action}`

示例：
- `container:read` - 查看容器
- `container:write` - 创建/修改容器
- `container:delete` - 删除容器

## 多租户隔离

系统实现了完整的多租户资源隔离：

1. **数据隔离**：每个资源都关联到特定租户
2. **访问控制**：用户只能访问自己租户的资源
3. **权限检查**：所有操作都会验证租户访问权限
4. **审计日志**：记录跨租户访问尝试

### 租户隔离验证

```go
// 用户只能访问自己租户的资源
hasAccess, err := rbacService.CheckTenantAccess(ctx, userID, tenantID)
if !hasAccess {
    return errors.New("无权访问该租户资源")
}
```

## 测试

### 运行单元测试

```bash
go test -v ./internal/database/ -run "TestUserRole|TestUserModel|TestTenantModel"
```

### 运行属性测试

```bash
# 权限隔离属性测试
go test -v ./internal/database/ -run TestProperty18_UserPermissionIsolation_Mock

# 多租户隔离属性测试
go test -v ./internal/security/ -run TestMultiTenantIsolation
```

## 性能优化

### 数据库连接池配置

```go
sqlDB, _ := database.DB.DB()
sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期
```

### 查询优化建议

1. 使用索引字段进行查询
2. 避免 N+1 查询问题，使用 `Preload`
3. 对大量数据使用分页查询
4. 定期清理旧的审计日志

## 安全建议

1. **密码加密**：使用 bcrypt 等算法加密用户密码
2. **SQL注入防护**：使用 GORM 的参数化查询
3. **权限最小化**：遵循最小权限原则
4. **审计日志**：记录所有敏感操作
5. **定期备份**：定期备份数据库
6. **SSL连接**：生产环境使用 SSL 连接数据库

## 数据迁移

### 从现有系统迁移

如果需要从现有系统迁移数据：

1. 导出现有数据
2. 创建租户和用户
3. 导入数据并关联到相应租户
4. 验证数据完整性
5. 测试权限和访问控制

### 版本升级

数据库表结构变更时：

1. 使用 `AutoMigrate()` 自动迁移
2. 备份数据库
3. 测试迁移脚本
4. 在生产环境执行迁移

## 故障排查

### 常见问题

**Q: 连接数据库失败**
```
A: 检查数据库配置、网络连接、防火墙设置
```

**Q: 权限检查总是失败**
```
A: 确认已执行 InitDefaultData() 初始化权限数据
```

**Q: 审计日志过多导致性能问题**
```
A: 定期清理旧日志，使用 DeleteOldLogs() 方法
```

## 贡献指南

欢迎贡献代码！请遵循以下规范：

1. 编写单元测试
2. 编写属性测试验证核心逻辑
3. 更新文档
4. 遵循 Go 代码规范

## 许可证

本项目采用 MIT 许可证。
