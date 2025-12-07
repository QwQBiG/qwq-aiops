// Package registry 提供服务注册与发现功能
// 支持服务实例的注册、注销、发现和健康检查
package registry

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ServiceRegistry 服务注册中心
// 提供服务实例的注册、发现、健康检查等核心功能
// 支持多种服务发现策略和负载均衡算法
type ServiceRegistry struct {
	mu                sync.RWMutex          // 读写锁，保护并发访问
	services          map[string]*ServiceInstance // 服务实例映射表，key为服务ID
	watchers          []ServiceWatcher      // 服务状态变化监听器列表
	healthCheckTicker *time.Ticker          // 健康检查定时器
	ctx               context.Context       // 上下文，用于控制goroutine生命周期
	cancel            context.CancelFunc    // 取消函数，用于停止健康检查
}

// ServiceInstance 服务实例信息
// 包含服务的基本信息、健康状态、负载均衡配置等
type ServiceInstance struct {
	ID          string            `json:"id"`          // 服务实例唯一标识符
	Name        string            `json:"name"`        // 服务名称
	Address     string            `json:"address"`     // 服务地址（IP或域名）
	Port        int               `json:"port"`        // 服务端口
	Health      string            `json:"health"`      // 健康检查URL
	Status      ServiceStatus     `json:"status"`      // 当前服务状态
	Metadata    map[string]string `json:"metadata"`    // 服务元数据，存储额外信息
	Tags        []string          `json:"tags"`        // 服务标签，用于分类和过滤
	LastSeen    time.Time         `json:"last_seen"`   // 最后一次心跳时间
	Version     string            `json:"version"`     // 服务版本号
	Weight      int               `json:"weight"`      // 负载均衡权重，数值越大权重越高
	FailCount   int               `json:"fail_count"`  // 连续失败次数，用于熔断判断
	MaxFails    int               `json:"max_fails"`   // 最大失败次数阈值
}

// ServiceStatus 服务状态枚举
type ServiceStatus string

const (
	StatusHealthy   ServiceStatus = "healthy"
	StatusUnhealthy ServiceStatus = "unhealthy"
	StatusUnknown   ServiceStatus = "unknown"
	StatusDraining  ServiceStatus = "draining" // 正在排空流量
)

// ServiceWatcher 服务监听器接口
// 用于监听服务注册、注销和状态变化事件
// 实现此接口可以接收服务变化通知，用于负载均衡器更新、日志记录等
type ServiceWatcher interface {
	OnServiceRegistered(service *ServiceInstance)                                    // 服务注册时触发
	OnServiceDeregistered(serviceID string)                                          // 服务注销时触发
	OnServiceStatusChanged(service *ServiceInstance, oldStatus ServiceStatus)       // 服务状态变化时触发
}

// RegistrationRequest 服务注册请求
// 包含注册服务实例所需的所有信息
type RegistrationRequest struct {
	Name     string            `json:"name"`      // 服务名称，必填
	Address  string            `json:"address"`   // 服务地址，必填
	Port     int               `json:"port"`      // 服务端口，必填
	Health   string            `json:"health"`    // 健康检查URL，可选
	Metadata map[string]string `json:"metadata"`  // 服务元数据，可选
	Tags     []string          `json:"tags"`      // 服务标签，可选
	Version  string            `json:"version"`   // 服务版本，可选
	Weight   int               `json:"weight"`    // 负载均衡权重，默认100
	MaxFails int               `json:"max_fails"` // 最大失败次数，默认3
}

// NewServiceRegistry 创建新的服务注册中心
func NewServiceRegistry() *ServiceRegistry {
	ctx, cancel := context.WithCancel(context.Background())
	
	registry := &ServiceRegistry{
		services: make(map[string]*ServiceInstance),
		watchers: make([]ServiceWatcher, 0),
		ctx:      ctx,
		cancel:   cancel,
	}
	
	// 启动健康检查
	registry.startHealthChecker()
	
	return registry
}

