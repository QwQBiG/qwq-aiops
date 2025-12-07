package website

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// sslService SSL 证书服务实现
type sslService struct {
	db *gorm.DB
}

// NewSSLService 创建 SSL 服务实例
func NewSSLService(db *gorm.DB) SSLService {
	return &sslService{db: db}
}

// CreateSSLCert 创建 SSL 证书记录
func (s *sslService) CreateSSLCert(ctx context.Context, cert *SSLCert) error {
	if err := s.db.WithContext(ctx).Create(cert).Error; err != nil {
		return fmt.Errorf("failed to create ssl cert: %w", err)
	}
	return nil
}

// GetSSLCert 获取 SSL 证书
func (s *sslService) GetSSLCert(ctx context.Context, id uint) (*SSLCert, error) {
	var cert SSLCert
	if err := s.db.WithContext(ctx).First(&cert, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSSLCertNotFound
		}
		return nil, fmt.Errorf("failed to get ssl cert: %w", err)
	}
	return &cert, nil
}

// GetSSLCertByDomain 根据域名获取证书
func (s *sslService) GetSSLCertByDomain(ctx context.Context, domain string) (*SSLCert, error) {
	var cert SSLCert
	if err := s.db.WithContext(ctx).
		Where("domain = ?", domain).
		Order("created_at DESC").
		First(&cert).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSSLCertNotFound
		}
		return nil, fmt.Errorf("failed to get ssl cert by domain: %w", err)
	}
	return &cert, nil
}

// ListSSLCerts 列出 SSL 证书
func (s *sslService) ListSSLCerts(ctx context.Context, userID, tenantID uint) ([]*SSLCert, error) {
	var certs []*SSLCert
	
	query := s.db.WithContext(ctx).Model(&SSLCert{})
	
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Order("created_at DESC").Find(&certs).Error; err != nil {
		return nil, fmt.Errorf("failed to list ssl certs: %w", err)
	}

	return certs, nil
}

// UpdateSSLCert 更新 SSL 证书
func (s *sslService) UpdateSSLCert(ctx context.Context, cert *SSLCert) error {
	// 检查证书是否存在
	var existing SSLCert
	if err := s.db.WithContext(ctx).First(&existing, cert.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSSLCertNotFound
		}
		return fmt.Errorf("failed to check ssl cert existence: %w", err)
	}

	if err := s.db.WithContext(ctx).Save(cert).Error; err != nil {
		return fmt.Errorf("failed to update ssl cert: %w", err)
	}
	return nil
}

// DeleteSSLCert 删除 SSL 证书
func (s *sslService) DeleteSSLCert(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&SSLCert{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete ssl cert: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrSSLCertNotFound
	}
	return nil
}

