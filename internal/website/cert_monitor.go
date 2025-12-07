package website

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CertMonitor 证书监控器
type CertMonitor struct {
	db         *gorm.DB
	sslService SSLService
	interval   time.Duration
	stopChan   chan struct{}
}

// NewCertMonitor 创建证书监控器
func NewCertMonitor(db *gorm.DB, interval time.Duration) *CertMonitor {
	return &CertMonitor{
		db:         db,
		sslService: NewSSLService(db),
		interval:   interval,
		stopChan:   make(chan struct{}),
	}
}

// Start 启动监控
func (m *CertMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// 立即执行一次检查
	m.checkAndRenew(ctx)

	for {
		select {
		case <-ticker.C:
			m.checkAndRenew(ctx)
		case <-m.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop 停止监控
func (m *CertMonitor) Stop() {
	close(m.stopChan)
}

// checkAndRenew 检查并续期证书
func (m *CertMonitor) checkAndRenew(ctx context.Context) {
	// 获取即将过期的证书
	certs, err := m.sslService.CheckExpiry(ctx)
	if err != nil {
		fmt.Printf("failed to check certificate expiry: %v\n", err)
		return
	}

	if len(certs) == 0 {
		return
	}

	fmt.Printf("found %d certificates that need renewal\n", len(certs))

	// 续期证书
	for _, cert := range certs {
		if err := m.renewCertificate(ctx, cert); err != nil {
			fmt.Printf("failed to renew certificate %d (%s): %v\n", cert.ID, cert.Domain, err)
			continue
		}
		fmt.Printf("successfully renewed certificate %d (%s)\n", cert.ID, cert.Domain)
	}
}

// renewCertificate 续期单个证书
func (m *CertMonitor) renewCertificate(ctx context.Context, cert *SSLCert) error {
	// 检查是否需要续期
	if cert.ExpiryDate == nil {
		return fmt.Errorf("certificate has no expiry date")
	}

	daysUntilExpiry := int(time.Until(*cert.ExpiryDate).Hours() / 24)
	if daysUntilExpiry > cert.RenewDaysBefore {
		return nil // 还不需要续期
	}

	// 执行续期
	if err := m.sslService.RenewCertificate(ctx, cert.ID); err != nil {
		return err
	}

	// 重载 Nginx 以应用新证书
	if err := reloadNginx(); err != nil {
		fmt.Printf("warning: failed to reload nginx after certificate renewal: %v\n", err)
	}

	return nil
}

// GetExpiringCertificates 获取即将过期的证书列表
func (m *CertMonitor) GetExpiringCertificates(ctx context.Context, days int) ([]*SSLCert, error) {
	var certs []*SSLCert
	
	expiryThreshold := time.Now().AddDate(0, 0, days)
	
	if err := m.db.WithContext(ctx).
		Where("status = ? AND expiry_date <= ?", SSLStatusValid, expiryThreshold).
		Order("expiry_date ASC").
		Find(&certs).Error; err != nil {
		return nil, fmt.Errorf("failed to get expiring certificates: %w", err)
	}

	return certs, nil
}

// GetCertificateStats 获取证书统计信息
func (m *CertMonitor) GetCertificateStats(ctx context.Context) (*CertStats, error) {
	var stats CertStats

	// 总证书数
	if err := m.db.WithContext(ctx).Model(&SSLCert{}).Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// 有效证书数
	if err := m.db.WithContext(ctx).Model(&SSLCert{}).
		Where("status = ?", SSLStatusValid).
		Count(&stats.Valid).Error; err != nil {
		return nil, err
	}

	// 已过期证书数
	if err := m.db.WithContext(ctx).Model(&SSLCert{}).
		Where("status = ? OR (expiry_date IS NOT NULL AND expiry_date < ?)", 
			SSLStatusExpired, time.Now()).
		Count(&stats.Expired).Error; err != nil {
		return nil, err
	}

	// 即将过期证书数（30天内）
	expiryThreshold := time.Now().AddDate(0, 0, 30)
	if err := m.db.WithContext(ctx).Model(&SSLCert{}).
		Where("status = ? AND expiry_date IS NOT NULL AND expiry_date <= ? AND expiry_date > ?", 
			SSLStatusValid, expiryThreshold, time.Now()).
		Count(&stats.ExpiringSoon).Error; err != nil {
		return nil, err
	}

	// 错误状态证书数
	if err := m.db.WithContext(ctx).Model(&SSLCert{}).
		Where("status = ?", SSLStatusError).
		Count(&stats.Error).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// CertStats 证书统计信息
type CertStats struct {
	Total        int64 `json:"total"`         // 总证书数
	Valid        int64 `json:"valid"`         // 有效证书数
	Expired      int64 `json:"expired"`       // 已过期证书数
	ExpiringSoon int64 `json:"expiring_soon"` // 即将过期证书数
	Error        int64 `json:"error"`         // 错误状态证书数
}
