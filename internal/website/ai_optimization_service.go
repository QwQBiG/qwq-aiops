package website

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// aiOptimizationService AI 优化服务实现
type aiOptimizationService struct {
	db             *gorm.DB
	websiteService WebsiteService
	proxyService   ProxyService
}

// NewAIOptimizationService 创建 AI 优化服务实例
func NewAIOptimizationService(db *gorm.DB, websiteService WebsiteService, proxyService ProxyService) AIOptimizationService {
	return &aiOptimizationService{
		db:             db,
		websiteService: websiteService,
		proxyService:   proxyService,
	}
}

// AnalyzeWebsiteConfig 分析网站配置
func (s *aiOptimizationService) AnalyzeWebsiteConfig(ctx context.Context, websiteID uint) (*ConfigAnalysis, error) {
	website, err := s.websiteService.GetWebsite(ctx, websiteID)
	if err != nil {
		return nil, err
	}

	// 检测配置问题
	issues, err := s.detectIssuesForWebsite(ctx, website)
	if err != nil {
		return nil, err
	}

	// 生成优化建议
	suggestions, err := s.generateSuggestionsForWebsite(ctx, website)
	if err != nil {
		return nil, err
	}

	// 计算配置评分
	score := s.calculateConfigScore(issues)

	return &ConfigAnalysis{
		WebsiteID:   websiteID,
		Score:       score,
		Issues:      issues,
		Suggestions: suggestions,
		AnalyzedAt:  time.Now().Format(time.RFC3339),
	}, nil
}

// DetectConfigIssues 检测配置问题
func (s *aiOptimizationService) DetectConfigIssues(ctx context.Context, config string) ([]*ConfigIssue, error) {
	var issues []*ConfigIssue

	// 检查常见的配置问题
	
	// 1. 检查是否缺少安全头
	if !strings.Contains(config, "X-Frame-Options") {
		issues = append(issues, &ConfigIssue{
			Severity:    "warning",
			Category:    "security",
			Title:       "缺少 X-Frame-Options 安全头",
			Description: "建议添加 X-Frame-Options 头以防止点击劫持攻击",
			Location:    "server block",
			CanAutoFix:  true,
		})
	}

	if !strings.Contains(config, "X-Content-Type-Options") {
		issues = append(issues, &ConfigIssue{
			Severity:    "warning",
			Category:    "security",
			Title:       "缺少 X-Content-Type-Options 安全头",
			Description: "建议添加 X-Content-Type-Options: nosniff 以防止 MIME 类型嗅探",
			Location:    "server block",
			CanAutoFix:  true,
		})
	}

	// 2. 检查 SSL 配置
	if strings.Contains(config, "ssl_protocols") {
		if strings.Contains(config, "TLSv1 ") || strings.Contains(config, "TLSv1.1") {
			issues = append(issues, &ConfigIssue{
				Severity:    "critical",
				Category:    "security",
				Title:       "使用了不安全的 TLS 协议版本",
				Description: "TLSv1 和 TLSv1.1 已被弃用，建议只使用 TLSv1.2 和 TLSv1.3",
				Location:    "ssl_protocols directive",
				CanAutoFix:  true,
			})
		}
	}

	// 3. 检查缓存配置
	if !strings.Contains(config, "proxy_cache") && strings.Contains(config, "proxy_pass") {
		issues = append(issues, &ConfigIssue{
			Severity:    "info",
			Category:    "performance",
			Title:       "未配置代理缓存",
			Description: "启用代理缓存可以显著提高性能",
			Location:    "location block",
			CanAutoFix:  false,
		})
	}

	// 4. 检查 Gzip 压缩
	if !strings.Contains(config, "gzip on") {
		issues = append(issues, &ConfigIssue{
			Severity:    "warning",
			Category:    "performance",
			Title:       "未启用 Gzip 压缩",
			Description: "启用 Gzip 压缩可以减少传输数据量，提高加载速度",
			Location:    "server block",
			CanAutoFix:  true,
		})
	}

	return issues, nil
}

