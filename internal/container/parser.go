package container

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeParser Docker Compose 解析器
type ComposeParser struct {
	// 支持的 Compose 文件版本
	supportedVersions []string
}

// NewComposeParser 创建 Compose 解析器实例
func NewComposeParser() *ComposeParser {
	return &ComposeParser{
		supportedVersions: []string{"3", "3.0", "3.1", "3.2", "3.3", "3.4", "3.5", "3.6", "3.7", "3.8", "3.9"},
	}
}

// Parse 解析 Docker Compose 文件内容
func (p *ComposeParser) Parse(content string) (*ComposeConfig, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("compose content is empty")
	}

	var config ComposeConfig
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// 验证版本
	if config.Version == "" {
		return nil, fmt.Errorf("compose version is required")
	}

	if !p.isVersionSupported(config.Version) {
		return nil, fmt.Errorf("unsupported compose version: %s", config.Version)
	}

	// 验证服务定义
	if len(config.Services) == 0 {
		return nil, fmt.Errorf("at least one service is required")
	}

	return &config, nil
}

// Render 将 ComposeConfig 渲染为 YAML 字符串
func (p *ComposeParser) Render(config *ComposeConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("config is nil")
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to render compose config: %w", err)
	}

	return string(data), nil
}

// Validate 验证 Compose 配置
func (p *ComposeParser) Validate(config *ComposeConfig) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []*ValidationError{},
	}

	// 验证版本
	if config.Version == "" {
		result.Valid = false
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "version",
			Message: "version is required",
		})
	} else if !p.isVersionSupported(config.Version) {
		result.Valid = false
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "version",
			Message: fmt.Sprintf("unsupported version: %s", config.Version),
		})
	}

	// 验证服务
	if len(config.Services) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "services",
			Message: "at least one service is required",
		})
	}

	// 验证每个服务
	for serviceName, service := range config.Services {
		errors := p.validateService(serviceName, service)
		result.Errors = append(result.Errors, errors...)
		if len(errors) > 0 {
			result.Valid = false
		}
	}

	// 验证网络引用
	for serviceName, service := range config.Services {
		errors := p.validateNetworkReferences(serviceName, service, config.Networks)
		result.Errors = append(result.Errors, errors...)
		if len(errors) > 0 {
			result.Valid = false
		}
	}

	// 验证卷引用
	for serviceName, service := range config.Services {
		errors := p.validateVolumeReferences(serviceName, service, config.Volumes)
		result.Errors = append(result.Errors, errors...)
		if len(errors) > 0 {
			result.Valid = false
		}
	}

	return result
}

// validateService 验证单个服务配置
func (p *ComposeParser) validateService(serviceName string, service *Service) []*ValidationError {
	errors := []*ValidationError{}

	// 验证镜像或构建配置
	if service.Image == "" && service.Build == nil {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("services.%s", serviceName),
			Message: "either 'image' or 'build' must be specified",
		})
	}

	// 验证端口格式
	for _, port := range service.Ports {
		if !p.isValidPortMapping(port) {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("services.%s.ports", serviceName),
				Message: fmt.Sprintf("invalid port mapping: %s", port),
			})
		}
	}

	// 验证重启策略
	if service.Restart != "" {
		validRestartPolicies := []string{"no", "always", "on-failure", "unless-stopped"}
		if !contains(validRestartPolicies, service.Restart) {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("services.%s.restart", serviceName),
				Message: fmt.Sprintf("invalid restart policy: %s", service.Restart),
			})
		}
	}

	// 验证健康检查
	if service.HealthCheck != nil {
		if service.HealthCheck.Test == nil {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("services.%s.healthcheck", serviceName),
				Message: "healthcheck test is required",
			})
		}
	}

	return errors
}

// validateNetworkReferences 验证网络引用
func (p *ComposeParser) validateNetworkReferences(serviceName string, service *Service, networks map[string]*Network) []*ValidationError {
	errors := []*ValidationError{}

	if service.Networks == nil {
		return errors
	}

	// 处理数组形式的网络
	if networkList, ok := service.Networks.([]interface{}); ok {
		for _, network := range networkList {
			networkName := fmt.Sprintf("%v", network)
			if _, exists := networks[networkName]; !exists && networkName != "default" {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("services.%s.networks", serviceName),
					Message: fmt.Sprintf("network '%s' is not defined", networkName),
				})
			}
		}
	}

	// 处理映射形式的网络
	if networkMap, ok := service.Networks.(map[string]interface{}); ok {
		for networkName := range networkMap {
			if _, exists := networks[networkName]; !exists && networkName != "default" {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("services.%s.networks", serviceName),
					Message: fmt.Sprintf("network '%s' is not defined", networkName),
				})
			}
		}
	}

	return errors
}

// validateVolumeReferences 验证卷引用
func (p *ComposeParser) validateVolumeReferences(serviceName string, service *Service, volumes map[string]*Volume) []*ValidationError {
	errors := []*ValidationError{}

	for _, volumeMount := range service.Volumes {
		// 解析卷挂载格式：source:target[:mode]
		parts := strings.Split(volumeMount, ":")
		if len(parts) < 2 {
			continue // 可能是绑定挂载，跳过
		}

		source := parts[0]
		
		// 如果是命名卷（不是路径），检查是否已定义
		if !strings.HasPrefix(source, "/") && !strings.HasPrefix(source, "./") && !strings.HasPrefix(source, "../") {
			if _, exists := volumes[source]; !exists {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("services.%s.volumes", serviceName),
					Message: fmt.Sprintf("volume '%s' is not defined", source),
				})
			}
		}
	}

	return errors
}

