package website

import (
	"context"
	"errors"
)

var (
	// ErrWebsiteNotFound 网站未找到
	ErrWebsiteNotFound = errors.New("website not found")
	// ErrWebsiteExists 网站已存在
	ErrWebsiteExists = errors.New("website already exists")
	// ErrInvalidDomain 无效的域名
	ErrInvalidDomain = errors.New("invalid domain")
	// ErrSSLCertNotFound SSL证书未找到
	ErrSSLCertNotFound = errors.New("ssl certificate not found")
	// ErrSSLCertExpired SSL证书已过期
	ErrSSLCertExpired = errors.New("ssl certificate expired")
	// ErrProxyConfigNotFound 代理配置未找到
	ErrProxyConfigNotFound = errors.New("proxy config not found")
	// ErrDNSRecordNotFound DNS记录未找到
	ErrDNSRecordNotFound = errors.New("dns record not found")
	// ErrInvalidBackend 无效的后端地址
	ErrInvalidBackend = errors.New("invalid backend address")
)

// WebsiteService 网站管理服务接口
type WebsiteService interface {
	// CreateWebsite 创建网站
	CreateWebsite(ctx context.Context, website *Website) error
	
	// GetWebsite 获取网站信息
	GetWebsite(ctx context.Context, id uint) (*Website, error)
	
	// GetWebsiteByDomain 根据域名获取网站
	GetWebsiteByDomain(ctx context.Context, domain string) (*Website, error)
	
	// ListWebsites 列出网站
	ListWebsites(ctx context.Context, userID, tenantID uint, page, pageSize int) ([]*Website, int64, error)
	
	// UpdateWebsite 更新网站
	UpdateWebsite(ctx context.Context, website *Website) error
	
	// DeleteWebsite 删除网站
	DeleteWebsite(ctx context.Context, id uint) error
	
	// EnableSSL 启用 SSL
	EnableSSL(ctx context.Context, websiteID, certID uint) error
	
	// DisableSSL 禁用 SSL
	DisableSSL(ctx context.Context, websiteID uint) error
}

// ProxyService 反向代理服务接口
type ProxyService interface {
	// CreateProxyConfig 创建代理配置
	CreateProxyConfig(ctx context.Context, config *ProxyConfig) error
	
	// GetProxyConfig 获取代理配置
	GetProxyConfig(ctx context.Context, id uint) (*ProxyConfig, error)
	
	// ListProxyConfigs 列出代理配置
	ListProxyConfigs(ctx context.Context, userID, tenantID uint) ([]*ProxyConfig, error)
	
	// UpdateProxyConfig 更新代理配置
	UpdateProxyConfig(ctx context.Context, config *ProxyConfig) error
	
	// DeleteProxyConfig 删除代理配置
	DeleteProxyConfig(ctx context.Context, id uint) error
	
	// GenerateNginxConfig 生成 Nginx 配置
	GenerateNginxConfig(ctx context.Context, website *Website) (string, error)
	
	// ValidateConfig 验证配置
	ValidateConfig(ctx context.Context, config string) error
	
	// ReloadNginx 重载 Nginx
	ReloadNginx(ctx context.Context) error
}

// SSLService SSL 证书服务接口
type SSLService interface {
	// CreateSSLCert 创建 SSL 证书记录
	CreateSSLCert(ctx context.Context, cert *SSLCert) error
	
	// GetSSLCert 获取 SSL 证书
	GetSSLCert(ctx context.Context, id uint) (*SSLCert, error)
	
	// GetSSLCertByDomain 根据域名获取证书
	GetSSLCertByDomain(ctx context.Context, domain string) (*SSLCert, error)
	
	// ListSSLCerts 列出 SSL 证书
	ListSSLCerts(ctx context.Context, userID, tenantID uint) ([]*SSLCert, error)
	
	// UpdateSSLCert 更新 SSL 证书
	UpdateSSLCert(ctx context.Context, cert *SSLCert) error
	
	// DeleteSSLCert 删除 SSL 证书
	DeleteSSLCert(ctx context.Context, id uint) error
	
	// RequestCertificate 申请证书
	RequestCertificate(ctx context.Context, domain, email string, provider SSLProvider) (*SSLCert, error)
	
	// RenewCertificate 续期证书
	RenewCertificate(ctx context.Context, certID uint) error
	
	// CheckExpiry 检查证书过期状态
	CheckExpiry(ctx context.Context) ([]*SSLCert, error)
	
	// AutoRenew 自动续期即将过期的证书
	AutoRenew(ctx context.Context) error
}

