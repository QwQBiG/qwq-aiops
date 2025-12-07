package website

import (
	"encoding/json"
	"fmt"
	"strings"
)

// NginxConfigGenerator Nginx 配置生成器
type NginxConfigGenerator struct {
	website *Website
}

// NewNginxConfigGenerator 创建 Nginx 配置生成器
func NewNginxConfigGenerator(website *Website) *NginxConfigGenerator {
	return &NginxConfigGenerator{website: website}
}

// Generate 生成完整的 Nginx 配置
func (g *NginxConfigGenerator) Generate() (string, error) {
	if g.website.ProxyConfig == nil {
		return "", ErrProxyConfigNotFound
	}

	var builder strings.Builder

	// 生成 upstream 配置（如果有多个后端）
	upstreamConfig, err := g.generateUpstream()
	if err != nil {
		return "", err
	}
	if upstreamConfig != "" {
		builder.WriteString(upstreamConfig)
		builder.WriteString("\n")
	}

	// 生成 server 配置
	serverConfig, err := g.generateServer()
	if err != nil {
		return "", err
	}
	builder.WriteString(serverConfig)

	return builder.String(), nil
}

// generateUpstream 生成 upstream 配置
func (g *NginxConfigGenerator) generateUpstream() (string, error) {
	config := g.website.ProxyConfig
	
	// 尝试解析后端地址为 JSON 数组
	var backends []string
	if err := json.Unmarshal([]byte(config.Backend), &backends); err != nil {
		// 如果不是 JSON 数组，则作为单个后端处理
		return "", nil
	}

	if len(backends) <= 1 {
		return "", nil
	}

	var builder strings.Builder
	upstreamName := fmt.Sprintf("backend_%s", sanitizeName(g.website.Domain))

	builder.WriteString(fmt.Sprintf("upstream %s {\n", upstreamName))

	// 负载均衡方法
	switch config.LoadBalanceMethod {
	case LoadBalanceLeastConn:
		builder.WriteString("    least_conn;\n")
	case LoadBalanceIPHash:
		builder.WriteString("    ip_hash;\n")
	case LoadBalanceWeighted:
		// 加权轮询需要在每个 server 后面指定权重
		builder.WriteString("    # weighted round robin\n")
	default:
		// 默认轮询，不需要额外配置
	}

	// 添加后端服务器
	for i, backend := range backends {
		weight := ""
		if config.LoadBalanceMethod == LoadBalanceWeighted {
			// 简单示例：第一个服务器权重为 3，其他为 1
			if i == 0 {
				weight = " weight=3"
			}
		}

		// 健康检查参数
		healthCheck := ""
		if config.HealthCheckEnabled {
			healthCheck = fmt.Sprintf(" max_fails=3 fail_timeout=%ds", config.HealthCheckInterval)
		}

		builder.WriteString(fmt.Sprintf("    server %s%s%s;\n", backend, weight, healthCheck))
	}

	// keepalive 连接
	builder.WriteString("    keepalive 32;\n")

	builder.WriteString("}\n")

	return builder.String(), nil
}

// generateServer 生成 server 配置
func (g *NginxConfigGenerator) generateServer() (string, error) {
	var builder strings.Builder
	config := g.website.ProxyConfig

	builder.WriteString(fmt.Sprintf("# Configuration for %s\n", g.website.Name))
	builder.WriteString("server {\n")

	// HTTP 监听
	builder.WriteString("    listen 80;\n")
	builder.WriteString(fmt.Sprintf("    server_name %s", g.website.Domain))
	
	// 添加域名别名
	if g.website.Aliases != "" {
		var aliases []string
		if err := json.Unmarshal([]byte(g.website.Aliases), &aliases); err == nil {
			for _, alias := range aliases {
				builder.WriteString(fmt.Sprintf(" %s", alias))
			}
		}
	}
	builder.WriteString(";\n\n")

	// HTTPS 配置
	if g.website.SSLEnabled && g.website.SSLCert != nil {
		builder.WriteString(g.generateSSLConfig())
	}

	// 安全头
	builder.WriteString(g.generateSecurityHeaders())

	// Gzip 压缩
	builder.WriteString(g.generateGzipConfig())

	// 日志配置
	builder.WriteString(g.generateLogConfig())

	// Location 配置
	builder.WriteString(g.generateLocationConfig())

	// 自定义配置
	if config.CustomConfig != "" {
		builder.WriteString("\n    # Custom configuration\n")
		builder.WriteString(g.indentConfig(config.CustomConfig, 1))
		builder.WriteString("\n")
	}

	builder.WriteString("}\n")

	// 如果启用了 SSL，添加 HTTP 到 HTTPS 的重定向
	if g.website.SSLEnabled {
		builder.WriteString("\n")
		builder.WriteString(g.generateHTTPRedirect())
	}

	return builder.String(), nil
}

// generateSSLConfig 生成 SSL 配置
func (g *NginxConfigGenerator) generateSSLConfig() string {
	cert := g.website.SSLCert
	var builder strings.Builder

	builder.WriteString("    # HTTPS configuration\n")
	builder.WriteString("    listen 443 ssl http2;\n")
	builder.WriteString(fmt.Sprintf("    ssl_certificate %s;\n", cert.CertPath))
	builder.WriteString(fmt.Sprintf("    ssl_certificate_key %s;\n", cert.KeyPath))
	builder.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n")
	builder.WriteString("    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';\n")
	builder.WriteString("    ssl_prefer_server_ciphers on;\n")
	builder.WriteString("    ssl_session_cache shared:SSL:10m;\n")
	builder.WriteString("    ssl_session_timeout 10m;\n")
	builder.WriteString("    ssl_stapling on;\n")
	builder.WriteString("    ssl_stapling_verify on;\n\n")

	return builder.String()
}

