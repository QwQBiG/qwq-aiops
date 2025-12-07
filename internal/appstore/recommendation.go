package appstore

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// UserBehaviorType 用户行为类型
type UserBehaviorType string

const (
	BehaviorView      UserBehaviorType = "view"      // 查看应用
	BehaviorInstall   UserBehaviorType = "install"   // 安装应用
	BehaviorUninstall UserBehaviorType = "uninstall" // 卸载应用
	BehaviorSearch    UserBehaviorType = "search"    // 搜索应用
	BehaviorFeedback  UserBehaviorType = "feedback"  // 反馈评价
)

// UserBehavior 用户行为记录
type UserBehavior struct {
	ID         uint             `json:"id" gorm:"primaryKey"`
	UserID     uint             `json:"user_id" gorm:"not null;index"`
	TenantID   uint             `json:"tenant_id" gorm:"index"`
	Type       UserBehaviorType `json:"type" gorm:"not null;index"`
	TemplateID uint             `json:"template_id" gorm:"index"`
	Metadata   string           `json:"metadata" gorm:"type:text"` // 额外元数据（JSON）
	CreatedAt  time.Time        `json:"created_at" gorm:"index"`
}

// UserFeedback 用户反馈
type UserFeedback struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"not null;index"`
	TemplateID uint      `json:"template_id" gorm:"not null;index"`
	Rating     int       `json:"rating" gorm:"not null"` // 评分 1-5
	Comment    string    `json:"comment" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
}

// AppRecommendation 应用推荐结果
type AppRecommendation struct {
	Template   *AppTemplate `json:"template"`
	Score      float64      `json:"score"`       // 推荐分数
	Reason     string       `json:"reason"`      // 推荐理由
	Confidence float64      `json:"confidence"`  // 置信度
}

// UserContext 用户上下文
type UserContext struct {
	UserID           uint              `json:"user_id"`
	TenantID         uint              `json:"tenant_id"`
	InstalledApps    []uint            `json:"installed_apps"`     // 已安装应用ID列表
	RecentSearches   []string          `json:"recent_searches"`    // 最近搜索关键词
	PreferredCategory AppCategory      `json:"preferred_category"` // 偏好分类
	CustomTags       []string          `json:"custom_tags"`        // 自定义标签
}

// RecommendationService AI 推荐服务接口
type RecommendationService interface {
	// 记录用户行为
	RecordBehavior(ctx context.Context, behavior *UserBehavior) error
	
	// 记录用户反馈
	RecordFeedback(ctx context.Context, feedback *UserFeedback) error
	
	// 获取推荐应用
	GetRecommendations(ctx context.Context, userContext *UserContext, limit int) ([]*AppRecommendation, error)
	
	// 分析用户需求
	AnalyzeUserNeeds(ctx context.Context, userID uint) (*UserProfile, error)
	
	// 计算应用相关性评分
	CalculateRelevanceScore(ctx context.Context, template *AppTemplate, userContext *UserContext) (float64, error)
}

// UserProfile 用户画像
type UserProfile struct {
	UserID              uint                   `json:"user_id"`
	PreferredCategories map[AppCategory]float64 `json:"preferred_categories"` // 分类偏好权重
	FrequentTags        map[string]int         `json:"frequent_tags"`        // 常用标签频次
	InstallHistory      []uint                 `json:"install_history"`      // 安装历史
	SearchPatterns      []string               `json:"search_patterns"`      // 搜索模式
	AvgRating           float64                `json:"avg_rating"`           // 平均评分
	ActivityLevel       float64                `json:"activity_level"`       // 活跃度
}

// recommendationServiceImpl 推荐服务实现
type recommendationServiceImpl struct {
	db              *gorm.DB
	appStoreService AppStoreService
}

// NewRecommendationService 创建推荐服务实例
func NewRecommendationService(db *gorm.DB, appStoreService AppStoreService) RecommendationService {
	return &recommendationServiceImpl{
		db:              db,
		appStoreService: appStoreService,
	}
}

// RecordBehavior 记录用户行为
func (s *recommendationServiceImpl) RecordBehavior(ctx context.Context, behavior *UserBehavior) error {
	if behavior == nil {
		return fmt.Errorf("behavior is nil")
	}

	if err := s.db.WithContext(ctx).Create(behavior).Error; err != nil {
		return fmt.Errorf("failed to record behavior: %w", err)
	}

	return nil
}

