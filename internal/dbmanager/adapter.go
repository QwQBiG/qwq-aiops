package dbmanager

import (
	"context"
	"time"
)

// DatabaseAdapter 数据库适配器接口
// 定义了所有数据库类型需要实现的通用操作
type DatabaseAdapter interface {
	// Connect 建立数据库连接
	Connect(ctx context.Context, config *DatabaseConnection) error
	
	// Disconnect 断开数据库连接
	Disconnect(ctx context.Context) error
	
	// Ping 测试连接是否有效
	Ping(ctx context.Context) error
	
	// ExecuteQuery 执行查询语句
	ExecuteQuery(ctx context.Context, sql string, timeout time.Duration) (*QueryResult, error)
	
	// ExecuteCommand 执行命令语句（INSERT, UPDATE, DELETE等）
	ExecuteCommand(ctx context.Context, sql string, timeout time.Duration) (*QueryResult, error)
	
	// ListDatabases 列出所有数据库
	ListDatabases(ctx context.Context) ([]DatabaseInfo, error)
	
	// ListTables 列出指定数据库的所有表
	ListTables(ctx context.Context, database string) ([]TableInfo, error)
	
	// GetTableSchema 获取表结构信息
	GetTableSchema(ctx context.Context, database, table string) ([]ColumnInfo, error)
	
	// GetTableIndexes 获取表索引信息
	GetTableIndexes(ctx context.Context, database, table string) ([]IndexInfo, error)
	
	// GetExecutionPlan 获取SQL执行计划
	GetExecutionPlan(ctx context.Context, sql string) (*ExecutionPlan, error)
	
	// Backup 执行数据库备份
	Backup(ctx context.Context, config *BackupConfig) error
	
	// Restore 执行数据库恢复
	Restore(ctx context.Context, backupPath string, targetDatabase string) error
}

// AdapterFactory 适配器工厂
type AdapterFactory struct {
	adapters map[DatabaseType]func() DatabaseAdapter
}

// NewAdapterFactory 创建适配器工厂
func NewAdapterFactory() *AdapterFactory {
	factory := &AdapterFactory{
		adapters: make(map[DatabaseType]func() DatabaseAdapter),
	}
	
	// 注册各种数据库适配器
	factory.Register(DatabaseTypeMySQL, func() DatabaseAdapter {
		return &MySQLAdapter{}
	})
	
	factory.Register(DatabaseTypePostgreSQL, func() DatabaseAdapter {
		return &PostgreSQLAdapter{}
	})
	
	factory.Register(DatabaseTypeRedis, func() DatabaseAdapter {
		return &RedisAdapter{}
	})
	
	factory.Register(DatabaseTypeMongoDB, func() DatabaseAdapter {
		return &MongoDBAdapter{}
	})
	
	return factory
}

// Register 注册数据库适配器
func (f *AdapterFactory) Register(dbType DatabaseType, creator func() DatabaseAdapter) {
	f.adapters[dbType] = creator
}

// Create 创建指定类型的数据库适配器
func (f *AdapterFactory) Create(dbType DatabaseType) (DatabaseAdapter, error) {
	creator, exists := f.adapters[dbType]
	if !exists {
		return nil, ErrUnsupportedDatabaseType
	}
	return creator(), nil
}

// GetSupportedTypes 获取支持的数据库类型列表
func (f *AdapterFactory) GetSupportedTypes() []DatabaseType {
	types := make([]DatabaseType, 0, len(f.adapters))
	for dbType := range f.adapters {
		types = append(types, dbType)
	}
	return types
}
