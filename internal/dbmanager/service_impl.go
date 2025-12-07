package dbmanager

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// DatabaseServiceImpl 数据库管理服务实现
type DatabaseServiceImpl struct {
	db                *gorm.DB
	connectionManager *ConnectionManager
}

// NewDatabaseService 创建数据库管理服务
func NewDatabaseService(db *gorm.DB, encryptionKey string) DatabaseService {
	return &DatabaseServiceImpl{
		db:                db,
		connectionManager: NewConnectionManager(encryptionKey),
	}
}

// CreateConnection 创建数据库连接配置
func (s *DatabaseServiceImpl) CreateConnection(ctx context.Context, conn *DatabaseConnection) error {
	// 加密密码
	if conn.Password != "" {
		encryptedPassword, err := s.connectionManager.EncryptPassword(conn.Password)
		if err != nil {
			return fmt.Errorf("加密密码失败: %w", err)
		}
		conn.Password = encryptedPassword
	}
	
	// 保存到数据库
	if err := s.db.WithContext(ctx).Create(conn).Error; err != nil {
		return fmt.Errorf("保存连接配置失败: %w", err)
	}
	
	return nil
}

// UpdateConnection 更新数据库连接配置
func (s *DatabaseServiceImpl) UpdateConnection(ctx context.Context, id uint, conn *DatabaseConnection) error {
	// 查找现有连接
	var existing DatabaseConnection
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return fmt.Errorf("连接不存在: %w", err)
	}
	
	// 如果密码有变化，重新加密
	if conn.Password != "" && conn.Password != existing.Password {
		encryptedPassword, err := s.connectionManager.EncryptPassword(conn.Password)
		if err != nil {
			return fmt.Errorf("加密密码失败: %w", err)
		}
		conn.Password = encryptedPassword
	} else {
		conn.Password = existing.Password
	}
	
	// 更新数据库
	conn.ID = id
	if err := s.db.WithContext(ctx).Save(conn).Error; err != nil {
		return fmt.Errorf("更新连接配置失败: %w", err)
	}
	
	// 关闭旧连接
	s.connectionManager.CloseConnection(ctx, id)
	
	return nil
}

// DeleteConnection 删除数据库连接配置
func (s *DatabaseServiceImpl) DeleteConnection(ctx context.Context, id uint) error {
	// 关闭连接
	s.connectionManager.CloseConnection(ctx, id)
	
	// 删除配置
	if err := s.db.WithContext(ctx).Delete(&DatabaseConnection{}, id).Error; err != nil {
		return fmt.Errorf("删除连接配置失败: %w", err)
	}
	
	return nil
}

// GetConnection 获取数据库连接配置
func (s *DatabaseServiceImpl) GetConnection(ctx context.Context, id uint) (*DatabaseConnection, error) {
	var conn DatabaseConnection
	if err := s.db.WithContext(ctx).First(&conn, id).Error; err != nil {
		return nil, fmt.Errorf("连接不存在: %w", err)
	}
	
	// 不返回密码
	conn.Password = ""
	
	return &conn, nil
}

// ListConnections 列出数据库连接配置
func (s *DatabaseServiceImpl) ListConnections(ctx context.Context, userID, tenantID uint) ([]*DatabaseConnection, error) {
	var connections []*DatabaseConnection
	query := s.db.WithContext(ctx)
	
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	
	if err := query.Find(&connections).Error; err != nil {
		return nil, fmt.Errorf("查询连接列表失败: %w", err)
	}
	
	// 不返回密码
	for _, conn := range connections {
		conn.Password = ""
	}
	
	return connections, nil
}

// TestConnection 测试数据库连接
func (s *DatabaseServiceImpl) TestConnection(ctx context.Context, conn *DatabaseConnection) error {
	// 解密密码
	if conn.Password != "" {
		decryptedPassword, err := s.connectionManager.DecryptPassword(conn.Password)
		if err != nil {
			return fmt.Errorf("解密密码失败: %w", err)
		}
		conn.Password = decryptedPassword
	}
	
	return s.connectionManager.TestConnection(ctx, conn)
}

