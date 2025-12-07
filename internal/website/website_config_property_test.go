package website

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: enhanced-aiops-platform, Property 10: 网站配置自动化**
// **Validates: Requirements 4.1**
//
// Property 10: 网站配置自动化
// *For any* 新添加的网站，系统应该能自动配置反向代理和负载均衡
//
// 这个属性测试验证：
// 1. 为任何有效的网站配置生成有效的 Nginx 配置
// 2. 生成的配置包含必要的反向代理设置
// 3. 生成的配置包含负载均衡配置（当有多个后端时）
// 4. 生成的配置包含安全头和优化设置
// 5. 配置生成是确定性的（相同输入产生相同输出）

// genValidDomain 生成有效的域名
// 组合随机标识符和顶级域名（TLD）生成类似 "example.com" 的域名
func genValidDomain() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),                                    // 生成域名主体部分
		gen.OneConstOf("com", "net", "org", "io", "dev"),   // 随机选择顶级域名
	).Map(func(values []interface{}) string {
		name := values[0].(string)
		tld := values[1].(string)
		return fmt.Sprintf("%s.%s", name, tld)
	})
}

// genValidBackend 生成有效的后端地址
// 生成格式为 "http://hostname:port" 或 "https://hostname:port" 的后端服务地址
func genValidBackend() gopter.Gen {
	return gopter.CombineGens(
		gen.OneConstOf("http", "https"),  // 随机选择协议
		gen.Identifier(),                  // 生成主机名
		gen.IntRange(3000, 9000),         // 生成端口号（常用应用端口范围）
	).Map(func(values []interface{}) string {
		protocol := values[0].(string)
		host := values[1].(string)
		port := values[2].(int)
		return fmt.Sprintf("%s://%s:%d", protocol, host, port)
	})
}

// genValidLoadBalanceMethod 生成有效的负载均衡方法
func genValidLoadBalanceMethod() gopter.Gen {
	return gen.OneConstOf(
		LoadBalanceRoundRobin,
		LoadBalanceLeastConn,
		LoadBalanceIPHash,
		LoadBalanceWeighted,
	)
}

// genProxyConfig 生成有效的代理配置
// multiBackend: true 生成多后端配置（用于负载均衡测试），false 生成单后端配置
func genProxyConfig(multiBackend bool) gopter.Gen {
	if multiBackend {
		// 生成多个后端的配置（用于负载均衡场景）
		return gopter.CombineGens(
			gen.Identifier(),                                 // 配置名称
			genValidLoadBalanceMethod(),                      // 负载均衡方法
			gen.SliceOfN(3, genValidBackend()),              // 生成3个后端服务器
			gen.IntRange(30, 120),                           // 超时时间（秒）
			gen.IntRange(10485760, 52428800),                // 请求体大小限制（10MB - 50MB）
		).Map(func(values []interface{}) *ProxyConfig {
			name := values[0].(string)
			lbMethod := values[1].(LoadBalanceMethod)
			backends := values[2].([]string)
			timeout := values[3].(int)
			maxBodySize := int64(values[4].(int))
			
			// 将后端列表转换为 JSON 数组
			backendsJSON, _ := json.Marshal(backends)
			
			return &ProxyConfig{
				ID:                  1,
				Name:                name,
				ProxyType:           ProxyTypeReverse,
				Backend:             string(backendsJSON),
				LoadBalanceMethod:   lbMethod,
				HealthCheckEnabled:  true,
				HealthCheckPath:     "/health",
				HealthCheckInterval: 30,
				Timeout:             timeout,
				MaxBodySize:         maxBodySize,
				UserID:              1,
				TenantID:            1,
			}
		})
	}
	
	// 生成单个后端的配置（简单反向代理场景）
	return gopter.CombineGens(
		gen.Identifier(),                  // 配置名称
		genValidBackend(),                 // 单个后端地址
		gen.IntRange(30, 120),            // 超时时间（秒）
		gen.IntRange(10485760, 52428800), // 请求体大小限制
	).Map(func(values []interface{}) *ProxyConfig {
		name := values[0].(string)
		backend := values[1].(string)
		timeout := values[2].(int)
		maxBodySize := int64(values[3].(int))
		
		return &ProxyConfig{
			ID:                  1,
			Name:                name,
			ProxyType:           ProxyTypeReverse,
			Backend:             backend,
			LoadBalanceMethod:   LoadBalanceRoundRobin,
			HealthCheckEnabled:  true,
			HealthCheckPath:     "/health",
			HealthCheckInterval: 30,
			Timeout:             timeout,
			MaxBodySize:         maxBodySize,
			UserID:              1,
			TenantID:            1,
		}
	})
}

