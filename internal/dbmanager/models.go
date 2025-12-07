package dbmanager

import (
	"time"
)

// DatabaseType 数据库类型
type DatabaseType string

const (
	DatabaseTypeMySQL      DatabaseType = "mysql"
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
	DatabaseTypeRedis      DatabaseType = "redis"
	DatabaseTypeMongoDB    DatabaseType = "mongodb"
)

// ConnectionStatus 连接状态
type ConnectionStatus string

const (
	ConnectionStatusConnected    ConnectionStatus = "connected"
	ConnectionStatusDisconnected ConnectionStatus = "disconnected"
	ConnectionStatusError        ConnectionStatus = "error"
)

// DatabaseConnection 数据库连接配置模型
type DatabaseConnection struct {
	ID          uint             `json:"id" gorm:"primaryKey"`
	Name        string           `json:"name" gorm:"not null;index"`        // 连接名称
	Type        DatabaseType     `json:"type" gorm:"not null;index"`        // 数据库类型
	Host        string           `json:"host" gorm:"not null"`              // 主机地址
	Port        int              `json:"port" gorm:"not null"`              // 端口
	Username    string           `json:"username"`                          // 用户名
	Password    string           `json:"-"`                                 // 密码（不返回给前端）
	Database    string           `json:"database"`                          // 数据库名
	SSLEnabled  bool             `json:"ssl_enabled" gorm:"default:false"`  // 是否启用SSL
	SSLCert     string           `json:"ssl_cert,omitempty"`                // SSL证书路径
	Status      ConnectionStatus `json:"status" gorm:"default:disconnected"` // 连接状态
	LastError   string           `json:"last_error,omitempty" gorm:"type:text"` // 最后错误信息
	UserID      uint             `json:"user_id" gorm:"index"`              // 所属用户
	TenantID    uint             `json:"tenant_id" gorm:"index"`            // 所属租户
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	LastUsedAt  *time.Time       `json:"last_used_at,omitempty"`            // 最后使用时间
}

// QueryResult SQL查询结果
type QueryResult struct {
	Columns      []string                 `json:"columns"`       // 列名
	Rows         []map[string]interface{} `json:"rows"`          // 数据行
	RowsAffected int64                    `json:"rows_affected"` // 影响行数
	ExecutionTime float64                 `json:"execution_time"` // 执行时间（毫秒）
	Error        string                   `json:"error,omitempty"` // 错误信息
}

// QueryRequest SQL查询请求
type QueryRequest struct {
	ConnectionID uint   `json:"connection_id" binding:"required"` // 连接ID
	SQL          string `json:"sql" binding:"required"`           // SQL语句
	Database     string `json:"database,omitempty"`               // 数据库名（可选）
	Timeout      int    `json:"timeout,omitempty"`                // 超时时间（秒）
	MaxRows      int    `json:"max_rows,omitempty"`               // 最大返回行数
}

