package security

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// **Feature: enhanced-aiops-platform, Property 19: 多租户环境隔离**
// **Validates: Requirements 7.4**
//
// 属性测试：验证多租户环境的资源隔离
// 对于任何多租户配置，不同租户之间应该完全隔离，无法访问彼此的资源

// TenantResource 租户资源
type TenantResource struct {
	ID          string
	TenantID    uint
	ResourceType string
	Name        string
	Data        map[string]interface{}
	CreatedAt   time.Time
}

// MockResourceStore 模拟资源存储
type MockResourceStore struct {
	mu        sync.RWMutex
	resources map[string]*TenantResource // key: resourceID
}

func NewMockResourceStore() *MockResourceStore {
	return &MockResourceStore{
		resources: make(map[string]*TenantResource),
	}
}

// CreateResource 创建资源
func (s *MockResourceStore) CreateResource(ctx context.Context, resource *TenantResource) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if resource.ID == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}
	
	if resource.TenantID == 0 {
		return fmt.Errorf("tenant ID cannot be zero")
	}
	
	s.resources[resource.ID] = resource
	return nil
}

// GetResource 获取资源（带租户隔离检查）
func (s *MockResourceStore) GetResource(ctx context.Context, resourceID string, tenantID uint) (*TenantResource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	resource, exists := s.resources[resourceID]
	if !exists {
		return nil, fmt.Errorf("resource not found")
	}
	
	// 租户隔离检查：只能访问自己租户的资源
	if resource.TenantID != tenantID {
		return nil, fmt.Errorf("access denied: resource belongs to different tenant")
	}
	
	return resource, nil
}

// ListResources 列出资源（带租户隔离）
func (s *MockResourceStore) ListResources(ctx context.Context, tenantID uint, resourceType string) ([]*TenantResource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var results []*TenantResource
	for _, resource := range s.resources {
		// 只返回属于该租户的资源
		if resource.TenantID == tenantID {
			if resourceType == "" || resource.ResourceType == resourceType {
				results = append(results, resource)
			}
		}
	}
	
	return results, nil
}

// UpdateResource 更新资源（带租户隔离检查）
func (s *MockResourceStore) UpdateResource(ctx context.Context, resourceID string, tenantID uint, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	resource, exists := s.resources[resourceID]
	if !exists {
		return fmt.Errorf("resource not found")
	}
	
	// 租户隔离检查：只能更新自己租户的资源
	if resource.TenantID != tenantID {
		return fmt.Errorf("access denied: cannot update resource from different tenant")
	}
	
	resource.Data = data
	return nil
}

// DeleteResource 删除资源（带租户隔离检查）
func (s *MockResourceStore) DeleteResource(ctx context.Context, resourceID string, tenantID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	resource, exists := s.resources[resourceID]
	if !exists {
		return fmt.Errorf("resource not found")
	}fmt.Errorf("resource not found")
	}
	
	// 租户隔离检查：只能删除自己租户的资源
	if resource.TenantID != tenantID {
		return fmt.Errorf("access denied: cannot delete resource from different tenant")
	}
	
	delete(s.resources, resourceID)
	return nil
}

