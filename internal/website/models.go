package website

import (
	"time"

	"gorm.io/gorm"
)

// WebsiteStatus 网站状态
type WebsiteStatus string

const (
	StatusActive   WebsiteStatus = "active"   // 活跃
	StatusInactive WebsiteStatus = "inactive" // 未激活
	StatusError    WebsiteStatus = "error"    // 错误
)

// ProxyType 代理类型
type ProxyType string

const (
	ProxyTypeHTTP    ProxyType = "http"    // HTTP 代理
	ProxyTypeReverse ProxyType = "reverse" // 反向代理
	ProxyTypeStream  ProxyType = "stream"  // TCP/UDP 流代理
)

// LoadBalanceMethod 负载均衡方法
type LoadBalanceMethod string

const (
	LoadBalanceRoundRobin  LoadBalanceMethod = "round_robin"  // 轮询
	LoadBalanceLeastConn   LoadBalanceMethod = "least_conn"   // 最少连接
	LoadBalanceIPHash      LoadBalanceMethod = "ip_hash"      // IP 哈希
	LoadBalanceWeighted    LoadBalanceMethod = "weighted"     // 加权轮询
)

// SSLProvider SSL 证书提供商
type SSLProvider string

const (
	SSLProviderLetsEncrypt SSLProvider = "letsencrypt" // Let's Encrypt
	SSLProviderManual      SSLProvider = "manual"      // 手动上传
	SSLProviderSelfSigned  SSLProvider = "self_signed" // 自签名
)

// SSLStatus SSL 证书状态
type SSLStatus string

const (
	SSLStatusValid   SSLStatus = "valid"   // 有效
	SSLStatusExpired SSLStatus = "expired" // 已过期
	SSLStatusPending SSLStatus = "pending" // 待申请
	SSLStatusError   SSLStatus = "error"   // 错误
)

// DNSRecordType DNS 记录类型
type DNSRecordType string

const (
	DNSRecordA     DNSRecordType = "A"     // IPv4 地址
	DNSRecordAAAA  DNSRecordType = "AAAA"  // IPv6 地址
	DNSRecordCNAME DNSRecordType = "CNAME" // 别名
	DNSRecordMX    DNSRecordType = "MX"    // 邮件交换
	DNSRecordTXT   DNSRecordType = "TXT"   // 文本记录
	DNSRecordNS    DNSRecordType = "NS"    // 名称服务器
)

// Website 网站模型
type Website struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	Name            string         `json:"name" gorm:"not null;index"`                // 网站名称
	Domain          string         `json:"domain" gorm:"unique;not null;index"`       // 主域名
	Aliases         string         `json:"aliases" gorm:"type:text"`                  // 域名别名（JSON数组）
	Status          WebsiteStatus  `json:"status" gorm:"default:inactive;index"`      // 网站状态
	SSLEnabled      bool           `json:"ssl_enabled" gorm:"default:false"`          // 是否启用 SSL
	SSLCertID       *uint          `json:"ssl_cert_id,omitempty" gorm:"index"`        // SSL 证书 ID
	SSLCert         *SSLCert       `json:"ssl_cert,omitempty" gorm:"foreignKey:SSLCertID"`
	ProxyConfigID   *uint          `json:"proxy_config_id,omitempty" gorm:"index"`    // 代理配置 ID
	ProxyConfig     *ProxyConfig   `json:"proxy_config,omitempty" gorm:"foreignKey:ProxyConfigID"`
	UserID          uint           `json:"user_id" gorm:"not null;index"`             // 所属用户
	TenantID        uint           `json:"tenant_id" gorm:"not null;index"`           // 所属租户
	Description     string         `json:"description" gorm:"type:text"`              // 描述
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// ProxyConfig 反向代理配置模型
type ProxyConfig struct {
	ID                  uint              `json:"id" gorm:"primaryKey"`
	Name                string            `json:"name" gorm:"not null;index"`           // 配置名称
	ProxyType           ProxyType         `json:"proxy_type" gorm:"default:reverse"`    // 代理类型
	Backend             string            `json:"backend" gorm:"not null"`              // 后端地址（单个或JSON数组）
	LoadBalanceMethod   LoadBalanceMethod `json:"load_balance_method" gorm:"default:round_robin"` // 负载均衡方法
	HealthCheckEnabled  bool              `json:"health_check_enabled" gorm:"default:true"`       // 是否启用健康检查
	HealthCheckPath     string            `json:"health_check_path" gorm:"default:/"`             // 健康检查路径
	HealthCheckInterval int               `json:"health_check_interval" gorm:"default:30"`        // 健康检查间隔（秒）
	Timeout             int               `json:"timeout" gorm:"default:60"`                      // 超时时间（秒）
	MaxBodySize         int64             `json:"max_body_size" gorm:"default:10485760"`          // 最大请求体大小（字节）
	CustomConfig        string            `json:"custom_config" gorm:"type:text"`                 // 自定义 Nginx 配置
	UserID              uint              `json:"user_id" gorm:"not null;index"`                  // 所属用户
	TenantID            uint              `json:"tenant_id" gorm:"not null;index"`                // 所属租户
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	DeletedAt           gorm.DeletedAt    `json:"-" gorm:"index"`
}