// DatabaseInfo 数据库信息
type DatabaseInfo struct {
	Name      string    `json:"name"`       // 数据库名
	Size      int64     `json:"size"`       // 大小（字节）
	Tables    int       `json:"tables"`     // 表数量
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// TableInfo 表信息
type TableInfo struct {
	Name       string    `json:"name"`        // 表名
	Engine     string    `json:"engine"`      // 存储引擎
	Rows       int64     `json:"rows"`        // 行数
	DataSize   int64     `json:"data_size"`   // 数据大小（字节）
	IndexSize  int64     `json:"index_size"`  // 索引大小（字节）
	CreatedAt  time.Time `json:"created_at"`  // 创建时间
	UpdatedAt  time.Time `json:"updated_at"`  // 更新时间
	Comment    string    `json:"comment"`     // 表注释
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Name         string `json:"name"`          // 列名
	Type         string `json:"type"`          // 数据类型
	Nullable     bool   `json:"nullable"`      // 是否可为空
	Key          string `json:"key"`           // 键类型（PRI, UNI, MUL）
	Default      string `json:"default"`       // 默认值
	Extra        string `json:"extra"`         // 额外信息
	Comment      string `json:"comment"`       // 列注释
}

// IndexInfo 索引信息
type IndexInfo struct {
	Name       string   `json:"name"`        // 索引名
	Columns    []string `json:"columns"`     // 索引列
	Unique     bool     `json:"unique"`      // 是否唯一索引
	Type       string   `json:"type"`        // 索引类型
	Comment    string   `json:"comment"`     // 索引注释
}

// QueryOptimization 查询优化建议
type QueryOptimization struct {
	OriginalSQL     string   `json:"original_sql"`      // 原始SQL
	OptimizedSQL    string   `json:"optimized_sql"`     // 优化后的SQL
	Suggestions     []string `json:"suggestions"`       // 优化建议
	EstimatedImprovement float64 `json:"estimated_improvement"` // 预计性能提升（百分比）
	IndexRecommendations []IndexRecommendation `json:"index_recommendations"` // 索引推荐
}

// IndexRecommendation 索引推荐
type IndexRecommendation struct {
	Table       string   `json:"table"`        // 表名
	Columns     []string `json:"columns"`      // 推荐索引的列
	Reason      string   `json:"reason"`       // 推荐原因
	Priority    string   `json:"priority"`     // 优先级（high, medium, low）
	CreateSQL   string   `json:"create_sql"`   // 创建索引的SQL
}

// ExecutionPlan 执行计划
type ExecutionPlan struct {
	Steps []ExecutionStep `json:"steps"` // 执行步骤
}

// ExecutionStep 执行计划步骤
type ExecutionStep struct {
	ID          int     `json:"id"`           // 步骤ID
	SelectType  string  `json:"select_type"`  // 查询类型
	Table       string  `json:"table"`        // 表名
	Type        string  `json:"type"`         // 访问类型
	PossibleKeys string `json:"possible_keys"` // 可能使用的索引
	Key         string  `json:"key"`          // 实际使用的索引
	KeyLen      string  `json:"key_len"`      // 索引长度
	Ref         string  `json:"ref"`          // 引用
	Rows        int64   `json:"rows"`         // 扫描行数
	Filtered    float64 `json:"filtered"`     // 过滤百分比
	Extra       string  `json:"extra"`        // 额外信息
}

// BackupConfig 数据库备份配置
type BackupConfig struct {
	ID           uint         `json:"id" gorm:"primaryKey"`
	ConnectionID uint         `json:"connection_id" gorm:"not null;index"` // 数据库连接ID
	Name         string       `json:"name" gorm:"not null"`                // 备份名称
	Schedule     string       `json:"schedule"`                            // 备份计划（Cron表达式）
	Enabled      bool         `json:"enabled" gorm:"default:true"`         // 是否启用
	BackupPath   string       `json:"backup_path" gorm:"not null"`         // 备份存储路径
	Compression  bool         `json:"compression" gorm:"default:true"`     // 是否压缩
	Retention    int          `json:"retention" gorm:"default:7"`          // 保留天数
	UserID       uint         `json:"user_id" gorm:"index"`                // 所属用户
	TenantID     uint         `json:"tenant_id" gorm:"index"`              // 所属租户
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// BackupRecord 备份记录
type BackupRecord struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	ConfigID     uint      `json:"config_id" gorm:"not null;index"`     // 备份配置ID
	ConnectionID uint      `json:"connection_id" gorm:"not null;index"` // 数据库连接ID
	FilePath     string    `json:"file_path" gorm:"not null"`           // 备份文件路径
	FileSize     int64     `json:"file_size"`                           // 文件大小（字节）
	Status       string    `json:"status" gorm:"index"`                 // 状态：success, failed, running
	ErrorMsg     string    `json:"error_msg,omitempty" gorm:"type:text"` // 错误信息
	StartTime    time.Time `json:"start_time"`                          // 开始时间
	EndTime      *time.Time `json:"end_time,omitempty"`                 // 结束时间
	Duration     int       `json:"duration"`                            // 持续时间（秒）
	UserID       uint      `json:"user_id" gorm:"index"`                // 所属用户
	TenantID     uint      `json:"tenant_id" gorm:"index"`              // 所属租户
	CreatedAt    time.Time `json:"created_at"`
}

// TableName 指定表名
func (DatabaseConnection) TableName() string {
	return "database_connections"
}

func (BackupConfig) TableName() string {
	return "database_backup_configs"
}

func (BackupRecord) TableName() string {
	return "database_backup_records"
}
