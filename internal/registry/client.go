package registry

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ServiceDiscoveryClient 服务发现客户端
type ServiceDiscoveryClient struct {
	registry        *ServiceRegistry
	loadBalancer    LoadBalancer
	lbManager       *LoadBalancerManager
	cache           map[string][]*ServiceInstance
	cacheMu         sync.RWMutex
	cacheExpiry     time.Duration
	lastCacheUpdate map[string]time.Time
	watchers        map[string][]ServiceChangeCallback
	watchersMu      sync.RWMutex
}

// ServiceChangeCallback 服务变化回调函数
type ServiceChangeCallback func(serviceName string, instances []*ServiceInstance)

// NewServiceDiscoveryClient 创建服务发现客户端
func NewServiceDiscoveryClient(registry *ServiceRegistry) *ServiceDiscoveryClient {
	client := &ServiceDiscoveryClient{
		registry:        registry,
		lbManager:       NewLoadBalancerManager(),
		cache:           make(map[string][]*ServiceInstance),
		cacheExpiry:     30 * time.Second,
		lastCacheUpdate: make(map[string]time.Time),
		watchers:        make(map[string][]ServiceChangeCallback),
	}
	
	// 设置默认负载均衡器
	client.loadBalancer = client.lbManager.GetBalancer("round_robin")
	
	// 注册为服务监听器
	registry.AddWatcher(client)
	
	return client
}

// SelectInstance 选择服务实例
func (sdc *ServiceDiscoveryClient) SelectInstance(serviceName string, key string) (*ServiceInstance, error) {
	instances, err := sdc.GetInstances(serviceName)
	if err != nil {
		return nil, err
	}
	
	if len(instances) == 0 {
		return nil, fmt.Errorf("服务 %s 没有可用实例", serviceName)
	}
	
	return sdc.loadBalancer.Select(instances, key)
}

// SelectInstanceWithTags 根据标签选择服务实例
func (sdc *ServiceDiscoveryClient) SelectInstanceWithTags(serviceName string, tags []string, key string) (*ServiceInstance, error) {
	instances, err := sdc.GetInstancesWithTags(serviceName, tags)
	if err != nil {
		return nil, err
	}
	
	if len(instances) == 0 {
		return nil, fmt.Errorf("服务 %s 没有匹配标签 %v 的可用实例", serviceName, tags)
	}
	
	return sdc.loadBalancer.Select(instances, key)
}

// GetInstances 获取服务实例（带缓存）
func (sdc *ServiceDiscoveryClient) GetInstances(serviceName string) ([]*ServiceInstance, error) {
	// 检查缓存
	sdc.cacheMu.RLock()
	if instances, exists := sdc.cache[serviceName]; exists {
		if lastUpdate, ok := sdc.lastCacheUpdate[serviceName]; ok {
			if time.Since(lastUpdate) < sdc.cacheExpiry {
				sdc.cacheMu.RUnlock()
				return instances, nil
			}
		}
	}
	sdc.cacheMu.RUnlock()
	
	// 从注册中心获取最新数据
	instances, err := sdc.registry.Discover(serviceName)
	if err != nil {
		return nil, err
	}
	
	// 更新缓存
	sdc.cacheMu.Lock()
	sdc.cache[serviceName] = instances
	sdc.lastCacheUpdate[serviceName] = time.Now()
	sdc.cacheMu.Unlock()
	
	return instances, nil
}

// GetInstancesWithTags 根据标签获取服务实例（带缓存）
func (sdc *ServiceDiscoveryClient) GetInstancesWithTags(serviceName string, tags []string) ([]*ServiceInstance, error) {
	cacheKey := fmt.Sprintf("%s:%v", serviceName, tags)
	
	// 检查缓存
	sdc.cacheMu.RLock()
	if instances, exists := sdc.cache[cacheKey]; exists {
		if lastUpdate, ok := sdc.lastCacheUpdate[cacheKey]; ok {
			if time.Since(lastUpdate) < sdc.cacheExpiry {
				sdc.cacheMu.RUnlock()
				return instances, nil
			}
		}
	}
	sdc.cacheMu.RUnlock()
	
	// 从注册中心获取最新数据
	instances, err := sdc.registry.DiscoverWithTags(serviceName, tags)
	if err != nil {
		return nil, err
	}
	
	// 更新缓存
	sdc.cacheMu.Lock()
	sdc.cache[cacheKey] = instances
	sdc.lastCacheUpdate[cacheKey] = time.Now()
	sdc.cacheMu.Unlock()
	
	return instances, nil
}

// SetLoadBalancer 设置负载均衡器
func (sdc *ServiceDiscoveryClient) SetLoadBalancer(balancerName string) error {
	balancer := sdc.lbManager.GetBalancer(balancerName)
	if balancer == nil {
		return fmt.Errorf("未知的负载均衡器: %s", balancerName)
	}
	
	sdc.loadBalancer = balancer
	return nil
}

// GetLoadBalancer 获取当前负载均衡器
func (sdc *ServiceDiscoveryClient) GetLoadBalancer() LoadBalancer {
	return sdc.loadBalancer
}

// WatchService 监听服务变化
func (sdc *ServiceDiscoveryClient) WatchService(serviceName string, callback ServiceChangeCallback) {
	sdc.watchersMu.Lock()
	defer sdc.watchersMu.Unlock()
	
	if _, exists := sdc.watchers[serviceName]; !exists {
		sdc.watchers[serviceName] = make([]ServiceChangeCallback, 0)
	}
	
	sdc.watchers[serviceName] = append(sdc.watchers[serviceName], callback)
}

