package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// DB 全局数据库实例
	DB *gorm.DB
)

// Config 数据库配置
type Config struct {
	Type     string // sqlite 或 postgres
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	FilePath string // SQLite 文件路径
	Debug    bool   // 是否开启调试模式
}

// Init 初始化数据库连接
func Init(cfg Config) error {
	var dialector gorm.Dialector
	var err error

	// 配置日志级别
	logLevel := logger.Silent
	if cfg.Debug {
		logLevel = logger.Info
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// 根据数据库类型选择驱动
	switch cfg.Type {
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
		dialector = postgres.Open(dsn)
	case "sqlite":
		// SQLite需要CGO支持，在Windows环境下可能不可用
		// 建议使用PostgreSQL作为生产数据库
		return fmt.Errorf("SQLite需要CGO支持，请使用PostgreSQL数据库")
	default:
		return fmt.Errorf("不支持的数据库类型: %s (支持: postgres)", cfg.Type)
	}

	// 打开数据库连接
	DB, err = gorm.Open(dialector, gormConfig)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 配置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期

	log.Printf("✅ 数据库连接成功 (类型: %s)", cfg.Type)

	return nil
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 自动迁移所有模型
	err := DB.AutoMigrate(
		&User{},
		&Tenant{},
		&Permission{},
		&RolePermission{},
		&AuditLog{},
	)

	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	log.Println("✅ 数据库表结构迁移完成")

	return nil
}

// InitDefaultData 初始化默认数据
func InitDefaultData() error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 创建默认租户
	var defaultTenant Tenant
	result := DB.Where("code = ?", "default").First(&defaultTenant)
	if result.Error == gorm.ErrRecordNotFound {
		defaultTenant = Tenant{
			Name:    "默认租户",
			Code:    "default",
			Enabled: true,
		}
		if err := DB.Create(&defaultTenant).Error; err != nil {
			return fmt.Errorf("创建默认租户失败: %w", err)
		}
		log.Println("✅ 创建默认租户")
	}

	// 创建默认权限
	defaultPermissions := []Permission{
		{Name: "container:read", Resource: "container", Action: "read", Description: "查看容器"},
		{Name: "container:write", Resource: "container", Action: "write", Description: "创建/修改容器"},
		{Name: "container:delete", Resource: "container", Action: "delete", Description: "删除容器"},
		{Name: "application:read", Resource: "application", Action: "read", Description: "查看应用"},
		{Name: "application:write", Resource: "application", Action: "write", Description: "安装/修改应用"},
		{Name: "application:delete", Resource: "application", Action: "delete", Description: "卸载应用"},
		{Name: "website:read", Resource: "website", Action: "read", Description: "查看网站"},
		{Name: "website:write", Resource: "website", Action: "write", Description: "创建/修改网站"},
		{Name: "website:delete", Resource: "website", Action: "delete", Description: "删除网站"},
		{Name: "database:read", Resource: "database", Action: "read", Description: "查看数据库"},
		{Name: "database:write", Resource: "database", Action: "write", Description: "操作数据库"},
		{Name: "database:delete", Resource: "database", Action: "delete", Description: "删除数据库"},
		{Name: "backup:read", Resource: "backup", Action: "read", Description: "查看备份"},
		{Name: "backup:write", Resource: "backup", Action: "write", Description: "创建/修改备份"},
		{Name: "backup:delete", Resource: "backup", Action: "delete", Description: "删除备份"},
		{Name: "user:read", Resource: "user", Action: "read", Description: "查看用户"},
		{Name: "user:write", Resource: "user", Action: "write", Description: "创建/修改用户"},
		{Name: "user:delete", Resource: "user", Action: "delete", Description: "删除用户"},
		{Name: "tenant:read", Resource: "tenant", Action: "read", Description: "查看租户"},
		{Name: "tenant:write", Resource: "tenant", Action: "write", Description: "创建/修改租户"},
		{Name: "tenant:delete", Resource: "tenant", Action: "delete", Description: "删除租户"},
	}

	for _, perm := range defaultPermissions {
		var existingPerm Permission
		result := DB.Where("name = ?", perm.Name).First(&existingPerm)
		if result.Error == gorm.ErrRecordNotFound {
			if err := DB.Create(&perm).Error; err != nil {
				return fmt.Errorf("创建权限 %s 失败: %w", perm.Name, err)
			}
		}
	}
	log.Println("✅ 初始化默认权限")

	// 为管理员角色分配所有权限
	var allPermissions []Permission
	DB.Find(&allPermissions)
	for _, perm := range allPermissions {
		var existingRP RolePermission
		result := DB.Where("role = ? AND permission_id = ?", RoleAdmin, perm.ID).First(&existingRP)
		if result.Error == gorm.ErrRecordNotFound {
			rp := RolePermission{
				Role:         RoleAdmin,
				PermissionID: perm.ID,
			}
			if err := DB.Create(&rp).Error; err != nil {
				return fmt.Errorf("分配管理员权限失败: %w", err)
			}
		}
	}

	// 为普通用户角色分配读权限
	for _, perm := range allPermissions {
		if perm.Action == "read" {
			var existingRP RolePermission
			result := DB.Where("role = ? AND permission_id = ?", RoleUser, perm.ID).First(&existingRP)
			if result.Error == gorm.ErrRecordNotFound {
				rp := RolePermission{
					Role:         RoleUser,
					PermissionID: perm.ID,
				}
				if err := DB.Create(&rp).Error; err != nil {
					return fmt.Errorf("分配用户权限失败: %w", err)
				}
			}
		}
	}

	// 为只读用户角色分配读权限
	for _, perm := range allPermissions {
		if perm.Action == "read" {
			var existingRP RolePermission
			result := DB.Where("role = ? AND permission_id = ?", RoleReadOnly, perm.ID).First(&existingRP)
			if result.Error == gorm.ErrRecordNotFound {
				rp := RolePermission{
					Role:         RoleReadOnly,
					PermissionID: perm.ID,
				}
				if err := DB.Create(&rp).Error; err != nil {
					return fmt.Errorf("分配只读权限失败: %w", err)
				}
			}
		}
	}

	log.Println("✅ 初始化角色权限关联")

	return nil
}

// Close 关闭数据库连接
func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
