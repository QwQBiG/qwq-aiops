package dbmanager

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLAdapter MySQL数据库适配器
type MySQLAdapter struct {
	db *sql.DB
	config *DatabaseConnection
}

// Connect 建立MySQL连接
func (a *MySQLAdapter) Connect(ctx context.Context, config *DatabaseConnection) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		config.Username, config.Password, config.Host, config.Port, config.Database)
	
	if config.SSLEnabled {
		dsn += "&tls=true"
	}
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("打开MySQL连接失败: %w", err)
	}
	
	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// 测试连接
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("MySQL连接测试失败: %w", err)
	}
	
	a.db = db
	a.config = config
	return nil
}

// Disconnect 断开MySQL连接
func (a *MySQLAdapter) Disconnect(ctx context.Context) error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// Ping 测试连接是否有效
func (a *MySQLAdapter) Ping(ctx context.Context) error {
	if a.db == nil {
		return ErrConnectionFailed
	}
	return a.db.PingContext(ctx)
}

// ExecuteQuery 执行查询语句
func (a *MySQLAdapter) ExecuteQuery(ctx context.Context, sql string, timeout time.Duration) (*QueryResult, error) {
	if a.db == nil {
		return nil, ErrConnectionFailed
	}
	
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	
	startTime := time.Now()
	rows, err := a.db.QueryContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()
	
	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %w", err)
	}
	
	// 读取数据
	result := &QueryResult{
		Columns: columns,
		Rows:    make([]map[string]interface{}, 0),
	}
	
	for rows.Next() {
		// 创建扫描目标
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("扫描行数据失败: %w", err)
		}
		
		// 构建行数据
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// 处理字节数组
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		result.Rows = append(result.Rows, row)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取结果集失败: %w", err)
	}
	
	result.ExecutionTime = float64(time.Since(startTime).Milliseconds())
	return result, nil
}

// ExecuteCommand 执行命令语句
func (a *MySQLAdapter) ExecuteCommand(ctx context.Context, sql string, timeout time.Duration) (*QueryResult, error) {
	if a.db == nil {
		return nil, ErrConnectionFailed
	}
	
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	
	startTime := time.Now()
	result, err := a.db.ExecContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("执行命令失败: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	
	return &QueryResult{
		RowsAffected:  rowsAffected,
		ExecutionTime: float64(time.Since(startTime).Milliseconds()),
	}, nil
}

// ListDatabases 列出所有数据库
func (a *MySQLAdapter) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	if a.db == nil {
		return nil, ErrConnectionFailed
	}
	
	query := `
		SELECT 
			SCHEMA_NAME as name,
			COALESCE(SUM(DATA_LENGTH + INDEX_LENGTH), 0) as size,
			COUNT(DISTINCT TABLE_NAME) as tables,
			CREATE_TIME as created_at
		FROM information_schema.SCHEMATA
		LEFT JOIN information_schema.TABLES ON SCHEMA_NAME = TABLE_SCHEMA
		WHERE SCHEMA_NAME NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		GROUP BY SCHEMA_NAME, CREATE_TIME
		ORDER BY SCHEMA_NAME
	`
	
	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询数据库列表失败: %w", err)
	}
	defer rows.Close()
	
	var databases []DatabaseInfo
	for rows.Next() {
		var db DatabaseInfo
		var createdAt sql.NullTime
		if err := rows.Scan(&db.Name, &db.Size, &db.Tables, &createdAt); err != nil {
			return nil, fmt.Errorf("扫描数据库信息失败: %w", err)
		}
		if createdAt.Valid {
			db.CreatedAt = createdAt.Time
		}
		databases = append(databases, db)
	}
	
	return databases, nil
}

// ListTables 列出指定数据库的所有表
func (a *MySQLAdapter) ListTables(ctx context.Context, database string) ([]TableInfo, error) {
	if a.db == nil {
		return nil, ErrConnectionFailed
	}
	
	query := `
		SELECT 
			TABLE_NAME as name,
			ENGINE as engine,
			TABLE_ROWS as rows,
			DATA_LENGTH as data_size,
			INDEX_LENGTH as index_size,
			CREATE_TIME as created_at,
			UPDATE_TIME as updated_at,
			TABLE_COMMENT as comment
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME
	`
	
	rows, err := a.db.QueryContext(ctx, query, database)
	if err != nil {
		return nil, fmt.Errorf("查询表列表失败: %w", err)
	}
	defer rows.Close()
	
	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(&table.Name, &table.Engine, &table.Rows, &table.DataSize, 
			&table.IndexSize, &createdAt, &updatedAt, &table.Comment); err != nil {
			return nil, fmt.Errorf("扫描表信息失败: %w", err)
		}
		if createdAt.Valid {
			table.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			table.UpdatedAt = updatedAt.Time
		}
		tables = append(tables, table)
	}
	
	return tables, nil
}

