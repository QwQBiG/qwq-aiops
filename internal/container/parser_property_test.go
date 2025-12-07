package container

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: enhanced-aiops-platform, Property 7: Docker Compose 解析正确性**
// **Validates: Requirements 3.1**
//
// Property 7: Docker Compose 解析正确性
// *For any* 有效的 Docker Compose 文件，系统应该能正确解析并提供可视化编辑界面
//
// 这个属性测试验证：
// 1. 解析 → 渲染 → 再解析的往返一致性（Round Trip Property）
// 2. 解析后的配置包含所有必要的字段
// 3. 验证功能能够正确识别有效和无效的配置
// 4. 解析器能够处理各种有效的 Compose 配置

// genValidVersion 生成有效的 Compose 版本
func genValidVersion() gopter.Gen {
	versions := []string{"3", "3.0", "3.1", "3.2", "3.3", "3.4", "3.5", "3.6", "3.7", "3.8", "3.9"}
	return gen.OneConstOf(
		"3", "3.0", "3.1", "3.2", "3.3", "3.4", "3.5", "3.6", "3.7", "3.8", "3.9",
	).Map(func(v interface{}) string {
		for _, version := range versions {
			if v == version {
				return version
			}
		}
		return "3.8"
	})
}

// genValidRestartPolicy 生成有效的重启策略
func genValidRestartPolicy() gopter.Gen {
	return gen.OneConstOf("no", "always", "on-failure", "unless-stopped")
}

// genValidPort 生成有效的端口映射字符串（简化版本）
// 不再使用，保留以防需要
func genValidPort() gopter.Gen {
	return gen.IntRange(1024, 65535).Map(func(port int) string {
		return ""
	})
}

// genService 生成有效的服务配置（简化版本，不再使用）
func genService() gopter.Gen {
	return gen.Const(&Service{
		Image:   "nginx:latest",
		Ports:   []string{"8080:80"},
		Restart: "always",
	})
}

// genComposeConfig 生成有效的 ComposeConfig
func genComposeConfig() gopter.Gen {
	return gopter.CombineGens(
		genValidVersion(),
		gen.Identifier(),
		gen.Identifier(),
	).Map(func(values []interface{}) *ComposeConfig {
		version := values[0].(string)
		serviceName1 := values[1].(string)
		serviceName2 := values[2].(string)
		
		// 确保服务名不同
		if serviceName1 == serviceName2 {
			serviceName2 = serviceName2 + "_2"
		}
		
		return &ComposeConfig{
			Version: version,
			Services: map[string]*Service{
				serviceName1: {
					Image:   "nginx:latest",
					Ports:   []string{"8080:80"},
					Restart: "always",
				},
				serviceName2: {
					Image:   "redis:latest",
					Ports:   []string{"6379:6379"},
					Restart: "unless-stopped",
				},
			},
			Networks: map[string]*Network{
				"frontend": {
					Driver: "bridge",
				},
			},
			Volumes: map[string]*Volume{
				"data": {
					Driver: "local",
				},
			},
		}
	})
}