// generateSecurityHeaders 生成安全头
func (g *NginxConfigGenerator) generateSecurityHeaders() string {
	var builder strings.Builder

	builder.WriteString("    # Security headers\n")
	builder.WriteString("    add_header X-Frame-Options \"SAMEORIGIN\" always;\n")
	builder.WriteString("    add_header X-Content-Type-Options \"nosniff\" always;\n")
	builder.WriteString("    add_header X-XSS-Protection \"1; mode=block\" always;\n")
	
	if g.website.SSLEnabled {
		builder.WriteString("    add_header Strict-Transport-Security \"max-age=31536000; includeSubDomains\" always;\n")
	}
	
	builder.WriteString("\n")

	return builder.String()
}

// generateGzipConfig 生成 Gzip 配置
func (g *NginxConfigGenerator) generateGzipConfig() string {
	var builder strings.Builder

	builder.WriteString("    # Gzip compression\n")
	builder.WriteString("    gzip on;\n")
	builder.WriteString("    gzip_vary on;\n")
	builder.WriteString("    gzip_proxied any;\n")
	builder.WriteString("    gzip_comp_level 6;\n")
	builder.WriteString("    gzip_types text/plain text/css text/xml text/javascript application/json application/javascript application/xml+rss application/rss+xml font/truetype font/opentype application/vnd.ms-fontobject image/svg+xml;\n\n")

	return builder.String()
}

// generateLogConfig 生成日志配置
func (g *NginxConfigGenerator) generateLogConfig() string {
	var builder strings.Builder
	logName := sanitizeName(g.website.Domain)

	builder.WriteString("    # Logging\n")
	builder.WriteString(fmt.Sprintf("    access_log /var/log/nginx/%s_access.log;\n", logName))
	builder.WriteString(fmt.Sprintf("    error_log /var/log/nginx/%s_error.log;\n\n", logName))

	return builder.String()
}

// generateLocationConfig 生成 location 配置
func (g *NginxConfigGenerator) generateLocationConfig() string {
	var builder strings.Builder
	config := g.website.ProxyConfig

	builder.WriteString("    location / {\n")

	// 确定代理目标
	proxyPass := config.Backend
	
	// 如果有多个后端，使用 upstream
	var backends []string
	if err := json.Unmarshal([]byte(config.Backend), &backends); err == nil && len(backends) > 1 {
		upstreamName := fmt.Sprintf("backend_%s", sanitizeName(g.website.Domain))
		proxyPass = fmt.Sprintf("http://%s", upstreamName)
	}

	builder.WriteString(fmt.Sprintf("        proxy_pass %s;\n", proxyPass))
	
	// 代理头
	builder.WriteString("        proxy_set_header Host $host;\n")
	builder.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
	builder.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	builder.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
	builder.WriteString("        proxy_set_header X-Forwarded-Host $host;\n")
	builder.WriteString("        proxy_set_header X-Forwarded-Port $server_port;\n\n")

	// 超时配置
	builder.WriteString(fmt.Sprintf("        proxy_connect_timeout %ds;\n", config.Timeout))
	builder.WriteString(fmt.Sprintf("        proxy_send_timeout %ds;\n", config.Timeout))
	builder.WriteString(fmt.Sprintf("        proxy_read_timeout %ds;\n\n", config.Timeout))

	// 缓冲配置
	builder.WriteString("        proxy_buffering on;\n")
	builder.WriteString("        proxy_buffer_size 4k;\n")
	builder.WriteString("        proxy_buffers 8 4k;\n")
	builder.WriteString("        proxy_busy_buffers_size 8k;\n\n")

	// 请求体大小限制
	maxBodySizeMB := config.MaxBodySize / (1024 * 1024)
	builder.WriteString(fmt.Sprintf("        client_max_body_size %dm;\n", maxBodySizeMB))

	builder.WriteString("    }\n")

	return builder.String()
}

// generateHTTPRedirect 生成 HTTP 到 HTTPS 的重定向
func (g *NginxConfigGenerator) generateHTTPRedirect() string {
	var builder strings.Builder

	builder.WriteString("# HTTP to HTTPS redirect\n")
	builder.WriteString("server {\n")
	builder.WriteString("    listen 80;\n")
	builder.WriteString(fmt.Sprintf("    server_name %s", g.website.Domain))
	
	if g.website.Aliases != "" {
		var aliases []string
		if err := json.Unmarshal([]byte(g.website.Aliases), &aliases); err == nil {
			for _, alias := range aliases {
				builder.WriteString(fmt.Sprintf(" %s", alias))
			}
		}
	}
	builder.WriteString(";\n")
	builder.WriteString("    return 301 https://$server_name$request_uri;\n")
	builder.WriteString("}\n")

	return builder.String()
}

// indentConfig 缩进配置文本
func (g *NginxConfigGenerator) indentConfig(config string, level int) string {
	indent := strings.Repeat("    ", level)
	lines := strings.Split(config, "\n")
	
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, indent+line)
		}
	}
	
	return strings.Join(result, "\n")
}

// sanitizeName 清理名称，使其适合用作文件名或标识符
func sanitizeName(name string) string {
	// 替换特殊字符为下划线
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ":", "_")
	return name
}
