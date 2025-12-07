package database

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// AuditService 审计日志服务
type AuditService struct {
	db *gorm.DB
}

// NewAuditService 创建审计服务实例
func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// LogAction 记录操作日志
func (s *AuditService) LogAction(ctx context.Context, log *AuditLog) error {
	return s.db.WithContext(ctx).Create(log).Error
}

// LogSuccess 记录成功操作
func (s *AuditService) LogSuccess(ctx context.Context, userID, tenantID uint, action, resource, resourceID string, details interface{}, ipAddress, userAgent string) error {
	detailsJSON, _ := json.Marshal(details)

	log := &AuditLog{
		UserID:     userID,
		TenantID:   tenantID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    string(detailsJSON),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Status:     "success",
	}

	return s.LogAction(ctx, log)
}

// LogFailure 记录失败操作
func (s *AuditService) LogFailure(ctx context.Context, userID, tenantID uint, action, resource, resourceID string, details interface{}, errorMsg, ipAddress, userAgent string) error {
	detailsJSON, _ := json.Marshal(details)

	log := &AuditLog{
		UserID:     userID,
		TenantID:   tenantID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    string(detailsJSON),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Status:     "failed",
		ErrorMsg:   errorMsg,
	}

	return s.LogAction(ctx, log)
}

// QueryLogs 查询审计日志
func (s *AuditService) QueryLogs(ctx context.Context, filter AuditLogFilter) ([]AuditLog, int64, error) {
	var logs []AuditLog
	var total int64

	query := s.db.WithContext(ctx).Model(&AuditLog{})

	// 应用过滤条件
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.TenantID > 0 {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.Resource != "" {
		query = query.Where("resource = ?", filter.Resource)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if !filter.StartTime.IsZero() {
		query = query.Where("created_at >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query = query.Where("created_at <= ?", filter.EndTime)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Preload("User").Order("created_at DESC").Offset(offset).Limit(filter.PageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetLogsByUser 获取用户的操作日志
func (s *AuditService) GetLogsByUser(ctx context.Context, userID uint, page, pageSize int) ([]AuditLog, int64, error) {
	filter := AuditLogFilter{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	}
	return s.QueryLogs(ctx, filter)
}

// GetLogsByTenant 获取租户的操作日志
func (s *AuditService) GetLogsByTenant(ctx context.Context, tenantID uint, page, pageSize int) ([]AuditLog, int64, error) {
	filter := AuditLogFilter{
		TenantID: tenantID,
		Page:     page,
		PageSize: pageSize,
	}
	return s.QueryLogs(ctx, filter)
}

// GetRecentLogs 获取最近的操作日志
func (s *AuditService) GetRecentLogs(ctx context.Context, limit int) ([]AuditLog, error) {
	var logs []AuditLog
	if err := s.db.WithContext(ctx).Preload("User").Order("created_at DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// DeleteOldLogs 删除旧的审计日志（数据清理）
func (s *AuditService) DeleteOldLogs(ctx context.Context, beforeDate time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Where("created_at < ?", beforeDate).Delete(&AuditLog{})
	return result.RowsAffected, result.Error
}

// AuditLogFilter 审计日志查询过滤器
type AuditLogFilter struct {
	UserID    uint
	TenantID  uint
	Action    string
	Resource  string
	Status    string
	StartTime time.Time
	EndTime   time.Time
	Page      int
	PageSize  int
}
