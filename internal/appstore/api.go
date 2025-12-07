package appstore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// APIService 应用商店 API 服务
type APIService struct {
	appStoreService AppStoreService
	installerService InstallerService
	recommendationService RecommendationService
}

// NewAPIService 创建 API 服务实例
func NewAPIService(db *gorm.DB) *APIService {
	appStoreService := NewAppStoreService(db)
	installerService := NewInstallerService(appStoreService)
	recommendationService := NewRecommendationService(appStoreService)
	
	return &APIService{
		appStoreService: appStoreService,
		installerService: installerService,
		recommendationService: recommendationService,
	}
}

// RegisterRoutes 注册路由
func (s *APIService) RegisterRoutes(router *gin.RouterGroup) {
	// 模板管理路由
	templates := router.Group("/templates")
	{
		templates.GET("", s.ListTemplates)           // 列出模板
		templates.GET("/:id", s.GetTemplate)         // 获取模板详情
		templates.POST("", s.CreateTemplate)         // 创建模板
		templates.PUT("/:id", s.UpdateTemplate)      // 更新模板
		templates.DELETE("/:id", s.DeleteTemplate)   // 删除模板
		templates.POST("/:id/validate", s.ValidateTemplate) // 验证模板
		templates.POST("/:id/render", s.RenderTemplate)     // 渲染模板
	}
	
	// 应用实例管理路由
	instances := router.Group("/instances")
	{
		instances.GET("", s.ListInstances)           // 列出实例
		instances.GET("/:id", s.GetInstance)         // 获取实例详情
		instances.POST("", s.InstallApplication)     // 安装应用
		instances.PUT("/:id", s.UpdateInstance)      // 更新实例
		instances.DELETE("/:id", s.UninstallApplication) // 卸载应用
		instances.GET("/:id/status", s.GetInstanceStatus) // 获取实例状态
	}
	
	// 安装管理路由
	install := router.Group("/install")
	{
		install.POST("/check-dependencies", s.CheckDependencies) // 检查依赖
		install.POST("/detect-conflicts", s.DetectConflicts)     // 检测冲突
		install.GET("/progress/:id", s.GetInstallProgress)       // 获取安装进度
		install.POST("/rollback/:id", s.RollbackInstallation)    // 回滚安装
	}
	
	// 搜索和推荐路由
	search := router.Group("/search")
	{
		search.GET("", s.SearchTemplates)            // 搜索模板
		search.GET("/recommendations", s.GetRecommendations) // 获取推荐
	}
	
	// 初始化路由
	router.POST("/init", s.InitBuiltinTemplates)     // 初始化内置模板
}

// ListTemplates 列出模板
// @Summary 列出应用模板
// @Description 获取应用模板列表，支持按分类和状态筛选
// @Tags templates
// @Accept json
// @Produce json
// @Param category query string false "应用分类"
// @Param status query string false "模板状态"
// @Success 200 {object} Response{data=[]AppTemplate}
// @Router /appstore/templates [get]
func (s *APIService) ListTemplates(c *gin.Context) {
	category := AppCategory(c.Query("category"))
	status := TemplateStatus(c.Query("status"))
	
	templates, err := s.appStoreService.ListTemplates(c.Request.Context(), category, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(templates))
}

// GetTemplate 获取模板详情
// @Summary 获取模板详情
// @Description 根据ID获取应用模板的详细信息
// @Tags templates
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} Response{data=AppTemplate}
// @Router /appstore/templates/{id} [get]
func (s *APIService) GetTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(fmt.Errorf("invalid template id")))
		return
	}
	
	template, err := s.appStoreService.GetTemplate(c.Request.Context(), uint(id))
	if err != nil {
		if err == ErrTemplateNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse(err))
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		}
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(template))
}

// CreateTemplate 创建模板
// @Summary 创建应用模板
// @Description 创建新的应用模板
// @Tags templates
// @Accept json
// @Produce json
// @Param template body AppTemplate true "模板信息"
// @Success 201 {object} Response{data=AppTemplate}
// @Router /appstore/templates [post]
func (s *APIService) CreateTemplate(c *gin.Context) {
	var template AppTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(err))
		return
	}
	
	if err := s.appStoreService.CreateTemplate(c.Request.Context(), &template); err != nil {
		if err == ErrTemplateAlreadyExists {
			c.JSON(http.StatusConflict, ErrorResponse(err))
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		}
		return
	}
	
	c.JSON(http.StatusCreated, SuccessResponse(template))
}

// UpdateTemplate 更新模板
// @Summary 更新应用模板
// @Description 更新现有的应用模板
// @Tags templates
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param template body AppTemplate true "模板信息"
// @Success 200 {object} Response{data=AppTemplate}
// @Router /appstore/templates/{id} [put]
func (s *APIService) UpdateTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(fmt.Errorf("invalid template id")))
		return
	}
	
	var template AppTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(err))
		return
	}
	
	template.ID = uint(id)
	
	if err := s.appStoreService.UpdateTemplate(c.Request.Context(), &template); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(template))
}

