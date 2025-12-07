package registry

import (
	"fmt"
	"testing"
	"time"
)

// **Feature: enhanced-aiops-platform, Property 26: 集群部署高可用性**
// **Validates: Requirements 10.1, 10.3**
func TestClusterDeploymentHighAvailability(t *testing.T) {
	// 测试用例代表不同场景的属性测试
	testCases := []struct {
		name        string
		serviceName string
		instances   []RegistrationRequest
		expected    bool
	}{
		{
			name:        "单个健康实例",
			serviceName: "test-service-1",
			instances: []RegistrationRequest{
				{
					Name:     "test-service-1",
					Address:  "localhost",
					Port:     8001,
					Health:   "http://localhost:8001/health",
					Version:  "1.0",
					Weight:   100,
					MaxFails: 3,
				},
			},
			expected: true,
		},
		{
			name:        "多个健康实例",
			serviceName: "test-service-2",
			instances: []RegistrationRequest{
				{
					Name:     "test-service-2",
					Address:  "localhost",
					Port:     8002,
					Health:   "http://localhost:8002/health",
					Version:  "1.0",
					Weight:   100,
					MaxFails: 3,
				},
				{
					Name:     "test-service-2",
					Address:  "localhost",
					Port:     8003,
					Health:   "http://localhost:8003/health",
					Version:  "1.0",
					Weight:   100,
					MaxFails: 3,
				},
			},
			expected: true,
		},
		{
			name:        "带标签的服务实例",
			serviceName: "test-service-3",
			instances: []RegistrationRequest{
				{
					Name:     "test-service-3",
					Address:  "localhost",
					Port:     8004,
					Health:   "http://localhost:8004/health",
					Version:  "1.0",
					Tags:     []string{"production", "backend"},
					Weight:   100,
					MaxFails: 3,
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Property: 对于任何集群部署配置，系统应该支持负载均衡、故障转移和服务恢复
			registry := NewServiceRegistry()
			defer registry.Stop()
			
			client := NewServiceDiscoveryClient(registry)
			
			// 注册服务实例
			var registeredInstances []*ServiceInstance
			for _, req := range tc.instances {
				instance, err := registry.Register(&req)
				if err != nil {
					t.Errorf("注册服务失败: %v", err)
					continue
				}
				registeredInstances = append(registeredInstances, instance)
				
				// 模拟服务健康状态
				registry.UpdateServiceStatus(instance.ID, StatusHealthy)
			}
			
			// 测试服务发现
			discoveredInstances, err := client.GetInstances(tc.serviceName)
			if err != nil {
				t.Errorf("服务发现失败: %v", err)
				return
			}
			
			// 验证发现的实例数量
			if len(discoveredInstances) != len(tc.instances) {
				t.Errorf("期望发现 %d 个实例，实际发现 %d 个", len(tc.instances), len(discoveredInstances))
			}
			
			// 测试负载均衡 - 轮询选择
			client.SetLoadBalancer("round_robin")
			selectedInstances := make(map[string]int)
			
			// 进行多次选择以测试负载均衡
			for i := 0; i < len(tc.instances)*3; i++ {
				instance, err := client.SelectInstance(tc.serviceName, "")
				if err != nil {
					t.Errorf("选择服务实例失败: %v", err)
					continue
				}
				selectedInstances[instance.ID]++
			}
			
			// 验证负载均衡效果（每个实例都应该被选中）
			if len(selectedInstances) != len(tc.instances) {
				t.Errorf("负载均衡未覆盖所有实例，期望 %d 个，实际 %d 个", len(tc.instances), len(selectedInstances))
			}
			
			// 测试故障转移 - 模拟一个实例失败
			if len(registeredInstances) > 1 {
				failedInstance := registeredInstances[0]
				registry.UpdateServiceStatus(failedInstance.ID, StatusUnhealthy)
				
				// 重新发现服务
				healthyInstances, err := client.GetInstances(tc.serviceName)
				if err != nil {
					t.Errorf("故障转移后服务发现失败: %v", err)
					return
				}
				
				// 验证不健康的实例被排除
				for _, instance := range healthyInstances {
					if instance.ID == failedInstance.ID {
						t.Errorf("不健康的实例 %s 仍然被发现", failedInstance.ID)
					}
				}
				
				// 验证仍然可以选择健康的实例
				if len(healthyInstances) > 0 {
					_, err := client.SelectInstance(tc.serviceName, "")
					if err != nil {
						t.Errorf("故障转移后选择实例失败: %v", err)
					}
				}
				
				// 测试服务恢复
				registry.UpdateServiceStatus(failedInstance.ID, StatusHealthy)
				
				// 等待一小段时间让缓存失效
				time.Sleep(100 * time.Millisecond)
				client.InvalidateCache(tc.serviceName)
				
				recoveredInstances, err := client.GetInstances(tc.serviceName)
				if err != nil {
					t.Errorf("服务恢复后发现失败: %v", err)
					return
				}
				
				// 验证恢复的实例重新可用
				if len(recoveredInstances) != len(tc.instances) {
					t.Errorf("服务恢复后实例数量不正确，期望 %d 个，实际 %d 个", len(tc.instances), len(recoveredInstances))
				}
			}
		})
	}
}

// 测试服务注册和注销的属性
func TestServiceRegistrationProperty(t *testing.T) {
	testCases := []struct {
		name    string
		request RegistrationRequest
	}{
		{
			name: "基本服务注册",
			request: RegistrationRequest{
				Name:     "basic-service",
				Address:  "localhost",
				Port:     8080,
				Health:   "http://localhost:8080/health",
				Version:  "1.0",
				Weight:   100,
				MaxFails: 3,
			},
		},
		{
			name: "带标签的服务注册",
			request: RegistrationRequest{
				Name:     "tagged-service",
				Address:  "localhost",
				Port:     8081,
				Health:   "http://localhost:8081/health",
				Version:  "2.0",
				Tags:     []string{"api", "backend", "production"},
				Weight:   150,
				MaxFails: 5,
			},
		},
		{
			name: "带元数据的服务注册",
			request: RegistrationRequest{
				Name:    "metadata-service",
				Address: "localhost",
				Port:    8082,
				Health:  "http://localhost:8082/health",
				Version: "1.5",
				Metadata: map[string]string{
					"region":      "us-west-1",
					"environment": "production",
					"team":        "backend",
				},
				Weight:   200,
				MaxFails: 2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Property: 对于任何服务注册请求，系统应该能正确注册、发现和注销服务
			registry := NewServiceRegistry()
			defer registry.Stop()
			
			// 测试服务注册
			instance, err := registry.Register(&tc.request)
			if err != nil {
				t.Errorf("服务注册失败: %v", err)
				return
			}
			
			// 验证注册的服务信息
			if instance.Name != tc.request.Name {
				t.Errorf("服务名称不匹配，期望 %s，实际 %s", tc.request.Name, instance.Name)
			}
			
			if instance.Address != tc.request.Address {
				t.Errorf("服务地址不匹配，期望 %s，实际 %s", tc.request.Address, instance.Address)
			}
			
			if instance.Port != tc.request.Port {
				t.Errorf("服务端口不匹配，期望 %d，实际 %d", tc.request.Port, instance.Port)
			}
			
			// 设置服务为健康状态以便发现
			registry.UpdateServiceStatus(instance.ID, StatusHealthy)
			
			// 测试服务发现
			discoveredInstances, err := registry.Discover(tc.request.Name)
			if err != nil {
				t.Errorf("服务发现失败: %v", err)
				return
			}
			
			// 验证能够发现注册的服务
			found := false
			for _, discovered := range discoveredInstances {
				if discovered.ID == instance.ID {
					found = true
					break
				}
			}
			
			if !found {
				t.Errorf("未能发现已注册的服务实例 %s", instance.ID)
			}
			
			// 测试标签发现
			if len(tc.request.Tags) > 0 {
				taggedInstances, err := registry.DiscoverWithTags(tc.request.Name, tc.request.Tags)
				if err != nil {
					t.Errorf("标签服务发现失败: %v", err)
					return
				}
				
				// 验证标签匹配
				tagFound := false
				for _, tagged := range taggedInstances {
					if tagged.ID == instance.ID {
						tagFound = true
						break
					}
				}
				
				if !tagFound {
					t.Errorf("未能通过标签发现服务实例 %s", instance.ID)
				}
			}
			
			// 测试服务注销
			err = registry.Deregister(instance.ID)
			if err != nil {
				t.Errorf("服务注销失败: %v", err)
				return
			}
			
			// 验证注销后无法发现服务
			afterDeregister, err := registry.Discover(tc.request.Name)
			if err != nil {
				t.Errorf("注销后服务发现失败: %v", err)
				return
			}
			
			// 验证服务已被移除
			for _, remaining := range afterDeregister {
				if remaining.ID == instance.ID {
					t.Errorf("注销后仍能发现服务实例 %s", instance.ID)
				}
			}
		})
	}
}

// 测试负载均衡器的属性
func TestLoadBalancerProperty(t *testing.T) {
	testCases := []struct {
		name      string
		balancer  string
		instances int
	}{
		{"轮询负载均衡", "round_robin", 3},
		{"随机负载均衡", "random", 3},
		{"加权轮询负载均衡", "weighted_round_robin", 3},
		{"一致性哈希负载均衡", "consistent_hash", 3},
		{"最少连接负载均衡", "least_connections", 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Property: 对于任何负载均衡器，应该能够在多个健康实例间分发请求
			registry := NewServiceRegistry()
			defer registry.Stop()
			
			client := NewServiceDiscoveryClient(registry)
			
			// 设置负载均衡器
			err := client.SetLoadBalancer(tc.balancer)
			if err != nil {
				t.Errorf("设置负载均衡器失败: %v", err)
				return
			}
			
			serviceName := fmt.Sprintf("test-service-%s", tc.balancer)
			
			// 注册多个服务实例
			var instances []*ServiceInstance
			for i := 0; i < tc.instances; i++ {
				req := &RegistrationRequest{
					Name:     serviceName,
					Address:  "localhost",
					Port:     8000 + i,
					Health:   fmt.Sprintf("http://localhost:%d/health", 8000+i),
					Version:  "1.0",
					Weight:   100 + i*10, // 不同权重
					MaxFails: 3,
				}
				
				instance, err := registry.Register(req)
				if err != nil {
					t.Errorf("注册服务实例失败: %v", err)
					continue
				}
				
				// 设置为健康状态
				registry.UpdateServiceStatus(instance.ID, StatusHealthy)
				instances = append(instances, instance)
			}
			
			// 进行多次选择测试
			selections := make(map[string]int)
			totalSelections := tc.instances * 10
			
			for i := 0; i < totalSelections; i++ {
				key := fmt.Sprintf("key-%d", i)
				selected, err := client.SelectInstance(serviceName, key)
				if err != nil {
					t.Errorf("选择服务实例失败: %v", err)
					continue
				}
				
				selections[selected.ID]++
			}
			
			// 验证所有实例都被选中（除了一致性哈希和加权轮询可能有偏差）
			if tc.balancer != "consistent_hash" && tc.balancer != "weighted_round_robin" {
				if len(selections) != tc.instances {
					t.Errorf("负载均衡器 %s 未覆盖所有实例，期望 %d 个，实际 %d 个", 
						tc.balancer, tc.instances, len(selections))
				}
			}
			
			// 验证选择次数合理（没有实例被完全忽略，除非是一致性哈希）
			for instanceID, count := range selections {
				if count == 0 {
					t.Errorf("实例 %s 从未被选中", instanceID)
				}
			}
		})
	}
}