// UnwatchService 取消监听服务变化
func (sdc *ServiceDiscoveryClient) UnwatchService(serviceName string) {
	sdc.watchersMu.Lock()
	defer sdc.watchersMu.Unlock()
	
	delete(sdc.watchers, serviceName)
}

// InvalidateCache 使缓存失效
func (sdc *ServiceDiscoveryClient) InvalidateCache(serviceName string) {
	sdc.cacheMu.Lock()
	defer sdc.cacheMu.Unlock()
	
	// 删除相关缓存项
	keysToDelete := make([]string, 0)
	for key := range sdc.cache {
		if key == serviceName || (len(key) > len(serviceName) && key[:len(serviceName)] == serviceName) {
			keysToDelete = append(keysToDelete, key)
		}
	}
	
	for _, key := range keysToDelete {
		delete(sdc.cache, key)
		delete(sdc.lastCacheUpdate, key)
	}
}

// GetServiceURL 获取服务URL
func (sdc *ServiceDiscoveryClient) GetServiceURL(serviceName string, key string) (string, error) {
	instance, err := sdc.SelectInstance(serviceName, key)
	if err != nil {
		return "", err
	}
	
	return fmt.Sprintf("http://%s:%d", instance.Address, instance.Port), nil
}

// GetServiceURLWithTags 根据标签获取服务URL
func (sdc *ServiceDiscoveryClient) GetServiceURLWithTags(serviceName string, tags []string, key string) (string, error) {
	instance, err := sdc.SelectInstanceWithTags(serviceName, tags, key)
	if err != nil {
		return "", err
	}
	
	return fmt.Sprintf("http://%s:%d", instance.Address, instance.Port), nil
}

// HealthCheck 执行健康检查
func (sdc *ServiceDiscoveryClient) HealthCheck(ctx context.Context, serviceName string) error {
	instances, err := sdc.GetInstances(serviceName)
	if err != nil {
		return err
	}
	
	if len(instances) == 0 {
		return fmt.Errorf("服务 %s 没有可用实例", serviceName)
	}
	
	// 检查至少有一个健康的实例
	for _, instance := range instances {
		if instance.Status == StatusHealthy {
			return nil
		}
	}
	
	return fmt.Errorf("服务 %s 没有健康的实例", serviceName)
}

// GetServiceStats 获取服务统计信息
func (sdc *ServiceDiscoveryClient) GetServiceStats(serviceName string) (map[string]interface{}, error) {
	instances, err := sdc.GetInstances(serviceName)
	if err != nil {
		return nil, err
	}
	
	stats := map[string]interface{}{
		"total_instances":   len(instances),
		"healthy_instances": 0,
		"load_balancer":     sdc.loadBalancer.Name(),
		"cache_hit":         false,
	}
	
	// 检查缓存命中
	sdc.cacheMu.RLock()
	if _, exists := sdc.cache[serviceName]; exists {
		if lastUpdate, ok := sdc.lastCacheUpdate[serviceName]; ok {
			if time.Since(lastUpdate) < sdc.cacheExpiry {
				stats["cache_hit"] = true
			}
		}
	}
	sdc.cacheMu.RUnlock()
	
	// 统计健康实例
	for _, instance := range instances {
		if instance.Status == StatusHealthy {
			stats["healthy_instances"] = stats["healthy_instances"].(int) + 1
		}
	}
	
	return stats, nil
}

// 实现ServiceWatcher接口
func (sdc *ServiceDiscoveryClient) OnServiceRegistered(service *ServiceInstance) {
	sdc.InvalidateCache(service.Name)
	sdc.notifyWatchers(service.Name)
}

func (sdc *ServiceDiscoveryClient) OnServiceDeregistered(serviceID string) {
	// 需要找到服务名称来使缓存失效
	if service, err := sdc.registry.GetService(serviceID); err == nil {
		sdc.InvalidateCache(service.Name)
		sdc.notifyWatchers(service.Name)
	}
}

func (sdc *ServiceDiscoveryClient) OnServiceStatusChanged(service *ServiceInstance, oldStatus ServiceStatus) {
	sdc.InvalidateCache(service.Name)
	sdc.notifyWatchers(service.Name)
}

// notifyWatchers 通知监听器
func (sdc *ServiceDiscoveryClient) notifyWatchers(serviceName string) {
	sdc.watchersMu.RLock()
	callbacks, exists := sdc.watchers[serviceName]
	sdc.watchersMu.RUnlock()
	
	if !exists {
		return
	}
	
	// 获取最新的服务实例
	instances, err := sdc.registry.Discover(serviceName)
	if err != nil {
		return
	}
	
	// 通知所有监听器
	for _, callback := range callbacks {
		go callback(serviceName, instances)
	}
}

// SetCacheExpiry 设置缓存过期时间
func (sdc *ServiceDiscoveryClient) SetCacheExpiry(expiry time.Duration) {
	sdc.cacheExpiry = expiry
}

// ClearCache 清空所有缓存
func (sdc *ServiceDiscoveryClient) ClearCache() {
	sdc.cacheMu.Lock()
	defer sdc.cacheMu.Unlock()
	
	sdc.cache = make(map[string][]*ServiceInstance)
	sdc.lastCacheUpdate = make(map[string]time.Time)
}