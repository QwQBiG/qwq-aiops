# Website Management Module

网站管理模块提供了完整的网站、SSL 证书、DNS 和反向代理管理功能。

## 功能特性

### 1. 网站管理
- 创建、更新、删除网站配置
- 支持域名别名
- 网站状态监控
- 多租户隔离

### 2. 反向代理配置
- 自动生成 Nginx 配置
- 支持多种负载均衡策略：
  - 轮询 (Round Robin)
  - 最少连接 (Least Connections)
  - IP 哈希 (IP Hash)
  - 加权轮询 (Weighted Round Robin)
- 健康检查配置
- 自定义 Nginx 配置
- 配置验证和自动重载

### 3. SSL 证书管理
- Let's Encrypt 自动申请和续期
- 自签名证书生成
- 手动证书上传
- 证书过期监控
- 自动续期调度器

### 4. DNS 管理
- DNS 记录 CRUD 操作
- 支持多种记录类型：A、AAAA、CNAME、MX、TXT、NS
- DNS 解析验证
- 多提供商支持：
  - 阿里云 DNS
  - 腾讯云 DNS
  - Cloudflare DNS
- 与 DNS 提供商同步

### 5. AI 配置优化
- 自动检测配置问题
- 生成优化建议
- 自动修复常见问题
- 性能分析
- 安全加固建议

## 使用示例

### 创建网站

```go
package main

import (
    "context"
    "github.com/yourusername/qwq/internal/website"
    "gorm.io/gorm"
)

func main() {
    // 初始化数据库连接
    var db *gorm.DB
    // ... 初始化 db

    // 创建服务
    websiteService := website.NewWebsiteService(db)
    proxyService := website.NewProxyService(db)
    sslService := website.NewSSLService(db)

    ctx := context.Background()

    // 1. 创建代理配置
    proxyConfig := &website.ProxyConfig{
        Name:                "example-proxy",
        ProxyType:           website.ProxyTypeReverse,
        Backend:             "http://localhost:3000",
        LoadBalanceMethod:   website.LoadBalanceRoundRobin,
        HealthCheckEnabled:  true,
        HealthCheckPath:     "/health",
        HealthCheckInterval: 30,
        Timeout:             60,
        MaxBodySize:         10485760,
        UserID:              1,
        TenantID:            1,
    }
    proxyService.CreateProxyConfig(ctx, proxyConfig)

    // 2. 申请 SSL 证书
    cert, err := sslService.RequestCertificate(
        ctx,
        "example.com",
        "admin@example.com",
        website.SSLProviderLetsEncrypt,
    )
    if err != nil {
        panic(err)
    }

    // 3. 创建网站
    site := &website.Website{
        Name:          "Example Website",
        Domain:        "example.com",
        Status:        website.StatusActive,
        SSLEnabled:    true,
        SSLCertID:     &cert.ID,
        ProxyConfigID: &proxyConfig.ID,
        UserID:        1,
        TenantID:      1,
    }
    websiteService.CreateWebsite(ctx, site)

    // 4. 生成 Nginx 配置
    nginxConfig, err := proxyService.GenerateNginxConfig(ctx, site)
    if err != nil {
        panic(err)
    }

    // 5. 写入配置文件并启用
    website.WriteNginxConfig(site.Domain, nginxConfig)
    website.EnableNginxSite(site.Domain)
    website.ReloadNginx()
}
```

### 证书自动续期

```go
package main

import (
    "context"
    "time"
    "github.com/yourusername/qwq/internal/website"
    "gorm.io/gorm"
)

func main() {
    var db *gorm.DB
    // ... 初始化 db

    // 创建证书监控器
    monitor := website.NewCertMonitor(db, 24*time.Hour) // 每天检查一次

    ctx := context.Background()

    // 启动监控（在后台运行）
    go monitor.Start(ctx)

    // 获取证书统计
    stats, err := monitor.GetCertificateStats(ctx)
    if err != nil {
        panic(err)
    }

    println("Total certificates:", stats.Total)
    println("Valid certificates:", stats.Valid)
    println("Expiring soon:", stats.ExpiringSoon)
}
```

### DNS 管理

