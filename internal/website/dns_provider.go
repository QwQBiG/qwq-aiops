package website

import (
	"context"
	"fmt"
)

// DNSProvider DNS 提供商接口
type DNSProvider interface {
	// AddRecord 添加 DNS 记录
	AddRecord(ctx context.Context, record *DNSRecord) (string, error)
	
	// UpdateRecord 更新 DNS 记录
	UpdateRecord(ctx context.Context, record *DNSRecord) error
	
	// DeleteRecord 删除 DNS 记录
	DeleteRecord(ctx context.Context, providerID string) error
	
	// GetRecord 获取 DNS 记录
	GetRecord(ctx context.Context, providerID string) (*DNSRecord, error)
	
	// ListRecords 列出 DNS 记录
	ListRecords(ctx context.Context, domain string) ([]*DNSRecord, error)
	
	// GetName 获取提供商名称
	GetName() string
}

// DNSProviderConfig DNS 提供商配置
type DNSProviderConfig struct {
	Provider    string            `json:"provider"`     // 提供商名称
	AccessKeyID string            `json:"access_key_id"` // 访问密钥 ID
	AccessKeySecret string        `json:"access_key_secret"` // 访问密钥
	Region      string            `json:"region,omitempty"` // 区域（某些提供商需要）
	Endpoint    string            `json:"endpoint,omitempty"` // 自定义端点
	Extra       map[string]string `json:"extra,omitempty"` // 额外配置
}

// NewDNSProvider 创建 DNS 提供商实例
func NewDNSProvider(config *DNSProviderConfig) (DNSProvider, error) {
	switch config.Provider {
	case "aliyun":
		return NewAliyunDNSProvider(config)
	case "tencent":
		return NewTencentDNSProvider(config)
	case "cloudflare":
		return NewCloudflareDNSProvider(config)
	default:
		return nil, fmt.Errorf("unsupported dns provider: %s", config.Provider)
	}
}

// DNSProviderManager DNS 提供商管理器
type DNSProviderManager struct {
	providers map[string]DNSProvider
}

// NewDNSProviderManager 创建 DNS 提供商管理器
func NewDNSProviderManager() *DNSProviderManager {
	return &DNSProviderManager{
		providers: make(map[string]DNSProvider),
	}
}

// RegisterProvider 注册提供商
func (m *DNSProviderManager) RegisterProvider(name string, provider DNSProvider) {
	m.providers[name] = provider
}

// GetProvider 获取提供商
func (m *DNSProviderManager) GetProvider(name string) (DNSProvider, error) {
	provider, ok := m.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

// ListProviders 列出所有提供商
func (m *DNSProviderManager) ListProviders() []string {
	var names []string
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}