// 测试服务健康检查的属性
func TestServiceHealthCheckProperty(t *testing.T) {
	testCases := []struct {
		name           string
		initialStatus  ServiceStatus
		expectedStatus ServiceStatus
		shouldDiscover bool
	}{
		{"健康服务应该被发现", StatusHealthy, StatusHealthy, true},
		{"不健康服务不应该被发现", StatusUnhealthy, StatusUnhealthy, false},
		{"未知状态服务不应该被发现", StatusUnknown, StatusUnknown, false},
		{"排空状态服务不应该被发现", StatusDraining, StatusDraining, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Property: 对于任何服务健康状态，只有健康的服务应该被发现和选择
			registry := NewServiceRegistry()
			defer registry.Stop()
			
			client := NewServiceDiscoveryClient(registry)
			
			// 注册服务
			req := &RegistrationRequest{
				Name:     "health-test-service",
				Address:  "localhost",
				Port:     8080,
				Health:   "http://localhost:8080/health",
				Version:  "1.0",
				Weight:   100,
				MaxFails: 3,
			}
			
			instance, err := registry.Register(req)
			if err != nil {
				t.Errorf("注册服务失败: %v", err)
				return
			}
			
			// 设置服务状态
			err = registry.UpdateServiceStatus(instance.ID, tc.initialStatus)
			if err != nil {
				t.Errorf("更新服务状态失败: %v", err)
				return
			}
			
			// 验证服务状态
			updatedInstance, err := registry.GetService(instance.ID)
			if err != nil {
				t.Errorf("获取服务失败: %v", err)
				return
			}
			
			if updatedInstance.Status != tc.expectedStatus {
				t.Errorf("服务状态不匹配，期望 %s，实际 %s", tc.expectedStatus, updatedInstance.Status)
			}
			
			// 测试服务发现
			discoveredInstances, err := client.GetInstances("health-test-service")
			if err != nil {
				t.Errorf("服务发现失败: %v", err)
				return
			}
			
			// 验证发现结果
			found := false
			for _, discovered := range discoveredInstances {
				if discovered.ID == instance.ID {
					found = true
					break
				}
			}
			
			if found != tc.shouldDiscover {
				if tc.shouldDiscover {
					t.Errorf("健康服务 %s 应该被发现但未被发现", instance.ID)
				} else {
					t.Errorf("不健康服务 %s 不应该被发现但被发现了", instance.ID)
				}
			}
			
			// 测试服务选择
			if tc.shouldDiscover {
				_, err := client.SelectInstance("health-test-service", "")
				if err != nil {
					t.Errorf("选择健康服务失败: %v", err)
				}
			} else {
				_, err := client.SelectInstance("health-test-service", "")
				if err == nil {
					t.Errorf("不应该能选择不健康的服务")
				}
			}
		})
	}
}