```go
package main

import (
    "context"
    "github.com/yourusername/qwq/internal/website"
    "gorm.io/gorm"
)

func main() {
    var db *gorm.DB
    // ... 初始化 db

    dnsService := website.NewDNSService(db)
    ctx := context.Background()

    // 创建 DNS 记录
    record := &website.DNSRecord{
        Domain:   "example.com",
        Type:     website.DNSRecordA,
        Name:     "www",
        Value:    "192.168.1.1",
        TTL:      600,
        Provider: "aliyun",
        UserID:   1,
        TenantID: 1,
    }
    dnsService.CreateDNSRecord(ctx, record)

    // 验证 DNS 解析
    verified, err := dnsService.VerifyDNS(
        ctx,
        "www.example.com",
        "A",
        "192.168.1.1",
    )
    if err != nil {
        panic(err)
    }
    println("DNS verified:", verified)
}
```

### AI 配置优化

```go
package main

import (
    "context"
    "github.com/yourusername/qwq/internal/website"
    "gorm.io/gorm"
)

func main() {
    var db *gorm.DB
    // ... 初始化 db

    websiteService := website.NewWebsiteService(db)
    proxyService := website.NewProxyService(db)
    aiService := website.NewAIOptimizationService(db, websiteService, proxyService)

    ctx := context.Background()

    // 分析网站配置
    analysis, err := aiService.AnalyzeWebsiteConfig(ctx, 1)
    if err != nil {
        panic(err)
    }

    println("Configuration score:", analysis.Score)
    println("Issues found:", len(analysis.Issues))
    println("Suggestions:", len(analysis.Suggestions))

    // 自动修复常见问题
    result, err := aiService.AutoFixCommonIssues(ctx, 1)
    if err != nil {
        panic(err)
    }

    println("Fixed issues:", len(result.FixedIssues))
    println("Failed issues:", len(result.FailedIssues))

    // 性能分析
    perfAnalysis, err := aiService.AnalyzePerformance(ctx, 1)
    if err != nil {
        panic(err)
    }

    println("Response time:", perfAnalysis.ResponseTime, "ms")
    println("Throughput:", perfAnalysis.Throughput, "req/s")
}
```

## 数据模型

### Website
- 网站基本信息
- 域名和别名
- SSL 和代理配置关联
- 多租户支持

### ProxyConfig
- 反向代理配置
- 负载均衡策略
- 健康检查设置
- 自定义 Nginx 配置

### SSLCert
- SSL 证书信息
- 证书提供商
- 自动续期配置
- 证书文件路径

### DNSRecord
- DNS 记录信息
- 多种记录类型
- DNS 提供商集成
- TTL 和优先级配置

## 配置要求

### Nginx
- 需要安装 Nginx
- 配置目录：`/etc/nginx/sites-available` 和 `/etc/nginx/sites-enabled`
- 需要有写入权限

### SSL 证书
- 证书存储目录：`/etc/qwq/ssl`
- Let's Encrypt 挑战目录：`/var/www/html/.well-known/acme-challenge`

### DNS 提供商
- 阿里云：需要 AccessKey ID 和 Secret
- 腾讯云：需要 SecretId 和 SecretKey
- Cloudflare：需要 API Token

## 安全注意事项

1. **私钥保护**：SSL 私钥文件权限设置为 0600
2. **凭证管理**：DNS 提供商凭证应加密存储
3. **权限隔离**：多租户环境下严格执行资源隔离
4. **审计日志**：所有操作都应记录审计日志

## 性能优化

1. **配置缓存**：Nginx 配置生成结果可以缓存
2. **批量操作**：DNS 记录同步支持批量处理
3. **异步处理**：证书申请和续期使用异步任务
4. **连接池**：DNS 提供商 API 调用使用连接池

## 故障排查

### Nginx 配置错误
```bash
# 测试配置
nginx -t

# 查看错误日志
tail -f /var/log/nginx/error.log
```

### SSL 证书问题
```bash
# 检查证书有效期
openssl x509 -in /etc/qwq/ssl/example_com.crt -noout -dates

# 验证证书和私钥匹配
openssl x509 -noout -modulus -in cert.crt | openssl md5
openssl rsa -noout -modulus -in key.key | openssl md5
```

### DNS 解析问题
```bash
# 查询 DNS 记录
dig example.com A
nslookup example.com

# 追踪 DNS 解析路径
dig +trace example.com
```

## 未来计划

- [ ] 支持更多 DNS 提供商
- [ ] Kubernetes Ingress 集成
- [ ] WAF (Web Application Firewall) 集成
- [ ] CDN 集成
- [ ] 更智能的 AI 优化建议
- [ ] 配置模板系统
- [ ] 批量网站管理
- [ ] 网站性能监控集成
