package dbmanager

import (
	"context"
	"errors"
)

var (
	// ErrConnectionNotFound 连接不存在
	ErrConnectionNotFound = errors.New("数据库连接不存在")
	
	// ErrConnectionFailed 连接失败
	ErrConnectionFailed = errors.New("数据库连接失败")
	
	// ErrUnsupportedDatabaseType 不支持的数据库类型
	ErrUnsupportedDatabaseType = errors.New("不支持的数据库类型")
	
	// ErrQueryTimeout 查询超时
	ErrQueryTimeout = errors.New("查询执行超时")
	
	// ErrInvalidSQL SQL语句无效
	ErrInvalidSQL = errors.New("SQL语句无效")
	
	// ErrPermissionDenied 权限不足
	ErrPermissionDenied = errors.New("权限不足")
	
	// ErrBackupFailed 备份失败
	ErrBackupFailed = errors.New("数据库备份失败")
	
	// ErrRestoreFailed 恢复失败
	ErrRestoreFailed = errors.New("数据库恢复失败")
)

// DatabaseService 数据库管理服务接口
type DatabaseService interface {
	// 连接管理
	CreateConnection(ctx context.Context, conn *DatabaseConnection) error
	UpdateConnection(ctx context.Context, id uint, conn *DatabaseConnection) error
	DeleteConnection(ctx context.Context, id uint) error
	GetConnection(ctx context.Context, id uint) (*DatabaseConnection, error)
	ListConnections(ctx context.Context, userID, tenantID uint) ([]*DatabaseConnection, error)
	TestConnection(ctx context.Context, conn *DatabaseConnection) error
	
	// SQL执行
	ExecuteQuery(ctx context.Context, req *QueryRequest) (*QueryResult, error)
	
	// 数据库操作
	ListDatabases(ctx context.Context, connectionID uint) ([]DatabaseInfo, error)
	ListTables(ctx context.Context, connectionID uint, database string) ([]TableInfo, error)
	GetTableSchema(ctx context.Context, connectionID uint, database, table string) ([]ColumnInfo, error)
	GetTableIndexes(ctx context.Context, connectionID uint, database, table string) ([]IndexInfo, error)
	
	// AI查询优化
	OptimizeQuery(ctx context.Context, connectionID uint, sql string) (*QueryOptimization, error)
	GetExecutionPlan(ctx context.Context, connectionID uint, sql string) (*ExecutionPlan, error)
	
	// 备份管理
	CreateBackupConfig(ctx context.Context, config *BackupConfig) error
	UpdateBackupConfig(ctx context.Context, id uint, config *BackupConfig) error
	DeleteBackupConfig(ctx context.Context, id uint) error
	ListBackupConfigs(ctx context.Context, userID, tenantID uint) ([]*BackupConfig, error)
	ExecuteBackup(ctx context.Context, configID uint) error
	ListBackupRecords(ctx context.Context, configID uint) ([]*BackupRecord, error)
	RestoreBackup(ctx context.Context, recordID uint, targetDatabase string) error
}
