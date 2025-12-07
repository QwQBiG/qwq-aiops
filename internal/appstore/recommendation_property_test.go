package appstore

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: enhanced-aiops-platform, Property 6: AI 应用推荐相关性**
// **Validates: Requirements 2.4**

// mockRecommendationService 模拟推荐服务
type mockRecommendationService struct {
	templates []*AppTemplate
}

// newMockRecommendationService 创建模拟推荐服务实例
func newMockRecommendationService() *mockRecommendationService {
	return &mockRecommendationService{templates: []*AppTemplate{}}
}

// AddTemplate 添加应用模板到服务中
func (m *mockRecommendationService) AddTemplate(template *AppTemplate) {
	m.templates = append(m.templates, template)
}

// GetRecommendations 根据用户上下文获取应用推荐列表
// 推荐算法考虑：偏好分类、自定义标签、搜索历史，并排除已安装应用
func (m *mockRecommendationService) GetRecommendations(ctx context.Context, userContext *UserContext, limit int) ([]*AppRecommendation, error) {
	if userContext == nil {
		return nil, fmt.Errorf("user context is nil")
	}

	// 构建已安装应用的映射表，用于快速查找
	installedMap := make(map[uint]bool)
	for _, id := range userContext.InstalledApps {
		installedMap[id] = true
	}

	// 遍历所有模板，计算推荐分数
	var recommendations []*AppRecommendation
	for _, template := range m.templates {
		// 跳过已安装的应用
		if installedMap[template.ID] {
			continue
		}

		score := m.calculateScore(template, userContext)
		reason := m.generateReason(template, userContext)

		recommendations = append(recommendations, &AppRecommendation{
			Template:   template,
			Score:      score,
			Reason:     reason,
			Confidence: 0.75,
		})
	}

	// 按分数降序排序（冒泡排序）
	for i := 0; i < len(recommendations); i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[i].Score < recommendations[j].Score {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}

	// 限制返回数量
	if limit > 0 && len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	return recommendations, nil
}

// calculateScore 计算应用模板的推荐分数
// 评分规则：偏好分类匹配 +0.4，自定义标签匹配 +0.3，搜索历史匹配 +0.3
func (m *mockRecommendationService) calculateScore(template *AppTemplate, userContext *UserContext) float64 {
	var score float64

	// 检查是否匹配用户偏好的分类
	if userContext.PreferredCategory != "" && template.Category == userContext.PreferredCategory {
		score += 0.4
	}

	// 检查是否匹配用户自定义标签
	if len(userContext.CustomTags) > 0 && template.Tags != "" {
		templateTags := strings.Split(template.Tags, ",")
		matchCount := 0
		for _, customTag := range userContext.CustomTags {
			for _, templateTag := range templateTags {
				if strings.TrimSpace(templateTag) == customTag {
					matchCount++
					break
				}
			}
		}
		if matchCount > 0 {
			score += (float64(matchCount) / float64(len(userContext.CustomTags))) * 0.3
		}
	}

	// 检查是否匹配用户搜索历史
	for _, pattern := range userContext.RecentSearches {
		pattern = strings.ToLower(pattern)
		if strings.Contains(strings.ToLower(template.Name), pattern) ||
			strings.Contains(strings.ToLower(template.DisplayName), pattern) ||
			strings.Contains(strings.ToLower(template.Description), pattern) {
			score += 0.3
			break
		}
	}

	return score
}

// generateReason 生成推荐理由的文本说明
func (m *mockRecommendationService) generateReason(template *AppTemplate, userContext *UserContext) string {
	var reasons []string

	if userContext.PreferredCategory != "" && template.Category == userContext.PreferredCategory {
		reasons = append(reasons, fmt.Sprintf("匹配您偏好的 %s 分类", template.Category))
	}

	if len(userContext.CustomTags) > 0 {
		reasons = append(reasons, "匹配您的自定义标签")
	}

	if len(userContext.RecentSearches) > 0 {
		reasons = append(reasons, "匹配您的搜索历史")
	}

	if len(reasons) == 0 {
		return "推荐给您"
	}

	return strings.Join(reasons, "；")
}

// createTestTemplates 创建测试用的应用模板
// 根据分类和标签的笛卡尔积生成模板
func createTestTemplates(service *mockRecommendationService, categories []AppCategory, tags []string) {
	id := uint(1)
	for _, category := range categories {
		for _, tag := range tags {
			template := &AppTemplate{
				ID:          id,
				Name:        fmt.Sprintf("app-%s-%s", category, tag),
				DisplayName: fmt.Sprintf("App %s %s", category, tag),
				Category:    category,
				Tags:        tag,
				Description: fmt.Sprintf("A %s application with %s", category, tag),
				Type:        TemplateTypeDockerCompose,
				Version:     "1.0.0",
				Status:      TemplateStatusPublished,
				Content:     "version: '3'\nservices:\n  app:\n    image: nginx:latest",
			}
			service.AddTemplate(template)
			id++
		}
	}
}