// GenerateOptimizationSuggestions 生成优化建议
func (s *aiOptimizationService) GenerateOptimizationSuggestions(ctx context.Context, websiteID uint) ([]*OptimizationSuggestion, error) {
	website, err := s.websiteService.GetWebsite(ctx, websiteID)
	if err != nil {
		return nil, err
	}

	return s.generateSuggestionsForWebsite(ctx, website)
}

// AutoFixCommonIssues 自动修复常见问题
func (s *aiOptimizationService) AutoFixCommonIssues(ctx context.Context, websiteID uint) (*FixResult, error) {
	website, err := s.websiteService.GetWebsite(ctx, websiteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get website: %w", err)
	}

	if website.ProxyConfig == nil {
		return &FixResult{
			Success: false,
			Message: "no proxy config found",
		}, nil
	}

	// 生成新的配置
	config, err := s.proxyService.GenerateNginxConfig(ctx, website)
	if err != nil {
		return &FixResult{
			Success: false,
			Message: fmt.Sprintf("failed to generate config: %v", err),
		}, nil
	}

	// 检测问题
	issues, err := s.DetectConfigIssues(ctx, config)
	if err != nil {
		return &FixResult{
			Success: false,
			Message: fmt.Sprintf("failed to detect issues: %v", err),
		}, nil
	}

	var fixedIssues []string
	var failedIssues []string

	// 修复可以自动修复的问题
	for _, issue := range issues {
		if issue.CanAutoFix {
			// 应用修复
			config = s.applyFix(config, issue)
			fixedIssues = append(fixedIssues, issue.Title)
		} else {
			failedIssues = append(failedIssues, issue.Title)
		}
	}

	// 更新配置
	if len(fixedIssues) > 0 {
		website.ProxyConfig.CustomConfig = config
		if err := s.proxyService.UpdateProxyConfig(ctx, website.ProxyConfig); err != nil {
			return &FixResult{
				Success:      false,
				FixedIssues:  fixedIssues,
				FailedIssues: failedIssues,
				Message:      fmt.Sprintf("failed to update config: %v", err),
			}, nil
		}
	}

	return &FixResult{
		Success:      true,
		FixedIssues:  fixedIssues,
		FailedIssues: failedIssues,
		Message:      fmt.Sprintf("successfully fixed %d issues", len(fixedIssues)),
	}, nil
}

// AnalyzePerformance 分析性能
func (s *aiOptimizationService) AnalyzePerformance(ctx context.Context, websiteID uint) (*PerformanceAnalysis, error) {
	// TODO: 实现实际的性能分析逻辑
	// 这里需要收集实际的性能指标
	
	return &PerformanceAnalysis{
		WebsiteID:    websiteID,
		ResponseTime: 150.5,
		Throughput:   1000.0,
		ErrorRate:    0.01,
		Bottlenecks: []string{
			"数据库查询较慢",
			"静态资源未启用缓存",
		},
		Recommendations: []string{
			"启用 Redis 缓存",
			"优化数据库索引",
			"启用 CDN 加速",
			"压缩静态资源",
		},
		Metrics: map[string]interface{}{
			"avg_response_time": 150.5,
			"p95_response_time": 300.0,
			"p99_response_time": 500.0,
			"requests_per_sec":  1000.0,
		},
	}, nil
}

// 辅助方法

// detectIssuesForWebsite 检测网站的配置问题
func (s *aiOptimizationService) detectIssuesForWebsite(ctx context.Context, website *Website) ([]*ConfigIssue, error) {
	var issues []*ConfigIssue

	// 检查 SSL 配置
	if !website.SSLEnabled {
		issues = append(issues, &ConfigIssue{
			Severity:    "warning",
			Category:    "security",
			Title:       "未启用 HTTPS",
			Description: "建议启用 HTTPS 以保护用户数据安全",
			Location:    "website config",
			CanAutoFix:  false,
		})
	}

	// 检查代理配置
	if website.ProxyConfig != nil {
		config, err := s.proxyService.GenerateNginxConfig(ctx, website)
		if err == nil {
			configIssues, _ := s.DetectConfigIssues(ctx, config)
			issues = append(issues, configIssues...)
		}
	}

	return issues, nil
}

