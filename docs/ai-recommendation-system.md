# AI 应用推荐系统

## 概述

AI 应用推荐系统是 qwq AIOps 平台的核心功能之一，通过分析用户行为、偏好和需求，为用户提供个性化的应用推荐。

## 核心功能

### 1. 用户行为分析

系统会自动记录和分析以下用户行为：

- **查看行为**: 用户浏览应用详情
- **安装行为**: 用户安装应用
- **卸载行为**: 用户卸载应用
- **搜索行为**: 用户搜索关键词
- **反馈行为**: 用户评分和评论

### 2. 需求识别

基于用户行为数据，系统会构建用户画像，包括：

- **分类偏好**: 用户偏好的应用分类（Web服务器、数据库、开发工具等）
- **标签频次**: 用户常用的应用标签
- **安装历史**: 用户的应用安装记录
- **搜索模式**: 用户的搜索关键词模式
- **活跃度**: 用户的活跃程度

### 3. 智能推荐算法

推荐算法采用多维度评分机制：

#### 评分维度及权重

1. **分类匹配度** (30%)
   - 基于用户历史行为中的分类偏好
   - 计算用户对各分类的兴趣权重

2. **标签匹配度** (25%)
   - 匹配应用标签与用户常用标签
   - 计算标签重叠率

3. **搜索关键词匹配** (20%)
   - 匹配应用名称、描述与用户搜索历史
   - 支持模糊匹配

4. **用户偏好分类** (15%)
   - 用户主动设置的偏好分类
   - 直接匹配加分

5. **自定义标签匹配** (10%)
   - 用户自定义的兴趣标签
   - 精确匹配计分

6. **热门度加成** (最高 10%)
   - 基于全局安装数量
   - 热门应用获得额外加分

7. **评分加成** (最高 5%)
   - 基于用户评分
   - 高评分应用获得额外加分

#### 置信度计算

推荐置信度基于以下因素：

- **用户活跃度**: 用户行为数据的丰富程度
- **数据完整性**: 用户画像的完整程度
- **基础置信度**: 50% 起始值

### 4. 个性化推荐

系统支持多种个性化场景：

#### 新用户推荐
- 基于热门度和评分
- 推荐通用性强的应用
- 较低的置信度

#### 活跃用户推荐
- 基于丰富的历史数据
- 高度个性化的推荐
- 较高的置信度

#### 企业用户推荐
- 考虑租户隔离
- 团队协作相关应用
- 企业级应用优先

### 5. 反馈学习

系统通过用户反馈持续优化：

- **评分反馈**: 1-5 星评分
- **评论反馈**: 文字评论
- **行为反馈**: 安装/卸载行为
- **隐式反馈**: 使用时长、频率等

## 使用示例

### 记录用户行为

```go
// 记录用户查看行为
viewBehavior := &UserBehavior{
    UserID:     1,
    TenantID:   1,
    Type:       BehaviorView,
    TemplateID: 1,
    CreatedAt:  time.Now(),
}
err := appStoreService.RecordUserBehavior(ctx, viewBehavior)

// 记录用户搜索行为
searchBehavior := &UserBehavior{
    UserID:   1,
    Type:     BehaviorSearch,
    Metadata: `{"query": "nginx"}`,
}
err = appStoreService.RecordUserBehavior(ctx, searchBehavior)
```

### 记录用户反馈

```go
feedback := &UserFeedback{
    UserID:     1,
    TemplateID: 1,
    Rating:     5,
    Comment:    "非常好用的应用！",
}
err := appStoreService.RecordUserFeedback(ctx, feedback)
```

### 获取个性化推荐

```go
userContext := &UserContext{
    UserID:            1,
    TenantID:          1,
    InstalledApps:     []uint{1, 2},
    RecentSearches:    []string{"nginx", "database"},
    PreferredCategory: CategoryWebServer,
    CustomTags:        []string{"high-performance"},
}

recommendations, err := appStoreService.GetRecommendations(ctx, userContext, 5)

for _, rec := range recommendations {
    fmt.Printf("推荐: %s (分数: %.2f)\n", rec.Template.DisplayName, rec.Score)
    fmt.Printf("理由: %s\n", rec.Reason)
    fmt.Printf("置信度: %.2f\n", rec.Confidence)
}
```

