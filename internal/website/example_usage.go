package website

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ExampleWebsiteManagement 网站管理示例
func ExampleWebsiteManagement(db *gorm.DB) error {
	ctx := context.Background()

	// 创建服务实例
	websiteService := NewWebsiteService(db)
	proxyService := NewProxyService(db)
	sslService := NewSSLService(db)
	dnsService := NewDNSService(db)
	aiService := NewAIOptimizationService(db, websiteService, proxyService)

	// 1. 创建代理配置
	fmt.Println("=== 创建代理配置 ===")
	proxyConfig := &ProxyConfig{
		Name:                "example-proxy",
		ProxyType:           ProxyTypeReverse,
		Backend:             "http://localhost:3000",
		LoadBalanceMethod:   LoadBalanceRoundRobin,
		HealthCheckEnabled:  true,
		HealthCheckPath:     "/health",
		HealthCheckInterval: 30,
		Timeout:             60,
		MaxBodySize:         10485760,
		UserID:              1,
		TenantID:            1,
	}

	if err := proxyService.CreateProxyConfig(ctx, proxyConfig); err != nil {
		return fmt.Errorf("failed to create proxy config: %w", err)
	}
	fmt.Printf("代理配置创建成功，ID: %d\n", proxyConfig.ID)

	// 2. 申请 SSL 证书（使用自签名证书作为示例）
	fmt.Println("\n=== 申请 SSL 证书 ===")
	cert, err := sslService.RequestCertificate(
		ctx,
		"example.com",
		"admin@example.com",
		SSLProviderSelfSigned, // 使用自签名证书便于测试
	)
	if err != nil {
		return fmt.Errorf("failed to request certificate: %w", err)
	}
	fmt.Printf("SSL 证书申请成功，ID: %d\n", cert.ID)

	// 3. 创建网站
	fmt.Println("\n=== 创建网站 ===")
	site := &Website{
		Name:          "Example Website",
		Domain:        "example.com",
		Aliases:       `["www.example.com"]`,
		Status:        StatusActive,
		SSLEnabled:    true,
		SSLCertID:     &cert.ID,
		ProxyConfigID: &proxyConfig.ID,
		UserID:        1,
		TenantID:      1,
		Description:   "示例网站",
	}

	if err := websiteService.CreateWebsite(ctx, site); err != nil {
		return fmt.Errorf("failed to create website: %w", err)
	}
	fmt.Printf("网站创建成功，ID: %d\n", site.ID)

	// 4. 生成 Nginx 配置
	fmt.Println("\n=== 生成 Nginx 配置 ===")
	nginxConfig, err := proxyService.GenerateNginxConfig(ctx, site)
	if err != nil {
		return fmt.Errorf("failed to generate nginx config: %w", err)
	}
	fmt.Println("Nginx 配置生成成功：")
	fmt.Println(nginxConfig)

	// 5. 创建 DNS 记录
	fmt.Println("\n=== 创建 DNS 记录 ===")
	dnsRecord := &DNSRecord{
		Domain:   "example.com",
		Type:     DNSRecordA,
		Name:     "www",
		Value:    "192.168.1.1",
		TTL:      600,
		UserID:   1,
		TenantID: 1,
	}

	if err := dnsService.CreateDNSRecord(ctx, dnsRecord); err != nil {
		return fmt.Errorf("failed to create dns record: %w", err)
	}
	fmt.Printf("DNS 记录创建成功，ID: %d\n", dnsRecord.ID)

	// 6. AI 配置分析
	fmt.Println("\n=== AI 配置分析 ===")
	analysis, err := aiService.AnalyzeWebsiteConfig(ctx, site.ID)
	if err != nil {
		return fmt.Errorf("failed to analyze config: %w", err)
	}
	fmt.Printf("配置评分: %d/100\n", analysis.Score)
	fmt.Printf("发现问题: %d 个\n", len(analysis.Issues))
	fmt.Printf("优化建议: %d 条\n", len(analysis.Suggestions))

	// 显示问题详情
	if len(analysis.Issues) > 0 {
		fmt.Println("\n问题列表:")
		for i, issue := range analysis.Issues {
			fmt.Printf("%d. [%s] %s: %s\n", i+1, issue.Severity, issue.Title, issue.Description)
		}
	}

	// 显示优化建议
	if len(analysis.Suggestions) > 0 {
		fmt.Println("\n优化建议:")
		for i, suggestion := range analysis.Suggestions {
			fmt.Printf("%d. [%s] %s: %s\n", i+1, suggestion.Impact, suggestion.Title, suggestion.Description)
		}
	}

	// 7. 自动修复问题
	fmt.Println("\n=== 自动修复配置问题 ===")
	fixResult, err := aiService.AutoFixCommonIssues(ctx, site.ID)
	if err != nil {
		return fmt.Errorf("failed to auto fix: %w", err)
	}
	fmt.Printf("修复成功: %d 个问题\n", len(fixResult.FixedIssues))
	fmt.Printf("修复失败: %d 个问题\n", len(fixResult.FailedIssues))

	// 8. 性能分析
	fmt.Println("\n=== 性能分析 ===")
	perfAnalysis, err := aiService.AnalyzePerformance(ctx, site.ID)
	if err != nil {
		return fmt.Errorf("failed to analyze performance: %w", err)
	}
	fmt.Printf("平均响应时间: %.2f ms\n", perfAnalysis.ResponseTime)
	fmt.Printf("吞吐量: %.2f req/s\n", perfAnalysis.Throughput)
	fmt.Printf("错误率: %.2f%%\n", perfAnalysis.ErrorRate*100)

	return nil
}