// DeleteTemplate 删除模板
// @Summary 删除应用模板
// @Description 删除指定的应用模板（软删除）
// @Tags templates
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} Response
// @Router /appstore/templates/{id} [delete]
func (s *APIService) DeleteTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(fmt.Errorf("invalid template id")))
		return
	}
	
	if err := s.appStoreService.DeleteTemplate(c.Request.Context(), uint(id)); err != nil {
		if err == ErrTemplateNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse(err))
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		}
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(nil))
}

// ValidateTemplate 验证模板
// @Summary 验证应用模板
// @Description 验证模板配置是否正确
// @Tags templates
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} Response{data=map[string]interface{}}
// @Router /appstore/templates/{id}/validate [post]
func (s *APIService) ValidateTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(fmt.Errorf("invalid template id")))
		return
	}
	
	template, err := s.appStoreService.GetTemplate(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse(err))
		return
	}
	
	if err := s.appStoreService.ValidateTemplate(c.Request.Context(), template); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"valid": true,
	})
}

// RenderTemplate 渲染模板
// @Summary 渲染应用模板
// @Description 使用参数渲染模板内容
// @Tags templates
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param params body map[string]interface{} true "渲染参数"
// @Success 200 {object} Response{data=string}
// @Router /appstore/templates/{id}/render [post]
func (s *APIService) RenderTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(fmt.Errorf("invalid template id")))
		return
	}
	
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(err))
		return
	}
	
	rendered, err := s.appStoreService.RenderTemplate(c.Request.Context(), uint(id), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"content": rendered,
	}))
}

// ListInstances 列出应用实例
// @Summary 列出应用实例
// @Description 获取应用实例列表
// @Tags instances
// @Accept json
// @Produce json
// @Param user_id query int false "用户ID"
// @Param tenant_id query int false "租户ID"
// @Success 200 {object} Response{data=[]ApplicationInstance}
// @Router /appstore/instances [get]
func (s *APIService) ListInstances(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 32)
	tenantID, _ := strconv.ParseUint(c.Query("tenant_id"), 10, 32)
	
	instances, err := s.appStoreService.ListInstances(c.Request.Context(), uint(userID), uint(tenantID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(instances))
}

// GetInstance 获取应用实例详情
// @Summary 获取应用实例详情
// @Description 根据ID获取应用实例的详细信息
// @Tags instances
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Success 200 {object} Response{data=ApplicationInstance}
// @Router /appstore/instances/{id} [get]
func (s *APIService) GetInstance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(fmt.Errorf("invalid instance id")))
		return
	}
	
	instance, err := s.appStoreService.GetInstance(c.Request.Context(), uint(id))
	if err != nil {
		if err == ErrInstanceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse(err))
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		}
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(instance))
}

// InstallApplication 安装应用
// @Summary 安装应用
// @Description 安装新的应用实例
// @Tags instances
// @Accept json
// @Produce json
// @Param request body InstallRequest true "安装请求"
// @Success 202 {object} Response{data=InstallResult}
// @Router /appstore/instances [post]
func (s *APIService) InstallApplication(c *gin.Context) {
	var req InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(err))
		return
	}
	
	// 从上下文获取用户信息（实际应该从认证中间件获取）
	// 这里简化处理
	if req.UserID == 0 {
		req.UserID = 1 // 默认用户
	}
	if req.TenantID == 0 {
		req.TenantID = 1 // 默认租户
	}
	
	result, err := s.installerService.Install(c.Request.Context(), &req)
	if err != nil {
		if err == ErrDependencyNotMet || err == ErrPortConflict {
			c.JSON(http.StatusConflict, ErrorResponse(err))
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		}
		return
	}
	
	c.JSON(http.StatusAccepted, SuccessResponse(result))
}

// UpdateInstance 更新应用实例
// @Summary 更新应用实例
// @Description 更新现有的应用实例
// @Tags instances
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Param instance body ApplicationInstance true "实例信息"
// @Success 200 {object} Response{data=ApplicationInstance}
// @Router /appstore/instances/{id} [put]
func (s *APIService) UpdateInstance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(fmt.Errorf("invalid instance id")))
		return
	}
	
	var instance ApplicationInstance
	if err := c.ShouldBindJSON(&instance); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(err))
		return
	}
	
	instance.ID = uint(id)
	
	if err := s.appStoreService.UpdateInstance(c.Request.Context(), &instance); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(instance))
}

// UninstallApplication 卸载应用
// @Summary 卸载应用
// @Description 卸载指定的应用实例
// @Tags instances
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Param force query bool false "是否强制卸载"
// @Success 200 {object} Response
// @Router /appstore/instances/{id} [delete]
func (s *APIService) UninstallApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(fmt.Errorf("invalid instance id")))
		return
	}
	
	force := c.Query("force") == "true"
	
	req := &UninstallRequest{
		InstanceID: uint(id),
		Force:      force,
	}
	
	if err := s.installerService.Uninstall(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(nil))
}