// RecordFeedback 记录用户反馈
func (s *recommendationServiceImpl) RecordFeedback(ctx context.Context, feedback *UserFeedback) error {
	if feedback == nil {
		return fmt.Errorf("feedback is nil")
	}

	// 验证评分范围
	if feedback.Rating < 1 || feedback.Rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	if err := s.db.WithContext(ctx).Create(feedback).Error; err != nil {
		return fmt.Errorf("failed to record feedback: %w", err)
	}

	// 同时记录为行为
	behavior := &UserBehavior{
		UserID:     feedback.UserID,
		Type:       BehaviorFeedback,
		TemplateID: feedback.TemplateID,
		Metadata:   fmt.Sprintf(`{"rating": %d}`, feedback.Rating),
		CreatedAt:  time.Now(),
	}

	return s.RecordBehavior(ctx, behavior)
}

// GetRecommendations 获取推荐应用
func (s *recommendationServiceImpl) GetRecommendations(ctx context.Context, userContext *UserContext, limit int) ([]*AppRecommendation, error) {
	if userContext == nil {
		return nil, fmt.Errorf("user context is nil")
	}

	// 获取用户画像
	profile, err := s.AnalyzeUserNeeds(ctx, userContext.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze user needs: %w", err)
	}

	// 获取所有已发布的模板
	templates, err := s.appStoreService.ListTemplates(ctx, "", TemplateStatusPublished)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	// 过滤已安装的应用
	installedMap := make(map[uint]bool)
	for _, id := range userContext.InstalledApps {
		installedMap[id] = true
	}

	var recommendations []*AppRecommendation
	for _, template := range templates {
		// 跳过已安装的应用
		if installedMap[template.ID] {
			continue
		}

		// 计算推荐分数
		score, reason := s.calculateScore(template, userContext, profile)
		
		// 计算置信度
		confidence := s.calculateConfidence(profile)

		recommendations = append(recommendations, &AppRecommendation{
			Template:   template,
			Score:      score,
			Reason:     reason,
			Confidence: confidence,
		})
	}

	// 按分数排序
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	// 限制返回数量
	if limit > 0 && len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	return recommendations, nil
}

// AnalyzeUserNeeds 分析用户需求
func (s *recommendationServiceImpl) AnalyzeUserNeeds(ctx context.Context, userID uint) (*UserProfile, error) {
	profile := &UserProfile{
		UserID:              userID,
		PreferredCategories: make(map[AppCategory]float64),
		FrequentTags:        make(map[string]int),
		InstallHistory:      []uint{},
		SearchPatterns:      []string{},
	}

	// 获取用户行为历史（最近30天）
	var behaviors []UserBehavior
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND created_at >= ?", userID, thirtyDaysAgo).
		Order("created_at DESC").
		Find(&behaviors).Error; err != nil {
		return nil, fmt.Errorf("failed to get user behaviors: %w", err)
	}

	// 分析行为模式
	categoryCount := make(map[AppCategory]int)
	totalBehaviors := len(behaviors)

	for _, behavior := range behaviors {
		// 获取模板信息
		template, err := s.appStoreService.GetTemplate(ctx, behavior.TemplateID)
		if err != nil {
			continue
		}

		// 统计分类偏好
		categoryCount[template.Category]++

		// 记录安装历史
		if behavior.Type == BehaviorInstall {
			profile.InstallHistory = append(profile.InstallHistory, behavior.TemplateID)
		}

		// 统计标签频次
		if template.Tags != "" {
			tags := strings.Split(template.Tags, ",")
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					profile.FrequentTags[tag]++
				}
			}
		}

		// 记录搜索模式
		if behavior.Type == BehaviorSearch && behavior.Metadata != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(behavior.Metadata), &metadata); err == nil {
				if query, ok := metadata["query"].(string); ok {
					profile.SearchPatterns = append(profile.SearchPatterns, query)
				}
			}
		}
	}

	// 计算分类偏好权重
	for category, count := range categoryCount {
		if totalBehaviors > 0 {
			profile.PreferredCategories[category] = float64(count) / float64(totalBehaviors)
		}
	}

	// 获取用户平均评分
	var avgRating float64
	if err := s.db.WithContext(ctx).
		Model(&UserFeedback{}).
		Where("user_id = ?", userID).
		Select("AVG(rating)").
		Scan(&avgRating).Error; err == nil {
		profile.AvgRating = avgRating
	}

	// 计算活跃度（基于最近行为数量）
	profile.ActivityLevel = math.Min(float64(totalBehaviors)/100.0, 1.0)

	return profile, nil
}

