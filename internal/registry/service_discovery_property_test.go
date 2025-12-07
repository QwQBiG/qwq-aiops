package registry

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// **Feature: enhanced-aiops-platform, Property 26: 集群部署高可用性*
// **Validates: Requirements 10.1, 10.3**
//
// Property: 对于任何集群部署配置，系统应该支持负载均衡、故障转移和服务恢复
// 这个属性测试验证以下关键特性：
// 1. 负载均衡：多个实例之间能够均匀分配请求
// 2. 故障转移：当实例失败时，系统能够自动切换到健康实�?
// 3. 服务恢复：失败的实例恢复后能够重新加入服务池
func TestServiceDiscoveryHighAvailabilityProperty(t *testing.T) {
	// 初始化随机数生成�?
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Property 1: 服务注册后必须能够被发现
	t.Run("注册的健康服务必须能够被发现", func(t *testing.T) {
		// 运行100次随机测�?
		for iteration := 0; iteration < 100; iteration++ {
			serviceName := fmt.Sprintf("test-service-%d", rnd.Intn(1000))
			instanceCount := rnd.Intn(10) + 1 // 1-10个实�?
			registry := NewServiceRegistry()
			defer registry.Stop()

			// 注册多个服务实例
			registeredIDs := make([]string, 0, instanceCount)
			for i := 0; i < instanceCount; i++ {
				req := &RegistrationRequest{
					Name:     serviceName,
					Address:  "localhost",
					Port:     8000 + i,
					Health:   fmt.Sprintf("http://localhost:%d/health", 8000+i),
					Version:  "1.0",
					Weight:   100,
					MaxFails: 3,
				}

				instance, err := registry.Register(req)
				if err != nil {
					return false
				}

				// 设置为健康状�?
				registry.UpdateServiceStatus(instance.ID, StatusHealthy)
				registeredIDs = append(registeredIDs, instance.ID)
			}

			// 验证所有实例都能被发现
			discoveredInstances, err := registry.Discover(serviceName)
			if err != nil {
				return false
			}

			// 验证发现的实例数量正�?
			if len(discoveredInstances) != instanceCount {
				return false
			}

			// 验证所有注册的实例都在发现列表�?
			discoveredMap := make(map[string]bool)
			for _, instance := range discoveredInstances {
				discoveredMap[instance.ID] = true
			}

			for _, id := range registeredIDs {
				if !discoveredMap[id] {
					return false
				}
			}

			return true
		},
		gen.Identifier().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(1, 10)gen.IntRange(1, 10),
	))

	// Property 2: 负载均衡器必须在所有健康实例间分配请求
	properties.Property("负载均衡器必须覆盖所有健康实�?, prop.ForAll(
		func(serviceName string, instanceCount int, requestCount int) bool {
			if instanceCount == 0 || requestCount == 0 {
				return true // 跳过无效输入
			}

			registry := NewServiceRegistry()
			defer registry.Stop()

			client := NewServiceDiscoveryClient(registry)
			client.SetLoadBalancer("round_robin")

			// 注册多个服务实例
			for i := 0; i < instanceCount; i++ {
				req := &RegistrationRequest{
					Name:     serviceName,
					Address:  "localhost",
					Port:     8000 + i,
					Health:   fmt.Sprintf("http://localhost:%d/health", 8000+i),
					Version:  "1.0",
					Weight:   100,
					MaxFails: 3,
				}

				instance, err := registry.Register(req)
				if err != nil {
					return false
				}

				registry.UpdateServiceStatus(instance.ID, StatusHealthy)
			}

			// 进行多次请求选择
			selectedInstances := make(map[string]int)
			for i := 0; i < requestCount; i++ {
				instance, err := client.SelectInstance(serviceName, "")
				if err != nil {
					return false
				}
				selectedInstances[instance.ID]++
			}

			// 验证所有实例都被选中至少一次（如果请求数足够）
			if requestCount >= instanceCount {
				if len(selectedInstances) != instanceCount {
					return false
				}
			}

			return true
		},
		gen.Identifier().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(1, 5),
		gen.IntRange(10, 50),
	))

	// Property 3: 故障转移 - 不健康的实例不应该被选中
	properties.Property("不健康的实例不应该被发现和选择", prop.ForAll(
		func(serviceName string, healthyCount int, unhealthyCount int) bool {
			if healthyCount == 0 {
				return true // 至少需要一个健康实�?
			}

			registry := NewServiceRegistry()
			defer registry.Stop()

			client := NewServiceDiscoveryClient(registry)

			// 注册健康实例
			healthyIDs := make([]string, 0, healthyCount)
			for i := 0; i < healthyCount; i++ {
				req := &RegistrationRequest{
					Name:     serviceName,
					Address:  "localhost",
					Port:     8000 + i,
					Health:   fmt.Sprintf("http://localhost:%d/health", 8000+i),
					Version:  "1.0",
					Weight:   100,
					MaxFails: 3,
				}

				instance, err := registry.Register(req)
				if err != nil {
					return false
				}

				registry.UpdateServiceStatus(instance.ID, StatusHealthy)
				healthyIDs = append(healthyIDs, instance.ID)
			}

			// 注册不健康实�?
			unhealthyIDs := make([]string, 0, unhealthyCount)
			for i := 0; i < unhealthyCount; i++ {
				req := &RegistrationRequest{
					Name:     serviceName,
					Address:  "localhost",
					Port:     9000 + i,
					Health:   fmt.Sprintf("http://localhost:%d/health", 9000+i),
					Version:  "1.0",
					Weight:   100,
					MaxFails: 3,
				}

				instance, err := registry.Register(req)
				if err != nil {
					return false
				}

				registry.UpdateServiceStatus(instance.ID, StatusUnhealthy)
				unhealthyIDs = append(unhealthyIDs, instance.ID)
			}

			// 验证只能发现健康实例
			discoveredInstances, err := client.GetInstances(serviceName)
			if err != nil {
				return false
			}

			if len(discoveredInstances) != healthyCount {
				return false
			}

			// 验证发现的都是健康实�?
			for _, instance := range discoveredInstances {
				found := false
				for _, healthyID := range healthyIDs {
					if instance.ID == healthyID {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}

			// 验证选择的实例都是健康的
			for i := 0; i < 10; i++ {
				selected, err := client.SelectInstance(serviceName, "")
				if err != nil {
					return false
				}

				// 确保选中的不是不健康实例
				for _, unhealthyID := range unhealthyIDs {
					if selected.ID == unhealthyID {
						return false
					}
				}
			}

			return true
		},
		gen.Identifier().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(1, 5),
		gen.IntRange(0, 5),
	))

	// Property 4: 服务恢复 - 失败的实例恢复后应该重新可用
	properties.Property("恢复的实例应该重新加入服务池", prop.ForAll(
		func(serviceName string, instanceCount int) bool {
			if instanceCount < 2 {
				return true // 至少需�?个实例来测试恢复
			}

			registry := NewServiceRegistry()
			defer registry.Stop()

			client := NewServiceDiscoveryClient(registry)

			// 注册多个服务实例
			instances := make([]*ServiceInstance, 0, instanceCount)
			for i := 0; i < instanceCount; i++ {
				req := &RegistrationRequest{
					Name:     serviceName,
					Address:  "localhost",
					Port:     8000 + i,
					Health:   fmt.Sprintf("http://localhost:%d/health", 8000+i),
					Version:  "1.0",
					Weight:   100,
					MaxFails: 3,
				}

				instance, err := registry.Register(req)
				if err != nil {
					return false
				}

				registry.UpdateServiceStatus(instance.ID, StatusHealthy)
				instances = append(instances, instance)
			}

			// 模拟第一个实例失�?
			failedInstance := instances[0]
			registry.UpdateServiceStatus(failedInstance.ID, StatusUnhealthy)

			// 清除缓存以获取最新状�?
			client.InvalidateCache(serviceName)

			// 验证失败后的实例数量
			afterFailure, err := client.GetInstances(serviceName)
			if err != nil {
				return false
			}

			if len(afterFailure) != instanceCount-1 {
				return false
			}

			// 恢复失败的实�?
			registry.UpdateServiceStatus(failedInstance.ID, StatusHealthy)

			// 清除缓存
			time.Sleep(10 * time.Millisecond)
			client.InvalidateCache(serviceName)

			// 验证恢复后的实例数量
			afterRecovery, err := client.GetInstances(serviceName)
			if err != nil {
				return false
			}

			if len(afterRecovery) != instanceCount {
				return false
			}

			// 验证恢复的实例在列表�?
			found := false
			for _, instance := range afterRecovery {
				if instance.ID == failedInstance.ID {
					found = true
					break
				}
			}

			return found
		},
		gen.Identifier().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(2, 8),
	))

	// Property 5: 服务注销后不应该被发�?
	properties.Property("注销的服务不应该被发�?, prop.ForAll(
		func(serviceName string, instanceCount int, deregisterIndex int) bool {
			if instanceCount == 0 {
				return true
			}

			// 确保索引有效
			deregisterIndex = deregisterIndex % instanceCount

			registry := NewServiceRegistry()
			defer registry.Stop()

			// 注册多个服务实例
			instances := make([]*ServiceInstance, 0, instanceCount)
			for i := 0; i < instanceCount; i++ {
				req := &RegistrationRequest{
					Name:     serviceName,
					Address:  "localhost",
					Port:     8000 + i,
					Health:   fmt.Sprintf("http://localhost:%d/health", 8000+i),
					Version:  "1.0",
					Weight:   100,
					MaxFails: 3,
				}

				instance, err := registry.Register(req)
				if err != nil {
					return false
				}

				registry.UpdateServiceStatus(instance.ID, StatusHealthy)
				instances = append(instances, instance)
			}

			// 注销指定的实�?
			deregisteredInstance := instances[deregisterIndex]
			err := registry.Deregister(deregisteredInstance.ID)
			if err != nil {
				return false
			}

			// 验证注销后的实例数量
			afterDeregister, err := registry.Discover(serviceName)
			if err != nil {
				return false
			}

			if len(afterDeregister) != instanceCount-1 {
				return false
			}

			// 验证注销的实例不在列表中
			for _, instance := range afterDeregister {
				if instance.ID == deregisteredInstance.ID {
					return false
				}
			}

			return true
		},
		gen.Identifier().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(1, 10),
		gen.IntRange(0, 100),
	))

	// Property 6: 标签过滤必须正确工作
	properties.Property("标签过滤必须返回匹配的实�?, prop.ForAll(
		func(serviceName string, withTagCount int, withoutTagCount int) bool {
			registry := NewServiceRegistry()
			defer registry.Stop()

			targetTag := "production"

			// 注册带标签的实例
			for i := 0; i < withTagCount; i++ {
				req := &RegistrationRequest{
					Name:     serviceName,
					Address:  "localhost",
					Port:     8000 + i,
					Health:   fmt.Sprintf("http://localhost:%d/health", 8000+i),
					Version:  "1.0",
					Tags:     []string{targetTag, "backend"},
					Weight:   100,
					MaxFails: 3,
				}

				instance, err := registry.Register(req)
				if err != nil {
					return false
				}

				registry.UpdateServiceStatus(instance.ID, StatusHealthy)
			}

			// 注册不带标签的实�?
			for i := 0; i < withoutTagCount; i++ {
				req := &RegistrationRequest{
					Name:     serviceName,
					Address:  "localhost",
					Port:     9000 + i,
					Health:   fmt.Sprintf("http://localhost:%d/health", 9000+i),
					Version:  "1.0",
					Tags:     []string{"development"},
					Weight:   100,
					MaxFails: 3,
				}

				instance, err := registry.Register(req)
				if err != nil {
					return false
				}

				registry.UpdateServiceStatus(instance.ID, StatusHealthy)
			}

			// 使用标签过滤
			taggedInstances, err := registry.DiscoverWithTags(serviceName, []string{targetTag})
			if err != nil {
				return false
			}

			// 验证返回的实例数�?
			if len(taggedInstances) != withTagCount {
				return false
			}

			// 验证所有返回的实例都有目标标签
			for _, instance := range taggedInstances {
				hasTag := false
				for _, tag := range instance.Tags {
					if tag == targetTag {
						hasTag = true
						break
					}
				}
				if !hasTag {
					return false
				}
			}

			return true
		},
		gen.Identifier().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(0, 5),
		gen.IntRange(0, 5),
	))

	// 运行所有属性测试，每个属性测�?00�?
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestServiceDiscoveryConsistencyProperty 测试服务发现的一致性属�?
func TestServiceDiscoveryConsistencyProperty(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: 并发注册和发现必须保持一致�?
	properties.Property("并发操作必须保持数据一致�?, prop.ForAll(
		func(serviceName string, operationCount int) bool {
			if operationCount == 0 {
				return true
			}

			registry := NewServiceRegistry()
			defer registry.Stop()

			// 并发注册服务
			done := make(chan bool, operationCount)
			registeredIDs := make(chan string, operationCount)

			for i := 0; i < operationCount; i++ {
				go func(index int) {
					req := &RegistrationRequest{
						Name:     serviceName,
						Address:  "localhost",
						Port:     8000 + index,
						Health:   fmt.Sprintf("http://localhost:%d/health", 8000+index),
						Version:  "1.0",
						Weight:   100,
						MaxFails: 3,
					}

					instance, err := registry.Register(req)
					if err == nil {
						registry.UpdateServiceStatus(instance.ID, StatusHealthy)
						registeredIDs <- instance.ID
					}
					done <- true
				}(i)
			}

			// 等待所有注册完�?
			for i := 0; i < operationCount; i++ {
				<-done
			}
			close(registeredIDs)

			// 收集所有注册的ID
			expectedIDs := make(map[string]bool)
			for id := range registeredIDs {
				expectedIDs[id] = true
			}

			// 验证所有注册的实例都能被发�?
			discoveredInstances, err := registry.Discover(serviceName)
			if err != nil {
				return false
			}

			if len(discoveredInstances) != len(expectedIDs) {
				return false
			}

			// 验证发现的实例ID都在预期列表�?
			for _, instance := range discoveredInstances {
				if !expectedIDs[instance.ID] {
					return false
				}
			}

			return true
		},
		gen.Identifier().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(1, 20),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