// ExampleCertificateMonitoring 证书监控示例
func ExampleCertificateMonitoring(db *gorm.DB) {
	// 创建证书监控器，每小时检查一次
	monitor := NewCertMonitor(db, 1*time.Hour)

	ctx := context.Background()

	// 获取证书统计
	stats, err := monitor.GetCertificateStats(ctx)
	if err != nil {
		fmt.Printf("获取证书统计失败: %v\n", err)
		return
	}

	fmt.Println("=== 证书统计 ===")
	fmt.Printf("总证书数: %d\n", stats.Total)
	fmt.Printf("有效证书: %d\n", stats.Valid)
	fmt.Printf("已过期: %d\n", stats.Expired)
	fmt.Printf("即将过期: %d\n", stats.ExpiringSoon)
	fmt.Printf("错误状态: %d\n", stats.Error)

	// 获取即将过期的证书（30天内）
	expiringCerts, err := monitor.GetExpiringCertificates(ctx, 30)
	if err != nil {
		fmt.Printf("获取即将过期证书失败: %v\n", err)
		return
	}

	if len(expiringCerts) > 0 {
		fmt.Println("\n即将过期的证书:")
		for _, cert := range expiringCerts {
			daysLeft := int(time.Until(*cert.ExpiryDate).Hours() / 24)
			fmt.Printf("- %s (剩余 %d 天)\n", cert.Domain, daysLeft)
		}
	}

	// 启动后台监控（在实际应用中）
	// go monitor.Start(ctx)
}

// ExampleDNSManagement DNS 管理示例
func ExampleDNSManagement(db *gorm.DB) error {
	ctx := context.Background()
	dnsService := NewDNSService(db)

	// 创建多种类型的 DNS 记录
	records := []*DNSRecord{
		{
			Domain:   "example.com",
			Type:     DNSRecordA,
			Name:     "@",
			Value:    "192.168.1.1",
			TTL:      600,
			UserID:   1,
			TenantID: 1,
		},
		{
			Domain:   "example.com",
			Type:     DNSRecordCNAME,
			Name:     "www",
			Value:    "example.com",
			TTL:      600,
			UserID:   1,
			TenantID: 1,
		},
		{
			Domain:   "example.com",
			Type:     DNSRecordMX,
			Name:     "@",
			Value:    "mail.example.com",
			TTL:      600,
			Priority: 10,
			UserID:   1,
			TenantID: 1,
		},
		{
			Domain:   "example.com",
			Type:     DNSRecordTXT,
			Name:     "@",
			Value:    "v=spf1 include:_spf.example.com ~all",
			TTL:      600,
			UserID:   1,
			TenantID: 1,
		},
	}

	fmt.Println("=== 创建 DNS 记录 ===")
	for _, record := range records {
		if err := dnsService.CreateDNSRecord(ctx, record); err != nil {
			return fmt.Errorf("failed to create dns record: %w", err)
		}
		fmt.Printf("创建 %s 记录: %s.%s -> %s\n", record.Type, record.Name, record.Domain, record.Value)
	}

	// 列出所有记录
	fmt.Println("\n=== 列出 DNS 记录 ===")
	allRecords, err := dnsService.ListDNSRecords(ctx, "example.com", 1, 1)
	if err != nil {
		return fmt.Errorf("failed to list dns records: %w", err)
	}

	for _, record := range allRecords {
		fmt.Printf("%s %s.%s -> %s (TTL: %d)\n", 
			record.Type, record.Name, record.Domain, record.Value, record.TTL)
	}

	return nil
}

// ExampleLoadBalancing 负