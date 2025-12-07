package website

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// proxyService 代理服务实现
type proxyService struct {
	db *gorm.DB
}

// NewProxyService 创建代理服务实例
func NewProxyService(db *gorm.DB) ProxyService {
	return &proxyService{db: db}
}

// CreateProxyConfig 创建代理配置
func (s *proxyService) CreateProxyConfig(ctx context.Context, config *ProxyConfig) error {
	// 验证后端地址
	if config.Backend == "" {
		return ErrInvalidBackend
	}

	if err := s.db.WithContext(ctx).Create(config).Error; err != nil {
		return fmt.Errorf("failed to create proxy config: %w", err)
	}
	return nil
}

// GetProxyConfig 获取代理配置
func (s *proxyService) GetProxyConfig(ctx context.Context, id uint) (*ProxyConfig, error) {
	var config ProxyConfig
	if err := s.db.WithContext(ctx).First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrProxyConfigNotFound
		}
		return nil, fmt.Errorf("failed to get proxy config: %w", err)
	}
	return &config, nil
}

// ListProxyConfigs 列出代理配置
func (s *proxyService) ListProxyConfigs(ctx context.Context, userID, tenantID uint) ([]*ProxyConfig, error) {
	var configs []*ProxyConfig
	
	query := s.db.WithContext(ctx).Model(&ProxyConfig{})
	
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to list proxy configs: %w", err)
	}

	return configs, nil
}

// UpdateProxyConfig 更新代理配置
func (s *proxyService) UpdateProxyConfig(ctx context.Context, config *ProxyConfig) error {
	// 验证后端地址
	if config.Backend == "" {
		return ErrInvalidBackend
	}

	// 检查配置是否存在
	var existing ProxyConfig
	if err := s.db.WithContext(ctx).First(&existing, config.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrProxyConfigNotFound
		}
		return fmt.Errorf("failed to check proxy config existence: %w", err)
	}

	if err := s.db.WithContext(ctx).Save(config).Error; err != nil {
		return fmt.Errorf("failed to update proxy config: %w", err)
	}
	return nil
}

// DeleteProxyConfig 删除代理配置
func (s *proxyService) DeleteProxyConfig(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&ProxyConfig{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete proxy config: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrProxyConfigNotFound
	}
	return nil
}

// GenerateNginxConfig 生成 Nginx 配置
func (s *proxyService) GenerateNginxConfig(ctx context.Context, website *Website) (string, error) {
	if website.ProxyConfig == nil {
		return "", ErrProxyConfigNotFound
	}

	generator := NewNginxConfigGenerator(website)
	return generator.Generate()
}

// ValidateConfig 验证配置
func (s *proxyService) ValidateConfig(ctx context.Context, config string) error {
	return validateNginxConfig(config)
}

// ReloadNginx 重载 Nginx
func (s *proxyService) ReloadNginx(ctx context.Context) error {
	return reloadNginx()
}