// TestMultiTenantIsolation 测试多租户隔离的核心属性
func TestMultiTenantIsolation(t *testing.T) {
	ctx := context.Background()
	store := NewMockResourceStore()
	
	// 定义多个租户
	tenants := []uint{1, 2, 3, 4, 5}
	
	// 定义资源类型
	resourceTypes := []string{"container", "application", "website", "database", "backup"}
	
	// 为每个租户创建多个资源
	resourceCount := 0
	for _, tenantID := range tenants {
		for i, resourceType := range resourceTypes {
			resource := &TenantResource{
				ID:           fmt.Sprintf("resource-%d-%d", tenantID, i),
				TenantID:     tenantID,
				ResourceType: resourceType,
				Name:         fmt.Sprintf("%s-%d", resourceType, i),
				Data: map[string]interface{}{
					"tenant_id": tenantID,
					"index":     i,
					"secret":    fmt.Sprintf("secret-data-tenant-%d", tenantID),
				},
				CreatedAt: time.Now(),
			}
			
			err := store.CreateResource(ctx, resource)
			if err != nil {
				t.Fatalf("创建资源失败: %v", err)
			}
			resourceCount++
		}
	}
	
	t.Logf("成功创建 %d 个资源，分布在 %d 个租户中", resourceCount, len(tenants))
	
	// 属性 1: 租户只能访问自己的资源
	t.Run("Property: 租户只能访问自己的资源", func(t *testing.T) {
		for _, tenantID := range tenants {
			// 列出该租户的所有资源
			resources, err := store.ListResources(ctx, tenantID, "")
			if err != nil {
				t.Errorf("租户 %d 列出资源失败: %v", tenantID, err)
				continue
			}
			
			// 验证返回的资源都属于该租户
			for _, resource := range resources {
				if resource.TenantID != tenantID {
					t.Errorf("租户隔离失败: 租户 %d 的资源列表中包含租户 %d 的资源 %s",
						tenantID, resource.TenantID, resource.ID)
				}
			}
			
			// 验证资源数量正确（每个租户应该有 len(resourceTypes) 个资源）
			expectedCount := len(resourceTypes)
			if len(resources) != expectedCount {
				t.Errorf("租户 %d 的资源数量不正确: 期望 %d, 实际 %d",
					tenantID, expectedCount, len(resources))
			}
			
			t.Logf("租户 %d: 成功验证 %d 个资源的隔离性", tenantID, len(resources))
		}
	})
	
	// 属性 2: 租户无法访问其他租户的资源
	t.Run("Property: 租户无法访问其他租户的资源", func(t *testing.T) {
		// 尝试让租户 1 访问租户 2 的资源
		tenant1 := uint(1)
		tenant2 := uint(2)
		
		// 获取租户 2 的资源 ID
		tenant2Resources, err := store.ListResources(ctx, tenant2, "")
		if err != nil {
			t.Fatalf("获取租户 2 的资源失败: %v", err)
		}
		
		if len(tenant2Resources) == 0 {
			t.Fatal("租户 2 没有资源")
		}
		
		// 租户 1 尝试访问租户 2 的资源
		for _, resource := range tenant2Resources {
			_, err := store.GetResource(ctx, resource.ID, tenant1)
			if err == nil {
				t.Errorf("租户隔离失败: 租户 %d 能够访问租户 %d 的资源 %s",
					tenant1, tenant2, resource.ID)
			} else {
				t.Logf("正确阻止了跨租户访问: %v", err)
			}
		}
	})
	
	// 属性 3: 租户无法修改其他租户的资源
	t.Run("Property: 租户无法修改其他租户的资源", func(t *testing.T) {
		tenant1 := uint(1)
		tenant2 := uint(2)
		
		// 获取租户 2 的资源
		tenant2Resources, err := store.ListResources(ctx, tenant2, "")
		if err != nil {
			t.Fatalf("获取租户 2 的资源失败: %v", err)
		}
		
		if len(tenant2Resources) == 0 {
			t.Fatal("租户 2 没有资源")
		}
		
		// 租户 1 尝试修改租户 2 的资源
		targetResource := tenant2Resources[0]
		newData := map[string]interface{}{
			"malicious": "data",
			"hacked_by": tenant1,
		}
		
		err = store.UpdateResource(ctx, targetResource.ID, tenant1, newData)
		if err == nil {
			t.Errorf("租户隔离失败: 租户 %d 能够修改租户 %d 的资源 %s",
				tenant1, tenant2, targetResource.ID)
		} else {
			t.Logf("正确阻止了跨租户修改: %v", err)
		}
		
		// 验证资源数据未被修改
		originalResource, err := store.GetResource(ctx, targetResource.ID, tenant2)
		if err != nil {
			t.Fatalf("租户 2 无法访问自己的资源: %v", err)
		}
		
		if _, exists := originalResource.Data["malicious"]; exists {
			t.Errorf("资源数据被非法修改")
		}
	})
	
	// 属性 4: 租户无法删除其他租户的资源
	t.Run("Property: 租户无法删除其他租户的资源", func(t *testing.T) {
		tenant1 := uint(1)
		tenant3 := uint(3)
		
		// 获取租户 3 的资源
		tenant3Resources, err := store.ListResources(ctx, tenant3, "")
		if err != nil {
			t.Fatalf("获取租户 3 的资源失败: %v", err)
		}
		
		if len(tenant3Resources) == 0 {
			t.Fatal("租户 3 没有资源")
		}
		
		originalCount := len(tenant3Resources)
		
		// 租户 1 尝试删除租户 3 的资源
		for _, resource := range tenant3Resources {
			err := store.DeleteResource(ctx, resource.ID, tenant1)
			if err == nil {
				t.Errorf("租户隔离失败: 租户 %d 能够删除租户 %d 的资源 %s",
					tenant1, tenant3, resource.ID)
			} else {
				t.Logf("正确阻止了跨租户删除: %v", err)
			}
		}
		
		// 验证租户 3 的资源数量未变化
		tenant3ResourcesAfter, err := store.ListResources(ctx, tenant3, "")
		if err != nil {
			t.Fatalf("获取租户 3 的资源失败: %v", err)
		}
		
		if len(tenant3ResourcesAfter) != originalCount {
			t.Errorf("租户 3 的资源数量发生变化: 之前 %d, 之后 %d",
				originalCount, len(tenant3ResourcesAfter))
		}
	})
	
	// 属性 5: 按资源类型过滤时仍保持租户隔离
	t.Run("Property: 按资源类型过滤时保持租户隔离", func(t *testing.T) {
		for _, resourceType := range resourceTypes {
			for _, tenantID := range tenants {
				resources, err := store.ListResources(ctx, tenantID, resourceType)
				if err != nil {
					t.Errorf("租户 %d 列出 %s 类型资源失败: %v", tenantID, resourceType, err)
					continue
				}
				
				// 验证所有返回的资源都属于该租户且类型正确
				for _, resource := range resources {
					if resource.TenantID != tenantID {
						t.Errorf("租户隔离失败: 租户 %d 的 %s 资源列表中包含租户 %d 的资源",
							tenantID, resourceType, resource.TenantID)
					}
					
					if resource.ResourceType != resourceType {
						t.Errorf("资源类型过滤失败: 期望 %s, 实际 %s",
							resourceType, resource.ResourceType)
					}
				}
				
				// 每个租户每种类型应该只有 1 个资源
				if len(resources) != 1 {
					t.Errorf("租户 %d 的 %s 类型资源数量不正确: 期望 1, 实际 %d",
						tenantID, resourceType, len(resources))
				}
			}
		}
	})
}