// ExecuteQuery 执行SQL查询
func (s *DatabaseServiceImpl) ExecuteQuery(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	// 获取连接配置
	var conn DatabaseConnection
	if err := s.db.WithContext(ctx).First(&conn, req.ConnectionID).Error; err != nil {
		return nil, fmt.Errorf("连接不存在: %w", err)
	}
	
	// 解密密码
	if conn.Password != "" {
		decryptedPassword, err := s.connectionManager.DecryptPassword(conn.Password)
		if err != nil {
			return nil, fmt.Errorf("解密密码失败: %w", err)
		}
		conn.Password = decryptedPassword
	}
	
	// 获取或创建连接
	adapter, err := s.connectionManager.GetConnection(ctx, req.ConnectionID)
	if err != nil {
		// 连接不存在，创建新连接
		adapter, err = s.connectionManager.CreateConnection(ctx, &conn)
		if err != nil {
			return nil, err
		}
	}
	
	// 设置超时
	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second // 默认30秒超时
	}
	
	// 执行查询
	result, err := adapter.ExecuteQuery(ctx, req.SQL, timeout)
	if err != nil {
		return nil, err
	}
	
	// 限制返回行数
	if req.MaxRows > 0 && len(result.Rows) > req.MaxRows {
		result.Rows = result.Rows[:req.MaxRows]
	}
	
	// 更新最后使用时间
	now := time.Now()
	s.db.WithContext(ctx).Model(&DatabaseConnection{}).
		Where("id = ?", req.ConnectionID).
		Update("last_used_at", now)
	
	return result, nil
}

