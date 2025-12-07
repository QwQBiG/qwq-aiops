package website

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// generateSelfSignedCert 生成自签名证书
func generateSelfSignedCert(domain string) (*CertificateBundle, error) {
	// 生成私钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// 生成序列号
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// 证书模板
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // 1年有效期

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"qwq AIOps Platform"},
			CommonName:   domain,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain},
	}

	// 创建自签名证书
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// 编码证书
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	// 编码私钥
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	return &CertificateBundle{
		Domain:      domain,
		Certificate: certPEM,
		PrivateKey:  keyPEM,
		NotBefore:   notBefore,
		NotAfter:    notAfter,
	}, nil
}

// UploadManualCertificate 上传手动证书
func UploadManualCertificate(domain string, certPEM, keyPEM []byte) (*CertificateBundle, error) {
	// 验证证书格式
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// 验证私钥格式
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode private key PEM")
	}

	// 尝试解析不同类型的私钥
	var privateKey interface{}
	privateKey, err = x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		privateKey, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			privateKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key: %w", err)
			}
		}
	}

	// 验证私钥和证书是否匹配
	if err := validateKeyPair(cert, privateKey); err != nil {
		return nil, fmt.Errorf("certificate and private key do not match: %w", err)
	}

	return &CertificateBundle{
		Domain:      domain,
		Certificate: certPEM,
		PrivateKey:  keyPEM,
		NotBefore:   cert.NotBefore,
		NotAfter:    cert.NotAfter,
	}, nil
}

// validateKeyPair 验证证书和私钥是否匹配
func validateKeyPair(cert *x509.Certificate, privateKey interface{}) error {
	switch key := privateKey.(type) {
	case *ecdsa.PrivateKey:
		pubKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			return fmt.Errorf("certificate public key type mismatch")
		}
		if !key.PublicKey.Equal(pubKey) {
			return fmt.Errorf("public keys do not match")
		}
	default:
		// 对于其他类型的密钥，暂时跳过验证
		return nil
	}
	return nil
}
