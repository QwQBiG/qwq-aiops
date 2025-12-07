# 数据库管理服务

数据库管理服务提供了完整的数据库连接管理、SQL查询执行、AI查询优化和自动备份功能。

## 功能特性

### 1. 多数据库支持
- MySQL
- PostgreSQL
- Redis（待完善）
- MongoDB（待完善）

### 2. 连接管理
- 安全的连接配置存储（密码加密）
- 连接池管理
- 连接状态监控
- 连接测试

### 3. SQL查询执行
- 查询执行和结果返回
- 查询超时控制
- 结果行数限制
- 分页查询支持
- 批量查询执行

### 4. AI查询优化
- 执行计划分析
- 性能瓶颈识别
- 索引推荐
- SQL重写建议
- 性能提升估算

### 5. 数据库备份
- 自动备份调度
- 多种备份策略
- 备份压缩
- 备份验证
- 快速恢复
- 过期备份清理

## 架构设计

### 适配器模式
使用适配器模式支持多种数据库类型，每种数据库类型实现统一的 `DatabaseAdapter` 接口：

```go
type DatabaseAdapter interface {
    Connect(ctx context.Context, config *DatabaseConnection) error
    Disconnect(ctx context.Context) error
    Ping(ctx context.Context) error
    ExecuteQuery(ctx context.Context, sql string, timeout time.Duration) (*QueryResult, error)
    // ... 其他方法
}
```

### 连接管理器
`ConnectionManager` 负责管理所有数据库连接：
- 连接池管理
- 连接复用
- 连接健康检查
- 密码加密/解密

### 查询引擎
`QueryEngine` 提供高级查询功能：
- SQL验证
- 查询解析
- 分页支持
- 批量执行
- 查询缓存

### AI优化器
`AIQueryOptimizer` 提供智能查询优化：
- 执行计划分析
- 性能问题识别
- 索引推荐生成
- SQL重写建议

### 备份管理器
`BackupManager` 处理数据库备份：
- 定时备份调度
- 多数据库备份支持
- 备份压缩
- 备份恢复
- 备份验证

## 使用示例

### 1. 创建数据库服务

```go
import (
    "github.com/yourusername/qwq/internal/database"
    "github.com/yourusername/qwq/internal/dbmanager"
)

// 初始化数据库
db := database.DB

// 创建数据库管理服务
encryptionKey := "your-32-byte-encryption-key-here"
dbService := dbmanager.NewDatabaseService(db, encryptionKey)
```

### 2. 创建数据库连接

```go
conn := &dbmanager.DatabaseConnection{
    Name:     "生产数据库",
    Type:     dbmanager.DatabaseTypeMySQL,
    Host:     "localhost",
    Port:     3306,
    Username: "root",
    Password: "password",
    Database: "myapp",
    UserID:   1,
    TenantID: 1,
}

err := dbService.CreateConnection(ctx, conn)
```

### 3. 执行SQL查询

```go
req := &dbmanager.QueryRequest{
    ConnectionID: conn.ID,
    SQL:          "SELECT * FROM users WHERE status = 'active'",
    Timeout:      30,
    MaxRows:      100,
}

result, err := dbService.ExecuteQuery(ctx, req)
if err != nil {
    log.Fatal(err)
}

// 处理结果
for _, row := range result.Rows {
    fmt.Printf("用户: %v\n", row)
}
```

### 4. AI查询优化

```go
optimization, err := dbService.OptimizeQuery(ctx, conn.ID, 
    "SELECT * FROM orders WHERE user_id = 123 ORDER BY created_at DESC")

if err != nil {
    log.Fatal(err)
}

fmt.Printf("优化建议:\n")
for _, suggestion := range optimization.Suggestions {
    fmt.Printf("- %s\n", suggestion)
}

fmt.Printf("\n索引推荐:\n")
for _, idx := range optimization.IndexRecommendations {
    fmt.Printf("- %s: %s\n", idx.Table, idx.CreateSQL)
}
```

### 5. 配置自动备份

```go
backupConfig := &dbmanager.BackupConfig{
    ConnectionID: conn.ID,
    Name:         "每日备份",
    Schedule:     "0 2 * * *", // 每天凌晨2点
    Enabled:      true,
    BackupPath:   "/backups",
    Compression:  true,
    Retention:    7, // 保留7天
    UserID:       1,
    TenantID:     1,
}

err := dbService.CreateBackupConfig(ctx, backupConfig)
```

