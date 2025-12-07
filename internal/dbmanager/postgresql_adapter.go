package dbmanager

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgreSQLAdapter PostgreSQL数据库适配器
type PostgreSQLAdapter struct {
	db *sql.DB
	config *DatabaseConnection
}

// Connect 建立PostgreSQL连接
func (a *PostgreSQLAdapter) Connect(ctx context.Context, config *DatabaseConnection) error {
	sslMode := "disable"
	if config.SSLEnabled {
		sslMode = "require"
	}
	
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.Database, sslMode)
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("打开PostgreSQL连接失败: %w", err)
	}
	
	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// 测试连接
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("PostgreSQL连接测试失败: %w", err)
	}
	
	a.db = db
	a.config = config
	return nil
}

// Disconnect 断开PostgreSQL连接
func (a *PostgreSQLAdapter) Disconnect(ctx context.Context) error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// Ping 测试连接是否有效
func (a *PostgreSQLAdapter) Ping(ctx context.Context) error {
	if a.db == nil {
		return ErrConnectionFailed
	}
	return a.db.PingContext(ctx)
}

// ExecuteQuery 执行查询语句
func (a *PostgreSQLAdapter) ExecuteQuery(ctx context.Context, sql string, timeout time.Duration) (*QueryResult, error) {
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
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("扫描行数据失败: %w", err)
		}
		
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
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
func (a *PostgreSQLAdapter) ExecuteCommand(ctx context.Context, sql string, timeout time.Duration) (*QueryResult, error) {
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
func (a *PostgreSQLAdapter) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	if a.db == nil {
		return nil, ErrConnectionFailed
	}
	
	query := `
		SELECT 
			datname as name,
			pg_database_size(datname) as size,
			(SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'public') as tables,
			(SELECT datcreated FROM pg_database WHERE datname = d.datname) as created_at
		FROM pg_database d
		WHERE datistemplate = false
		ORDER BY datname
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
func (a *PostgreSQLAdapter) ListTables(ctx context.Context, database string) ([]TableInfo, error) {
	if a.db == nil {
		return nil, ErrConnectionFailed
	}
	
	query := `
		SELECT 
			tablename as name,
			'heap' as engine,
			0 as rows,
			pg_total_relation_size(schemaname||'.'||tablename) as data_size,
			pg_indexes_size(schemaname||'.'||tablename) as index_size,
			NOW() as created_at,
			NOW() as updated_at,
			obj_description((schemaname||'.'||tablename)::regclass) as comment
		FROM pg_tables
		WHERE schemaname = 'public'
		ORDER BY tablename
	`
	
	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询表列表失败: %w", err)
	}
	defer rows.Close()
	
	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		var comment sql.NullString
		if err := rows.Scan(&table.Name, &table.Engine, &table.Rows, &table.DataSize,
			&table.IndexSize, &table.CreatedAt, &table.UpdatedAt, &comment); err != nil {
			return nil, fmt.Errorf("扫描表信息失败: %w", err)
		}
		if comment.Valid {
			table.Comment = comment.String
		}
		tables = append(tables, table)
	}
	
	return tables, nil
}

// GetTableSchema 获取表结构信息
func (a *PostgreSQLAdapter) GetTableSchema(ctx context.Context, database, table string) ([]ColumnInfo, error) {
	if a.db == nil {
		return nil, ErrConnectionFailed
	}
	
	query := `
		SELECT 
			column_name as name,
			data_type as type,
			is_nullable = 'YES' as nullable,
			CASE WHEN column_name IN (
				SELECT a.attname
				FROM pg_index i
				JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
				WHERE i.indrelid = $1::regclass AND i.indisprimary
			) THEN 'PRI' ELSE '' END as key_type,
			column_default as default_value,
			'' as extra,
			'' as comment
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`
	
	rows, err := a.db.QueryContext(ctx, query, table)
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
func (a *PostgreSQLAdapter) GetTableIndexes(ctx context.Context, database, table string) ([]IndexInfo, error) {
	if a.db == nil {
		return nil, ErrConnectionFailed
	}
	
	query := `
		SELECT 
			i.relname as name,
			a.attname as column_name,
			ix.indisunique as is_unique,
			am.amname as type,
			'' as comment
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		JOIN pg_am am ON i.relam = am.oid
		WHERE t.relname = $1
		ORDER BY i.relname, a.attnum
	`
	
	rows, err := a.db.QueryContext(ctx, query, table)
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
func (a *PostgreSQLAdapter) GetExecutionPlan(ctx context.Context, sql string) (*ExecutionPlan, error) {
	if a.db == nil {
		return nil, ErrConnectionFailed
	}
	
	explainSQL := "EXPLAIN (FORMAT JSON) " + sql
	var planJSON string
	err := a.db.QueryRowContext(ctx, explainSQL).Scan(&planJSON)
	if err != nil {
		return nil, fmt.Errorf("获取执行计划失败: %w", err)
	}
	
	// PostgreSQL的执行计划是JSON格式，这里简化处理
	// 实际应该解析JSON并转换为ExecutionPlan结构
	return &ExecutionPlan{Steps: []ExecutionStep{}}, nil
}

// Backup 执行数据库备份
func (a *PostgreSQLAdapter) Backup(ctx context.Context, config *BackupConfig) error {
	// PostgreSQL备份通常使用pg_dump命令
	return fmt.Errorf("PostgreSQL备份功能待实现")
}

// Restore 执行数据库恢复
func (a *PostgreSQLAdapter) Restore(ctx context.Context, backupPath string, targetDatabase string) error {
	// PostgreSQL恢复通常使用psql命令导入SQL文件
	return fmt.Errorf("PostgreSQL恢复功能待实现")
}