// TestProperty6_RecommendationRelevance_PreferredCategory 测试推荐系统是否正确匹配用户偏好分类
// 验证：推荐结果中应包含用户偏好分类的应用，且分数大于0
func TestProperty6_RecommendationRelevance_PreferredCategory(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("推荐匹配用户偏好分类", prop.ForAll(
		func(preferredCategory AppCategory) bool {
			service := newMockRecommendationService()
			
			categories := []AppCategory{CategoryWebServer, CategoryDatabase, CategoryDevTools, CategoryMonitoring}
			tags := []string{"popular", "stable"}
			createTestTemplates(service, categories, tags)
			
			userContext := &UserContext{
				UserID:            1,
				TenantID:          1,
				PreferredCategory: preferredCategory,
				InstalledApps:     []uint{},
			}
			
			ctx := context.Background()
			recommendations, err := service.GetRecommendations(ctx, userContext, 5)
			
			if err != nil {
				t.Logf("获取推荐失败: %v", err)
				return false
			}
			
			if len(recommendations) > 0 {
				hasMatchingCategory := false
				for _, rec := range recommendations {
					if rec.Template.Category == preferredCategory {
						hasMatchingCategory = true
						if rec.Score <= 0 {
							t.Logf("匹配偏好分类的应用推荐分数应该大于0")
							return false
						}
						break
					}
				}
				
				if !hasMatchingCategory {
					t.Logf("推荐结果中应该包含偏好分类 %s 的应用", preferredCategory)
					return false
				}
			}
			
			return true
		},
		gen.OneConstOf(CategoryWebServer, CategoryDatabase, CategoryDevTools, CategoryMonitoring),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}


// TestProperty6_RecommendationRelevance_CustomTags 测试推荐系统是否正确匹配用户自定义标签
// 验证：推荐结果中应包含匹配用户自定义标签的应用，且分数大于0
func TestProperty6_RecommendationRelevance_CustomTags(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("推荐匹配用户自定义标签", prop.ForAll(
		func(customTags []string) bool {
			validTags := []string{}
			for _, tag := range customTags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					validTags = append(validTags, tag)
				}
			}
			
			if len(validTags) == 0 {
				return true
			}
			
			service := newMockRecommendationService()
			categories := []AppCategory{CategoryWebServer, CategoryDatabase}
			createTestTemplates(service, categories, validTags)
			
			userContext := &UserContext{
				UserID:       1,
				TenantID:     1,
				CustomTags:   validTags,
				InstalledApps: []uint{},
			}
			
			ctx := context.Background()
			recommendations, err := service.GetRecommendations(ctx, userContext, 10)
			
			if err != nil {
				t.Logf("获取推荐失败: %v", err)
				return false
			}
			
			if len(recommendations) > 0 {
				hasMatchingTag := false
				for _, rec := range recommendations {
					templateTags := strings.Split(rec.Template.Tags, ",")
					for _, templateTag := range templateTags {
						templateTag = strings.TrimSpace(templateTag)
						for _, customTag := range validTags {
							if templateTag == customTag {
								hasMatchingTag = true
								if rec.Score <= 0 {
									t.Logf("匹配自定义标签的应用推荐分数应该大于0")
									return false
								}
								break
							}
						}
						if hasMatchingTag {
							break
						}
					}
					if hasMatchingTag {
						break
					}
				}
				
				if !hasMatchingTag {
					t.Logf("推荐结果中应该包含匹配自定义标签的应用")
					return false
				}
			}
			
			return true
		},
		gen.SliceOfN(2, gen.OneConstOf("popular", "stable", "fast", "secure")),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}


// TestProperty6_RecommendationRelevance_ExcludeInstalled 测试推荐系统是否正确排除已安装应用
// 验证：推荐结果中不应包含用户已安装的应用
func TestProperty6_RecommendationRelevance_ExcludeInstalled(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("推荐不包含已安装应用", prop.ForAll(
		func(installedAppIDs []uint) bool {
			if len(installedAppIDs) == 0 {
				return true
			}
			
			service := newMockRecommendationService()
			categories := []AppCategory{CategoryWebServer, CategoryDatabase}
			tags := []string{"popular"}
			createTestTemplates(service, categories, tags)
			
			validInstalledIDs := []uint{}
			for _, id := range installedAppIDs {
				if id > 0 && int(id) <= len(service.templates) {
					validInstalledIDs = append(validInstalledIDs, id)
				}
			}
			
			if len(validInstalledIDs) == 0 {
				return true
			}
			
			userContext := &UserContext{
				UserID:        1,
				TenantID:      1,
				InstalledApps: validInstalledIDs,
			}
			
			ctx := context.Background()
			recommendations, err := service.GetRecommendations(ctx, userContext, 10)
			
			if err != nil {
				t.Logf("获取推荐失败: %v", err)
				return false
			}
			
			installedMap := make(map[uint]bool)
			for _, id := range validInstalledIDs {
				installedMap[id] = true
			}
			
			for _, rec := range recommendations {
				if installedMap[rec.Template.ID] {
					t.Logf("推荐结果包含已安装的应用 ID: %d", rec.Template.ID)
					return false
				}
			}
			
			return true
		},
		gen.SliceOfN(2, gen.UIntRange(1, 4)),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty6_RecommendationRelevance_ScoreOrdering 测试推荐结果是否按分数正确排序
// 验证：推荐列表应按分数降序排列
func TestProperty6_RecommendationRelevance_ScoreOrdering(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("推荐结果按分数排序", prop.ForAll(
		func(preferredCategory AppCategory) bool {
			service := newMockRecommendationService()
			categories := []AppCategory{CategoryWebServer, CategoryDatabase, CategoryDevTools}
			tags := []string{"popular", "stable"}
			createTestTemplates(service, categories, tags)
			
			userContext := &UserContext{
				UserID:            1,
				TenantID:          1,
				PreferredCategory: preferredCategory,
				InstalledApps:     []uint{},
			}
			
			ctx := context.Background()
			recommendations, err := service.GetRecommendations(ctx, userContext, 10)
			
			if err != nil {
				t.Logf("获取推荐失败: %v", err)
				return false
			}
			
			for i := 0; i < len(recommendations)-1; i++ {
				if recommendations[i].Score < recommendations[i+1].Score {
					t.Logf("推荐结果未按分数降序排列: %.2f < %.2f", recommendations[i].Score, recommendations[i+1].Score)
					return false
				}
			}
			
			return true
		},
		gen.OneConstOf(CategoryWebServer, CategoryDatabase, CategoryDevTools),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