// DNSService DNS 管理服务接口
type DNSService interface {
	// CreateDNSRecord 创建 DNS 记录
	CreateDNSRecord(ctx context.Context, record *DNSRecord) error
	
	// GetDNSRecord 获取 DNS 记录
	GetDNSRecord(ctx context.Context, id uint) (*DNSRecord, error)
	
	// ListDNSRecords 列出 DNS 记录
	ListDNSRecords(ctx context.Context, domain string, userID, tenantID uint) ([]*DNSRecord, error)
	
	// UpdateDNSRecord 更新 DNS 记录
	UpdateDNSRecord(ctx context.Context, record *DNSRecord) error
	
	// DeleteDNSRecord 删除 DNS 记录
	DeleteDNSRecord(ctx context.Context, id uint) error
	
	// VerifyDNS 验证 DNS 解析
	VerifyDNS(ctx context.Context, domain, recordType, expectedValue string) (bool, error)
	
	// SyncWithProvider 与 DNS 提供商同步
	SyncWithProvider(ctx context.Context, domain, provider string) error
}

// AIOptimizationService AI 网站配置优化服务接口
type AIOptimizationService interface {
	// AnalyzeWebsiteConfig 分析网站配置
	AnalyzeWebsiteConfig(ctx context.Context, websiteID uint) (*ConfigAnalysis, error)
	
	// DetectConfigIssues 检测配置问题
	DetectConfigIssues(ctx context.Context, config string) ([]*ConfigIssue, error)
	
	// GenerateOptimizationSuggestions 生成优化建议
	GenerateOptimizationSuggestions(ctx context.Context, websiteID uint) ([]*OptimizationSuggestion, error)
	
	// AutoFixCommonIssues 自动修复常见问题
	AutoFixCommonIssues(ctx context.Context, websiteID uint) (*FixResult, error)
	
	// AnalyzePerformance 分析性能
	AnalyzePerformance(ctx context.Context, websiteID uint) (*PerformanceAnalysis, error)
}

// ConfigAnalysis 配置分析结果
type ConfigAnalysis struct {
	WebsiteID   uint            `json:"website_id"`
	Score       int             `json:"score"`        // 配置评分 0-100
	Issues      []*ConfigIssue  `json:"issues"`       // 发现的问题
	Suggestions []*OptimizationSuggestion `json:"suggestions"` // 优化建议
	AnalyzedAt  string          `json:"analyzed_at"`
}

// ConfigIssue 配置问题
type ConfigIssue struct {
	Severity    string `json:"severity"`    // 严重程度: critical, warning, info
	Category    string `json:"category"`    // 问题类别: security, performance, compatibility
	Title       string `json:"title"`       // 问题标题
	Description string `json:"description"` // 问题描述
	Location    string `json:"location"`    // 问题位置
	CanAutoFix  bool   `json:"can_auto_fix"` // 是否可以自动修复
}

// OptimizationSuggestion 优化建议
type OptimizationSuggestion struct {
	Category    string `json:"category"`    // 建议类别: performance, security, seo
	Title       string `json:"title"`       // 建议标题
	Description string `json:"description"` // 建议描述
	Impact      string `json:"impact"`      // 影响程度: high, medium, low
	Effort      string `json:"effort"`      // 实施难度: easy, medium, hard
	Action      string `json:"action"`      // 建议操作
}

// FixResult 修复结果
type FixResult struct {
	Success      bool     `json:"success"`
	FixedIssues  []string `json:"fixed_issues"`  // 已修复的问题
	FailedIssues []string `json:"failed_issues"` // 修复失败的问题
	Message      string   `json:"message"`
}

// PerformanceAnalysis 性能分析结果
type PerformanceAnalysis struct {
	WebsiteID       uint              `json:"website_id"`
	ResponseTime    float64           `json:"response_time"`    // 平均响应时间（毫秒）
	Throughput      float64           `json:"throughput"`       // 吞吐量（请求/秒）
	ErrorRate       float64           `json:"error_rate"`       // 错误率
	Bottlenecks     []string          `json:"bottlenecks"`      // 性能瓶颈
	Recommendations []string          `json:"recommendations"`  // 性能优化建议
	Metrics         map[string]interface{} `json:"metrics"` // 详细指标
}