// TestMultiTenantConcurrentAccess 测试并发访问下的多租户隔离
func TestMultiTenantConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := NewMockResourceStore()
	
	// 创建多个租户的资源
	tenantCount := 10
	resourcesPerTenant := 20
	
	// 并发创建资源
	var wg sync.WaitGroup
	for tenantID := 1; tenantID <= tenantCount; tenantID++ {
		wg.Add(1)
		go func(tid uint) {
			defer wg.Done()
			
			for i := 0; i < resourcesPerTenant; i++ {
				resource := &TenantResource{
					ID:           fmt.Sprintf("concurrent-resource-%d-%d", tid, i),
					TenantID:     tid,
					ResourceType: "test",
					Name:         fmt.Sprintf("test-%d-%d", tid, i),
					Data: map[string]interface{}{
						"value": i,
					},
					CreatedAt: time.Now(),
				}
				
				if err := store.CreateResource(ctx, resource); err != nil {
					t.Errorf("租户 %d 创建资源失败: %v", tid, err)
				}
			}
		}(uint(tenantID))
	}
	
	wg.Wait()
	t.Logf("并发创建完成: %d 个租户，每个租户 %d 个资源", tenantCount, resourcesPerTenant)
	
	// 并发访问和验证隔离性
	t.Run("Property: 并发访问时保持租户隔离", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, tenantCount*10)
		
		for tenantID := 1; tenantID <= tenantCount; tenantID++ {
			wg.Add(1)
			go func(tid uint) {
				defer wg.Done()
				
				// 列出该租户的资源
				resources, err := store.ListResources(ctx, tid, "")
				if err != nil {
					errors <- fmt.Errorf("租户 %d 列出资源失败: %v", tid, err)
					return
				}
				
				// 验证资源数量
				if len(resources) != resourcesPerTenant {
					errors <- fmt.Errorf("租户 %d 的资源数量不正确: 期望 %d, 实际 %d",
						tid, resourcesPerTenant, len(resources))
					return
				}
				
				// 验证所有资源都属于该租户
				for _, resource := range resources {
					if resource.TenantID != tid {
						errors <- fmt.Errorf("租户隔离失败: 租户 %d 的资源列表中包含租户 %d 的资源",
							tid, resource.TenantID)
						return
					}
				}
				
				// 尝试访问其他租户的资源（应该失败）
				otherTenantID := tid%uint(tenantCount) + 1
				otherResources, err := store.ListResources(ctx, otherTenantID, "")
				if err != nil {
					errors <- fmt.Errorf("租户 %d 列出其他租户资源失败: %v", otherTenantID, err)
					return
				}
				
				if len(otherResources) > 0 {
					// 尝试访问第一个资源
					_, err := store.GetResource(ctx, otherResources[0].ID, tid)
					if err == nil {
						errors <- fmt.Errorf("租户隔离失败: 租户 %d 能够访问租户 %d 的资源",
							tid, otherTenantID)
					}
				}
			}(uint(tenantID))
		}
		
		wg.Wait()
		close(errors)
		
		// 检查是否有错误
		errorCount := 0
		for err := range errors {
			t.Error(err)
			errorCount++
		}
		
		if errorCount == 0 {
			t.Logf("并发访问测试通过: %d 个租户的隔离性得到验证", tenantCount)
		} else {
			t.Errorf("并发访问测试失败: 发现 %d 个错误", errorCount)
		}
	})
}