// Register 注册服务实例
func (r *ServiceRegistry) Register(req *RegistrationRequest) (*ServiceInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// 生成服务实例ID
	instanceID := fmt.Sprintf("%s-%s-%d-%d", req.Name, req.Address, req.Port, time.Now().Unix())
	
	// 设置默认值
	if req.Weight <= 0 {
		req.Weight = 100
	}
	if req.MaxFails <= 0 {
		req.MaxFails = 3
	}
	
	instance := &ServiceInstance{
		ID:        instanceID,
		Name:      req.Name,
		Address:   req.Address,
		Port:      req.Port,
		Health:    req.Health,
		Status:    StatusUnknown,
		Metadata:  req.Metadata,
		Tags:      req.Tags,
		LastSeen:  time.Now(),
		Version:   req.Version,
		Weight:    req.Weight,
		FailCount: 0,
		MaxFails:  req.MaxFails,
	}
	
	r.services[instanceID] = instance
	
	// 通知监听器
	for _, watcher := range r.watchers {
		watcher.OnServiceRegistered(instance)
	}
	
	return instance, nil
}

// Deregister 注销服务实例
func (r *ServiceRegistry) Deregister(serviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.services[serviceID]; !exists {
		return fmt.Errorf("服务实例不存在: %s", serviceID)
	}
	
	delete(r.services, serviceID)
	
	// 通知监听器
	for _, watcher := range r.watchers {
		watcher.OnServiceDeregistered(serviceID)
	}
	
	return nil
}

// Discover 发现服务实例
func (r *ServiceRegistry) Discover(serviceName string) ([]*ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var instances []*ServiceInstance
	
	for _, instance := range r.services {
		if instance.Name == serviceName && instance.Status == StatusHealthy {
			instances = append(instances, instance)
		}
	}
	
	return instances, nil
}

// DiscoverWithTags 根据标签发现服务实例
func (r *ServiceRegistry) DiscoverWithTags(serviceName string, tags []string) ([]*ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var instances []*ServiceInstance
	
	for _, instance := range r.services {
		if instance.Name == serviceName && instance.Status == StatusHealthy {
			// 检查是否包含所有必需的标签
			if r.hasAllTags(instance.Tags, tags) {
				instances = append(instances, instance)
			}
		}
	}
	
	return instances, nil
}

// GetService 获取特定服务实例
func (r *ServiceRegistry) GetService(serviceID string) (*ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	instance, exists := r.services[serviceID]
	if !exists {
		return nil, fmt.Errorf("服务实例不存在: %s", serviceID)
	}
	
	return instance, nil
}

// ListServices 列出所有服务实例
func (r *ServiceRegistry) ListServices() map[string]*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	services := make(map[string]*ServiceInstance)
	for id, instance := range r.services {
		services[id] = instance
	}
	
	return services
}

// AddWatcher 添加服务监听器
func (r *ServiceRegistry) AddWatcher(watcher ServiceWatcher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.watchers = append(r.watchers, watcher)
}

// UpdateServiceStatus 更新服务状态
func (r *ServiceRegistry) UpdateServiceStatus(serviceID string, status ServiceStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	instance, exists := r.services[serviceID]
	if !exists {
		return fmt.Errorf("服务实例不存在: %s", serviceID)
	}
	
	oldStatus := instance.Status
	instance.Status = status
	instance.LastSeen = time.Now()
	
	// 重置失败计数（如果状态变为健康）
	if status == StatusHealthy {
		instance.FailCount = 0
	}
	
	// 通知监听器状态变化
	if oldStatus != status {
		for _, watcher := range r.watchers {
			watcher.OnServiceStatusChanged(instance, oldStatus)
		}
	}
	
	return nil
}

// Heartbeat 服务心跳
func (r *ServiceRegistry) Heartbeat(serviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	instance, exists := r.services[serviceID]
	if !exists {
		return fmt.Errorf("服务实例不存在: %s", serviceID)
	}
	
	instance.LastSeen = time.Now()
	
	// 如果服务之前是不健康的，现在收到心跳，可以考虑恢复
	if instance.Status == StatusUnhealthy {
		instance.Status = StatusHealthy
		instance.FailCount = 0
		
		// 通知监听器状态变化
		for _, watcher := range r.watchers {
			watcher.OnServiceStatusChanged(instance, StatusUnhealthy)
		}
	}
	
	return nil
}