// genWebsite 生成有效的网站配置
// withSSL: 是否启用 SSL/HTTPS
// multiBackend: 是否使用多个后端服务器（负载均衡）
func genWebsite(withSSL bool, multiBackend bool) gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),              // 网站名称
		genValidDomain(),              // 域名
		genProxyConfig(multiBackend),  // 代理配置
	).Map(func(values []interface{}) *Website {
		name := values[0].(string)
		domain := values[1].(string)
		proxyConfig := values[2].(*ProxyConfig)
		
		website := &Website{
			ID:            1,
			Name:          name,
			Domain:        domain,
			Status:        StatusActive,
			SSLEnabled:    withSSL,
			ProxyConfig:   proxyConfig,
			ProxyConfigID: &proxyConfig.ID,
			UserID:        1,
			TenantID:      1,
		}
		
		// 如果启用 SSL，添加证书信息
		if withSSL {
			certID := uint(1)
			website.SSLCertID = &certID
			website.SSLCert = &SSLCert{
				ID:       1,
				Domain:   domain,
				Provider: SSLProviderLetsEncrypt,
				Status:   SSLStatusValid,
				CertPath: fmt.Sprintf("/etc/ssl/certs/%s.crt", domain),
				KeyPath:  fmt.Sprintf("/etc/ssl/private/%s.key", domain),
				UserID:   1,
				TenantID: 1,
			}
		}
		
		return website
	})
}

