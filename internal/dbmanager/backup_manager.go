package dbmanager

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/robfig/cron/v3"
)

// BackupManager 备份管理器
type BackupManager struct {
	service   DatabaseService
	scheduler *cron.Cron
	backupDir string
}

// NewBackupManager 创建备份管理器
func NewBackupManager(service DatabaseService, backupDir string) *BackupManager {
	if backupDir == "" {
		backupDir = "./backups"
	}
	
	// 确保备份目录存在
	os.MkdirAll(backupDir, 0755)
	
	return &BackupManager{
		service:   service,
		scheduler: cron.New(),
		backupDir: backupDir,
	}
}

// Start 启动备份调度器
func (bm *BackupManager) Start() {
	bm.scheduler.Start()
}

// Stop 停止备份调度器
func (bm *BackupManager) Stop() {
	bm.scheduler.Stop()
}

// ScheduleBackup 调度备份任务
func (bm *BackupManager) ScheduleBackup(ctx context.Context, config *BackupConfig) error {
	if config.Schedule == "" {
		return fmt.Errorf("备份计划不能为空")
	}
	
	// 添加定时任务
	_, err := bm.scheduler.AddFunc(config.Schedule, func() {
		if err := bm.service.ExecuteBackup(context.Background(), config.ID); err != nil {
			// 记录错误日志
			fmt.Printf("执行备份失败: %v\n", err)
		}
	})
	
	return err
}

// ExecuteBackup 执行数据库备份
func (bm *BackupManager) ExecuteBackup(ctx context.Context, conn *DatabaseConnection, config *BackupConfig) error {
	switch conn.Type {
	case DatabaseTypeMySQL:
		return bm.backupMySQL(ctx, conn, config)
	case DatabaseTypePostgreSQL:
		return bm.backupPostgreSQL(ctx, conn, config)
	case DatabaseTypeRedis:
		return bm.backupRedis(ctx, conn, config)
	case DatabaseTypeMongoDB:
		return bm.backupMongoDB(ctx, conn, config)
	default:
		return fmt.Errorf("不支持的数据库类型: %s", conn.Type)
	}
}

// backupMySQL 备份MySQL数据库
func (bm *BackupManager) backupMySQL(ctx context.Context, conn *DatabaseConnection, config *BackupConfig) error {
	// 生成备份文件名
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("mysql_%s_%s.sql", conn.Database, timestamp)
	if config.Compression {
		filename += ".gz"
	}
	
	backupPath := filepath.Join(bm.backupDir, filename)
	
	// 构建mysqldump命令
	args := []string{
		fmt.Sprintf("--host=%s", conn.Host),
		fmt.Sprintf("--port=%d", conn.Port),
		fmt.Sprintf("--user=%s", conn.Username),
		fmt.Sprintf("--password=%s", conn.Password),
		"--single-transaction",
		"--routines",
		"--triggers",
		"--events",
		conn.Database,
	}
	
	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	
	// 创建输出文件
	outFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("创建备份文件失败: %w", err)
	}
	defer outFile.Close()
	
	// 如果需要压缩
	if config.Compression {
		gzipCmd := exec.CommandContext(ctx, "gzip")
		gzipCmd.Stdin, _ = cmd.StdoutPipe()
		gzipCmd.Stdout = outFile
		
		if err := gzipCmd.Start(); err != nil {
			return fmt.Errorf("启动压缩失败: %w", err)
		}
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("执行mysqldump失败: %w", err)
		}
		
		if err := gzipCmd.Wait(); err != nil {
			return fmt.Errorf("压缩失败: %w", err)
		}
	} else {
		cmd.Stdout = outFile
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("执行mysqldump失败: %w", err)
		}
	}
	
	// 获取文件大小
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("获取备份文件信息失败: %w", err)
	}
	
	// 更新备份配置中的文件路径
	config.BackupPath = backupPath
	
	fmt.Printf("MySQL备份完成: %s (大小: %d 字节)\n", backupPath, fileInfo.Size())
	
	return nil
}