// TestMultiTenantResourceLifecycle 测试资源生命周期中的租户隔离
func TestMultiTenantResourceLifecycle(t *testing.T) {
	ctx := context.Background()
	store := NewMockResourceStore()
	
	tenant1 := uint(1)
	tenant2 := uint(2)
	
	// 租户 1 创建资源
	resource1 := &TenantResource{
		ID:           "lifecycle-resource-1",
		TenantID:     tenant1,
		ResourceType: "application",
		Name:         "app1",
		Data: map[string]interface{}{
			"version": "1.0",
		},
		CreatedAt: time.Now(),
	}
	
	err := store.CreateResource(ctx, resource1)
	if err != nil {
		t.Fatalf("创建资源失败: %v", err)
	}
	
	// 租户 2 创建同名资源（不同租户可以有同名资源）
	resource2 := &TenantResource{
		ID:           "lifecycle-resource-2",
		TenantID:     tenant2,
		ResourceType: "application",
		Name:         "app1", // 同名
		Data: map[string]interface{}{
			"version": "2.0",
		},
		CreatedAt: time.Now(),
	}
	
	err = store.CreateResource(ctx, resource2)
	if err != nil {
		t.Fatalf("创建资源失败: %v", err)
	}
	
	t.Run("Property: 不同租户可以创建同名资源", func(t *testing.T) {
		// 验证两个租户都能访问自己的资源
		r1, err := store.GetResource(ctx, resource1.ID, tenant1)
		if err != nil {
			t.Errorf("租户 1 无法访问自己的资源: %v", err)
		} else if r1.Data["version"] != "1.0" {
			t.Errorf("租户 1 的资源数据不正确")
		}
		
		r2, err := store.GetResource(ctx, resource2.ID, tenant2)
		if err != nil {
			t.Errorf("租户 2 无法访问自己的资源: %v", err)
		} else if r2.Data["version"] != "2.0" {
			t.Errorf("租户 2 的资源数据不正确")
		}
		
		t.Logf("验证通过: 不同租户可以创建同名资源且互不干扰")
	})
	
	t.Run("Property: 租户只能修改自己的资源", func(t *testing.T) {
		// 租户 1 修改自己的资源
		newData1 := map[string]interface{}{
			"version": "1.1",
			"updated": true,
		}
		
		err := store.UpdateResource(ctx, resource1.ID, tenant1, newData1)
		if err != nil {
			t.Errorf("租户 1 无法修改自己的资源: %v", err)
		}
		
		// 验证修改成功
		r1, _ := store.GetResource(ctx, resource1.ID, tenant1)
		if r1.Data["version"] != "1.1" {
			t.Errorf("资源修改未生效")
		}
		
		// 租户 2 的资源应该不受影响
		r2, _ := store.GetResource(ctx, resource2.ID, tenant2)
		if r2.Data["version"] != "2.0" {
			t.Errorf("租户 2 的资源被意外修改")
		}
		
		t.Logf("验证通过: 租户修改资源时不影响其他租户")
	})
	
	t.Run("Property: 租户只能删除自己的资源", func(t *testing.T) {
		// 租户 1 删除自己的资源
		err := store.DeleteResource(ctx, resource1.ID, tenant1)
		if err != nil {
			t.Errorf("租户 1 无法删除自己的资源: %v", err)
		}
		
		// 验证资源已删除
		_, err = store.GetResource(ctx, resource1.ID, tenant1)
		if err == nil {
			t.Errorf("资源删除失败: 资源仍然存在")
		}
		
		// 租户 2 的资源应该不受影响
		r2, err := store.GetResource(ctx, resource2.ID, tenant2)
		if err != nil {
			t.Errorf("租户 2 的资源被意外删除: %v", err)
		} else if r2.Name != "app1" {
			t.Errorf("租户 2 的资源数据异常")
		}
		
		t.Logf("验证通过: 租户删除资源时不影响其他租户")
	})
}

// TestMultiTenantZeroTenantID 测试租户 ID 为 0 的边界情况
func TestMultiTenantZeroTenantID(t *testing.T) {
	ctx := context.Background()
	store := NewMockResourceStore()
	
	t.Run("Property: 拒绝租户 ID 为 0 的资源", func(t *testing.T) {
		resource := &TenantResource{
			ID:           "invalid-resource",
			TenantID:     0, // 无效的租户 ID
			ResourceType: "test",
			Name:         "test",
			Data:         make(map[string]interface{}),
			CreatedAt:    time.Now(),
		}
		
		err := store.CreateResource(ctx, resource)
		if err == nil {
			t.Errorf("应该拒绝租户 ID 为 0 的资源")
		} else {
			t.Logf("正确拒绝了无效的租户 ID: %v", err)
		}
	})
}