// generateSuggestionsForWebsite 为网站生成优化建议
func (s *aiOptimizationService) generateSuggestionsForWebsite(ctx context.Context, website *Website) ([]*OptimizationSuggestion, error) {
	var suggestions []*OptimizationSuggestion

	// SSL 建议
	if !website.SSLEnabled {
		suggestions = append(suggestions, &OptimizationSuggestion{
			Category:    "security",
			Title:       "启用 HTTPS",
			Description: "使用 Let's Encrypt 免费证书为网站启用 HTTPS",
			Impact:      "high",
			Effort:      "easy",
			Action:      "申请并配置 SSL 证书",
		})
	}

	// 性能建议
	if website.ProxyConfig != nil {
		if !website.ProxyConfig.HealthCheckEnabled {
			suggestions = append(suggestions, &OptimizationSuggestion{
				Category:    "performance",
				Title:       "启用健康检查",
				Description: "启用后端健康检查可以提高服务可用性",
				Impact:      "medium",
				Effort:      "easy",
				Action:      "在代理配置中启用健康检查",
			})
		}

		suggestions = append(suggestions, &OptimizationSuggestion{
			Category:    "performance",
			Title:       "启用 HTTP/2",
			Description: "HTTP/2 可以显著提高页面加载速度",
			Impact:      "high",
			Effort:      "easy",
			Action:      "在 Nginx 配置中添加 http2 参数",
		})
	}

	// SEO 建议
	suggestions = append(suggestions, &OptimizationSuggestion{
		Category:    "seo",
		Title:       "配置 robots.txt",
		Description: "添加 robots.txt 文件以控制搜索引擎爬虫",
		Impact:      "medium",
		Effort:      "easy",
		Action:      "创建并配置 robots.txt 文件",
	})

	return suggestions, nil
}

// calculateConfigScore 计算配置评分
func (s *aiOptimizationService) calculateConfigScore(issues []*ConfigIssue) int {
	score := 100

	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			score -= 20
		case "warning":
			score -= 10
		case "info":
			score -= 5
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

// applyFix 应用修复
func (s *aiOptimizationService) applyFix(config string, issue *ConfigIssue) string {
	switch issue.Title {
	case "缺少 X-Frame-Options 安全头":
		if !strings.Contains(config, "add_header X-Frame-Options") {
			// 在 server block 中添加
			config = strings.Replace(config, "server {", 
				"server {\n    add_header X-Frame-Options \"SAMEORIGIN\" always;", 1)
		}
	case "缺少 X-Content-Type-Options 安全头":
		if !strings.Contains(config, "add_header X-Content-Type-Options") {
			config = strings.Replace(config, "server {", 
				"server {\n    add_header X-Content-Type-Options \"nosniff\" always;", 1)
		}
	case "使用了不安全的 TLS 协议版本":
		config = strings.ReplaceAll(config, "ssl_protocols TLSv1 TLSv1.1 TLSv1.2 TLSv1.3", 
			"ssl_protocols TLSv1.2 TLSv1.3")
		config = strings.ReplaceAll(config, "ssl_protocols TLSv1.1 TLSv1.2 TLSv1.3", 
			"ssl_protocols TLSv1.2 TLSv1.3")
	case "未启用 Gzip 压缩":
		if !strings.Contains(config, "gzip on") {
			config = strings.Replace(config, "server {", 
				"server {\n    gzip on;\n    gzip_types text/plain text/css application/json application/javascript text/xml application/xml;", 1)
		}
	}

	return config
}