// GetTableSchema 获取表结构信息
func (a *MySQLAdapter) GetTableSchema(ctx context.Context, database, table string) ([]ColumnInfo, error) {
	if a.db == nil {
		return nil, ErrConnectionFailed
	}
	
	query := `
		SELECT 
			COLUMN_NAME as name,
			COLUMN_TYPE as type,
			IS_NULLABLE = 'YES' as nullable,
			COLUMN_KEY as key_type,
			COLUMN_DEFAULT as default_value,
			EXTRA as extra,
			COLUMN_COMMENT as comment
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`
	
	rows, err := a.db.QueryContext(ctx, query, database, table)
	if err != nil {
		return nil, fmt.Errorf("查询表结构失败: %w", err)
	}
	defer rows.Close()
	
	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var defaultValue sql.NullString
		if err := rows.Scan(&col.Name, &col.Type, &col.Nullable, &col.Key, 
			&defaultValue, &col.Extra, &col.Comment); err != nil {
			return nil, fmt.Errorf("扫描列信息失败: %w", err)
		}
		if defaultValue.Valid {
			col.Default = defaultValue.String
		}
		columns = append(columns, col)
	}
	
	return columns, nil
}

// GetTableIndexes 获取表索引信息
func (a *MySQLAdapter) GetTableIndexes(ctx context.Context, database, table string) ([]IndexInfo, error) {
	if a.db == nil {
		return nil, ErrConnectionFailed
	}
	
	query := `
		SELECT 
			INDEX_NAME as name,
			COLUMN_NAME as column_name,
			NON_UNIQUE = 0 as is_unique,
			INDEX_TYPE as type,
			INDEX_COMMENT as comment
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`
	
	rows, err := a.db.QueryContext(ctx, query, database, table)
	if err != nil {
		return nil, fmt.Errorf("查询索引信息失败: %w", err)
	}
	defer rows.Close()
	
	indexMap := make(map[string]*IndexInfo)
	for rows.Next() {
		var name, columnName, indexType, comment string
		var unique bool
		if err := rows.Scan(&name, &columnName, &unique, &indexType, &comment); err != nil {
			return nil, fmt.Errorf("扫描索引信息失败: %w", err)
		}
		
		if idx, exists := indexMap[name]; exists {
			idx.Columns = append(idx.Columns, columnName)
		} else {
			indexMap[name] = &IndexInfo{
				Name:    name,
				Columns: []string{columnName},
				Unique:  unique,
				Type:    indexType,
				Comment: comment,
			}
		}
	}
	
	var indexes []IndexInfo
	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}
	
	return indexes, nil
}

// GetExecutionPlan 获取SQL执行计划
func (a *MySQLAdapter) GetExecutionPlan(ctx context.Context, sqlQuery string) (*ExecutionPlan, error) {
	if a.db == nil {
		return nil, ErrConnectionFailed
	}
	
	explainSQL := "EXPLAIN " + sqlQuery
	rows, err := a.db.QueryContext(ctx, explainSQL)
	if err != nil {
		return nil, fmt.Errorf("获取执行计划失败: %w", err)
	}
	defer rows.Close()
	
	var steps []ExecutionStep
	for rows.Next() {
		var step ExecutionStep
		var possibleKeys, key, keyLen, ref, extra sql.NullString
		if err := rows.Scan(&step.ID, &step.SelectType, &step.Table, &step.Type,
			&possibleKeys, &key, &keyLen, &ref, &step.Rows, &step.Filtered, &extra); err != nil {
			return nil, fmt.Errorf("扫描执行计划失败: %w", err)
		}
		
		if possibleKeys.Valid {
			step.PossibleKeys = possibleKeys.String
		}
		if key.Valid {
			step.Key = key.String
		}
		if keyLen.Valid {
			step.KeyLen = keyLen.String
		}
		if ref.Valid {
			step.Ref = ref.String
		}
		if extra.Valid {
			step.Extra = extra.String
		}
		
		steps = append(steps, step)
	}
	
	return &ExecutionPlan{Steps: steps}, nil
}

// Backup 执行数据库备份
func (a *MySQLAdapter) Backup(ctx context.Context, config *BackupConfig) error {
	// MySQL备份通常使用mysqldump命令
	// 这里返回未实现错误，实际实现需要调用系统命令
	return fmt.Errorf("MySQL备份功能待实现")
}

// Restore 执行数据库恢复
func (a *MySQLAdapter) Restore(ctx context.Context, backupPath string, targetDatabase string) error {
	// MySQL恢复通常使用mysql命令导入SQL文件
	// 这里返回未实现错误，实际实现需要调用系统命令
	return fmt.Errorf("MySQL恢复功能待实现")
}
