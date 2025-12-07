package registry

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"sync"
	"time"
)

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	Select(instances []*ServiceInstance, key string) (*ServiceInstance, error)
	Name() string
}

// RoundRobinBalancer 轮询负载均衡器
type RoundRobinBalancer struct {
	mu      sync.Mutex
	counter map[string]int
}

// NewRoundRobinBalancer 创建轮询负载均衡器
func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{
		counter: make(map[string]int),
	}
}

// Select 选择服务实例
func (rb *RoundRobinBalancer) Select(instances []*ServiceInstance, key string) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("没有可用的服务实例")
	}
	
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	serviceName := instances[0].Name
	index := rb.counter[serviceName] % len(instances)
	rb.counter[serviceName]++
	
	return instances[index], nil
}

// Name 返回负载均衡器名称
func (rb *RoundRobinBalancer) Name() string {
	return "round_robin"
}

// RandomBalancer 随机负载均衡器
type RandomBalancer struct {
	rand *rand.Rand
	mu   sync.Mutex
}

// NewRandomBalancer 创建随机负载均衡器
func NewRandomBalancer() *RandomBalancer {
	return &RandomBalancer{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Select 选择服务实例
func (rb *RandomBalancer) Select(instances []*ServiceInstance, key string) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("没有可用的服务实例")
	}
	
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	index := rb.rand.Intn(len(instances))
	return instances[index], nil
}

// Name 返回负载均衡器名称
func (rb *RandomBalancer) Name() string {
	return "random"
}

// WeightedRoundRobinBalancer 加权轮询负载均衡器
type WeightedRoundRobinBalancer struct {
	mu      sync.Mutex
	weights map[string][]int // 服务名 -> 权重数组
}

// NewWeightedRoundRobinBalancer 创建加权轮询负载均衡器
func NewWeightedRoundRobinBalancer() *WeightedRoundRobinBalancer {
	return &WeightedRoundRobinBalancer{
		weights: make(map[string][]int),
	}
}

// Select 选择服务实例
func (wrb *WeightedRoundRobinBalancer) Select(instances []*ServiceInstance, key string) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("没有可用的服务实例")
	}
	
	wrb.mu.Lock()
	defer wrb.mu.Unlock()
	
	serviceName := instances[0].Name
	
	// 简化的加权轮询：根据权重比例选择
	totalWeight := 0
	for _, instance := range instances {
		weight := instance.Weight
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
	}
	
	// 轮询选择
	if _, exists := wrb.weights[serviceName]; !exists {
		wrb.weights[serviceName] = []int{0}
	}
	
	currentIndex := wrb.weights[serviceName][0] % totalWeight
	wrb.weights[serviceName][0]++
	
	// 根据权重找到对应的实例
	currentWeight := 0
	for _, instance := range instances {
		weight := instance.Weight
		if weight <= 0 {
			weight = 1
		}
		currentWeight += weight
		if currentIndex < currentWeight {
			return instance, nil
		}
	}
	
	// 默认返回第一个实例
	return instances[0], nil
}

// Name 返回负载均衡器名称
func (wrb *WeightedRoundRobinBalancer) Name() string {
	return "weighted_round_robin"
}

// ConsistentHashBalancer 一致性哈希负载均衡器
type ConsistentHashBalancer struct{}

// NewConsistentHashBalancer 创建一致性哈希负载均衡器
func NewConsistentHashBalancer() *ConsistentHashBalancer {
	return &ConsistentHashBalancer{}
}

// Select 选择服务实例
func (chb *ConsistentHashBalancer) Select(instances []*ServiceInstance, key string) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("没有可用的服务实例")
	}
	
	if key == "" {
		// 如果没有提供key，使用随机选择
		index := int(time.Now().UnixNano()) % len(instances)
		return instances[index], nil
	}
	
	// 计算key的哈希值
	hash := fnv.New32a()
	hash.Write([]byte(key))
	hashValue := hash.Sum32()
	
	// 选择实例
	index := int(hashValue) % len(instances)
	return instances[index], nil
}

// Name 返回负载均衡器名称
func (chb *ConsistentHashBalancer) Name() string {
	return "consistent_hash"
}

// LeastConnectionsBalancer 最少连接数负载均衡器
type LeastConnectionsBalancer struct {
	mu          sync.Mutex
	connections map[string]int // 实例ID -> 连接数
}

// NewLeastConnectionsBalancer 创建最少连接数负载均衡器
func NewLeastConnectionsBalancer() *LeastConnectionsBalancer {
	return &LeastConnectionsBalancer{
		connections: make(map[string]int),
	}
}

// Select 选择服务实例
func (lcb *LeastConnectionsBalancer) Select(instances []*ServiceInstance, key string) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("没有可用的服务实例")
	}
	
	lcb.mu.Lock()
	defer lcb.mu.Unlock()
	
	var selectedInstance *ServiceInstance
	minConnections := -1
	
	for _, instance := range instances {
		connections := lcb.connections[instance.ID]
		
		if minConnections == -1 || connections < minConnections {
			minConnections = connections
			selectedInstance = instance
		}
	}
	
	// 增加连接数
	if selectedInstance != nil {
		lcb.connections[selectedInstance.ID]++
	}
	
	return selectedInstance, nil
}

// ReleaseConnection 释放连接
func (lcb *LeastConnectionsBalancer) ReleaseConnection(instanceID string) {
	lcb.mu.Lock()
	defer lcb.mu.Unlock()
	
	if count, exists := lcb.connections[instanceID]; exists && count > 0 {
		lcb.connections[instanceID]--
	}
}

// Name 返回负载均衡器名称
func (lcb *LeastConnectionsBalancer) Name() string {
	return "least_connections"
}

// LoadBalancerManager 负载均衡器管理器
type LoadBalancerManager struct {
	balancers       map[string]LoadBalancer
	defaultBalancer LoadBalancer
}

// NewLoadBalancerManager 创建负载均衡器管理器
func NewLoadBalancerManager() *LoadBalancerManager {
	manager := &LoadBalancerManager{
		balancers: make(map[string]LoadBalancer),
	}
	
	// 注册默认负载均衡器
	manager.RegisterBalancer(NewRoundRobinBalancer())
	manager.RegisterBalancer(NewRandomBalancer())
	manager.RegisterBalancer(NewWeightedRoundRobinBalancer())
	manager.RegisterBalancer(NewConsistentHashBalancer())
	manager.RegisterBalancer(NewLeastConnectionsBalancer())
	
	// 设置默认负载均衡器
	manager.defaultBalancer = NewRoundRobinBalancer()
	
	return manager
}

// RegisterBalancer 注册负载均衡器
func (lbm *LoadBalancerManager) RegisterBalancer(balancer LoadBalancer) {
	lbm.balancers[balancer.Name()] = balancer
}

// GetBalancer 获取负载均衡器
func (lbm *LoadBalancerManager) GetBalancer(name string) LoadBalancer {
	if balancer, exists := lbm.balancers[name]; exists {
		return balancer
	}
	return lbm.defaultBalancer
}

// SetDefault 设置默认负载均衡器
func (lbm *LoadBalancerManager) SetDefault(name string) {
	if balancer, exists := lbm.balancers[name]; exists {
		lbm.defaultBalancer = balancer
	}
}

// ListBalancers 列出所有负载均衡器
func (lbm *LoadBalancerManager) ListBalancers() []string {
	var names []string
	for name := range lbm.balancers {
		names = append(names, name)
	}
	return names
}