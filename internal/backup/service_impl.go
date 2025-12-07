package backup

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

// BackupServiceImpl 备份服务实现
type BackupServiceImpl struct {
	db              *gorm.DB
	storageBackends map[StorageType]StorageBackend
}

// NewBackupService 创建备份服务
func NewBackupService(db *gorm.DB) BackupService {
	service := &BackupServiceImpl{
		db:              db,
		storageBackends: make(map[StorageType]StorageBackend),
	}
	
	// 注册存储后端
	service.registerStorageBackends()
	
	return service
}

// registerStorageBackends 注册存储后端
func (s *BackupServiceImpl) registerStorageBackends() {
	s.storageBackends[StorageTypeLocal] = NewLocalStorage()
	// TODO: 添加 S3, FTP, SFTP 存储后端
}

// CreatePolicy 创建备份策略
func (s *BackupServiceImpl) CreatePolicy(ctx context.Context, policy *BackupPolicy) error {
	// 验证存储后端
	if _, exists := s.storageBackends[policy.StorageType]; !exists {
		return ErrInvalidStorage
	}
	
	return s.db.WithContext(ctx).Create(policy).Error
}

// UpdatePolicy 更新备份策略
func (s *BackupServiceImpl) UpdatePolicy(ctx context.Context, id uint, policy *BackupPolicy) error {
	policy.ID = id
	return s.db.WithContext(ctx).Save(policy).Error
}

// DeletePolicy 删除备份策略
func (s *BackupServiceImpl) DeletePolicy(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&BackupPolicy{}, id).Error
}

// GetPolicy 获取备份策略
func (s *BackupServiceImpl) GetPolicy(ctx context.Context, id uint) (*BackupPolicy, error) {
	var policy BackupPolicy
	if err := s.db.WithContext(ctx).First(&policy, id).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

// ListPolicies 列出备份策略
func (s *BackupServiceImpl) ListPolicies(ctx context.Context, userID, tenantID uint) ([]*BackupPolicy, error) {
	var policies []*BackupPolicy
	query := s.db.WithContext(ctx)
	
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	
	if err := query.Find(&policies).Error; err != nil {
		return nil, err
	}
	
	return policies, nil
}

// ExecuteBackup 执行备份
func (s *BackupServiceImpl) ExecuteBackup(ctx context.Context, policyID uint) (*BackupJob, error) {
	// 获取策略
	policy, err := s.GetPolicy(ctx, policyID)
	if err != nil {
		return nil, err
	}
	
	// 创建备份任务
	job := &BackupJob{
		PolicyID:  policyID,
		Status:    StatusRunning,
		StartTime: time.Now(),
		UserID:    policy.UserID,
		TenantID:  policy.TenantID,
	}
	
	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, err
	}
	
	// 异步执行备份
	go s.performBackup(context.Background(), job, policy)
	
	return job, nil
}

// performBackup 执行备份操作
func (s *BackupServiceImpl) performBackup(ctx context.Context, job *BackupJob, policy *BackupPolicy) {
	// 根据备份类型执行备份
	var backupPath string
	var err error
	
	switch policy.Type {
	case BackupTypeDatabase:
		backupPath, err = s.backupDatabase(ctx, policy)
	case BackupTypeFiles:
		backupPath, err = s.backupFiles(ctx, policy)
	case BackupTypeContainer:
		backupPath, err = s.backupContainer(ctx, policy)
	default:
		err = fmt.Errorf("unsupported backup type: %s", policy.Type)
	}
	
	if err != nil {
		s.updateJobStatus(ctx, job.ID, StatusFailed, err.Error(), "")
		return
	}
	
	// 计算文件大小和校验和
	fileInfo, _ := os.Stat(backupPath)
	checksum, _ := s.calculateChecksum(backupPath)
	
	// 上传到存储后端
	storage := s.storageBackends[policy.StorageType]
	remotePath, err := storage.Upload(ctx, backupPath, policy.StorageConfig)
	if err != nil {
		s.updateJobStatus(ctx, job.ID, StatusFailed, err.Error(), "")
		return
	}
	
	// 更新任务状态
	s.updateJobStatus(ctx, job.ID, StatusCompleted, "", remotePath)
	
	// 更新文件信息
	s.db.Model(&BackupJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
		"file_size": fileInfo.Size(),
		"checksum":  checksum,
	})
	
	// 清理本地临时文件
	os.Remove(backupPath)
	
	// 清理过期备份
	s.cleanupOldBackups(ctx, policy)
}

// backupDatabase 备份数据库
func (s *BackupServiceImpl) backupDatabase(ctx context.Context, policy *BackupPolicy) (string, error) {
	// TODO: 实现数据库备份逻辑
	// 这里应该调用 dbmanager 包的备份功能
	return "", fmt.Errorf("database backup not implemented")
}

// backupFiles 备份文件
func (s *BackupServiceImpl) backupFiles(ctx context.Context, policy *BackupPolicy) (string, error) {
	// TODO: 实现文件备份逻辑
	return "", fmt.Errorf("file backup not implemented")
}