// GetInstanceStatus 获取实例状态
// @Summary 获取实例状态
// @Description 获取应用实例的运行状态
// @Tags instances
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Success 200 {object} Response{data=map[string]interface{}}
// @Router /appstore/instances/{id}/status [get]
func (s *APIService) GetInstanceStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(fmt.Errorf("invalid instance id")))
		return
	}
	
	instance, err := s.appStoreService.GetInstance(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"instance_id": instance.ID,
		"name":        instance.Name,
		"status":      instance.Status,
		"version":     instance.Version,
	}))
}

// CheckDependencies 检查依赖
// @Summary 检查应用依赖
// @Description 检查应用的依赖是否满足
// @Tags install
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "检查请求"
// @Success 200 {object} Response{data=[]DependencyCheck}
// @Router /appstore/install/check-dependencies [post]
func (s *APIService) CheckDependencies(c *gin.Context) {
	var req struct {
		TemplateID uint `json:"template_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(err))
		return
	}
	
	checks, err := s.installerService.CheckDependencies(c.Request.Context(), req.TemplateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(checks))
}

// DetectConflicts 检测冲突
// @Summary 检测安装冲突
// @Description 检测应用安装时可能的冲突
// @Tags install
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "检测请求"
// @Success 200 {object} Response{data=[]ConflictInfo}
// @Router /appstore/install/detect-conflicts [post]
func (s *APIService) DetectConflicts(c *gin.Context) {
	var req struct {
		TemplateID uint                   `json:"template_id" binding:"required"`
		Parameters map[string]interface{} `json:"parameters"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(err))
		return
	}
	
	conflicts, err := s.installerService.DetectConflicts(c.Request.Context(), req.TemplateID, req.Parameters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(conflicts))
}

// GetInstallProgress 获取安装进度
// @Summary 获取安装进度
// @Description 获取应用安装的进度信息
// @Tags install
// @Accept json
// @Produce json
// @Param id path string true "进度ID"
// @Success 200 {object} Response{data=InstallationProgress}
// @Router /appstore/install/progress/{id} [get]
func (s *APIService) GetInstallProgress(c *gin.Context) {
	progressID := c.Param("id")
	
	progress, err := s.installerService.GetProgress(c.Request.Context(), progressID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(progress))
}

// RollbackInstallation 回滚安装
// @Summary 回滚安装
// @Description 回滚失败的应用安装
// @Tags install
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Success 200 {object} Response
// @Router /appstore/install/rollback/{id} [post]
func (s *APIService) RollbackInstallation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(fmt.Errorf("invalid instance id")))
		return
	}
	
	if err := s.installerService.Rollback(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(nil))
}

// SearchTemplates 搜索模板
// @Summary 搜索应用模板
// @Description 根据关键词搜索应用模板
// @Tags search
// @Accept json
// @Produce json
// @Param q query string true "搜索关键词"
// @Param category query string false "应用分类"
// @Success 200 {object} Response{data=[]AppTemplate}
// @Router /appstore/search [get]
func (s *APIService) SearchTemplates(c *gin.Context) {
	query := c.Query("q")
	category := AppCategory(c.Query("category"))
	
	// 获取所有模板
	templates, err := s.appStoreService.ListTemplates(c.Request.Context(), category, TemplateStatusPublished)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	
	// 简单的关键词过滤
	var results []*AppTemplate
	for _, template := range templates {
		if query == "" || 
			contains(template.Name, query) || 
			contains(template.DisplayName, query) || 
			contains(template.Description, query) ||
			contains(template.Tags, query) {
			results = append(results, template)
		}
	}
	
	c.JSON(http.StatusOK, SuccessResponse(results))
}

// GetRecommendations 获取推荐
// @Summary 获取应用推荐
// @Description 根据用户上下文获取应用推荐
// @Tags search
// @Accept json
// @Produce json
// @Param user_id query int false "用户ID"
// @Param tenant_id query int false "租户ID"
// @Success 200 {object} Response{data=[]AppRecommendation}
// @Router /appstore/search/recommendations [get]
func (s *APIService) GetRecommendations(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 32)
	tenantID, _ := strconv.ParseUint(c.Query("tenant_id"), 10, 32)
	
	userContext := &UserContext{
		UserID:   uint(userID),
		TenantID: uint(tenantID),
	}
	
	recommendations, err := s.recommendationService.RecommendApplications(c.Request.Context(), userContext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(recommendations))
}

// InitBuiltinTemplates 初始化内置模板
// @Summary 初始化内置模板
// @Description 初始化系统内置的应用模板
// @Tags templates
// @Accept json
// @Produce json
// @Success 200 {object} Response
// @Router /appstore/init [post]
func (s *APIService) InitBuiltinTemplates(c *gin.Context) {
	if err := s.appStoreService.InitBuiltinTemplates(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"message": "内置模板初始化成功",
	}))
}

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SuccessResponse 成功响应
func SuccessResponse(data interface{}) Response {
	return Response{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

// ErrorResponse 错误响应
func ErrorResponse(err error) Response {
	return Response{
		Code:    1,
		Message: err.Error(),
	}
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 len(substr) == 0 || 
		 indexIgnoreCase(s, substr) >= 0)
}

// indexIgnoreCase 不区分大小写的字符串查找
func indexIgnoreCase(s, substr string) int {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// toLower 转换为小写
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}