### 6. 执行备份

```go
// 手动执行备份
err := dbService.ExecuteBackup(ctx, backupConfig.ID)

// 或使用备份管理器
backupManager := dbmanager.NewBackupManager(dbService, "/backups")
backupManager.Start() // 启动定时备份

// 调度备份任务
err = backupManager.ScheduleBackup(ctx, backupConfig)
```

### 7. 恢复备份

```go
// 列出备份记录
records, err := dbService.ListBackupRecords(ctx, backupConfig.ID)

// 恢复最新的备份
if len(records) > 0 {
    err = dbService.RestoreBackup(ctx, records[0].ID, "myapp_restored")
}
```

## API接口

### 连接管理
- `POST /api/database/connections` - 创建连接
- `GET /api/database/connections` - 列出连接
- `GET /api/database/connections/:id` - 获取连接详情
- `PUT /api/database/connections/:id` - 更新连接
- `DELETE /api/database/connections/:id` - 删除连接
- `POST /api/database/connections/test` - 测试连接

### SQL执行
- `POST /api/database/query` - 执行SQL查询

### 数据库操作
- `GET /api/database/connections/:id/databases` - 列出数据库
- `GET /api/database/connections/:id/databases/:database/tables` - 列出表
- `GET /api/database/connections/:id/databases/:database/tables/:table/schema` - 获取表结构
- `GET /api/database/connections/:id/databases/:database/tables/:table/indexes` - 获取索引

### AI优化
- `POST /api/database/connections/:id/optimize` - 优化查询
- `POST /api/database/connections/:id/explain` - 获取执行计划

### 备份管理
- `POST /api/database/backup/configs` - 创建备份配置
- `GET /api/database/backup/configs` - 列出备份配置
- `PUT /api/database/backup/configs/:id` - 更新备份配置
- `DELETE /api/database/backup/configs/:id` - 删除备份配置
- `POST /api/database/backup/configs/:id/execute` - 执行备份
- `GET /api/database/backup/configs/:id/records` - 列出备份记录
- `POST /api/database/backup/records/:id/restore` - 恢复备份

## 安全考虑

### 1. 密码加密
所有数据库密码使用AES-256-GCM加密存储，确保密码安全。

### 2. 连接隔离
每个租户的数据库连接完全隔离，防止跨租户访问。

### 3. SQL注入防护
- 使用参数化查询
- SQL语句验证
- 危险操作检测

### 4. 权限控制
- 基于角色的访问控制
- 操作审计日志
- 资源级权限检查

## 性能优化

### 1. 连接池
使用连接池管理数据库连接，避免频繁创建和销毁连接。

### 2. 查询缓存
对于相同的查询，可以使用缓存减少数据库负载。

### 3. 分页查询
大结果集自动分页，避免内存溢出。

### 4. 超时控制
所有查询都有超时限制，防止长时间占用资源。

## 待完善功能

### 1. Redis适配器
- 完整的Redis命令支持
- Redis集群支持
- 持久化管理

### 2. MongoDB适配器
- MongoDB查询语言支持
- 聚合管道
- 索引管理

### 3. 高级功能
- 数据库迁移工具
- 数据同步
- 读写分离
- 分库分表支持

## 依赖项

```bash
# MySQL驱动
go get github.com/go-sql-driver/mysql

# PostgreSQL驱动
go get github.com/lib/pq

# 定时任务
go get github.com/robfig/cron/v3

# Web框架
go get github.com/gin-gonic/gin
```

## 数据库迁移

在使用数据库管理服务之前，需要先运行数据库迁移：

```go
import "github.com/yourusername/qwq/internal/database"

// 自动迁移数据库表
err := database.DB.AutoMigrate(
    &dbmanager.DatabaseConnection{},
    &dbmanager.BackupConfig{},
    &dbmanager.BackupRecord{},
)
```

## 测试

运行单元测试：

```bash
go test ./internal/dbmanager/...
```

运行集成测试：

```bash
go test -tags=integration ./internal/dbmanager/...
```

## 贡献

欢迎提交问题和拉取请求来改进数据库管理服务。

## 许可证

MIT License