// TestProperty10_ConfigGeneration_BasicProxy 测试基本反向代理配置生成
// 验证系统能为任意有效的网站配置生成正确的 Nginx 反向代理配置
func TestProperty10_ConfigGeneration_BasicProxy(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	// Property 1: 为任何有效的网站配置生成有效的 Nginx 配置
	// 验证生成的配置包含必要的 server 块、域名和监听端口
	properties.Property("生成有效的Nginx配置", prop.ForAll(
		func(website *Website) bool {
			generator := NewNginxConfigGenerator(website)
			config, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 验证配置不为空
			if config == "" {
				t.Logf("生成的配置为空")
				return false
			}
			
			// 验证配置包含必要的 server 块
			if !strings.Contains(config, "server {") {
				t.Logf("配置缺少 server 块")
				return false
			}
			
			// 验证配置包含域名
			if !strings.Contains(config, website.Domain) {
				t.Logf("配置缺少域名: %s", website.Domain)
				return false
			}
			
			// 验证配置包含监听端口
			if !strings.Contains(config, "listen 80") {
				t.Logf("配置缺少 HTTP 监听端口")
				return false
			}
			
			return true
		},
		genWebsite(false, false), // 不启用 SSL，单个后端
	))
	
	// Property 2: 生成的配置包含反向代理设置
	// 验证包含 proxy_pass 指令和必要的代理头（Host, X-Real-IP 等）
	properties.Property("包含反向代理设置", prop.ForAll(
		func(website *Website) bool {
			generator := NewNginxConfigGenerator(website)
			config, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 验证包含 proxy_pass 指令
			if !strings.Contains(config, "proxy_pass") {
				t.Logf("配置缺少 proxy_pass 指令")
				return false
			}
			
			// 验证包含必要的代理头
			requiredHeaders := []string{
				"proxy_set_header Host",
				"proxy_set_header X-Real-IP",
				"proxy_set_header X-Forwarded-For",
				"proxy_set_header X-Forwarded-Proto",
			}
			
			for _, header := range requiredHeaders {
				if !strings.Contains(config, header) {
					t.Logf("配置缺少必要的代理头: %s", header)
					return false
				}
			}
			
			// 验证包含超时配置
			if !strings.Contains(config, "proxy_connect_timeout") {
				t.Logf("配置缺少超时设置")
				return false
			}
			
			return true
		},
		genWebsite(false, false),
	))
	
	// Property 3: 生成的配置包含安全头
	// 验证包含 X-Frame-Options、X-Content-Type-Options 等安全响应头
	properties.Property("包含安全头", prop.ForAll(
		func(website *Website) bool {
			generator := NewNginxConfigGenerator(website)
			config, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 验证包含安全头
			securityHeaders := []string{
				"X-Frame-Options",
				"X-Content-Type-Options",
				"X-XSS-Protection",
			}
			
			for _, header := range securityHeaders {
				if !strings.Contains(config, header) {
					t.Logf("配置缺少安全头: %s", header)
					return false
				}
			}
			
			return true
		},
		genWebsite(false, false),
	))
	
	// Property 4: 生成的配置包含 Gzip 压缩设置
	// 验证启用了 Gzip 压缩以优化传输性能
	properties.Property("包含Gzip压缩设置", prop.ForAll(
		func(website *Website) bool {
			generator := NewNginxConfigGenerator(website)
			config, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 验证包含 Gzip 配置
			if !strings.Contains(config, "gzip on") {
				t.Logf("配置缺少 Gzip 压缩设置")
				return false
			}
			
			return true
		},
		genWebsite(false, false),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty10_ConfigGeneration_LoadBalancing 测试负载均衡配置生成
// 验证多后端场景下正确生成 upstream 块和负载均衡配置
func TestProperty10_ConfigGeneration_LoadBalancing(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	// Property 5: 多个后端时生成 upstream 配置
	// 验证包含 upstream 块和多个 server 指令
	properties.Property("多后端生成upstream配置", prop.ForAll(
		func(website *Website) bool {
			generator := NewNginxConfigGenerator(website)
			config, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 验证包含 upstream 块
			if !strings.Contains(config, "upstream") {
				t.Logf("多后端配置缺少 upstream 块")
				return false
			}
			
			// 验证 upstream 名称包含域名
			upstreamName := fmt.Sprintf("backend_%s", sanitizeName(website.Domain))
			if !strings.Contains(config, upstreamName) {
				t.Logf("配置缺少正确的 upstream 名称: %s", upstreamName)
				return false
			}
			
			// 验证包含多个 server 指令
			serverCount := strings.Count(config, "server ")
			if serverCount < 3 { // upstream 中至少有3个后端 server
				t.Logf("upstream 中的 server 数量不足: %d", serverCount)
				return false
			}
			
			return true
		},
		genWebsite(false, true), // 多个后端
	))
	
	// Property 6: 负载均衡方法正确应用
	// 验证 least_conn、ip_hash、weighted 等负载均衡算法正确配置
	properties.Property("负载均衡方法正确应用", prop.ForAll(
		func(website *Website) bool {
			generator := NewNginxConfigGenerator(website)
			config, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 根据负载均衡方法验证配置
			switch website.ProxyConfig.LoadBalanceMethod {
			case LoadBalanceLeastConn:
				if !strings.Contains(config, "least_conn") {
					t.Logf("配置缺少 least_conn 指令")
					return false
				}
			case LoadBalanceIPHash:
				if !strings.Contains(config, "ip_hash") {
					t.Logf("配置缺少 ip_hash 指令")
					return false
				}
			case LoadBalanceWeighted:
				if !strings.Contains(config, "weight=") {
					t.Logf("加权轮询配置缺少 weight 参数")
					return false
				}
			}
			
			return true
		},
		genWebsite(false, true),
	))
	
	// Property 7: 健康检查配置正确生成
	// 验证包含 max_fails 和 fail_timeout 等健康检查参数
	properties.Property("健康检查配置正确", prop.ForAll(
		func(website *Website) bool {
			generator := NewNginxConfigGenerator(website)
			config, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 如果启用了健康检查，验证相关配置
			if website.ProxyConfig.HealthCheckEnabled {
				if !strings.Contains(config, "max_fails") {
					t.Logf("健康检查配置缺少 max_fails 参数")
					return false
				}
				if !strings.Contains(config, "fail_timeout") {
					t.Logf("健康检查配置缺少 fail_timeout 参数")
					return false
				}
			}
			
			return true
		},
		genWebsite(false, true),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty10_ConfigGeneration_SSL 测试 SSL 配置生成
// 验证启用 SSL 时正确生成 HTTPS 监听、证书路径和安全配置
func TestProperty10_ConfigGeneration_SSL(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	// Property 8: 启用 SSL 时生成 HTTPS 配置
	// 验证包含 443 端口监听、证书路径、SSL 协议和 HSTS 头
	properties.Property("SSL启用时生成HTTPS配置", prop.ForAll(
		func(website *Website) bool {
			generator := NewNginxConfigGenerator(website)
			config, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 验证包含 HTTPS 监听端口
			if !strings.Contains(config, "listen 443 ssl") {
				t.Logf("SSL 配置缺少 HTTPS 监听端口")
				return false
			}
			
			// 验证包含证书路径
			if !strings.Contains(config, "ssl_certificate") {
				t.Logf("SSL 配置缺少证书路径")
				return false
			}
			
			if !strings.Contains(config, "ssl_certificate_key") {
				t.Logf("SSL 配置缺少私钥路径")
				return false
			}
			
			// 验证包含 SSL 协议配置
			if !strings.Contains(config, "ssl_protocols") {
				t.Logf("SSL 配置缺少协议设置")
				return false
			}
			
			// 验证包含 HSTS 头
			if !strings.Contains(config, "Strict-Transport-Security") {
				t.Logf("SSL 配置缺少 HSTS 头")
				return false
			}
			
			return true
		},
		genWebsite(true, false), // 启用 SSL
	))
	
	// Property 9: 启用 SSL 时生成 HTTP 到 HTTPS 重定向
	// 验证包含 301 重定向和两个 server 块（HTTP 重定向 + HTTPS 服务）
	properties.Property("SSL启用时生成HTTP重定向", prop.ForAll(
		func(website *Website) bool {
			generator := NewNginxConfigGenerator(website)
			config, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 验证包含重定向配置
			if !strings.Contains(config, "return 301 https://") {
				t.Logf("SSL 配置缺少 HTTP 到 HTTPS 重定向")
				return false
			}
			
			// 验证有两个 server 块（一个用于 HTTPS，一个用于重定向）
			serverCount := strings.Count(config, "server {")
			if serverCount < 2 {
				t.Logf("SSL 配置应该包含至少2个 server 块，实际: %d", serverCount)
				return false
			}
			
			return true
		},
		genWebsite(true, false),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty10_ConfigGeneration_Deterministic 测试配置生成的确定性
// 验证配置生成是幂等的，相同输入总是产生相同输出
func TestProperty10_ConfigGeneration_Deterministic(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	// Property 10: 相同输入产生相同输出（确定性）
	// 多次生成配置应该得到完全一致的结果
	properties.Property("配置生成确定性", prop.ForAll(
		func(website *Website) bool {
			generator1 := NewNginxConfigGenerator(website)
			config1, err1 := generator1.Generate()
			
			if err1 != nil {
				t.Logf("第一次配置生成失败: %v", err1)
				return false
			}
			
			// 使用相同的输入再次生成
			generator2 := NewNginxConfigGenerator(website)
			config2, err2 := generator2.Generate()
			
			if err2 != nil {
				t.Logf("第二次配置生成失败: %v", err2)
				return false
			}
			
			// 两次生成的配置应该完全相同
			if config1 != config2 {
				t.Logf("两次生成的配置不一致")
				t.Logf("第一次:\n%s", config1)
				t.Logf("第二次:\n%s", config2)
				return false
			}
			
			return true
		},
		genWebsite(false, false),
	))
	
	// Property 11: 配置生成不应该修改输入对象
	// 验证生成过程是纯函数，不会产生副作用
	properties.Property("配置生成不修改输入", prop.ForAll(
		func(website *Website) bool {
			// 保存原始值
			originalDomain := website.Domain
			originalName := website.Name
			originalSSLEnabled := website.SSLEnabled
			
			generator := NewNginxConfigGenerator(website)
			_, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 验证输入对象未被修改
			if website.Domain != originalDomain {
				t.Logf("域名被修改: %s -> %s", originalDomain, website.Domain)
				return false
			}
			
			if website.Name != originalName {
				t.Logf("名称被修改: %s -> %s", originalName, website.Name)
				return false
			}
			
			if website.SSLEnabled != originalSSLEnabled {
				t.Logf("SSL 状态被修改: %v -> %v", originalSSLEnabled, website.SSLEnabled)
				return false
			}
			
			return true
		},
		genWebsite(false, false),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty10_ConfigGeneration_CustomConfig 测试自定义配置集成
// 验证用户自定义的 Nginx 指令能正确插入到生成的配置中
func TestProperty10_ConfigGeneration_CustomConfig(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	// Property 12: 自定义配置正确集成到生成的配置中
	// 验证自定义指令被包含且位于正确的 server 块内
	properties.Property("自定义配置正确集成", prop.ForAll(
		func(website *Website, customDirective string) bool {
			// 添加自定义配置
			website.ProxyConfig.CustomConfig = customDirective
			
			generator := NewNginxConfigGenerator(website)
			config, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 验证自定义配置被包含
			if !strings.Contains(config, customDirective) {
				t.Logf("生成的配置缺少自定义指令: %s", customDirective)
				return false
			}
			
			// 验证自定义配置在 server 块内
			serverBlockStart := strings.Index(config, "server {")
			serverBlockEnd := strings.LastIndex(config, "}")
			customConfigPos := strings.Index(config, customDirective)
			
			if customConfigPos < serverBlockStart || customConfigPos > serverBlockEnd {
				t.Logf("自定义配置不在 server 块内")
				return false
			}
			
			return true
		},
		genWebsite(false, false),
		gen.OneConstOf(
			"client_body_timeout 30s;",
			"proxy_cache_valid 200 1h;",
			"add_header X-Custom-Header \"value\";",
		),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty10_ConfigGeneration_Timeout 测试超时配置
// 验证代理超时和请求体大小限制等参数正确应用到配置中
func TestProperty10_ConfigGeneration_Timeout(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	// Property 13: 超时配置正确应用
	// 验证 proxy_connect_timeout、proxy_send_timeout、proxy_read_timeout 等参数
	properties.Property("超时配置正确应用", prop.ForAll(
		func(website *Website) bool {
			generator := NewNginxConfigGenerator(website)
			config, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 验证超时值在配置中
			timeoutStr := fmt.Sprintf("%ds", website.ProxyConfig.Timeout)
			if !strings.Contains(config, timeoutStr) {
				t.Logf("配置缺少超时设置: %s", timeoutStr)
				return false
			}
			
			// 验证包含所有超时类型
			timeoutTypes := []string{
				"proxy_connect_timeout",
				"proxy_send_timeout",
				"proxy_read_timeout",
			}
			
			for _, timeoutType := range timeoutTypes {
				if !strings.Contains(config, timeoutType) {
					t.Logf("配置缺少超时类型: %s", timeoutType)
					return false
				}
			}
			
			return true
		},
		genWebsite(false, false),
	))
	
	// Property 14: 请求体大小限制正确应用
	// 验证 client_max_body_size 指令正确设置，防止过大请求
	properties.Property("请求体大小限制正确", prop.ForAll(
		func(website *Website) bool {
			generator := NewNginxConfigGenerator(website)
			config, err := generator.Generate()
			
			if err != nil {
				t.Logf("配置生成失败: %v", err)
				return false
			}
			
			// 验证包含 client_max_body_size 指令
			if !strings.Contains(config, "client_max_body_size") {
				t.Logf("配置缺少请求体大小限制")
				return false
			}
			
			// 验证大小值合理（转换为 MB）
			maxBodySizeMB := website.ProxyConfig.MaxBodySize / (1024 * 1024)
			sizeStr := fmt.Sprintf("%dm", maxBodySizeMB)
			if !strings.Contains(config, sizeStr) {
				t.Logf("配置中的大小限制不正确，期望: %s", sizeStr)
				return false
			}
			
			return true
		},
		genWebsite(false, false),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