// ListDatabases 列出所有数据库
func (s *DatabaseServiceImpl) ListDatabases(ctx context.Context, connectionID uint) ([]DatabaseInfo, error) {
	adapter, err := s.getAdapter(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	
	return adapter.ListDatabases(ctx)
}

// ListTables 列出指定数据库的所有表
func (s *DatabaseServiceImpl) ListTables(ctx context.Context, connectionID uint, database string) ([]TableInfo, error) {
	adapter, err := s.getAdapter(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	
	return adapter.ListTables(ctx, database)
}

// GetTableSchema 获取表结构信息
func (s *DatabaseServiceImpl) GetTableSchema(ctx context.Context, connectionID uint, database, table string) ([]ColumnInfo, error) {
	adapter, err := s.getAdapter(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	
	return adapter.GetTableSchema(ctx, database, table)
}

// GetTableIndexes 获取表索引信息
func (s *DatabaseServiceImpl) GetTableIndexes(ctx context.Context, connectionID uint, database, table string) ([]IndexInfo, error) {
	adapter, err := s.getAdapter(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	
	return adapter.GetTableIndexes(ctx, database, table)
}

// OptimizeQuery AI查询优化（待实现）
func (s *DatabaseServiceImpl) OptimizeQuery(ctx context.Context, connectionID uint, sql string) (*QueryOptimization, error) {
	// TODO: 集成AI服务进行查询优化
	return &QueryOptimization{
		OriginalSQL:  sql,
		OptimizedSQL: sql,
		Suggestions:  []string{"AI查询优化功能待实现"},
	}, nil
}

// GetExecutionPlan 获取SQL执行计划
func (s *DatabaseServiceImpl) GetExecutionPlan(ctx context.Context, connectionID uint, sql string) (*ExecutionPlan, error) {
	adapter, err := s.getAdapter(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	
	return adapter.GetExecutionPlan(ctx, sql)
}

// CreateBackupConfig 创建备份配置
func (s *DatabaseServiceImpl) CreateBackupConfig(ctx context.Context, config *BackupConfig) error {
	return s.db.WithContext(ctx).Create(config).Error
}

// UpdateBackupConfig 更新备份配置
func (s *DatabaseServiceImpl) UpdateBackupConfig(ctx context.Context, id uint, config *BackupConfig) error {
	config.ID = id
	return s.db.WithContext(ctx).Save(config).Error
}

// DeleteBackupConfig 删除备份配置
func (s *DatabaseServiceImpl) DeleteBackupConfig(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&BackupConfig{}, id).Error
}

// ListBackupConfigs 列出备份配置
func (s *DatabaseServiceImpl) ListBackupConfigs(ctx context.Context, userID, tenantID uint) ([]*BackupConfig, error) {
	var configs []*BackupConfig
	query := s.db.WithContext(ctx)
	
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	
	if err := query.Find(&configs).Error; err != nil {
		return nil, err
	}
	
	return configs, nil
}

// ExecuteBackup 执行备份
func (s *DatabaseServiceImpl) ExecuteBackup(ctx context.Context, configID uint) error {
	// 获取备份配置
	var config BackupConfig
	if err := s.db.WithContext(ctx).First(&config, configID).Error; err != nil {
		return fmt.Errorf("备份配置不存在: %w", err)
	}
	
	// 创建备份记录
	record := &BackupRecord{
		ConfigID:     configID,
		ConnectionID: config.ConnectionID,
		Status:       "running",
		StartTime:    time.Now(),
		UserID:       config.UserID,
		TenantID:     config.TenantID,
	}
	
	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		return fmt.Errorf("创建备份记录失败: %w", err)
	}
	
	// 获取适配器
	adapter, err := s.getAdapter(ctx, config.ConnectionID)
	if err != nil {
		s.updateBackupRecord(ctx, record.ID, "failed", err.Error())
		return err
	}
	
	// 执行备份
	if err := adapter.Backup(ctx, &config); err != nil {
		s.updateBackupRecord(ctx, record.ID, "failed", err.Error())
		return fmt.Errorf("备份执行失败: %w", err)
	}
	
	// 更新备份记录
	s.updateBackupRecord(ctx, record.ID, "success", "")
	
	return nil
}

// ListBackupRecords 列出备份记录
func (s *DatabaseServiceImpl) ListBackupRecords(ctx context.Context, configID uint) ([]*BackupRecord, error) {
	var records []*BackupRecord
	if err := s.db.WithContext(ctx).
		Where("config_id = ?", configID).
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	
	return records, nil
}

// RestoreBackup 恢复备份
func (s *DatabaseServiceImpl) RestoreBackup(ctx context.Context, recordID uint, targetDatabase string) error {
	// 获取备份记录
	var record BackupRecord
	if err := s.db.WithContext(ctx).First(&record, recordID).Error; err != nil {
		return fmt.Errorf("备份记录不存在: %w", err)
	}
	
	// 获取适配器
	adapter, err := s.getAdapter(ctx, record.ConnectionID)
	if err != nil {
		return err
	}
	
	// 执行恢复
	return adapter.Restore(ctx, record.FilePath, targetDatabase)
}

// getAdapter 获取数据库适配器
func (s *DatabaseServiceImpl) getAdapter(ctx context.Context, connectionID uint) (DatabaseAdapter, error) {
	// 获取连接配置
	var conn DatabaseConnection
	if err := s.db.WithContext(ctx).First(&conn, connectionID).Error; err != nil {
		return nil, fmt.Errorf("连接不存在: %w", err)
	}
	
	// 解密密码
	if conn.Password != "" {
		decryptedPassword, err := s.connectionManager.DecryptPassword(conn.Password)
		if err != nil {
			return nil, fmt.Errorf("解密密码失败: %w", err)
		}
		conn.Password = decryptedPassword
	}
	
	// 获取或创建连接
	adapter, err := s.connectionManager.GetConnection(ctx, connectionID)
	if err != nil {
		// 连接不存在，创建新连接
		adapter, err = s.connectionManager.CreateConnection(ctx, &conn)
		if err != nil {
			return nil, err
		}
	}
	
	return adapter, nil
}

// updateBackupRecord 更新备份记录状态
func (s *DatabaseServiceImpl) updateBackupRecord(ctx context.Context, recordID uint, status, errorMsg string) {
	now := time.Now()
	updates := map[string]interface{}{
		"status":   status,
		"end_time": now,
	}
	
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}
	
	s.db.WithContext(ctx).Model(&BackupRecord{}).
		Where("id = ?", recordID).
		Updates(updates)
}