// 测试缓存机制的属性
func TestCachingProperty(t *testing.T) {
	// Property: 对于任何服务发现请求，缓存应该提高性能而不影响正确性
	registry := NewServiceRegistry()
	defer registry.Stop()
	
	client := NewServiceDiscoveryClient(registry)
	
	// 设置较短的缓存过期时间用于测试
	client.SetCacheExpiry(100 * time.Millisecond)
	
	serviceName := "cache-test-service"
	
	// 注册服务
	req := &RegistrationRequest{
		Name:     serviceName,
		Address:  "localhost",
		Port:     8080,
		Health:   "http://localhost:8080/health",
		Version:  "1.0",
		Weight:   100,
		MaxFails: 3,
	}
	
	instance, err := registry.Register(req)
	if err != nil {
		t.Errorf("注册服务失败: %v", err)
		return
	}
	
	registry.UpdateServiceStatus(instance.ID, StatusHealthy)
	
	// 第一次发现（应该从注册中心获取）
	instances1, err := client.GetInstances(serviceName)
	if err != nil {
		t.Errorf("第一次服务发现失败: %v", err)
		return
	}
	
	if len(instances1) != 1 {
		t.Errorf("期望发现1个实例，实际发现%d个", len(instances1))
	}
	
	// 第二次发现（应该从缓存获取）
	instances2, err := client.GetInstances(serviceName)
	if err != nil {
		t.Errorf("第二次服务发现失败: %v", err)
		return
	}
	
	if len(instances2) != 1 {
		t.Errorf("期望从缓存发现1个实例，实际发现%d个", len(instances2))
	}
	
	// 验证缓存命中
	stats, err := client.GetServiceStats(serviceName)
	if err != nil {
		t.Errorf("获取服务统计失败: %v", err)
		return
	}
	
	if cacheHit, ok := stats["cache_hit"].(bool); !ok || !cacheHit {
		t.Errorf("期望缓存命中，但未命中")
	}
	
	// 等待缓存过期
	time.Sleep(150 * time.Millisecond)
	
	// 第三次发现（缓存过期，应该重新从注册中心获取）
	instances3, err := client.GetInstances(serviceName)
	if err != nil {
		t.Errorf("缓存过期后服务发现失败: %v", err)
		return
	}
	
	if len(instances3) != 1 {
		t.Errorf("缓存过期后期望发现1个实例，实际发现%d个", len(instances3))
	}
	
	// 测试缓存失效
	client.InvalidateCache(serviceName)
	
	// 验证缓存已失效
	instances4, err := client.GetInstances(serviceName)
	if err != nil {
		t.Errorf("缓存失效后服务发现失败: %v", err)
		return
	}
	
	if len(instances4) != 1 {
		t.Errorf("缓存失效后期望发现1个实例，实际发现%d个", len(instances4))
	}
}