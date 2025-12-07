package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// RBACService RBAC 权限管理服务
type RBACService struct {
	db *gorm.DB
}

// NewRBACService 创建 RBAC 服务实例
func NewRBACService(db *gorm.DB) *RBACService {
	return &RBACService{db: db}
}

// CheckPermission 检查用户是否有指定权限
func (s *RBACService) CheckPermission(ctx context.Context, userID uint, resource, action string) (bool, error) {
	// 获取用户信息
	var user User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, fmt.Errorf("用户不存在")
		}
		return false, err
	}

	// 检查用户是否启用
	if !user.Enabled {
		return false, fmt.Errorf("用户已被禁用")
	}

	// 管理员拥有所有权限
	if user.Role == RoleAdmin {
		return true, nil
	}

	// 查询权限
	var permission Permission
	if err := s.db.WithContext(ctx).Where("resource = ? AND action = ?", resource, action).First(&permission).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, fmt.Errorf("权限不存在: %s:%s", resource, action)
		}
		return false, err
	}

	// 检查角色是否有该权限
	var rolePermission RolePermission
	err := s.db.WithContext(ctx).Where("role = ? AND permission_id = ?", user.Role, permission.ID).First(&rolePermission).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil // 没有权限
		}
		return false, err
	}

	return true, nil
}

// CheckTenantAccess 检查用户是否可以访问指定租户的资源
func (s *RBACService) CheckTenantAccess(ctx context.Context, userID, tenantID uint) (bool, error) {
	// 获取用户信息
	var user User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, fmt.Errorf("用户不存在")
		}
		return false, err
	}

	// 管理员可以访问所有租户
	if user.Role == RoleAdmin {
		return true, nil
	}

	// 检查用户是否属于该租户
	if user.TenantID != tenantID {
		return false, nil
	}

	return true, nil
}

// GetUserPermissions 获取用户的所有权限
func (s *RBACService) GetUserPermissions(ctx context.Context, userID uint) ([]Permission, error) {
	// 获取用户信息
	var user User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, err
	}

	// 管理员拥有所有权限
	if user.Role == RoleAdmin {
		var allPermissions []Permission
		if err := s.db.WithContext(ctx).Find(&allPermissions).Error; err != nil {
			return nil, err
		}
		return allPermissions, nil
	}

	// 查询角色权限
	var rolePermissions []RolePermission
	if err := s.db.WithContext(ctx).Preload("Permission").Where("role = ?", user.Role).Find(&rolePermissions).Error; err != nil {
		return nil, err
	}

	// 提取权限列表
	permissions := make([]Permission, 0, len(rolePermissions))
	for _, rp := range rolePermissions {
		if rp.Permission != nil {
			permissions = append(permissions, *rp.Permission)
		}
	}

	return permissions, nil
}

// AssignRolePermission 为角色分配权限
func (s *RBACService) AssignRolePermission(ctx context.Context, role UserRole, permissionID uint) error {
	// 检查权限是否存在
	var permission Permission
	if err := s.db.WithContext(ctx).First(&permission, permissionID).Error; err != nil {
		return fmt.Errorf("权限不存在")
	}

	// 检查是否已经分配
	var existing RolePermission
	result := s.db.WithContext(ctx).Where("role = ? AND permission_id = ?", role, permissionID).First(&existing)
	if result.Error == nil {
		return fmt.Errorf("权限已分配")
	}

	// 创建角色权限关联
	rp := RolePermission{
		Role:         role,
		PermissionID: permissionID,
	}

	return s.db.WithContext(ctx).Create(&rp).Error
}

// RevokeRolePermission 撤销角色权限
func (s *RBACService) RevokeRolePermission(ctx context.Context, role UserRole, permissionID uint) error {
	result := s.db.WithContext(ctx).Where("role = ? AND permission_id = ?", role, permissionID).Delete(&RolePermission{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("权限关联不存在")
	}
	return nil
}

// CreateUser 创建用户
func (s *RBACService) CreateUser(ctx context.Context, user *User) error {
	// 检查用户名是否已存在
	var existing User
	result := s.db.WithContext(ctx).Where("username = ?", user.Username).First(&existing)
	if result.Error == nil {
		return fmt.Errorf("用户名已存在")
	}

	// 检查邮箱是否已存在
	result = s.db.WithContext(ctx).Where("email = ?", user.Email).First(&existing)
	if result.Error == nil {
		return fmt.Errorf("邮箱已存在")
	}

	// 如果没有指定租户，使用默认租户
	if user.TenantID == 0 {
		var defaultTenant Tenant
		if err := s.db.WithContext(ctx).Where("code = ?", "default").First(&defaultTenant).Error; err == nil {
			user.TenantID = defaultTenant.ID
		}
	}

	// 创建用户
	return s.db.WithContext(ctx).Create(user).Error
}

// UpdateUser 更新用户信息
func (s *RBACService) UpdateUser(ctx context.Context, userID uint, updates map[string]interface{}) error {
	// 检查用户是否存在
	var user User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}

	// 更新用户
	return s.db.WithContext(ctx).Model(&user).Updates(updates).Error
}

// DeleteUser 删除用户（软删除）
func (s *RBACService) DeleteUser(ctx context.Context, userID uint) error {
	result := s.db.WithContext(ctx).Delete(&User{}, userID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("用户不存在")
	}
	return nil
}

// GetUser 获取用户信息
func (s *RBACService) GetUser(ctx context.Context, userID uint) (*User, error) {
	var user User
	if err := s.db.WithContext(ctx).Preload("Tenant").First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUsers 列出用户（支持分页和过滤）
func (s *RBACService) ListUsers(ctx context.Context, tenantID uint, page, pageSize int) ([]User, int64, error) {
	var users []User
	var total int64

	query := s.db.WithContext(ctx).Model(&User{})

	// 如果指定了租户ID，只查询该租户的用户
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Preload("Tenant").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// CreateTenant 创建租户
func (s *RBACService) CreateTenant(ctx context.Context, tenant *Tenant) error {
	// 检查租户代码是否已存在
	var existing Tenant
	result := s.db.WithContext(ctx).Where("code = ?", tenant.Code).First(&existing)
	if result.Error == nil {
		return fmt.Errorf("租户代码已存在")
	}

	return s.db.WithContext(ctx).Create(tenant).Error
}

// GetTenant 获取租户信息
func (s *RBACService) GetTenant(ctx context.Context, tenantID uint) (*Tenant, error) {
	var tenant Tenant
	if err := s.db.WithContext(ctx).First(&tenant, tenantID).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

// ListTenants 列出所有租户
func (s *RBACService) ListTenants(ctx context.Context) ([]Tenant, error) {
	var tenants []Tenant
	if err := s.db.WithContext(ctx).Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}
