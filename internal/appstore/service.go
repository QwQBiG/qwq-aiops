package appstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	// ErrTemplateNotFound 模板未找到
	ErrTemplateNotFound = errors.New("template not found")
	// ErrTemplateAlreadyExists 模板已存在
	ErrTemplateAlreadyExists = errors.New("template already exists")
	// ErrInstanceNotFound 实例未找到
	ErrInstanceNotFound = errors.New("instance not found")
)

// AppStoreService 应用商店服务接口
type AppStoreService interface {
	// 模板管理
	CreateTemplate(ctx context.Context, template *AppTemplate) error
	GetTemplate(ctx context.Context, id uint) (*AppTemplate, error)
	GetTemplateByName(ctx context.Context, name string) (*AppTemplate, error)
	ListTemplates(ctx context.Context, category AppCategory, status TemplateStatus) ([]*AppTemplate, error)
	UpdateTemplate(ctx context.Context, template *AppTemplate) error
	DeleteTemplate(ctx context.Context, id uint) error

	// 模板操作
	ValidateTemplate(ctx context.Context, template *AppTemplate) error
	RenderTemplate(ctx context.Context, templateID uint, params map[string]interface{}) (string, error)

	// 应用实例管理
	CreateInstance(ctx context.Context, instance *ApplicationInstance) error
	GetInstance(ctx context.Context, id uint) (*ApplicationInstance, error)
	ListInstances(ctx context.Context, userID, tenantID uint) ([]*ApplicationInstance, error)
	UpdateInstance(ctx context.Context, instance *ApplicationInstance) error
	DeleteInstance(ctx context.Context, id uint) error

	// 初始化内置模板
	InitBuiltinTemplates(ctx context.Context) error
}

// appStoreServiceImpl 应用商店服务实现
type appStoreServiceImpl struct {
	db              *gorm.DB
	templateService *TemplateService
}

// NewAppStoreService 创建应用商店服务实例
func NewAppStoreService(db *gorm.DB) AppStoreService {
	return &appStoreServiceImpl{
		db:              db,
		templateService: NewTemplateService(),
	}
}

// CreateTemplate 创建模板
func (s *appStoreServiceImpl) CreateTemplate(ctx context.Context, template *AppTemplate) error {
	if template == nil {
		return errors.New("template is nil")
	}

	// 检查模板名称是否已存在
	var count int64
	if err := s.db.WithContext(ctx).Model(&AppTemplate{}).
		Where("name = ? AND deleted_at IS NULL", template.Name).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check template existence: %w", err)
	}

	if count > 0 {
		return ErrTemplateAlreadyExists
	}

	// 验证模板
	if err := s.templateService.ValidateTemplate(template); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// 创建模板
	if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	return nil
}

// GetTemplate 获取模板
func (s *appStoreServiceImpl) GetTemplate(ctx context.Context, id uint) (*AppTemplate, error) {
	var template AppTemplate
	if err := s.db.WithContext(ctx).First(&template, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &template, nil
}

// GetTemplateByName 根据名称获取模板
func (s *appStoreServiceImpl) GetTemplateByName(ctx context.Context, name string) (*AppTemplate, error) {
	var template AppTemplate
	if err := s.db.WithContext(ctx).
		Where("name = ?", name).
		First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template by name: %w", err)
	}

	return &template, nil
}

// ListTemplates 列出模板
func (s *appStoreServiceImpl) ListTemplates(ctx context.Context, category AppCategory, status TemplateStatus) ([]*AppTemplate, error) {
	var templates []*AppTemplate
	query := s.db.WithContext(ctx).Model(&AppTemplate{})

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	return templates, nil
}

// UpdateTemplate 更新模板
func (s *appStoreServiceImpl) UpdateTemplate(ctx context.Context, template *AppTemplate) error {
	if template == nil {
		return errors.New("template is nil")
	}

	// 验证模板
	if err := s.templateService.ValidateTemplate(template); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// 更新模板
	if err := s.db.WithContext(ctx).Save(template).Error; err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	return nil
}

// DeleteTemplate 删除模板（软删除）
func (s *appStoreServiceImpl) DeleteTemplate(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&AppTemplate{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete template: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

// ValidateTemplate 验证模板
func (s *appStoreServiceImpl) ValidateTemplate(ctx context.Context, template *AppTemplate) error {
	return s.templateService.ValidateTemplate(template)
}

// RenderTemplate 渲染模板
func (s *appStoreServiceImpl) RenderTemplate(ctx context.Context, templateID uint, params map[string]interface{}) (string, error) {
	// 获取模板
	template, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return "", err
	}

	// 渲染模板
	rendered, err := s.templateService.RenderTemplate(template, params)
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return rendered, nil
}

// CreateInstance 创建应用实例
func (s *appStoreServiceImpl) CreateInstance(ctx context.Context, instance *ApplicationInstance) error {
	if instance == nil {
		return errors.New("instance is nil")
	}

	// 验证模板是否存在
	template, err := s.GetTemplate(ctx, instance.TemplateID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// 设置版本
	if instance.Version == "" {
		instance.Version = template.Version
	}

	// 创建实例
	if err := s.db.WithContext(ctx).Create(instance).Error; err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	return nil
}

// GetInstance 获取应用实例
func (s *appStoreServiceImpl) GetInstance(ctx context.Context, id uint) (*ApplicationInstance, error) {
	var instance ApplicationInstance
	if err := s.db.WithContext(ctx).
		Preload("Template").
		First(&instance, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return &instance, nil
}

// ListInstances 列出应用实例
func (s *appStoreServiceImpl) ListInstances(ctx context.Context, userID, tenantID uint) ([]*ApplicationInstance, error) {
	var instances []*ApplicationInstance
	query := s.db.WithContext(ctx).Model(&ApplicationInstance{}).Preload("Template")

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.Order("created_at DESC").Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	return instances, nil
}

// UpdateInstance 更新应用实例
func (s *appStoreServiceImpl) UpdateInstance(ctx context.Context, instance *ApplicationInstance) error {
	if instance == nil {
		return errors.New("instance is nil")
	}

	if err := s.db.WithContext(ctx).Save(instance).Error; err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	return nil
}

// DeleteInstance 删除应用实例（软删除）
func (s *appStoreServiceImpl) DeleteInstance(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&ApplicationInstance{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete instance: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrInstanceNotFound
	}

	return nil
}

// InitBuiltinTemplates 初始化内置模板
func (s *appStoreServiceImpl) InitBuiltinTemplates(ctx context.Context) error {
	builtinTemplates := GetBuiltinTemplates()

	for _, template := range builtinTemplates {
		// 检查模板是否已存在
		existing, err := s.GetTemplateByName(ctx, template.Name)
		if err != nil && !errors.Is(err, ErrTemplateNotFound) {
			return fmt.Errorf("failed to check template '%s': %w", template.Name, err)
		}

		if existing != nil {
			// 模板已存在，跳过
			continue
		}

		// 创建模板
		if err := s.CreateTemplate(ctx, template); err != nil {
			return fmt.Errorf("failed to create builtin template '%s': %w", template.Name, err)
		}
	}

	return nil
}

// ParseTemplateParameters 解析模板参数定义
func ParseTemplateParameters(parametersJSON string) ([]TemplateParameter, error) {
	if parametersJSON == "" {
		return []TemplateParameter{}, nil
	}

	var params []TemplateParameter
	if err := json.Unmarshal([]byte(parametersJSON), &params); err != nil {
		return nil, fmt.Errorf("failed to parse template parameters: %w", err)
	}

	return params, nil
}
