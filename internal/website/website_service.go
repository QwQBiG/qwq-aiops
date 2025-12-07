package website

import (
	"context"
	"fmt"
	"regexp"

	"gorm.io/gorm"
)

// websiteService 网站服务实现
type websiteService struct {
	db *gorm.DB
}

// NewWebsiteService 创建网站服务实例
func NewWebsiteService(db *gorm.DB) WebsiteService {
	return &websiteService{db: db}
}

// CreateWebsite 创建网站
func (s *websiteService) CreateWebsite(ctx context.Context, website *Website) error {
	// 验证域名格式
	if !isValidDomain(website.Domain) {
		return ErrInvalidDomain
	}

	// 检查域名是否已存在
	var count int64
	if err := s.db.WithContext(ctx).Model(&Website{}).
		Where("domain = ? AND tenant_id = ?", website.Domain, website.TenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check domain existence: %w", err)
	}
	if count > 0 {
		return ErrWebsiteExists
	}

	// 创建网站记录
	if err := s.db.WithContext(ctx).Create(website).Error; err != nil {
		return fmt.Errorf("failed to create website: %w", err)
	}

	return nil
}

// GetWebsite 获取网站信息
func (s *websiteService) GetWebsite(ctx context.Context, id uint) (*Website, error) {
	var website Website
	if err := s.db.WithContext(ctx).
		Preload("SSLCert").
		Preload("ProxyConfig").
		First(&website, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrWebsiteNotFound
		}
		return nil, fmt.Errorf("failed to get website: %w", err)
	}
	return &website, nil
}

// GetWebsiteByDomain 根据域名获取网站
func (s *websiteService) GetWebsiteByDomain(ctx context.Context, domain string) (*Website, error) {
	var website Website
	if err := s.db.WithContext(ctx).
		Preload("SSLCert").
		Preload("ProxyConfig").
		Where("domain = ?", domain).
		First(&website).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrWebsiteNotFound
		}
		return nil, fmt.Errorf("failed to get website by domain: %w", err)
	}
	return &website, nil
}

// ListWebsites 列出网站
func (s *websiteService) ListWebsites(ctx context.Context, userID, tenantID uint, page, pageSize int) ([]*Website, int64, error) {
	var websites []*Website
	var total int64

	query := s.db.WithContext(ctx).Model(&Website{})
	
	// 根据租户和用户过滤
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count websites: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.
		Preload("SSLCert").
		Preload("ProxyConfig").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&websites).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list websites: %w", err)
	}

	return websites, total, nil
}

// UpdateWebsite 更新网站
func (s *websiteService) UpdateWebsite(ctx context.Context, website *Website) error {
	// 验证域名格式
	if !isValidDomain(website.Domain) {
		return ErrInvalidDomain
	}

	// 检查网站是否存在
	var existing Website
	if err := s.db.WithContext(ctx).First(&existing, website.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrWebsiteNotFound
		}
		return fmt.Errorf("failed to check website existence: %w", err)
	}

	// 如果域名变更，检查新域名是否已被使用
	if existing.Domain != website.Domain {
		var count int64
		if err := s.db.WithContext(ctx).Model(&Website{}).
			Where("domain = ? AND tenant_id = ? AND id != ?", website.Domain, website.TenantID, website.ID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check domain existence: %w", err)
		}
		if count > 0 {
			return ErrWebsiteExists
		}
	}

	// 更新网站记录
	if err := s.db.WithContext(ctx).Save(website).Error; err != nil {
		return fmt.Errorf("failed to update website: %w", err)
	}

	return nil
}

// DeleteWebsite 删除网站
func (s *websiteService) DeleteWebsite(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&Website{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete website: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrWebsiteNotFound
	}
	return nil
}

// EnableSSL 启用 SSL
func (s *websiteService) EnableSSL(ctx context.Context, websiteID, certID uint) error {
	// 检查证书是否存在
	var cert SSLCert
	if err := s.db.WithContext(ctx).First(&cert, certID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSSLCertNotFound
		}
		return fmt.Errorf("failed to get ssl cert: %w", err)
	}

	// 检查证书状态
	if cert.Status != SSLStatusValid {
		return ErrSSLCertExpired
	}

	// 更新网站配置
	if err := s.db.WithContext(ctx).Model(&Website{}).
		Where("id = ?", websiteID).
		Updates(map[string]interface{}{
			"ssl_enabled": true,
			"ssl_cert_id": certID,
		}).Error; err != nil {
		return fmt.Errorf("failed to enable ssl: %w", err)
	}

	return nil
}

// DisableSSL 禁用 SSL
func (s *websiteService) DisableSSL(ctx context.Context, websiteID uint) error {
	if err := s.db.WithContext(ctx).Model(&Website{}).
		Where("id = ?", websiteID).
		Updates(map[string]interface{}{
			"ssl_enabled": false,
			"ssl_cert_id": nil,
		}).Error; err != nil {
		return fmt.Errorf("failed to disable ssl: %w", err)
	}
	return nil
}

// isValidDomain 验证域名格式
func isValidDomain(domain string) bool {
	// 简单的域名格式验证
	domainRegex := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	return domainRegex.MatchString(domain)
}