// RequestCertificate 申请证书
func (s *sslService) RequestCertificate(ctx context.Context, domain, email string, provider SSLProvider) (*SSLCert, error) {
	// 创建证书记录
	now := time.Now()
	autoRenew := true
	
	cert := &SSLCert{
		Domain:          domain,
		Provider:        provider,
		Status:          SSLStatusPending,
		Email:           email,
		AutoRenew:       &autoRenew,
		RenewDaysBefore: 30,
		IssueDate:       &now,
	}

	// 根据提供商申请证书
	switch provider {
	case SSLProviderLetsEncrypt:
		// 使用 Let's Encrypt 申请证书
		bundle, err := s.requestLetsEncryptCert(ctx, domain, email)
		if err != nil {
			cert.Status = SSLStatusError
			s.CreateSSLCert(ctx, cert)
			return nil, fmt.Errorf("failed to request certificate: %w", err)
		}

		// 保存证书文件
		certPath, keyPath, err := bundle.SaveToFile()
		if err != nil {
			cert.Status = SSLStatusError
			s.CreateSSLCert(ctx, cert)
			return nil, fmt.Errorf("failed to save certificate: %w", err)
		}

		// 更新证书信息
		cert.CertPath = certPath
		cert.KeyPath = keyPath
		cert.CertContent = string(bundle.Certificate)
		cert.KeyContent = string(bundle.PrivateKey)
		cert.ExpiryDate = &bundle.NotAfter
		cert.Status = SSLStatusValid

	case SSLProviderSelfSigned:
		// 生成自签名证书
		bundle, err := generateSelfSignedCert(domain)
		if err != nil {
			cert.Status = SSLStatusError
			s.CreateSSLCert(ctx, cert)
			return nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}

		certPath, keyPath, err := bundle.SaveToFile()
		if err != nil {
			cert.Status = SSLStatusError
			s.CreateSSLCert(ctx, cert)
			return nil, fmt.Errorf("failed to save certificate: %w", err)
		}

		cert.CertPath = certPath
		cert.KeyPath = keyPath
		cert.CertContent = string(bundle.Certificate)
		cert.KeyContent = string(bundle.PrivateKey)
		cert.ExpiryDate = &bundle.NotAfter
		cert.Status = SSLStatusValid

	case SSLProviderManual:
		// 手动上传证书，只创建记录
		expiryDate := now.AddDate(1, 0, 0) // 默认1年有效期
		cert.ExpiryDate = &expiryDate

	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	
	if err := s.CreateSSLCert(ctx, cert); err != nil {
		return nil, err
	}

	return cert, nil
}

// requestLetsEncryptCert 使用 Let's Encrypt 申请证书
func (s *sslService) requestLetsEncryptCert(ctx context.Context, domain, email string) (*CertificateBundle, error) {
	// 创建 ACME 客户端（使用生产环境）
	client, err := NewACMEClient(false)
	if err != nil {
		return nil, err
	}

	// 获取证书
	return client.ObtainCertificate(ctx, domain, email)
}

// RenewCertificate 续期证书
func (s *sslService) RenewCertificate(ctx context.Context, certID uint) error {
	cert, err := s.GetSSLCert(ctx, certID)
	if err != nil {
		return err
	}

	// 根据提供商续期证书
	switch cert.Provider {
	case SSLProviderLetsEncrypt:
		// 使用 Let's Encrypt 续期证书
		client, err := NewACMEClient(false)
		if err != nil {
			return fmt.Errorf("failed to create acme client: %w", err)
		}

		bundle, err := client.RenewCertificate(ctx, cert.Domain, cert.Email)
		if err != nil {
			cert.Status = SSLStatusError
			s.UpdateSSLCert(ctx, cert)
			return fmt.Errorf("failed to renew certificate: %w", err)
		}

		// 保存新证书
		certPath, keyPath, err := bundle.SaveToFile()
		if err != nil {
			return fmt.Errorf("failed to save certificate: %w", err)
		}

		// 更新证书信息
		now := time.Now()
		cert.CertPath = certPath
		cert.KeyPath = keyPath
		cert.CertContent = string(bundle.Certificate)
		cert.KeyContent = string(bundle.PrivateKey)
		cert.IssueDate = &now
		cert.ExpiryDate = &bundle.NotAfter
		cert.Status = SSLStatusValid

	case SSLProviderSelfSigned:
		// 重新生成自签名证书
		bundle, err := generateSelfSignedCert(cert.Domain)
		if err != nil {
			return fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}

		certPath, keyPath, err := bundle.SaveToFile()
		if err != nil {
			return fmt.Errorf("failed to save certificate: %w", err)
		}

		now := time.Now()
		cert.CertPath = certPath
		cert.KeyPath = keyPath
		cert.CertContent = string(bundle.Certificate)
		cert.KeyContent = string(bundle.PrivateKey)
		cert.IssueDate = &now
		cert.ExpiryDate = &bundle.NotAfter
		cert.Status = SSLStatusValid

	case SSLProviderManual:
		return fmt.Errorf("manual certificates cannot be auto-renewed")

	default:
		return fmt.Errorf("unsupported provider: %s", cert.Provider)
	}

	return s.UpdateSSLCert(ctx, cert)
}

// CheckExpiry 检查证书过期状态
func (s *sslService) CheckExpiry(ctx context.Context) ([]*SSLCert, error) {
	var certs []*SSLCert
	
	// 查找即将过期的证书（30天内）
	expiryThreshold := time.Now().AddDate(0, 0, 30)
	
	if err := s.db.WithContext(ctx).
		Where("status = ? AND expiry_date <= ? AND auto_renew = ?", 
			SSLStatusValid, expiryThreshold, true).
		Find(&certs).Error; err != nil {
		return nil, fmt.Errorf("failed to check expiry: %w", err)
	}

	return certs, nil
}

// AutoRenew 自动续期即将过期的证书
func (s *sslService) AutoRenew(ctx context.Context) error {
	certs, err := s.CheckExpiry(ctx)
	if err != nil {
		return err
	}

	for _, cert := range certs {
		// 检查是否需要续期
		if cert.ExpiryDate != nil {
			daysUntilExpiry := int(time.Until(*cert.ExpiryDate).Hours() / 24)
			if daysUntilExpiry <= cert.RenewDaysBefore {
				if err := s.RenewCertificate(ctx, cert.ID); err != nil {
					// 记录错误但继续处理其他证书
					fmt.Printf("failed to renew certificate %d: %v\n", cert.ID, err)
					continue
				}
			}
		}
	}

	return nil
}
