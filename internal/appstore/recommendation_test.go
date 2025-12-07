package appstore

import (
	"testing"
)

// TestUserBehaviorType 测试用户行为类型
func TestUserBehaviorType(t *testing.T) {
	t.Skip("跳过需要数据库的集成测试")
}

// TestUserFeedbackValidation 测试用户反馈验证
func TestUserFeedbackValidation(t *testing.T) {
	t.Skip("跳过需要数据库的集成测试")
}

// TestRecommendationStructure 测试推荐结构
func TestRecommendationStructure(t *testing.T) {
	// 测试推荐结果结构
	rec := &AppRecommendation{
		Template: &AppTemplate{
			Name:        "test-app",
			DisplayName: "Test App",
			Category:    CategoryWebServer,
		},
		Score:      0.85,
		Reason:     "匹配您的偏好",
		Confidence: 0.75,
	}
	
	if rec.Score != 0.85 {
		t.Errorf("expected score 0.85, got %f", rec.Score)
	}
	
	if rec.Confidence != 0.75 {
		t.Errorf("expected confidence 0.75, got %f", rec.Confidence)
	}
	
	if rec.Reason == "" {
		t.Error("reason should not be empty")
	}
}

// TestUserProfileStructure 测试用户画像结构
func TestUserProfileStructure(t *testing.T) {
	profile := &UserProfile{
		UserID:              1,
		PreferredCategories: map[AppCategory]float64{CategoryWebServer: 0.8},
		FrequentTags:        map[string]int{"web": 5, "proxy": 3},
		InstallHistory:      []uint{1, 2, 3},
		SearchPatterns:      []string{"nginx", "apache"},
		AvgRating:           4.5,
		ActivityLevel:       0.75,
	}
	
	if profile.UserID != 1 {
		t.Errorf("expected user ID 1, got %d", profile.UserID)
	}
	
	if len(profile.PreferredCategories) == 0 {
		t.Error("expected preferred categories")
	}
	
	if len(profile.FrequentTags) == 0 {
		t.Error("expected frequent tags")
	}
	
	if profile.ActivityLevel <= 0 || profile.ActivityLevel > 1 {
		t.Errorf("activity level should be between 0 and 1, got %f", profile.ActivityLevel)
	}
}

// TestUserContextStructure 测试用户上下文结构
func TestUserContextStructure(t *testing.T) {
	ctx := &UserContext{
		UserID:            1,
		TenantID:          1,
		InstalledApps:     []uint{1, 2, 3},
		RecentSearches:    []string{"nginx", "mysql"},
		PreferredCategory: CategoryWebServer,
		CustomTags:        []string{"high-performance", "easy-to-use"},
	}
	
	if ctx.UserID != 1 {
		t.Errorf("expected user ID 1, got %d", ctx.UserID)
	}
	
	if len(ctx.InstalledApps) != 3 {
		t.Errorf("expected 3 installed apps, got %d", len(ctx.InstalledApps))
	}
	
	if ctx.PreferredCategory != CategoryWebServer {
		t.Errorf("expected category %s, got %s", CategoryWebServer, ctx.PreferredCategory)
	}
}

// TestBehaviorTypes 测试行为类型
func TestBehaviorTypes(t *testing.T) {
	behaviors := []UserBehaviorType{
		BehaviorView,
		BehaviorInstall,
		BehaviorUninstall,
		BehaviorSearch,
		BehaviorFeedback,
	}
	
	if len(behaviors) != 5 {
		t.Errorf("expected 5 behavior types, got %d", len(behaviors))
	}
	
	// 验证行为类型值
	if BehaviorView != "view" {
		t.Errorf("expected BehaviorView to be 'view', got %s", BehaviorView)
	}
	
	if BehaviorInstall != "install" {
		t.Errorf("expected BehaviorInstall to be 'install', got %s", BehaviorInstall)
	}
}