// CalculateRelevanceScore 计算应用相关性评分
func (s *recommendationServiceImpl) CalculateRelevanceScore(ctx context.Context, template *AppTemplate, userContext *UserContext) (float64, error) {
	profile, err := s.AnalyzeUserNeeds(ctx, userContext.UserID)
	if err != nil {
		return 0, err
	}

	score, _ := s.calculateScore(template, userContext, profile)
	return score, nil
}

// calculateScore 计算推荐分数（内部方法）
func (s *recommendationServiceImpl) calculateScore(template *AppTemplate, userContext *UserContext, profile *UserProfile) (float64, string) {
	var score float64
	var reasons []string

	// 1. 分类匹配度（权重 0.3）
	if weight, ok := profile.PreferredCategories[template.Category]; ok {
		categoryScore := weight * 0.3
		score += categoryScore
		if categoryScore > 0.1 {
			reasons = append(reasons, fmt.Sprintf("匹配您偏好的 %s 分类", template.Category))
		}
	}

	// 2. 标签匹配度（权重 0.25）
	if template.Tags != "" {
		tags := strings.Split(template.Tags, ",")
		matchedTags := 0
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if count, ok := profile.FrequentTags[tag]; ok && count > 0 {
				matchedTags++
			}
		}
		if len(tags) > 0 {
			tagScore := (float64(matchedTags) / float64(len(tags))) * 0.25
			score += tagScore
			if matchedTags > 0 {
				reasons = append(reasons, fmt.Sprintf("包含您常用的 %d 个标签", matchedTags))
			}
		}
	}

	// 3. 搜索关键词匹配（权重 0.2）
	searchScore := 0.0
	for _, pattern := range profile.SearchPatterns {
		pattern = strings.ToLower(pattern)
		if strings.Contains(strings.ToLower(template.Name), pattern) ||
			strings.Contains(strings.ToLower(template.DisplayName), pattern) ||
			strings.Contains(strings.ToLower(template.Description), pattern) {
			searchScore = 0.2
			reasons = append(reasons, "匹配您的搜索历史")
			break
		}
	}
	score += searchScore

	// 4. 用户偏好分类（权重 0.15）
	if userContext.PreferredCategory != "" && template.Category == userContext.PreferredCategory {
		score += 0.15
		reasons = append(reasons, "符合您设置的偏好分类")
	}

	// 5. 自定义标签匹配（权重 0.1）
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
			customTagScore := (float64(matchCount) / float64(len(userContext.CustomTags))) * 0.1
			score += customTagScore
			reasons = append(reasons, "匹配您的自定义标签")
		}
	}

	// 6. 热门度加成（基于全局安装数）
	var installCount int64
	s.db.Model(&ApplicationInstance{}).Where("template_id = ?", template.ID).Count(&installCount)
	if installCount > 10 {
		popularityBonus := math.Min(float64(installCount)/1000.0, 0.1)
		score += popularityBonus
		if installCount > 50 {
			reasons = append(reasons, "热门应用")
		}
	}

	// 7. 评分加成
	var avgRating float64
	s.db.Model(&UserFeedback{}).
		Where("template_id = ?", template.ID).
		Select("AVG(rating)").
		Scan(&avgRating)
	if avgRating >= 4.0 {
		ratingBonus := (avgRating - 3.0) * 0.05
		score += ratingBonus
		reasons = append(reasons, fmt.Sprintf("高评分应用 (%.1f/5.0)", avgRating))
	}

	// 生成推荐理由
	reason := "推荐给您"
	if len(reasons) > 0 {
		reason = strings.Join(reasons, "；")
	}

	return score, reason
}

// calculateConfidence 计算置信度
func (s *recommendationServiceImpl) calculateConfidence(profile *UserProfile) float64 {
	// 基于用户活跃度和数据完整性计算置信度
	confidence := 0.5 // 基础置信度

	// 活跃度贡献
	confidence += profile.ActivityLevel * 0.3

	// 数据完整性贡献
	if len(profile.PreferredCategories) > 0 {
		confidence += 0.1
	}
	if len(profile.FrequentTags) > 0 {
		confidence += 0.05
	}
	if len(profile.InstallHistory) > 0 {
		confidence += 0.05
	}

	return math.Min(confidence, 1.0)
}

// TableName 指定表名
func (UserBehavior) TableName() string {
	return "user_behaviors"
}

func (UserFeedback) TableName() string {
	return "user_feedbacks"
}