// backupPostgreSQL 备份PostgreSQL数据库
func (bm *BackupManager) backupPostgreSQL(ctx context.Context, conn *DatabaseConnection, config *BackupConfig) error {
	// 生成备份文件名
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("postgresql_%s_%s.sql", conn.Database, timestamp)
	if config.Compression {
		filename += ".gz"
	}
	
	backupPath := filepath.Join(bm.backupDir, filename)
	
	// 设置环境变量（PostgreSQL使用环境变量传递密码）
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", conn.Password))
	
	// 构建pg_dump命令
	args := []string{
		fmt.Sprintf("--host=%s", conn.Host),
		fmt.Sprintf("--port=%d", conn.Port),
		fmt.Sprintf("--username=%s", conn.Username),
		"--format=plain",
		"--no-owner",
		"--no-acl",
		conn.Database,
	}
	
	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = env
	
	// 创建输出文件
	outFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("创建备份文件失败: %w", err)
	}
	defer outFile.Close()
	
	// 如果需要压缩
	if config.Compression {
		gzipCmd := exec.CommandContext(ctx, "gzip")
		gzipCmd.Stdin, _ = cmd.StdoutPipe()
		gzipCmd.Stdout = outFile
		
		if err := gzipCmd.Start(); err != nil {
			return fmt.Errorf("启动压缩失败: %w", err)
		}
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("执行pg_dump失败: %w", err)
		}
		
		if err := gzipCmd.Wait(); err != nil {
			return fmt.Errorf("压缩失败: %w", err)
		}
	} else {
		cmd.Stdout = outFile
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("执行pg_dump失败: %w", err)
		}
	}
	
	// 获取文件大小
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("获取备份文件信息失败: %w", err)
	}
	
	config.BackupPath = backupPath
	
	fmt.Printf("PostgreSQL备份完成: %s (大小: %d 字节)\n", backupPath, fileInfo.Size())
	
	return nil
}

// backupRedis 备份Redis数据库
func (bm *BackupManager) backupRedis(ctx context.Context, conn *DatabaseConnection, config *BackupConfig) error {
	// Redis备份通常通过SAVE或BGSAVE命令
	// 或者直接复制RDB文件
	return fmt.Errorf("Redis备份功能待实现")
}

// backupMongoDB 备份MongoDB数据库
func (bm *BackupManager) backupMongoDB(ctx context.Context, conn *DatabaseConnection, config *BackupConfig) error {
	// 生成备份目录名
	timestamp := time.Now().Format("20060102_150405")
	backupDir := filepath.Join(bm.backupDir, fmt.Sprintf("mongodb_%s_%s", conn.Database, timestamp))
	
	// 构建mongodump命令
	args := []string{
		fmt.Sprintf("--host=%s", conn.Host),
		fmt.Sprintf("--port=%d", conn.Port),
		fmt.Sprintf("--username=%s", conn.Username),
		fmt.Sprintf("--password=%s", conn.Password),
		fmt.Sprintf("--db=%s", conn.Database),
		fmt.Sprintf("--out=%s", backupDir),
	}
	
	if config.Compression {
		args = append(args, "--gzip")
	}
	
	cmd := exec.CommandContext(ctx, "mongodump", args...)
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行mongodump失败: %w", err)
	}
	
	config.BackupPath = backupDir
	
	fmt.Printf("MongoDB备份完成: %s\n", backupDir)
	
	return nil
}

// RestoreBackup 恢复数据库备份
func (bm *BackupManager) RestoreBackup(ctx context.Context, conn *DatabaseConnection, backupPath string) error {
	switch conn.Type {
	case DatabaseTypeMySQL:
		return bm.restoreMySQL(ctx, conn, backupPath)
	case DatabaseTypePostgreSQL:
		return bm.restorePostgreSQL(ctx, conn, backupPath)
	case DatabaseTypeRedis:
		return bm.restoreRedis(ctx, conn, backupPath)
	case DatabaseTypeMongoDB:
		return bm.restoreMongoDB(ctx, conn, backupPath)
	default:
		return fmt.Errorf("不支持的数据库类型: %s", conn.Type)
	}
}

// restoreMySQL 恢复MySQL数据库
func (bm *BackupManager) restoreMySQL(ctx context.Context, conn *DatabaseConnection, backupPath string) error {
	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}
	
	// 构建mysql命令
	args := []string{
		fmt.Sprintf("--host=%s", conn.Host),
		fmt.Sprintf("--port=%d", conn.Port),
		fmt.Sprintf("--user=%s", conn.Username),
		fmt.Sprintf("--password=%s", conn.Password),
		conn.Database,
	}
	
	cmd := exec.CommandContext(ctx, "mysql", args...)
	
	// 打开备份文件
	var inFile *os.File
	var err error
	
	// 检查是否为压缩文件
	if filepath.Ext(backupPath) == ".gz" {
		// 使用gunzip解压
		gunzipCmd := exec.CommandContext(ctx, "gunzip", "-c", backupPath)
		cmd.Stdin, _ = gunzipCmd.StdoutPipe()
		
		if err := gunzipCmd.Start(); err != nil {
			return fmt.Errorf("启动解压失败: %w", err)
		}
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("执行mysql恢复失败: %w", err)
		}
		
		if err := gunzipCmd.Wait(); err != nil {
			return fmt.Errorf("解压失败: %w", err)
		}
	} else {
		inFile, err = os.Open(backupPath)
		if err != nil {
			return fmt.Errorf("打开备份文件失败: %w", err)
		}
		defer inFile.Close()
		
		cmd.Stdin = inFile
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("执行mysql恢复失败: %w", err)
		}
	}
	
	fmt.Printf("MySQL恢复完成: %s\n", backupPath)
	
	return nil
}

