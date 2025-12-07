package website

import (
	"context"
	"fmt"
	"net"

	"gorm.io/gorm"
)

// dnsService DNS 服务实现
type dnsService struct {
	db *gorm.DB
}

// NewDNSService 创建 DNS 服务实例
func NewDNSService(db *gorm.DB) DNSService {
	return &dnsService{db: db}
}

// CreateDNSRecord 创建 DNS 记录
func (s *dnsService) CreateDNSRecord(ctx context.Context, record *DNSRecord) error {
	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		return fmt.Errorf("failed to create dns record: %w", err)
	}
	return nil
}

// GetDNSRecord 获取 DNS 记录
func (s *dnsService) GetDNSRecord(ctx context.Context, id uint) (*DNSRecord, error) {
	var record DNSRecord
	if err := s.db.WithContext(ctx).First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrDNSRecordNotFound
		}
		return nil, fmt.Errorf("failed to get dns record: %w", err)
	}
	return &record, nil
}

// ListDNSRecords 列出 DNS 记录
func (s *dnsService) ListDNSRecords(ctx context.Context, domain string, userID, tenantID uint) ([]*DNSRecord, error) {
	var records []*DNSRecord
	
	query := s.db.WithContext(ctx).Model(&DNSRecord{})
	
	if domain != "" {
		query = query.Where("domain = ?", domain)
	}
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to list dns records: %w", err)
	}

	return records, nil
}

// UpdateDNSRecord 更新 DNS 记录
func (s *dnsService) UpdateDNSRecord(ctx context.Context, record *DNSRecord) error {
	// 检查记录是否存在
	var existing DNSRecord
	if err := s.db.WithContext(ctx).First(&existing, record.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrDNSRecordNotFound
		}
		return fmt.Errorf("failed to check dns record existence: %w", err)
	}

	if err := s.db.WithContext(ctx).Save(record).Error; err != nil {
		return fmt.Errorf("failed to update dns record: %w", err)
	}
	return nil
}

// DeleteDNSRecord 删除 DNS 记录
func (s *dnsService) DeleteDNSRecord(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&DNSRecord{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete dns record: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrDNSRecordNotFound
	}
	return nil
}

// VerifyDNS 验证 DNS 解析
func (s *dnsService) VerifyDNS(ctx context.Context, domain, recordType, expectedValue string) (bool, error) {
	switch DNSRecordType(recordType) {
	case DNSRecordA:
		return s.verifyARecord(domain, expectedValue)
	case DNSRecordAAAA:
		return s.verifyAAAARecord(domain, expectedValue)
	case DNSRecordCNAME:
		return s.verifyCNAMERecord(domain, expectedValue)
	case DNSRecordTXT:
		return s.verifyTXTRecord(domain, expectedValue)
	case DNSRecordMX:
		return s.verifyMXRecord(domain, expectedValue)
	default:
		return false, fmt.Errorf("unsupported record type: %s", recordType)
	}
}

// verifyARecord 验证 A 记录
func (s *dnsService) verifyARecord(domain, expectedIP string) (bool, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return false, fmt.Errorf("failed to lookup A record: %w", err)
	}

	for _, ip := range ips {
		if ip.To4() != nil && ip.String() == expectedIP {
			return true, nil
		}
	}

	return false, nil
}

// verifyAAAARecord 验证 AAAA 记录
func (s *dnsService) verifyAAAARecord(domain, expectedIP string) (bool, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return false, fmt.Errorf("failed to lookup AAAA record: %w", err)
	}

	for _, ip := range ips {
		if ip.To16() != nil && ip.To4() == nil && ip.String() == expectedIP {
			return true, nil
		}
	}

	return false, nil
}

// verifyCNAMERecord 验证 CNAME 记录
func (s *dnsService) verifyCNAMERecord(domain, expectedCNAME string) (bool, error) {
	cname, err := net.LookupCNAME(domain)
	if err != nil {
		return false, fmt.Errorf("failed to lookup CNAME record: %w", err)
	}

	// 移除末尾的点
	if len(cname) > 0 && cname[len(cname)-1] == '.' {
		cname = cname[:len(cname)-1]
	}

	return cname == expectedCNAME, nil
}

// verifyTXTRecord 验证 TXT 记录
func (s *dnsService) verifyTXTRecord(domain, expectedValue string) (bool, error) {
	txts, err := net.LookupTXT(domain)
	if err != nil {
		return false, fmt.Errorf("failed to lookup TXT record: %w", err)
	}

	for _, txt := range txts {
		if txt == expectedValue {
			return true, nil
		}
	}

	return false, nil
}

// verifyMXRecord 验证 MX 记录
func (s *dnsService) verifyMXRecord(domain, expectedMX string) (bool, error) {
	mxs, err := net.LookupMX(domain)
	if err != nil {
		return false, fmt.Errorf("failed to lookup MX record: %w", err)
	}

	for _, mx := range mxs {
		host := mx.Host
		// 移除末尾的点
		if len(host) > 0 && host[len(host)-1] == '.' {
			host = host[:len(host)-1]
		}
		if host == expectedMX {
			return true, nil
		}
	}

	return false, nil
}

// SyncWithProvider 与 DNS 提供商同步
func (s *dnsService) SyncWithProvider(ctx context.Context, domain, provider string) error {
	// 获取提供商配置
	// TODO: 从配置中获取提供商凭证
	config := &DNSProviderConfig{
		Provider: provider,
		// AccessKeyID 和 AccessKeySecret 需要从配置或数据库中获取
	}

	dnsProvider, err := NewDNSProvider(config)
	if err != nil {
		return fmt.Errorf("failed to create dns provider: %w", err)
	}

	// 从提供商获取记录
	providerRecords, err := dnsProvider.ListRecords(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to list records from provider: %w", err)
	}

	// 获取本地记录
	localRecords, err := s.ListDNSRecords(ctx, domain, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to list local records: %w", err)
	}

	// 创建本地记录映射
	localRecordMap := make(map[string]*DNSRecord)
	for _, record := range localRecords {
		key := fmt.Sprintf("%s-%s-%s", record.Name, record.Type, record.Value)
		localRecordMap[key] = record
	}

	// 同步记录
	for _, providerRecord := range providerRecords {
		key := fmt.Sprintf("%s-%s-%s", providerRecord.Name, providerRecord.Type, providerRecord.Value)
		
		if localRecord, exists := localRecordMap[key]; exists {
			// 更新现有记录
			localRecord.ProviderID = providerRecord.ProviderID
			localRecord.TTL = providerRecord.TTL
			localRecord.Priority = providerRecord.Priority
			s.UpdateDNSRecord(ctx, localRecord)
		} else {
			// 创建新记录
			s.CreateDNSRecord(ctx, providerRecord)
		}
	}

	return nil
}
