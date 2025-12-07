package backup

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrPolicyNotFound 备份策略不存在
	ErrPolicyNotFound = errors.New("备份策略不存在")
	
	// ErrBackupFailed 备份失败
	ErrBackupFailed = errors.New("备份失败")
	
	// ErrRestoreFailed 恢复失败
	ErrRestoreFailed = errors.New("恢复失败")
	
	// ErrInvalidStorage 无效的存储后端
	ErrInvalidStorage = errors.New("无效的存储后端")
)

// BackupService 统一备份恢复服务接口
type BackupService interface {
	// 策略管理
	CreatePolicy(ctx context.Context, policy *BackupPolicy) error
	UpdatePolicy(ctx context.Context, id uint, policy *BackupPolicy) error
	DeletePolicy(ctx context.Context, id uint) error
	GetPolicy(ctx context.Context, id uint) (*BackupPolicy, error)
	ListPolicies(ctx context.Context, userID, tenantID uint) ([]*BackupPolicy, error)
	
	// 备份执行
	ExecuteBackup(ctx context.Context, policyID uint) (*BackupJob, error)
	ListBackupJobs(ctx context.Context, policyID uint) ([]*BackupJob, error)
	GetBackupJob(ctx context.Context, jobID uint) (*BackupJob, error)
	
	// 数据恢复
	RestoreBackup(ctx context.Context, jobID uint, target *RestoreTarget) (*RestoreJob, error)
	ListRestoreJobs(ctx context.Context, userID, tenantID uint) ([]*RestoreJob, error)
	GetRestoreJob(ctx context.Context, jobID uint) (*RestoreJob, error)
	
	// AI 监控
	ValidateBackup(ctx context.Context, jobID uint) (*ValidationResult, error)
	CheckBackupHealth(ctx context.Context, policyID uint) (*HealthReport, error)
}

// BackupType 备份类型
type BackupType string

const (
	BackupTypeDatabase  BackupType = "database"
	BackupTypeFiles     BackupType = "files"
	BackupTypeContainer BackupType = "container"
	BackupTypeSystem    BackupType = "system"
)

// StorageType 存储类型
type StorageType string

const (
	StorageTypeLocal StorageType = "local"
	StorageTypeS3    StorageType = "s3"
	StorageTypeFTP   StorageType = "ftp"
	StorageTypeSFTP  StorageType = "sftp"
)

// BackupStatus 备份状态
type BackupStatus string

const (
	StatusPending   BackupStatus = "pending"
	StatusRunning   BackupStatus = "running"
	StatusCompleted BackupStatus = "completed"
	StatusFailed    BackupStatus = "failed"
)

// BackupPolicy 备份策略
type BackupPolicy struct {
	ID          uint        `json:"id" gorm:"primaryKey"`
	Name        string      `json:"name" gorm:"not null"`
	Type        BackupType  `json:"type" gorm:"not null"`
	Schedule    string      `json:"schedule"`                       // Cron 表达式
	Enabled     bool        `json:"enabled" gorm:"default:true"`
	Retention   int         `json:"retention" gorm:"default:7"`     // 保留天数
	Compression bool        `json:"compression" gorm:"default:true"`
	Encryption  bool        `json:"encryption" gorm:"default:false"`
	
	// 存储配置
	StorageType   StorageType            `json:"storage_type" gorm:"not null"`
	StorageConfig map[string]interface{} `json:"storage_config" gorm:"type:jsonb"`
	
	// 备份源配置
	SourceType   string                 `json:"source_type"`   // database, files, container
	SourceConfig map[string]interface{} `json:"source_config" gorm:"type:jsonb"`
	
	UserID    uint      `json:"user_id" gorm:"index"`
	TenantID  uint      `json:"tenant_id" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BackupJob 备份任务
type BackupJob struct {
	ID         uint         `json:"id" gorm:"primaryKey"`
	PolicyID   uint         `json:"policy_id" gorm:"index"`
	Status     BackupStatus `json:"status" gorm:"index"`
	FilePath   string       `json:"file_path"`
	FileSize   int64        `json:"file_size"`
	StartTime  time.Time    `json:"start_time"`
	EndTime    *time.Time   `json:"end_time,omitempty"`
	Duration   int          `json:"duration"` // 秒
	ErrorMsg   string       `json:"error_msg,omitempty" gorm:"type:text"`
	Checksum   string       `json:"checksum"`
	UserID     uint         `json:"user_id" gorm:"index"`
	TenantID   uint         `json:"tenant_id" gorm:"index"`
	CreatedAt  time.Time    `json:"created_at"`
}

// RestoreTarget 恢复目标
type RestoreTarget struct {
	Type   string                 `json:"type"`   // database, files, container
	Config map[string]interface{} `json:"config"` // 目标配置
}

// RestoreJob 恢复任务
type RestoreJob struct {
	ID         uint         `json:"id" gorm:"primaryKey"`
	BackupID   uint         `json:"backup_id" gorm:"index"`
	Status     BackupStatus `json:"status" gorm:"index"`
	Target     string       `json:"target" gorm:"type:jsonb"`
	StartTime  time.Time    `json:"start_time"`
	EndTime    *time.Time   `json:"end_time,omitempty"`
	Duration   int          `json:"duration"` // 秒
	ErrorMsg   string       `json:"error_msg,omitempty" gorm:"type:text"`
	UserID     uint         `json:"user_id" gorm:"index"`
	TenantID   uint         `json:"tenant_id" gorm:"index"`
	CreatedAt  time.Time    `json:"created_at"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid      bool     `json:"valid"`
	Checksum   string   `json:"checksum"`
	FileSize   int64    `json:"file_size"`
	Errors     []string `json:"errors,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	ValidatedAt time.Time `json:"validated_at"`
}

// HealthReport 健康报告
type HealthReport struct {
	PolicyID        uint      `json:"policy_id"`
	TotalBackups    int       `json:"total_backups"`
	SuccessfulBackups int     `json:"successful_backups"`
	FailedBackups   int       `json:"failed_backups"`
	LastBackupTime  *time.Time `json:"last_backup_time,omitempty"`
	LastBackupStatus string    `json:"last_backup_status"`
	StorageUsed     int64     `json:"storage_used"` // 字节
	Issues          []string  `json:"issues,omitempty"`
	Recommendations []string  `json:"recommendations,omitempty"`
	CheckedAt       time.Time `json:"checked_at"`
}

// TableName 指定表名
func (BackupPolicy) TableName() string {
	return "backup_policies"
}

func (BackupJob) TableName() string {
	return "backup_jobs"
}

func (RestoreJob) TableName() string {
	return "restore_jobs"
}