// backupContainer 备份容器
func (s *BackupServiceImpl) backupContainer(ctx context.Context, policy *BackupPolicy) (string, error) {
	// TODO: 实现容器备份逻辑
	return "", fmt.Errorf("container backup not implemented")
}

// calculateChecksum 计算文件校验和
func (s *BackupServiceImpl) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// updateJobStatus 更新任务状态
func (s *BackupServiceImpl) updateJobStatus(ctx context.Context, jobID uint, status BackupStatus, errorMsg, filePath string) {
	now := time.Now()
	updates := map[string]interface{}{
		"status":   status,
		"end_time": now,
	}
	
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}
	if filePath != "" {
		updates["file_path"] = filePath
	}
	
	s.db.WithContext(ctx).Model(&BackupJob{}).Where("id = ?", jobID).Updates(updates)
	
	// 计算持续时间
	var job BackupJob
	if err := s.db.WithContext(ctx).First(&job, jobID).Error; err == nil {
		duration := int(now.Sub(job.StartTime).Seconds())
		s.db.Model(&BackupJob{}).Where("id = ?", jobID).Update("duration", duration)
	}
}

// cleanupOldBackups 清理过期备份
func (s *BackupServiceImpl) cleanupOldBackups(ctx context.Context, policy *BackupPolicy) {
	if policy.Retention <= 0 {
		return
	}
	
	cutoffTime := time.Now().AddDate(0, 0, -policy.Retention)
	
	var oldJobs []*BackupJob
	s.db.WithContext(ctx).
		Where("policy_id = ? AND created_at < ? AND status = ?", policy.ID, cutoffTime, StatusCompleted).
		Find(&oldJobs)
	
	storage := s.storageBackends[policy.StorageType]
	
	for _, job := range oldJobs {
		// 删除存储中的文件
		storage.Delete(ctx, job.FilePath, policy.StorageConfig)
		
		// 删除任务记录
		s.db.WithContext(ctx).Delete(job)
	}
}

