package appstore

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: enhanced-aiops-platform, Property 4: 应用安装冲突解决**
// **Validates: Requirements 2.3**
//
// Property 4: 应用安装冲突解决
// *For any* 应用安装请求，系统应该能自动检测并解决端口冲突、数据卷挂载等问题
//
// 这个属性测试验证：
// 1. 系统能够检测端口冲突
// 2. 系统能够检测数据卷冲突
// 3. 检测到的冲突包含正确的冲突信息
// 4. 可解决的冲突被标记为可解决
// 5. 冲突检测在所有安装请求中都有效

// mockAppStoreService 模拟应用商店服务
type mockAppStoreService struct {
	templates map[uint]*AppTemplate
	instances []*ApplicationInstance
}

func newMockAppStoreService() *mockAppStoreService {
	return &mockAppStoreService{
		templates: make(map[uint]*AppTemplate),
		instances: []*ApplicationInstance{},
	}
}

func (m *mockAppStoreService) GetTemplate(ctx context.Context, id uint) (*AppTemplate, error) {
	if template, ok := m.templates[id]; ok {
		return template, nil
	}
	return nil, fmt.Errorf("template not found: %d", id)
}

func (m *mockAppStoreService) RenderTemplate(ctx context.Context, templateID uint, params map[string]interface{}) (string, error) {
	template, err := m.GetTemplate(ctx, templateID)
	if err != nil {
		return "", err
	}
	
	// 简单的模板渲染：将参数注入到模板内容中
	rendered := template.Content
	
	// 如果参数中有端口，替换模板中的端口占位符
	if port, ok := params["port"]; ok {
		// 将端口转换为字符串
		portStr := fmt.Sprintf("%v", port)
		// 使用字符串替换而不是 fmt.Sprintf
		rendered = fmt.Sprintf(template.Content, portStr)
	}
	
	return rendered, nil
}

func (m *mockAppStoreService) ListInstances(ctx context.Context, userID, tenantID uint) ([]*ApplicationInstance, error) {
	return m.instances, nil
}

func (m *mockAppStoreService) CreateInstance(ctx context.Context, instance *ApplicationInstance) error {
	instance.ID = uint(len(m.instances) + 1)
	m.instances = append(m.instances, instance)
	return nil
}

func (m *mockAppStoreService) GetInstance(ctx context.Context, id uint) (*ApplicationInstance, error) {
	for _, instance := range m.instances {
		if instance.ID == id {
			return instance, nil
		}
	}
	return nil, fmt.Errorf("instance not found: %d", id)
}

func (m *mockAppStoreService) UpdateInstance(ctx context.Context, instance *ApplicationInstance) error {
	for i, inst := range m.instances {
		if inst.ID == instance.ID {
			m.instances[i] = instance
			return nil
		}
	}
	return fmt.Errorf("instance not found: %d", instance.ID)
}

