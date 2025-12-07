package appstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConflictChecker 冲突检测器
type ConflictChecker struct {
	appStoreService AppStoreService
}

// NewConflictChecker 创建冲突检测器实例
func NewConflictChecker(appStoreService AppStoreService) *ConflictChecker {
	return &ConflictChecker{
		appStoreService: appStoreService,
	}
}

// DetectConflicts 检测冲突
func (c *ConflictChecker) DetectConflicts(ctx context.Context, templateID uint, params map[string]interface{}) ([]ConflictInfo, error) {
	// 获取模板
	template, err := c.appStoreService.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// 渲染模板以获取实际配置
	rendered, err := c.appStoreService.RenderTemplate(ctx, templateID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	var conflicts []ConflictInfo

	// 根据模板类型检测不同的冲突
	switch template.Type {
	case TemplateTypeDockerCompose:
		composeConflicts, err := c.detectDockerComposeConflicts(ctx, rendered)
		if err != nil {
			return nil, fmt.Errorf("failed to detect docker-compose conflicts: %w", err)
		}
		conflicts = append(conflicts, composeConflicts...)
	case TemplateTypeHelmChart:
		// Helm Chart 冲突检测
		helmConflicts, err := c.detectHelmChartConflicts(ctx, rendered)
		if err != nil {
			return nil, fmt.Errorf("failed to detect helm chart conflicts: %w", err)
		}
		conflicts = append(conflicts, helmConflicts...)
	}

	return conflicts, nil
}

// detectDockerComposeConflicts 检测 Docker Compose 冲突
func (c *ConflictChecker) detectDockerComposeConflicts(ctx context.Context, rendered string) ([]ConflictInfo, error) {
	// 解析 Docker Compose 文件
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(rendered), &compose); err != nil {
		return nil, fmt.Errorf("failed to parse docker-compose: %w", err)
	}

	var conflicts []ConflictInfo

	// 获取所有已安装的实例
	instances, err := c.appStoreService.ListInstances(ctx, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	// 提取当前模板使用的端口
	currentPorts := c.extractPortsFromCompose(compose)

	// 提取当前模板使用的数据卷
	currentVolumes := c.extractVolumesFromCompose(compose)

	// 检查端口冲突
	for _, instance := range instances {
		if instance.Status == "running" || instance.Status == "installing" {
			// 获取实例的配置
			var instanceConfig map[string]interface{}
			if err := json.Unmarshal([]byte(instance.Config), &instanceConfig); err != nil {
				continue
			}

			// 渲染实例的模板以获取其使用的资源
			instanceRendered, err := c.appStoreService.RenderTemplate(ctx, instance.TemplateID, instanceConfig)
			if err != nil {
				continue
			}

			var instanceCompose map[string]interface{}
			if err := yaml.Unmarshal([]byte(instanceRendered), &instanceCompose); err != nil {
				continue
			}

			// 检查端口冲突
			instancePorts := c.extractPortsFromCompose(instanceCompose)
			for _, port := range currentPorts {
				for _, existingPort := range instancePorts {
					if port == existingPort {
						conflicts = append(conflicts, ConflictInfo{
							Type:        "port",
							Resource:    port,
							ExistingApp: instance.Name,
							Resolvable:  true,
							Suggestions: []string{
								fmt.Sprintf("Change port mapping to use a different host port"),
								fmt.Sprintf("Stop the conflicting application: %s", instance.Name),
							},
						})
					}
				}
			}

			// 检查数据卷冲突
			instanceVolumes := c.extractVolumesFromCompose(instanceCompose)
			for _, volume := range currentVolumes {
				for _, existingVolume := range instanceVolumes {
					if volume == existingVolume {
						conflicts = append(conflicts, ConflictInfo{
							Type:        "volume",
							Resource:    volume,
							ExistingApp: instance.Name,
							Resolvable:  true,
							Suggestions: []string{
								fmt.Sprintf("Use a different volume name or path"),
								fmt.Sprintf("Share the volume with application: %s", instance.Name),
							},
						})
					}
				}
			}
		}
	}

	return conflicts, nil
}

// detectHelmChartConflicts 检测 Helm Chart 冲突
func (c *ConflictChecker) detectHelmChartConflicts(ctx context.Context, rendered string) ([]ConflictInfo, error) {
	// Helm Chart 冲突检测逻辑
	// 这里可以检查 Service、Ingress 等资源的冲突
	
	var conflicts []ConflictInfo
	
	// 解析 Helm Chart
	var chart map[string]interface{}
	if err := yaml.Unmarshal([]byte(rendered), &chart); err != nil {
		return nil, fmt.Errorf("failed to parse helm chart: %w", err)
	}

	// 检查服务名称冲突
	// 检查 Ingress 主机名冲突
	// 等等...

	return conflicts, nil
}

// extractPortsFromCompose 从 Docker Compose 配置中提取端口
func (c *ConflictChecker) extractPortsFromCompose(compose map[string]interface{}) []string {
	var ports []string

	services, ok := compose["services"].(map[string]interface{})
	if !ok {
		return ports
	}

	for _, serviceConfig := range services {
		serviceMap, ok := serviceConfig.(map[string]interface{})
		if !ok {
			continue
		}

		// 检查 ports 字段
		if portsField, exists := serviceMap["ports"]; exists {
			switch v := portsField.(type) {
			case []interface{}:
				for _, port := range v {
					portStr := c.normalizePort(port)
					if portStr != "" {
						ports = append(ports, portStr)
					}
				}
			}
		}
	}

	return ports
}

// extractVolumesFromCompose 从 Docker Compose 配置中提取数据卷
func (c *ConflictChecker) extractVolumesFromCompose(compose map[string]interface{}) []string {
	var volumes []string

	services, ok := compose["services"].(map[string]interface{})
	if !ok {
		return volumes
	}

	for _, serviceConfig := range services {
		serviceMap, ok := serviceConfig.(map[string]interface{})
		if !ok {
			continue
		}

		// 检查 volumes 字段
		if volumesField, exists := serviceMap["volumes"]; exists {
			switch v := volumesField.(type) {
			case []interface{}:
				for _, volume := range v {
					volumeStr := c.normalizeVolume(volume)
					if volumeStr != "" {
						volumes = append(volumes, volumeStr)
					}
				}
			}
		}
	}

	return volumes
}

// normalizePort 规范化端口表示
func (c *ConflictChecker) normalizePort(port interface{}) string {
	switch v := port.(type) {
	case string:
		// 格式可能是 "8080:80" 或 "8080"
		parts := strings.Split(v, ":")
		if len(parts) > 0 {
			return parts[0] // 返回主机端口
		}
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.Itoa(int(v))
	}
	return ""
}

// normalizeVolume 规范化数据卷表示
func (c *ConflictChecker) normalizeVolume(volume interface{}) string {
	switch v := volume.(type) {
	case string:
		// 格式可能是 "/host/path:/container/path" 或 "volume_name:/container/path"
		parts := strings.Split(v, ":")
		if len(parts) > 0 {
			return parts[0] // 返回主机路径或卷名
		}
	}
	return ""
}

// ResolveConflict 解决冲突
func (c *ConflictChecker) ResolveConflict(ctx context.Context, conflict ConflictInfo, resolution string) error {
	if !conflict.Resolvable {
		return fmt.Errorf("conflict is not resolvable: %s", conflict.Resource)
	}

	// 根据冲突类型和解决方案执行相应的操作
	switch conflict.Type {
	case "port":
		// 端口冲突解决逻辑
		// 例如：自动分配新端口、停止冲突应用等
		return c.resolvePortConflict(ctx, conflict, resolution)
	case "volume":
		// 数据卷冲突解决逻辑
		return c.resolveVolumeConflict(ctx, conflict, resolution)
	case "service":
		// 服务名称冲突解决逻辑
		return c.resolveServiceConflict(ctx, conflict, resolution)
	default:
		return fmt.Errorf("unknown conflict type: %s", conflict.Type)
	}
}

// resolvePortConflict 解决端口冲突
func (c *ConflictChecker) resolvePortConflict(ctx context.Context, conflict ConflictInfo, resolution string) error {
	// 实现端口冲突解决逻辑
	// 例如：自动分配新端口
	return nil
}

// resolveVolumeConflict 解决数据卷冲突
func (c *ConflictChecker) resolveVolumeConflict(ctx context.Context, conflict ConflictInfo, resolution string) error {
	// 实现数据卷冲突解决逻辑
	return nil
}

// resolveServiceConflict 解决服务冲突
func (c *ConflictChecker) resolveServiceConflict(ctx context.Context, conflict ConflictInfo, resolution string) error {
	// 实现服务名称冲突解决逻辑
	return nil
}

// FindAvailablePort 查找可用端口
func (c *ConflictChecker) FindAvailablePort(ctx context.Context, startPort int) (int, error) {
	// 获取所有已使用的端口
	usedPorts := make(map[int]bool)

	instances, err := c.appStoreService.ListInstances(ctx, 0, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to list instances: %w", err)
	}

	for _, instance := range instances {
		if instance.Status == "running" || instance.Status == "installing" {
			var instanceConfig map[string]interface{}
			if err := json.Unmarshal([]byte(instance.Config), &instanceConfig); err != nil {
				continue
			}

			instanceRendered, err := c.appStoreService.RenderTemplate(ctx, instance.TemplateID, instanceConfig)
			if err != nil {
				continue
			}

			var instanceCompose map[string]interface{}
			if err := yaml.Unmarshal([]byte(instanceRendered), &instanceCompose); err != nil {
				continue
			}

			ports := c.extractPortsFromCompose(instanceCompose)
			for _, portStr := range ports {
				if port, err := strconv.Atoi(portStr); err == nil {
					usedPorts[port] = true
				}
			}
		}
	}

	// 从 startPort 开始查找可用端口
	for port := startPort; port < 65535; port++ {
		if !usedPorts[port] {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available port found")
}