// SSLCert SSL 证书模型
type SSLCert struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	Domain          string         `json:"domain" gorm:"not null;index"`              // 证书域名
	Provider        SSLProvider    `json:"provider" gorm:"not null"`                  // 证书提供商
	Status          SSLStatus      `json:"status" gorm:"default:pending;index"`       // 证书状态
	CertPath        string         `json:"cert_path"`                                 // 证书文件路径
	KeyPath         string         `json:"key_path"`                                  // 私钥文件路径
	CertContent     string         `json:"cert_content,omitempty" gorm:"type:text"`   // 证书内容
	KeyContent      string         `json:"key_content,omitempty" gorm:"type:text"`    // 私钥内容
	IssueDate       *time.Time     `json:"issue_date,omitempty"`                      // 签发日期
	ExpiryDate      *time.Time     `json:"expiry_date,omitempty" gorm:"index"`        // 过期日期
	AutoRenew       *bool          `json:"auto_renew" gorm:"default:true"`            // 是否自动续期（指针类型以区分未设置和false）
	RenewDaysBefore int            `json:"renew_days_before" gorm:"default:30"`       // 提前多少天续期
	Email           string         `json:"email"`                                     // 联系邮箱
	UserID          uint           `json:"user_id" gorm:"not null;index"`             // 所属用户
	TenantID        uint           `json:"tenant_id" gorm:"not null;index"`           // 所属租户
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// DNSRecord DNS 记录模型
type DNSRecord struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Domain    string         `json:"domain" gorm:"not null;index"`          // 域名
	Type      DNSRecordType  `json:"type" gorm:"not null;index"`            // 记录类型
	Name      string         `json:"name" gorm:"not null;index"`            // 记录名称（子域名）
	Value     string         `json:"value" gorm:"not null"`                 // 记录值
	TTL       int            `json:"ttl" gorm:"default:600"`                // TTL（秒）
	Priority  int            `json:"priority,omitempty"`                    // 优先级（MX记录）
	Provider  string         `json:"provider"`                              // DNS 提供商（aliyun, tencent等）
	ProviderID string        `json:"provider_id"`                           // 提供商记录ID
	UserID    uint           `json:"user_id" gorm:"not null;index"`         // 所属用户
	TenantID  uint           `json:"tenant_id" gorm:"not null;index"`       // 所属租户
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (Website) TableName() string {
	return "websites"
}

func (ProxyConfig) TableName() string {
	return "proxy_configs"
}

func (SSLCert) TableName() string {
	return "ssl_certs"
}

func (DNSRecord) TableName() string {
	return "dns_records"
}