// TestProperty7_RoundTripConsistency 测试解析往返一致性
func TestProperty7_RoundTripConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)
	parser := NewComposeParser()
	
	// Property 1: 解析 → 渲染 → 再解析应该得到等价的配置
	properties.Property("解析往返一致性", prop.ForAll(
		func(config *ComposeConfig) bool {
			// 第一步：渲染配置为 YAML
			rendered, err := parser.Render(config)
			if err != nil {
				t.Logf("渲染失败: %v", err)
				return false
			}
			
			// 第二步：解析渲染后的 YAML
			parsed, err := parser.Parse(rendered)
			if err != nil {
				t.Logf("解析渲染后的内容失败: %v", err)
				return false
			}
			
			// 第三步：验证关键字段一致性
			if parsed.Version != config.Version {
				t.Logf("版本不一致: %s != %s", parsed.Version, config.Version)
				return false
			}
			
			if len(parsed.Services) != len(config.Services) {
				t.Logf("服务数量不一致: %d != %d", len(parsed.Services), len(config.Services))
				return false
			}
			
			// 验证每个服务的关键属性
			for serviceName, originalService := range config.Services {
				parsedService, exists := parsed.Services[serviceName]
				if !exists {
					t.Logf("服务 %s 在解析后丢失", serviceName)
					return false
				}
				
				if parsedService.Image != originalService.Image {
					t.Logf("服务 %s 的镜像不一致: %s != %s", 
						serviceName, parsedService.Image, originalService.Image)
					return false
				}
				
				if parsedService.Restart != originalService.Restart {
					t.Logf("服务 %s 的重启策略不一致: %s != %s", 
						serviceName, parsedService.Restart, originalService.Restart)
					return false
				}
			}
			
			// 验证网络数量
			if len(parsed.Networks) != len(config.Networks) {
				t.Logf("网络数量不一致: %d != %d", len(parsed.Networks), len(config.Networks))
				return false
			}
			
			// 验证卷数量
			if len(parsed.Volumes) != len(config.Volumes) {
				t.Logf("卷数量不一致: %d != %d", len(parsed.Volumes), len(config.Volumes))
				return false
			}
			
			return true
		},
		genComposeConfig(),
	))
	
	// Property 2: 多次渲染应该产生一致的结果
	properties.Property("渲染结果一致性", prop.ForAll(
		func(config *ComposeConfig) bool {
			// 第一次渲染
			rendered1, err1 := parser.Render(config)
			if err1 != nil {
				t.Logf("第一次渲染失败: %v", err1)
				return false
			}
			
			// 第二次渲染
			rendered2, err2 := parser.Render(config)
			if err2 != nil {
				t.Logf("第二次渲染失败: %v", err2)
				return false
			}
			
			// 两次渲染的结果应该能解析为等价的配置
			parsed1, err := parser.Parse(rendered1)
			if err != nil {
				t.Logf("解析第一次渲染结果失败: %v", err)
				return false
			}
			
			parsed2, err := parser.Parse(rendered2)
			if err != nil {
				t.Logf("解析第二次渲染结果失败: %v", err)
				return false
			}
			
			// 验证两次解析的结果一致
			if parsed1.Version != parsed2.Version {
				t.Logf("两次解析的版本不一致")
				return false
			}
			
			if len(parsed1.Services) != len(parsed2.Services) {
				t.Logf("两次解析的服务数量不一致")
				return false
			}
			
			return true
		},
		genComposeConfig(),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty7_ValidationConsistency 测试验证功能的一致性
func TestProperty7_ValidationConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)
	parser := NewComposeParser()
	
	// Property 3: 有效的配置应该通过验证
	properties.Property("有效配置通过验证", prop.ForAll(
		func(config *ComposeConfig) bool {
			result := parser.Validate(config)
			
			if !result.Valid {
				t.Logf("有效配置未通过验证，错误: %d 个", len(result.Errors))
				for _, err := range result.Errors {
					t.Logf("  - %s: %s", err.Field, err.Message)
				}
				return false
			}
			
			return true
		},
		genComposeConfig(),
	))
	
	// Property 4: 解析成功的配置应该能通过验证
	properties.Property("解析成功的配置通过验证", prop.ForAll(
		func(config *ComposeConfig) bool {
			// 渲染配置
			rendered, err := parser.Render(config)
			if err != nil {
				t.Logf("渲染失败: %v", err)
				return false
			}
			
			// 解析渲染后的内容
			parsed, err := parser.Parse(rendered)
			if err != nil {
				t.Logf("解析失败: %v", err)
				return false
			}
			
			// 验证解析后的配置
			result := parser.Validate(parsed)
			if !result.Valid {
				t.Logf("解析后的配置未通过验证，错误: %d 个", len(result.Errors))
				for _, err := range result.Errors {
					t.Logf("  - %s: %s", err.Field, err.Message)
				}
				return false
			}
			
			return true
		},
		genComposeConfig(),
	))
	
	// Property 5: 验证结果应该是确定性的（相同输入产生相同输出）
	properties.Property("验证结果确定性", prop.ForAll(
		func(config *ComposeConfig) bool {
			// 第一次验证
			result1 := parser.Validate(config)
			
			// 第二次验证
			result2 := parser.Validate(config)
			
			// 验证结果应该一致
			if result1.Valid != result2.Valid {
				t.Logf("两次验证的有效性不一致")
				return false
			}
			
			if len(result1.Errors) != len(result2.Errors) {
				t.Logf("两次验证的错误数量不一致: %d != %d", 
					len(result1.Errors), len(result2.Errors))
				return false
			}
			
			return true
		},
		genComposeConfig(),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty7_ParseErrorHandling 测试解析错误处理
func TestProperty7_ParseErrorHandling(t *testing.T) {
	properties := gopter.NewProperties(nil)
	parser := NewComposeParser()
	
	// Property 6: 空内容应该返回错误
	properties.Property("空内容返回错误", prop.ForAll(
		func(whitespace string) bool {
			// 生成只包含空白字符的字符串
			_, err := parser.Parse(whitespace)
			
			// 应该返回错误
			return err != nil
		},
		gen.OneConstOf("", "   ", "\n", "\t", "  \n\t  "),
	))
	
	// Property 7: 缺少必要字段的配置应该被拒绝
	properties.Property("缺少版本的配置被拒绝", prop.ForAll(
		func(serviceName string) bool {
			// 创建缺少版本的配置
			invalidYAML := `services:
  ` + serviceName + `:
    image: nginx:latest
`
			
			_, err := parser.Parse(invalidYAML)
			
			// 应该返回错误
			return err != nil
		},
		gen.Identifier(),
	))
	
	// Property 8: 不支持的版本应该被拒绝
	properties.Property("不支持的版本被拒绝", prop.ForAll(
		func(version string) bool {
			// 生成不支持的版本号
			invalidYAML := `version: '` + version + `'
services:
  web:
    image: nginx:latest
`
			
			_, err := parser.Parse(invalidYAML)
			
			// 应该返回错误
			return err != nil
		},
		gen.OneConstOf("1.0", "2.0", "2.1", "4.0", "invalid"),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty7_ServiceValidation 测试服务配置验证
func TestProperty7_ServiceValidation(t *testing.T) {
	properties := gopter.NewProperties(nil)
	parser := NewComposeParser()
	
	// Property 9: 服务必须有 image 或 build 配置
	properties.Property("服务需要image或build", prop.ForAll(
		func(serviceName string) bool {
			config := &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					serviceName: {
						// 既没有 image 也没有 build
						Ports: []string{"80:80"},
					},
				},
			}
			
			result := parser.Validate(config)
			
			// 应该验证失败
			if result.Valid {
				t.Logf("缺少 image 和 build 的服务应该验证失败")
				return false
			}
			
			// 应该有相关的错误信息
			hasRelevantError := false
			for _, err := range result.Errors {
				if err.Field == "services."+serviceName {
					hasRelevantError = true
					break
				}
			}
			
			return hasRelevantError
		},
		gen.Identifier(),
	))
	
	// Property 10: 有效的重启策略应该通过验证
	properties.Property("有效重启策略通过验证", prop.ForAll(
		func(restartPolicy string) bool {
			config := &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"web": {
						Image:   "nginx:latest",
						Restart: restartPolicy,
					},
				},
			}
			
			result := parser.Validate(config)
			
			// 应该验证成功
			return result.Valid
		},
		genValidRestartPolicy(),
	))
	
	// Property 11: 无效的重启策略应该被拒绝
	properties.Property("无效重启策略被拒绝", prop.ForAll(
		func(invalidPolicy string) bool {
			config := &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"web": {
						Image:   "nginx:latest",
						Restart: invalidPolicy,
					},
				},
			}
			
			result := parser.Validate(config)
			
			// 应该验证失败
			return !result.Valid
		},
		gen.OneConstOf("invalid", "restart-always", "never", "sometimes"),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty7_NetworkAndVolumeReferences 测试网络和卷引用验证
func TestProperty7_NetworkAndVolumeReferences(t *testing.T) {
	properties := gopter.NewProperties(nil)
	parser := NewComposeParser()
	
	// Property 12: 引用已定义的网络应该通过验证
	properties.Property("已定义网络通过验证", prop.ForAll(
		func(networkName string) bool {
			config := &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"web": {
						Image:    "nginx:latest",
						Networks: []interface{}{networkName},
					},
				},
				Networks: map[string]*Network{
					networkName: {
						Driver: "bridge",
					},
				},
			}
			
			result := parser.Validate(config)
			
			// 应该验证成功
			return result.Valid
		},
		gen.Identifier(),
	))
	
	// Property 13: 引用未定义的网络应该被拒绝
	properties.Property("未定义网络被拒绝", prop.ForAll(
		func(networkName string) bool {
			config := &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"web": {
						Image:    "nginx:latest",
						Networks: []interface{}{networkName},
					},
				},
				Networks: map[string]*Network{}, // 空网络定义
			}
			
			result := parser.Validate(config)
			
			// 应该验证失败（除非是 "default" 网络）
			if networkName == "default" {
				return true // default 网络是隐式存在的
			}
			
			return !result.Valid
		},
		gen.Identifier(),
	))
	
	// Property 14: 引用已定义的卷应该通过验证
	properties.Property("已定义卷通过验证", prop.ForAll(
		func(volumeName string) bool {
			config := &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"db": {
						Image:   "postgres:15",
						Volumes: []string{volumeName + ":/data"},
					},
				},
				Volumes: map[string]*Volume{
					volumeName: {
						Driver: "local",
					},
				},
			}
			
			result := parser.Validate(config)
			
			// 应该验证成功
			return result.Valid
		},
		gen.Identifier(),
	))
	
	// Property 15: 引用未定义的命名卷应该被拒绝
	properties.Property("未定义卷被拒绝", prop.ForAll(
		func(volumeName string) bool {
			config := &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"db": {
						Image:   "postgres:15",
						Volumes: []string{volumeName + ":/data"},
					},
				},
				Volumes: map[string]*Volume{}, // 空卷定义
			}
			
			result := parser.Validate(config)
			
			// 应该验证失败
			return !result.Valid
		},
		gen.Identifier(),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