## 数据模型

### UserBehavior - 用户行为记录

```go
type UserBehavior struct {
    ID         uint             // 主键
    UserID     uint             // 用户ID
    TenantID   uint             // 租户ID
    Type       UserBehaviorType // 行为类型
    TemplateID uint             // 应用模板ID
    Metadata   string           // 额外元数据（JSON）
    CreatedAt  time.Time        // 创建时间
}
```

### UserFeedback - 用户反馈

```go
type UserFeedback struct {
    ID         uint      // 主键
    UserID     uint      // 用户ID
    TemplateID uint      // 应用模板ID
    Rating     int       // 评分 1-5
    Comment    string    // 评论
    CreatedAt  time.Time // 创建时间
}
```

### AppRecommendation - 推荐结果

```go
type AppRecommendation struct {
    Template   *AppTemplate // 应用模板
    Score      float64      // 推荐分数
    Reason     string       // 推荐理由
    Confidence float64      // 置信度
}
```

### UserProfile - 用户画像

```go
type UserProfile struct {
    UserID              uint                    // 用户ID
    PreferredCategories map[AppCategory]float64 // 分类偏好权重
    FrequentTags        map[string]int          // 常用标签频次
    InstallHistory      []uint                  // 安装历史
    SearchPatterns      []string                // 搜索模式
    AvgRating           float64                 // 平均评分
    ActivityLevel       float64                 // 活跃度
}
```

## API 接口

### 推荐服务接口

```go
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
```

## 推荐场景

### 场景1: 首页推荐

用户打开应用商店首页时，展示个性化推荐：

```go
recommendations, _ := appStoreService.GetRecommendations(ctx, userContext, 10)
// 展示前10个推荐应用
```

### 场景2: 搜索推荐

用户搜索时，记录搜索行为并影响后续推荐：

```go
searchBehavior := &UserBehavior{
    UserID:   userID,
    Type:     BehaviorSearch,
    Metadata: fmt.Sprintf(`{"query": "%s"}`, searchQuery),
}
appStoreService.RecordUserBehavior(ctx, searchBehavior)
```

### 场景3: 安装后推荐

用户安装应用后，推荐相关应用：

```go
// 记录安装行为
installBehavior := &UserBehavior{
    UserID:     userID,
    Type:       BehaviorInstall,
    TemplateID: templateID,
}
appStoreService.RecordUserBehavior(ctx, installBehavior)

// 获取相关推荐
userContext.InstalledApps = append(userContext.InstalledApps, templateID)
recommendations, _ := appStoreService.GetRecommendations(ctx, userContext, 5)
```

### 场景4: 评分后优化

用户评分后，系统学习用户偏好：

```go
feedback := &UserFeedback{
    UserID:     userID,
    TemplateID: templateID,
    Rating:     rating,
    Comment:    comment,
}
appStoreService.RecordUserFeedback(ctx, feedback)
// 后续推荐会考虑用户的评分偏好
```

## 性能优化

### 缓存策略

- 用户画像缓存（30分钟）
- 推荐结果缓存（10分钟）
- 热门应用缓存（1小时）

### 批量处理

- 批量记录用户行为
- 批量计算推荐分数
- 异步更新用户画像

### 数据清理

- 定期清理过期行为数据（保留90天）
- 归档历史反馈数据
- 压缩存储元数据

## 未来优化方向

1. **协同过滤**: 基于相似用户的推荐
2. **深度学习**: 使用神经网络模型
3. **实时推荐**: 基于流式数据的实时推荐
4. **A/B测试**: 推荐算法效果对比
5. **多目标优化**: 平衡准确性、多样性、新颖性

## 相关需求

- **Requirements 2.4**: AI 检测到用户需求时主动推荐相关应用和最佳实践
- **Property 6**: AI 应用推荐相关性 - 验证推荐的应用与用户需求相关

## 参考资料

- [推荐系统实践](https://book.douban.com/subject/10769749/)
- [协同过滤算法](https://en.wikipedia.org/wiki/Collaborative_filtering)
- [内容推荐算法](https://en.wikipedia.org/wiki/Recommender_system)