// isValidPortMapping 验证端口映射格式
func (p *ComposeParser) isValidPortMapping(port string) bool {
	// 支持的格式：
	// - "80"
	// - "8080:80"
	// - "127.0.0.1:8080:80"
	// - "8080-8090:80-90"
	portPattern := regexp.MustCompile(`^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:)?(\d+(-\d+)?:)?\d+(-\d+)?(/tcp|/udp)?$`)
	return portPattern.MatchString(port)
}

// isVersionSupported 检查版本是否支持
func (p *ComposeParser) isVersionSupported(version string) bool {
	return contains(p.supportedVersions, version)
}

// GetCompletions 获取自动补全建议
func (p *ComposeParser) GetCompletions(context string, position int) []*CompletionItem {
	completions := []*CompletionItem{}

	// 根据上下文提供不同的补全建议
	if strings.Contains(context, "services:") {
		completions = append(completions, p.getServiceCompletions()...)
	}

	if strings.Contains(context, "image:") {
		completions = append(completions, p.getImageCompletions()...)
	}

	if strings.Contains(context, "restart:") {
		completions = append(completions, p.getRestartPolicyCompletions()...)
	}

	if strings.Contains(context, "networks:") {
		completions = append(completions, p.getNetworkCompletions()...)
	}

	return completions
}

// getServiceCompletions 获取服务级别的补全建议
func (p *ComposeParser) getServiceCompletions() []*CompletionItem {
	return []*CompletionItem{
		{
			Label:         "image",
			Kind:          "property",
			Detail:        "Docker image",
			Documentation: "指定要使用的 Docker 镜像",
			InsertText:    "image: ",
		},
		{
			Label:         "build",
			Kind:          "property",
			Detail:        "Build configuration",
			Documentation: "构建配置，用于从 Dockerfile 构建镜像",
			InsertText:    "build:\n  context: .\n  dockerfile: Dockerfile",
		},
		{
			Label:         "ports",
			Kind:          "property",
			Detail:        "Port mappings",
			Documentation: "端口映射配置",
			InsertText:    "ports:\n  - \"8080:80\"",
		},
		{
			Label:         "volumes",
			Kind:          "property",
			Detail:        "Volume mounts",
			Documentation: "卷挂载配置",
			InsertText:    "volumes:\n  - ./data:/data",
		},
		{
			Label:         "environment",
			Kind:          "property",
			Detail:        "Environment variables",
			Documentation: "环境变量配置",
			InsertText:    "environment:\n  - KEY=value",
		},
		{
			Label:         "depends_on",
			Kind:          "property",
			Detail:        "Service dependencies",
			Documentation: "服务依赖关系",
			InsertText:    "depends_on:\n  - service_name",
		},
		{
			Label:         "restart",
			Kind:          "property",
			Detail:        "Restart policy",
			Documentation: "重启策略",
			InsertText:    "restart: always",
		},
		{
			Label:         "networks",
			Kind:          "property",
			Detail:        "Networks",
			Documentation: "网络配置",
			InsertText:    "networks:\n  - network_name",
		},
		{
			Label:         "healthcheck",
			Kind:          "property",
			Detail:        "Health check",
			Documentation: "健康检查配置",
			InsertText:    "healthcheck:\n  test: [\"CMD\", \"curl\", \"-f\", \"http://localhost\"]\n  interval: 30s\n  timeout: 10s\n  retries: 3",
		},
	}
}

// getImageCompletions 获取镜像补全建议
func (p *ComposeParser) getImageCompletions() []*CompletionItem {
	return []*CompletionItem{
		{Label: "nginx:latest", Kind: "value", Detail: "Nginx web server"},
		{Label: "mysql:8.0", Kind: "value", Detail: "MySQL database"},
		{Label: "postgres:15", Kind: "value", Detail: "PostgreSQL database"},
		{Label: "redis:7", Kind: "value", Detail: "Redis cache"},
		{Label: "mongo:6", Kind: "value", Detail: "MongoDB database"},
	}
}

// getRestartPolicyCompletions 获取重启策略补全建议
func (p *ComposeParser) getRestartPolicyCompletions() []*CompletionItem {
	return []*CompletionItem{
		{Label: "no", Kind: "value", Detail: "不自动重启", Documentation: "容器退出时不自动重启"},
		{Label: "always", Kind: "value", Detail: "总是重启", Documentation: "容器退出时总是自动重启"},
		{Label: "on-failure", Kind: "value", Detail: "失败时重启", Documentation: "仅在容器非正常退出时重启"},
		{Label: "unless-stopped", Kind: "value", Detail: "除非停止", Documentation: "除非手动停止，否则总是重启"},
	}
}

// getNetworkCompletions 获取网络补全建议
func (p *ComposeParser) getNetworkCompletions() []*CompletionItem {
	return []*CompletionItem{
		{
			Label:         "driver: bridge",
			Kind:          "value",
			Detail:        "Bridge network",
			Documentation: "桥接网络（默认）",
		},
		{
			Label:         "driver: host",
			Kind:          "value",
			Detail:        "Host network",
			Documentation: "主机网络模式",
		},
		{
			Label:         "driver: overlay",
			Kind:          "value",
			Detail:        "Overlay network",
			Documentation: "覆盖网络（用于 Swarm）",
		},
	}
}

// contains 检查字符串切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