// ListBackupJobs 列出备份任务
func (s *BackupServiceImpl) ListBackupJobs(ctx context.Context, policyID uint) ([]*BackupJob, error) {
	var jobs []*BackupJob
	if err := s.db.WithContext(ctx).
		Where("policy_id = ?", policyID).
		Order("created_at DESC").
		Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// GetBackupJob 获取备份任务
func (s *BackupServiceImpl) GetBackupJob(ctx context.Context, jobID uint) (*BackupJob, error) {
	var job BackupJob
	if err := s.db.WithContext(ctx).First(&job, jobID).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// RestoreBackup 恢复备份
func (s *BackupServiceImpl) RestoreBackup(ctx context.Context, jobID uint, target *RestoreTarget) (*RestoreJob, error) {
	// 获取备份任务
	backupJob, err := s.GetBackupJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	
	// 创建恢复任务
	restoreJob := &RestoreJob{
		BackupID:  jobID,
		Status:    StatusRunning,
		StartTime: time.Now(),
		UserID:    backupJob.UserID,
		TenantID:  backupJob.TenantID,
	}
	
	if err := s.db.WithContext(ctx).Create(restoreJob).Error; err != nil {
		return nil, err
	}
	
	// 异步执行恢复
	go s.performRestore(context.Background(), restoreJob, backupJob, target)
	
	return restoreJob, nil
}

// performRestore 执行恢复操作
func (s *BackupServiceImpl) performRestore(ctx context.Context, restoreJob *RestoreJob, backupJob *BackupJob, target *RestoreTarget) {
	// 获取策略
	policy, err := s.GetPolicy(ctx, backupJob.PolicyID)
	if err != nil {
		s.updateRestoreJobStatus(ctx, restoreJob.ID, StatusFailed, err.Error())
		return
	}
	
	// 从存储后端下载备份文件
	storage := s.storageBackends[policy.StorageType]
	localPath := filepath.Join(os.TempDir(), fmt.Sprintf("restore_%d_%d", restoreJob.ID, time.Now().Unix()))
	
	if err := storage.Download(ctx, backupJob.FilePath, localPath, policy.StorageConfig); err != nil {
		s.updateRestoreJobStatus(ctx, restoreJob.ID, StatusFailed, err.Error())
		return
	}
	defer os.Remove(localPath)
	
	// 验证备份文件
	checksum, _ := s.calculateChecksum(localPath)
	if checksum != backupJob.Checksum {
		s.updateRestoreJobStatus(ctx, restoreJob.ID, StatusFailed, "backup file checksum mismatch")
		return
	}
	
	// 根据类型执行恢复
	switch policy.Type {
	case BackupTypeDatabase:
		err = s.restoreDatabase(ctx, localPath, target)
	case BackupTypeFiles:
		err = s.restoreFiles(ctx, localPath, target)
	case BackupTypeContainer:
		err = s.restoreContainer(ctx, localPath, target)
	default:
		err = fmt.Errorf("unsupported backup type: %s", policy.Type)
	}
	
	if err != nil {
		s.updateRestoreJobStatus(ctx, restoreJob.ID, StatusFailed, err.Error())
		return
	}
	
	s.updateRestoreJobStatus(ctx, restoreJob.ID, StatusCompleted, "")
}

// restoreDatabase 恢复数据库
func (s *BackupServiceImpl) restoreDatabase(ctx context.Context, backupPath string, target *RestoreTarget) error {
	// TODO: 实现数据库恢复逻辑
	return fmt.Errorf("database restore not implemented")
}

// restoreFiles 恢复文件
func (s *BackupServiceImpl) restoreFiles(ctx context.Context, backupPath string, target *RestoreTarget) error {
	// TODO: 实现文件恢复逻辑
	return fmt.Errorf("file restore not implemented")
}

// restoreContainer 恢复容器
func (s *BackupServiceImpl) restoreContainer(ctx context.Context, backupPath string, target *RestoreTarget) error {
	// TODO: 实现容器恢复逻辑
	return fmt.Errorf("container restore not implemented")
}

// updateRestoreJobStatus 更新恢复任务状态
func (s *BackupServiceImpl) updateRestoreJobStatus(ctx context.Context, jobID uint, status BackupStatus, errorMsg string) {
	now := time.Now()
	updates := map[string]interface{}{
		"status":   status,
		"end_time": now,
	}
	
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}
	
	s.db.WithContext(ctx).Model(&RestoreJob{}).Where("id = ?", jobID).Updates(updates)
	
	// 计算持续时间
	var job RestoreJob
	if err := s.db.WithContext(ctx).First(&job, jobID).Error; err == nil {
		duration := int(now.Sub(job.StartTime).Seconds())
		s.db.Model(&RestoreJob{}).Where("id = ?", jobID).Update("duration", duration)
	}
}

// ListRestoreJobs 列出恢复任务
func (s *BackupServiceImpl) ListRestoreJobs(ctx context.Context, userID, tenantID uint) ([]*RestoreJob, error) {
	var jobs []*RestoreJob
	query := s.db.WithContext(ctx)
	
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	
	if err := query.Order("created_at DESC").Find(&jobs).Error; err != nil {
		return nil, err
	}
	
	return jobs, nil
}

// GetRestoreJob 获取恢复任务
func (s *BackupServiceImpl) GetRestoreJob(ctx context.Context, jobID uint) (*RestoreJob, error) {
	var job RestoreJob
	if err := s.db.WithContext(ctx).First(&job, jobID).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// ValidateBackup 验证备份
func (s *BackupServiceImpl) ValidateBackup(ctx context.Context, jobID uint) (*ValidationResult, error) {
	job, err := s.GetBackupJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	
	policy, err := s.GetPolicy(ctx, job.PolicyID)
	if err != nil {
		return nil, err
	}
	
	result := &ValidationResult{
		Valid:       true,
		Checksum:    job.Checksum,
		FileSize:    job.FileSize,
		ValidatedAt: time.Now(),
	}
	
	// 验证文件是否存在
	storage := s.storageBackends[policy.StorageType]
	exists, err := storage.Exists(ctx, job.FilePath, policy.StorageConfig)
	if err != nil || !exists {
		result.Valid = false
		result.Errors = append(result.Errors, "backup file not found")
		return result, nil
	}
	
	// TODO: 添加更多验证逻辑
	
	return result, nil
}

// CheckBackupHealth 检查备份健康状态
func (s *BackupServiceImpl) CheckBackupHealth(ctx context.Context, policyID uint) (*HealthReport, error) {
	policy, err := s.GetPolicy(ctx, policyID)
	if err != nil {
		return nil, err
	}
	
	// 统计备份任务
	var total, successful, failed int64
	s.db.WithContext(ctx).Model(&BackupJob{}).Where("policy_id = ?", policyID).Count(&total)
	s.db.WithContext(ctx).Model(&BackupJob{}).Where("policy_id = ? AND status = ?", policyID, StatusCompleted).Count(&successful)
	s.db.WithContext(ctx).Model(&BackupJob{}).Where("policy_id = ? AND status = ?", policyID, StatusFailed).Count(&failed)
	
	// 获取最后一次备份
	var lastJob BackupJob
	s.db.WithContext(ctx).Where("policy_id = ?", policyID).Order("created_at DESC").First(&lastJob)
	
	report := &HealthReport{
		PolicyID:          policyID,
		TotalBackups:      int(total),
		SuccessfulBackups: int(successful),
		FailedBackups:     int(failed),
		CheckedAt:         time.Now(),
	}
	
	if lastJob.ID > 0 {
		report.LastBackupTime = &lastJob.CreatedAt
		report.LastBackupStatus = string(lastJob.Status)
	}
	
	// AI 分析和建议
	if failed > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("有 %d 个备份任务失败", failed))
	}
	
	if !policy.Enabled {
		report.Issues = append(report.Issues, "备份策略已禁用")
	}
	
	if policy.Retention < 7 {
		report.Recommendations = append(report.Recommendations, "建议将备份保留期设置为至少7天")
	}
	
	if !policy.Encryption {
		report.Recommendations = append(report.Recommendations, "建议启用备份加密以提高安全性")
	}
	
	return report, nil
}