func (m *mockAppStoreService) DeleteInstance(ctx context.Context, id uint) error {
	for i, instance := range m.instances {
		if instance.ID == id {
			m.instances = append(m.instances[:i], m.instances[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("instance not found: %d", id)
}

func (m *mockAppStoreService) ValidateTemplate(ctx context.Context, template *AppTemplate) error {
	return nil
}

func (m *mockAppStoreService) CreateTemplate(ctx context.Context, template *AppTemplate) error {
	template.ID = uint(len(m.templates) + 1)
	m.templates[template.ID] = template
	return nil
}

func (m *mockAppStoreService) GetTemplateByName(ctx context.Context, name string) (*AppTemplate, error) {
	for _, template := range m.templates {
		if template.Name == name {
			return template, nil
		}
	}
	return nil, fmt.Errorf("template not found: %s", name)
}

func (m *mockAppStoreService) ListTemplates(ctx context.Context, category AppCategory, status TemplateStatus) ([]*AppTemplate, error) {
	var result []*AppTemplate
	for _, template := range m.templates {
		result = append(result, template)
	}
	return result, nil
}

func (m *mockAppStoreService) UpdateTemplate(ctx context.Context, template *AppTemplate) error {
	m.templates[template.ID] = template
	return nil
}

func (m *mockAppStoreService) DeleteTemplate(ctx context.Context, id uint) error {
	delete(m.templates, id)
	return nil
}

func (m *mockAppStoreService) InitBuiltinTemplates(ctx context.Context) error {
	return nil
}

// GetRecommendations 获取应用推荐（mock 实现）
func (m *mockAppStoreService) GetRecommendations(ctx context.Context, userContext *UserContext, limit int) ([]*AppRecommendation, error) {
	return []*AppRecommendation{}, nil
}

// RecordUserBehavior 记录用户行为（mock 实现）
func (m *mockAppStoreService) RecordUserBehavior(ctx context.Context, behavior *UserBehavior) error {
	return nil
}

// RecordUserFeedback 记录用户反馈（mock 实现）
func (m *mockAppStoreService) RecordUserFeedback(ctx context.Context, feedback *UserFeedback) error {
	return nil
}

// 创建一个使用特定端口的 Docker Compose 模板
func createDockerComposeTemplate(templateID uint, serviceName string) *AppTemplate {
	content := `version: '3'
services:
  ` + serviceName + `:
    image: nginx:latest
    ports:
      - "%s:80"
    volumes:
      - data:/var/www/html
volumes:
  data:`
	
	return &AppTemplate{
		ID:      templateID,
		Name:    serviceName,
		Type:    TemplateTypeDockerCompose,
		Version: "1.0.0",
		Content: content,
		Status:  TemplateStatusPublished,
	}
}

// 创建一个使用特定数据卷的 Docker Compose 模板
func createDockerComposeTemplateWithVolume(templateID uint, serviceName, volumeName string) *AppTemplate {
	content := `version: '3'
services:
  ` + serviceName + `:
    image: nginx:latest
    ports:
      - "8080:80"
    volumes:
      - ` + volumeName + `:/var/www/html
volumes:
  ` + volumeName + `:`
	
	return &AppTemplate{
		ID:      templateID,
		Name:    serviceName,
		Type:    TemplateTypeDockerCompose,
		Version: "1.0.0",
		Content: content,
		Status:  TemplateStatusPublished,
	}
}

// TestProperty4_ConflictDetection_PortConflicts 测试端口冲突检测
func TestProperty4_ConflictDetection_PortConflicts(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	// Property 1: 当两个应用使用相同端口时，应该检测到端口冲突
	properties.Property("检测端口冲突", prop.ForAll(
		func(port int) bool {
			ctx := context.Background()
			mockService := newMockAppStoreService()
			
			// 创建第一个模板和实例（使用指定端口）
			template1 := createDockerComposeTemplate(1, "nginx1")
			mockService.templates[1] = template1
			
			params1 := map[string]interface{}{"port": port}
			configJSON1, _ := json.Marshal(params1)
			
			instance1 := &ApplicationInstance{
				ID:         1,
				Name:       "nginx-instance-1",
				TemplateID: 1,
				Status:     "running",
				Config:     string(configJSON1),
			}
			mockService.instances = append(mockService.instances, instance1)
			
			// 创建第二个模板（尝试使用相同端口）
			template2 := createDockerComposeTemplate(2, "nginx2")
			mockService.templates[2] = template2
			
			params2 := map[string]interface{}{"port": port}
			
			// 检测冲突
			checker := NewConflictChecker(mockService)
			conflicts, err := checker.DetectConflicts(ctx, 2, params2)
			
			if err != nil {
				t.Logf("冲突检测错误: %v", err)
				return false
			}
			
			// 应该检测到至少一个端口冲突
			hasPortConflict := false
			for _, conflict := range conflicts {
				if conflict.Type == "port" && conflict.Resource == fmt.Sprintf("%d", port) {
					hasPortConflict = true
					// 验证冲突信息完整性
					if conflict.ExistingApp == "" {
						t.Logf("冲突信息缺少现有应用名称")
						return false
					}
					// 端口冲突应该是可解决的
					if !conflict.Resolvable {
						t.Logf("端口冲突应该标记为可解决")
						return false
					}
					// 应该提供解决建议
					if len(conflict.Suggestions) == 0 {
						t.Logf("冲突应该提供解决建议")
						return false
					}
				}
			}
			
			return hasPortConflict
		},
		gen.IntRange(8000, 9000), // 测试端口范围
	))
	
	// Property 2: 当应用使用不同端口时，不应该检测到冲突
	properties.Property("不同端口不冲突", prop.ForAll(
		func(port1, port2 int) bool {
			// 确保端口不同
			if port1 == port2 {
				return true // 跳过相同端口的情况
			}
			
			ctx := context.Background()
			mockService := newMockAppStoreService()
			
			// 创建第一个实例（使用 port1）
			template1 := createDockerComposeTemplate(1, "nginx1")
			mockService.templates[1] = template1
			
			params1 := map[string]interface{}{"port": port1}
			configJSON1, _ := json.Marshal(params1)
			
			instance1 := &ApplicationInstance{
				ID:         1,
				Name:       "nginx-instance-1",
				TemplateID: 1,
				Status:     "running",
				Config:     string(configJSON1),
			}
			mockService.instances = append(mockService.instances, instance1)
			
			// 创建第二个模板（使用 port2）
			template2 := createDockerComposeTemplate(2, "nginx2")
			mockService.templates[2] = template2
			
			params2 := map[string]interface{}{"port": port2}
			
			// 检测冲突
			checker := NewConflictChecker(mockService)
			conflicts, err := checker.DetectConflicts(ctx, 2, params2)
			
			if err != nil {
				t.Logf("冲突检测错误: %v", err)
				return false
			}
			
			// 不应该检测到端口冲突
			for _, conflict := range conflicts {
				if conflict.Type == "port" {
					t.Logf("不应该检测到端口冲突: %v", conflict)
					return false
				}
			}
			
			return true
		},
		gen.IntRange(8000, 8500),
		gen.IntRange(8501, 9000),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty4_ConflictDetection_VolumeConflicts 测试数据卷冲突检测
func TestProperty4_ConflictDetection_VolumeConflicts(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	// Property 3: 当两个应用使用相同数据卷时，应该检测到数据卷冲突
	properties.Property("检测数据卷冲突", prop.ForAll(
		func(volumeName string) bool {
			ctx := context.Background()
			mockService := newMockAppStoreService()
			
			// 创建第一个实例（使用指定数据卷）
			template1 := createDockerComposeTemplateWithVolume(1, "app1", volumeName)
			mockService.templates[1] = template1
			
			params1 := map[string]interface{}{}
			configJSON1, _ := json.Marshal(params1)
			
			instance1 := &ApplicationInstance{
				ID:         1,
				Name:       "app-instance-1",
				TemplateID: 1,
				Status:     "running",
				Config:     string(configJSON1),
			}
			mockService.instances = append(mockService.instances, instance1)
			
			// 创建第二个模板（尝试使用相同数据卷）
			template2 := createDockerComposeTemplateWithVolume(2, "app2", volumeName)
			mockService.templates[2] = template2
			
			params2 := map[string]interface{}{}
			
			// 检测冲突
			checker := NewConflictChecker(mockService)
			conflicts, err := checker.DetectConflicts(ctx, 2, params2)
			
			if err != nil {
				t.Logf("冲突检测错误: %v", err)
				return false
			}
			
			// 应该检测到至少一个数据卷冲突
			hasVolumeConflict := false
			for _, conflict := range conflicts {
				if conflict.Type == "volume" && conflict.Resource == volumeName {
					hasVolumeConflict = true
					// 验证冲突信息完整性
					if conflict.ExistingApp == "" {
						t.Logf("冲突信息缺少现有应用名称")
						return false
					}
					// 数据卷冲突应该是可解决的
					if !conflict.Resolvable {
						t.Logf("数据卷冲突应该标记为可解决")
						return false
					}
					// 应该提供解决建议
					if len(conflict.Suggestions) == 0 {
						t.Logf("冲突应该提供解决建议")
						return false
					}
				}
			}
			
			return hasVolumeConflict
		},
		gen.Identifier(), // 生成有效的标识符作为数据卷名称
	))
	
	// Property 4: 当应用使用不同数据卷时，不应该检测到冲突
	properties.Property("不同数据卷不冲突", prop.ForAll(
		func(volume1, volume2 string) bool {
			// 确保数据卷名称不同
			if volume1 == volume2 {
				return true // 跳过相同数据卷的情况
			}
			
			ctx := context.Background()
			mockService := newMockAppStoreService()
			
			// 创建第一个实例（使用 volume1）
			template1 := createDockerComposeTemplateWithVolume(1, "app1", volume1)
			mockService.templates[1] = template1
			
			params1 := map[string]interface{}{}
			configJSON1, _ := json.Marshal(params1)
			
			instance1 := &ApplicationInstance{
				ID:         1,
				Name:       "app-instance-1",
				TemplateID: 1,
				Status:     "running",
				Config:     string(configJSON1),
			}
			mockService.instances = append(mockService.instances, instance1)
			
			// 创建第二个模板（使用 volume2）
			template2 := createDockerComposeTemplateWithVolume(2, "app2", volume2)
			mockService.templates[2] = template2
			
			params2 := map[string]interface{}{}
			
			// 检测冲突
			checker := NewConflictChecker(mockService)
			conflicts, err := checker.DetectConflicts(ctx, 2, params2)
			
			if err != nil {
				t.Logf("冲突检测错误: %v", err)
				return false
			}
			
			// 不应该检测到数据卷冲突
			for _, conflict := range conflicts {
				if conflict.Type == "volume" {
					t.Logf("不应该检测到数据卷冲突: %v", conflict)
					return false
				}
			}
			
			return true
		},
		gen.Identifier(),
		gen.Identifier(),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty4_ConflictResolution 测试冲突解决
func TestProperty4_ConflictResolution(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	// Property 5: 查找可用端口应该返回未被使用的端口
	properties.Property("查找可用端口", prop.ForAll(
		func(usedPorts []int, startPort int) bool {
			// 确保 startPort 在有效范围内
			if startPort < 1024 || startPort > 65000 {
				return true // 跳过无效的起始端口
			}
			
			ctx := context.Background()
			mockService := newMockAppStoreService()
			
			// 去重端口列表
			uniquePorts := make(map[int]bool)
			for _, port := range usedPorts {
				if port >= 1024 && port <= 65535 {
					uniquePorts[port] = true
				}
			}
			
			// 创建使用指定端口的实例
			i := 0
			for port := range uniquePorts {
				i++
				template := createDockerComposeTemplate(uint(i), fmt.Sprintf("app%d", i))
				mockService.templates[uint(i)] = template
				
				params := map[string]interface{}{"port": port}
				configJSON, _ := json.Marshal(params)
				
				instance := &ApplicationInstance{
					ID:         uint(i),
					Name:       fmt.Sprintf("app-instance-%d", i),
					TemplateID: uint(i),
					Status:     "running",
					Config:     string(configJSON),
				}
				mockService.instances = append(mockService.instances, instance)
			}
			
			// 查找可用端口
			checker := NewConflictChecker(mockService)
			availablePort, err := checker.FindAvailablePort(ctx, startPort)
			
			if err != nil {
				t.Logf("查找可用端口错误: %v", err)
				return false
			}
			
			// 验证返回的端口未被使用
			if uniquePorts[availablePort] {
				t.Logf("返回的端口 %d 已被使用", availablePort)
				return false
			}
			
			// 验证返回的端口在有效范围内
			if availablePort < startPort || availablePort > 65535 {
				t.Logf("返回的端口 %d 不在有效范围内", availablePort)
				return false
			}
			
			return true
		},
		gen.SliceOfN(5, gen.IntRange(8000, 9000)), // 生成5个已使用的端口
		gen.IntRange(8000, 9000), // 起始端口
	))
	
	// Property 6: 冲突检测的一致性 - 相同的输入应该产生相同的结果
	properties.Property("冲突检测结果一致性", prop.ForAll(
		func(port int) bool {
			ctx := context.Background()
			mockService := newMockAppStoreService()
			
			// 创建一个使用指定端口的实例
			template1 := createDockerComposeTemplate(1, "nginx1")
			mockService.templates[1] = template1
			
			params1 := map[string]interface{}{"port": port}
			configJSON1, _ := json.Marshal(params1)
			
			instance1 := &ApplicationInstance{
				ID:         1,
				Name:       "nginx-instance-1",
				TemplateID: 1,
				Status:     "running",
				Config:     string(configJSON1),
			}
			mockService.instances = append(mockService.instances, instance1)
			
			// 创建第二个模板
			template2 := createDockerComposeTemplate(2, "nginx2")
			mockService.templates[2] = template2
			
			params2 := map[string]interface{}{"port": port}
			
			// 第一次检测冲突
			checker := NewConflictChecker(mockService)
			conflicts1, err1 := checker.DetectConflicts(ctx, 2, params2)
			
			// 第二次检测冲突（相同的输入）
			conflicts2, err2 := checker.DetectConflicts(ctx, 2, params2)
			
			// 两次检测应该产生相同的结果
			if (err1 == nil) != (err2 == nil) {
				t.Logf("两次检测的错误状态不一致")
				return false
			}
			
			if len(conflicts1) != len(conflicts2) {
				t.Logf("两次检测的冲突数量不一致: %d vs %d", len(conflicts1), len(conflicts2))
				return false
			}
			
			return true
		},
		gen.IntRange(8000, 9000),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty4_ConflictDetection_StoppedInstances 测试已停止实例不应该产生冲突
func TestProperty4_ConflictDetection_StoppedInstances(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	// Property 7: 已停止的实例不应该产生端口冲突
	properties.Property("已停止实例不产生冲突", prop.ForAll(
		func(port int) bool {
			ctx := context.Background()
			mockService := newMockAppStoreService()
			
			// 创建一个已停止的实例（使用指定端口）
			template1 := createDockerComposeTemplate(1, "nginx1")
			mockService.templates[1] = template1
			
			params1 := map[string]interface{}{"port": port}
			configJSON1, _ := json.Marshal(params1)
			
			instance1 := &ApplicationInstance{
				ID:         1,
				Name:       "nginx-instance-1",
				TemplateID: 1,
				Status:     "stopped", // 已停止
				Config:     string(configJSON1),
			}
			mockService.instances = append(mockService.instances, instance1)
			
			// 创建第二个模板（尝试使用相同端口）
			template2 := createDockerComposeTemplate(2, "nginx2")
			mockService.templates[2] = template2
			
			params2 := map[string]interface{}{"port": port}
			
			// 检测冲突
			checker := NewConflictChecker(mockService)
			conflicts, err := checker.DetectConflicts(ctx, 2, params2)
			
			if err != nil {
				t.Logf("冲突检测错误: %v", err)
				return false
			}
			
			// 不应该检测到端口冲突（因为第一个实例已停止）
			for _, conflict := range conflicts {
				if conflict.Type == "port" {
					t.Logf("已停止的实例不应该产生端口冲突")
					return false
				}
			}
			
			return true
		},
		gen.IntRange(8000, 9000),
	))
	
	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