// restorePostgreSQL 恢复PostgreSQL数据库
func (bm *BackupManager) restorePostgreSQL(ctx context.Context, conn *DatabaseConnection, backupPath string) error {
	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}
	
	// 设置环境变量
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", conn.Password))
	
	// 构建psql命令
	args := []string{
		fmt.Sprintf("--host=%s", conn.Host),
		fmt.Sprintf("--port=%d", conn.Port),
		fmt.Sprintf("--username=%s", conn.Username),
		fmt.Sprintf("--dbname=%s", conn.Database),
	}
	
	cmd := exec.CommandContext(ctx, "psql", args...)
	cmd.Env = env
	
	// 检查是否为压缩文件
	if filepath.Ext(backupPath) == ".gz" {
		gunzipCmd := exec.CommandContext(ctx, "gunzip", "-c", backupPath)
		cmd.Stdin, _ = gunzipCmd.StdoutPipe()
		
		if err := gunzipCmd.Start(); err != nil {
			return fmt.Errorf("启动解压失败: %w", err)
		}
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("执行psql恢复失败: %w", err)
		}
		
		if err := gunzipCmd.Wait(); err != nil {
			return fmt.Errorf("解压失败: %w", err)
		}
	} else {
		inFile, err := os.Open(backupPath)
		if err != nil {
			return fmt.Errorf("打开备份文件失败: %w", err)
		}
		defer inFile.Close()
		
		cmd.Stdin = inFile
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("执行psql恢复失败: %w", err)
		}
	}
	
	fmt.Printf("PostgreSQL恢复完成: %s\n", backupPath)
	
	return nil
}

// restoreRedis 恢复Redis数据库
func (bm *BackupManager) restoreRedis(ctx context.Context, conn *DatabaseConnection, backupPath string) error {
	return fmt.Errorf("Redis恢复功能待实现")
}

// restoreMongoDB 恢复MongoDB数据库
func (bm *BackupManager) restoreMongoDB(ctx context.Context, conn *DatabaseConnection, backupPath string) error {
	// 检查备份目录是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份目录不存在: %s", backupPath)
	}
	
	// 构建mongorestore命令
	args := []string{
		fmt.Sprintf("--host=%s", conn.Host),
		fmt.Sprintf("--port=%d", conn.Port),
		fmt.Sprintf("--username=%s", conn.Username),
		fmt.Sprintf("--password=%s", conn.Password),
		fmt.Sprintf("--db=%s", conn.Database),
		"--drop", // 恢复前删除现有数据
		backupPath,
	}
	
	// 检查是否为压缩备份
	if _, err := os.Stat(filepath.Join(backupPath, conn.Database+".metadata.json.gz")); err == nil {
		args = append(args, "--gzip")
	}
	
	cmd := exec.CommandContext(ctx, "mongorestore", args...)
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行mongorestore失败: %w", err)
	}
	
	fmt.Printf("MongoDB恢复完成: %s\n", backupPath)
	
	return nil
}

// CleanupOldBackups 清理过期备份
func (bm *BackupManager) CleanupOldBackups(ctx context.Context, config *BackupConfig) error {
	if config.Retention <= 0 {
		return nil // 不清理
	}
	
	// 计算过期时间
	expiryTime := time.Now().AddDate(0, 0, -config.Retention)
	
	// 遍历备份目录
	files, err := os.ReadDir(bm.backupDir)
	if err != nil {
		return fmt.Errorf("读取备份目录失败: %w", err)
	}
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		// 获取文件信息
		info, err := file.Info()
		if err != nil {
			continue
		}
		
		// 检查是否过期
		if info.ModTime().Before(expiryTime) {
			filePath := filepath.Join(bm.backupDir, file.Name())
			if err := os.Remove(filePath); err != nil {
				fmt.Printf("删除过期备份失败: %s, 错误: %v\n", filePath, err)
			} else {
				fmt.Printf("已删除过期备份: %s\n", filePath)
			}
		}
	}
	
	return nil
}

// ValidateBackup 验证备份完整性
func (bm *BackupManager) ValidateBackup(ctx context.Context, backupPath string) error {
	// 检查文件是否存在
	fileInfo, err := os.Stat(backupPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}
	
	// 检查文件大小
	if fileInfo.Size() == 0 {
		return fmt.Errorf("备份文件为空: %s", backupPath)
	}
	
	// 检查文件是否可读
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("无法读取备份文件: %w", err)
	}
	defer file.Close()
	
	// 如果是压缩文件，尝试读取前几个字节验证格式
	if filepath.Ext(backupPath) == ".gz" {
		header := make([]byte, 2)
		if _, err := file.Read(header); err != nil {
			return fmt.Errorf("读取备份文件头失败: %w", err)
		}
		
		// 检查gzip魔数
		if header[0] != 0x1f || header[1] != 0x8b {
			return fmt.Errorf("备份文件格式无效")
		}
	}
	
	fmt.Printf("备份文件验证通过: %s\n", backupPath)
	
	return nil
}
