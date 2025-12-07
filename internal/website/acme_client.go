package website

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/acme"
)

const (
	// LetsEncryptProductionURL Let's Encrypt 生产环境 URL
	LetsEncryptProductionURL = "https://acme-v02.api.letsencrypt.org/directory"
	// LetsEncryptStagingURL Let's Encrypt 测试环境 URL
	LetsEncryptStagingURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
	// CertStorageDir 证书存储目录
	CertStorageDir = "/etc/qwq/ssl"
)

// ACMEClient ACME 客户端
type ACMEClient struct {
	client      *acme.Client
	accountKey  crypto.Signer
	directoryURL string
}

// NewACMEClient 创建 ACME 客户端
func NewACMEClient(staging bool) (*ACMEClient, error) {
	// 生成账户密钥
	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate account key: %w", err)
	}

	// 选择目录 URL
	directoryURL := LetsEncryptProductionURL
	if staging {
		directoryURL = LetsEncryptStagingURL
	}

	client := &acme.Client{
		Key:          accountKey,
		DirectoryURL: directoryURL,
	}

	return &ACMEClient{
		client:       client,
		accountKey:   accountKey,
		directoryURL: directoryURL,
	}, nil
}

// Register 注册 ACME 账户
func (c *ACMEClient) Register(ctx context.Context, email string) error {
	account := &acme.Account{
		Contact: []string{"mailto:" + email},
	}

	_, err := c.client.Register(ctx, account, acme.AcceptTOS)
	if err != nil {
		// 如果账户已存在，忽略错误
		if err != acme.ErrAccountAlreadyExists {
			return fmt.Errorf("failed to register account: %w", err)
		}
	}

	return nil
}

// ObtainCertificate 获取证书
func (c *ACMEClient) ObtainCertificate(ctx context.Context, domain, email string) (*CertificateBundle, error) {
	// 注册账户
	if err := c.Register(ctx, email); err != nil {
		return nil, err
	}

	// 创建订单
	order, err := c.client.AuthorizeOrder(ctx, []acme.AuthzID{
		{Type: "dns", Value: domain},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// 完成挑战
	for _, authzURL := range order.AuthzURLs {
		if err := c.completeChallenge(ctx, authzURL); err != nil {
			return nil, fmt.Errorf("failed to complete challenge: %w", err)
		}
	}

	// 等待订单准备就绪
	order, err = c.client.WaitOrder(ctx, order.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for order: %w", err)
	}

	// 生成证书私钥
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cert key: %w", err)
	}

	// 创建 CSR
	csr, err := c.createCSR(certKey, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to create csr: %w", err)
	}

	// 完成订单
	der, _, err := c.client.CreateOrderCert(ctx, order.FinalizeURL, csr, true)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize order: %w", err)
	}

	// 解析证书
	cert, err := x509.ParseCertificate(der[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// 编码证书和私钥
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der[0],
	})

	keyBytes, err := x509.MarshalECPrivateKey(certKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	return &CertificateBundle{
		Domain:      domain,
		Certificate: certPEM,
		PrivateKey:  keyPEM,
		NotBefore:   cert.NotBefore,
		NotAfter:    cert.NotAfter,
	}, nil
}

// completeChallenge 完成挑战
func (c *ACMEClient) completeChallenge(ctx context.Context, authzURL string) error {
	// 获取授权
	authz, err := c.client.GetAuthorization(ctx, authzURL)
	if err != nil {
		return fmt.Errorf("failed to get authorization: %w", err)
	}

	// 如果已经验证，直接返回
	if authz.Status == acme.StatusValid {
		return nil
	}

	// 选择 HTTP-01 挑战
	var challenge *acme.Challenge
	for _, ch := range authz.Challenges {
		if ch.Type == "http-01" {
			challenge = ch
			break
		}
	}

	if challenge == nil {
		return fmt.Errorf("no http-01 challenge found")
	}

	// 获取挑战响应
	response, err := c.client.HTTP01ChallengeResponse(challenge.Token)
	if err != nil {
		return fmt.Errorf("failed to get challenge response: %w", err)
	}

	// 创建挑战文件
	challengePath := filepath.Join("/var/www/html/.well-known/acme-challenge", challenge.Token)
	if err := os.MkdirAll(filepath.Dir(challengePath), 0755); err != nil {
		return fmt.Errorf("failed to create challenge directory: %w", err)
	}

	if err := os.WriteFile(challengePath, []byte(response), 0644); err != nil {
		return fmt.Errorf("failed to write challenge file: %w", err)
	}
	defer os.Remove(challengePath)

	// 接受挑战
	if _, err := c.client.Accept(ctx, challenge); err != nil {
		return fmt.Errorf("failed to accept challenge: %w", err)
	}

	// 等待验证完成
	_, err = c.client.WaitAuthorization(ctx, authzURL)
	if err != nil {
		return fmt.Errorf("failed to wait for authorization: %w", err)
	}

	return nil
}

// createCSR 创建证书签名请求
func (c *ACMEClient) createCSR(key crypto.Signer, domain string) ([]byte, error) {
	template := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: domain},
		DNSNames: []string{domain},
	}

	return x509.CreateCertificateRequest(rand.Reader, template, key)
}

// RenewCertificate 续期证书
func (c *ACMEClient) RenewCertificate(ctx context.Context, domain, email string) (*CertificateBundle, error) {
	// 续期实际上就是重新获取证书
	return c.ObtainCertificate(ctx, domain, email)
}

// CertificateBundle 证书包
type CertificateBundle struct {
	Domain      string
	Certificate []byte
	PrivateKey  []byte
	NotBefore   time.Time
	NotAfter    time.Time
}

// SaveToFile 保存证书到文件
func (b *CertificateBundle) SaveToFile() (certPath, keyPath string, err error) {
	// 确保存储目录存在
	if err := os.MkdirAll(CertStorageDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 证书文件路径
	certPath = filepath.Join(CertStorageDir, sanitizeName(b.Domain)+".crt")
	keyPath = filepath.Join(CertStorageDir, sanitizeName(b.Domain)+".key")

	// 保存证书
	if err := os.WriteFile(certPath, b.Certificate, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write certificate: %w", err)
	}

	// 保存私钥（设置更严格的权限）
	if err := os.WriteFile(keyPath, b.PrivateKey, 0600); err != nil {
		return "", "", fmt.Errorf("failed to write private key: %w", err)
	}

	return certPath, keyPath, nil
}

// LoadFromFile 从文件加载证书
func LoadCertificateFromFile(domain string) (*CertificateBundle, error) {
	certPath := filepath.Join(CertStorageDir, sanitizeName(domain)+".crt")
	keyPath := filepath.Join(CertStorageDir, sanitizeName(domain)+".key")

	// 读取证书
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	// 读取私钥
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// 解析证书以获取有效期
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return &CertificateBundle{
		Domain:      domain,
		Certificate: certPEM,
		PrivateKey:  keyPEM,
		NotBefore:   cert.NotBefore,
		NotAfter:    cert.NotAfter,
	}, nil
}