// startHealthChecker 启动健康检查器
func (r *ServiceRegistry) startHealthChecker() {
	r.healthCheckTicker = time.NewTicker(30 * time.Second)
	
	go func() {
		for {
			select {
			case <-r.ctx.Done():
				return
			case <-r.healthCheckTicker.C:
				r.performHealthChecks()
			}
		}
	}()
}

// performHealthChecks 执行健康检查
func (r *ServiceRegistry) performHealthChecks() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	client := &http.Client{Timeout: 5 * time.Second}
	
	for _, instance := range r.services {
		if instance.Health == "" {
			continue
		}
		
		// 检查心跳超时（如果超过2分钟没有心跳，标记为不健康）
		if time.Since(instance.LastSeen) > 2*time.Minute {
			if instance.Status != StatusUnhealthy {
				oldStatus := instance.Status
				instance.Status = StatusUnhealthy
				instance.FailCount++
				
				// 通知监听器状态变化
				for _, watcher := range r.watchers {
					watcher.OnServiceStatusChanged(instance, oldStatus)
				}
			}
			continue
		}
		
		// 执行HTTP健康检查
		resp, err := client.Get(instance.Health)
		
		if err != nil {
			r.handleHealthCheckFailure(instance)
			continue
		}
		
		resp.Body.Close()
		
		if resp.StatusCode == 200 {
			r.handleHealthCheckSuccess(instance)
		} else {
			r.handleHealthCheckFailure(instance)
		}
	}
}

// handleHealthCheckSuccess 处理健康检查成功
func (r *ServiceRegistry) handleHealthCheckSuccess(instance *ServiceInstance) {
	oldStatus := instance.Status
	
	if instance.Status != StatusHealthy {
		instance.Status = StatusHealthy
		instance.FailCount = 0
		instance.LastSeen = time.Now()
		
		// 通知监听器状态变化
		for _, watcher := range r.watchers {
			watcher.OnServiceStatusChanged(instance, oldStatus)
		}
	}
}

// handleHealthCheckFailure 处理健康检查失败
func (r *ServiceRegistry) handleHealthCheckFailure(instance *ServiceInstance) {
	oldStatus := instance.Status
	instance.FailCount++
	
	// 如果连续失败次数超过阈值，标记为不健康
	if instance.FailCount >= instance.MaxFails {
		if instance.Status != StatusUnhealthy {
			instance.Status = StatusUnhealthy
			
			// 通知监听器状态变化
			for _, watcher := range r.watchers {
				watcher.OnServiceStatusChanged(instance, oldStatus)
			}
		}
	}
}

// hasAllTags 检查是否包含所有必需的标签
func (r *ServiceRegistry) hasAllTags(instanceTags, requiredTags []string) bool {
	if len(requiredTags) == 0 {
		return true
	}
	
	tagMap := make(map[string]bool)
	for _, tag := range instanceTags {
		tagMap[tag] = true
	}
	
	for _, requiredTag := range requiredTags {
		if !tagMap[requiredTag] {
			return false
		}
	}
	
	return true
}

// Stop 停止服务注册中心
func (r *ServiceRegistry) Stop() {
	if r.healthCheckTicker != nil {
		r.healthCheckTicker.Stop()
	}
	r.cancel()
}

// GetHealthyServices 获取所有健康的服务实例
func (r *ServiceRegistry) GetHealthyServices() map[string][]*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	services := make(map[string][]*ServiceInstance)
	
	for _, instance := range r.services {
		if instance.Status == StatusHealthy {
			services[instance.Name] = append(services[instance.Name], instance)
		}
	}
	
	return services
}

// GetServiceStats 获取服务统计信息
func (r *ServiceRegistry) GetServiceStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_services": len(r.services),
		"healthy_count":  0,
		"unhealthy_count": 0,
		"unknown_count":  0,
		"services_by_name": make(map[string]int),
	}
	
	servicesByName := make(map[string]int)
	
	for _, instance := range r.services {
		servicesByName[instance.Name]++
		
		switch instance.Status {
		case StatusHealthy:
			stats["healthy_count"] = stats["healthy_count"].(int) + 1
		case StatusUnhealthy:
			stats["unhealthy_count"] = stats["unhealthy_count"].(int) + 1
		case StatusUnknown:
			stats["unknown_count"] = stats["unknown_count"].(int) + 1
		}
	}
	
	stats["services_by_name"] = servicesByName
	
	return stats
}