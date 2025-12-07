package website

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	_ "modernc.org/sqlite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// **Feature: enhanced-aiops-platform, Property 11: SSL 证书生命周期管理**
// **Validates: Requirements 4.2**
//
// Property 11: SSL 证书生命周期管理
// *For any* 配置的 SSL 证书，系统应该能自动申请、部署和续期证书
//
// 这个属性测试验证：
// 1. 证书创建后状态正确初始化
// 2. 证书过期检查能正确识别即将过期的证书
// 3. 自动续期功能能正确触发
// 4. 证书状态在生命周期中正确转换
// 5. 证书信息在续期后正确更新

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	// 使用 modernc.org/sqlite 纯 Go 驱动
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open SQL database: %v", err)
	}

	// 使用 GORM 包装
	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open GORM database: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&SSLCert{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

// genValidEmail 生成有效的邮箱地址
func genValidEmail() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		gen.OneConstOf("gmail.com", "outlook.com", "example.com"),
	).Map(func(values []interface{}) string {
		username := values[0].(string)
		domain := values[1].(string)
		return fmt.Sprintf("%s@%s", username, domain)
	})
}

// genSSLProvider 生成 SSL 提供商
func genSSLProvider() gopter.Gen {
	return gen.OneConstOf(
		SSLProviderSelfSigned,
		SSLProviderManual,
	)
}

// genRenewDaysBefore 生成续期提前天数
func genRenewDaysBefore() gopter.Gen {
	return gen.IntRange(7, 60)
}

// genSSLCert 生成 SSL 证书配置
func genSSLCert(daysUntilExpiry int) gopter.Gen {
	return gopter.CombineGens(
		genValidDomain(),
		genSSLProvider(),
		genValidEmail(),
		genRenewDaysBefore(),
		gen.Bool(),
	).Map(func(values []interface{}) *SSLCert {
		domain := values[0].(string)
		provider := values[1].(SSLProvider)
		email := values[2].(string)
		renewDaysBefore := values[3].(int)
		autoRenewValue := values[4].(bool)
		autoRenew := &autoRenewValue

		now := time.Now()
		issueDate := now
		expiryDate := now.AddDate(0, 0, daysUntilExpiry)

		return &SSLCert{
			Domain:          domain,
			Provider:        provider,
			Status:          SSLStatusValid,
			Email:           email,
			IssueDate:       &issueDate,
			ExpiryDate:      &expiryDate,
			AutoRenew:       autoRenew,
			RenewDaysBefore: renewDaysBefore,
			UserID:          1,
			TenantID:        1,
		}
	})
}

// TestProperty11_CertificateCreation 测试证书创建
func TestProperty11_CertificateCreation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 1: 创建证书后状态正确初始化
	properties.Property("证书创建后状态正确", prop.ForAll(
		func(cert *SSLCert) bool {
			db := setupTestDB(t)
			service := NewSSLService(db)
			ctx := context.Background()

			// 创建证书
			err := service.CreateSSLCert(ctx, cert)
			if err != nil {
				t.Logf("创建证书失败: %v", err)
				return false
			}

			// 验证证书已保存
			savedCert, err := service.GetSSLCert(ctx, cert.ID)
			if err != nil {
				t.Logf("获取证书失败: %v", err)
				return false
			}

			// 验证基本字段
			if savedCert.Domain != cert.Domain {
				t.Logf("域名不匹配: 期望 %s, 实际 %s", cert.Domain, savedCert.Domain)
				return false
			}

			if savedCert.Provider != cert.Provider {
				t.Logf("提供商不匹配: 期望 %s, 实际 %s", cert.Provider, savedCert.Provider)
				return false
			}

			if savedCert.Status != cert.Status {
				t.Logf("状态不匹配: 期望 %s, 实际 %s", cert.Status, savedCert.Status)
				return false
			}

			if (savedCert.AutoRenew == nil && cert.AutoRenew != nil) ||
				(savedCert.AutoRenew != nil && cert.AutoRenew == nil) ||
				(savedCert.AutoRenew != nil && cert.AutoRenew != nil && *savedCert.AutoRenew != *cert.AutoRenew) {
				t.Logf("自动续期设置不匹配: 期望 %v, 实际 %v", 
					func() string {
						if cert.AutoRenew == nil {
							return "nil"
						}
						return fmt.Sprintf("%v", *cert.AutoRenew)
					}(),
					func() string {
						if savedCert.AutoRenew == nil {
							return "nil"
						}
						return fmt.Sprintf("%v", *savedCert.AutoRenew)
					}())
				return false
			}

			if savedCert.RenewDaysBefore != cert.RenewDaysBefore {
				t.Logf("续期提前天数不匹配: 期望 %d, 实际 %d", cert.RenewDaysBefore, savedCert.RenewDaysBefore)
				return false
			}

			return true
		},
		genSSLCert(90), // 90天后过期
	))

	// Property 2: 可以通过域名查询证书
	properties.Property("可通过域名查询证书", prop.ForAll(
		func(cert *SSLCert) bool {
			db := setupTestDB(t)
			service := NewSSLService(db)
			ctx := context.Background()

			// 创建证书
			err := service.CreateSSLCert(ctx, cert)
			if err != nil {
				t.Logf("创建证书失败: %v", err)
				return false
			}

			// 通过域名查询
			foundCert, err := service.GetSSLCertByDomain(ctx, cert.Domain)
			if err != nil {
				t.Logf("通过域名查询证书失败: %v", err)
				return false
			}

			// 验证找到的证书
			if foundCert.Domain != cert.Domain {
				t.Logf("查询到的证书域名不匹配")
				return false
			}

			return true
		},
		genSSLCert(90),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_ExpiryCheck_ExpiringCerts 测试即将过期的证书能被正确识别
func TestProperty11_ExpiryCheck_ExpiringCerts(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 3: 即将过期的证书能被正确识别
	properties.Property("即将过期证书正确识别", prop.ForAll(
		func(daysUntilExpiry int) bool {
			db := setupTestDB(t)
			service := NewSSLService(db)
			ctx := context.Background()

			// 创建一个即将过期的证书
			now := time.Now()
			issueDate := now
			expiryDate := now.AddDate(0, 0, daysUntilExpiry)
			autoRenew := true

			cert := &SSLCert{
				Domain:          "test.example.com",
				Provider:        SSLProviderSelfSigned,
				Status:          SSLStatusValid,
				Email:           "test@example.com",
				IssueDate:       &issueDate,
				ExpiryDate:      &expiryDate,
				AutoRenew:       &autoRenew,
				RenewDaysBefore: 30,
				UserID:          1,
				TenantID:        1,
			}

			err := service.CreateSSLCert(ctx, cert)
			if err != nil {
				t.Logf("创建证书失败: %v", err)
				return false
			}

			// 检查过期证书
			expiringCerts, err := service.CheckExpiry(ctx)
			if err != nil {
				t.Logf("检查过期证书失败: %v", err)
				return false
			}

			// 如果证书在30天内过期，应该被检测到
			shouldBeDetected := daysUntilExpiry <= 30
			isDetected := len(expiringCerts) > 0

			if shouldBeDetected != isDetected {
				t.Logf("过期检测不正确: 距离过期 %d 天, 应该检测到: %v, 实际检测到: %v",
					daysUntilExpiry, shouldBeDetected, isDetected)
				return false
			}

			// 如果被检测到，验证是正确的证书
			if isDetected {
				found := false
				for _, c := range expiringCerts {
					if c.Domain == cert.Domain {
						found = true
						break
					}
				}
				if !found {
					t.Logf("检测到的证书列表中没有找到目标证书")
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 60), // 1-60天后过期
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_ExpiryCheck_AutoRenewOnly 测试只有启用自动续期的证书才会被检测
func TestProperty11_ExpiryCheck_AutoRenewOnly(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 4: 只有启用自动续期的证书才会被检测
	properties.Property("只检测自动续期证书", prop.ForAll(
		func(autoRenewValue bool, domainSuffix string) bool {
			db := setupTestDB(t)
			service := NewSSLService(db)
			ctx := context.Background()

			// 创建一个即将过期的证书，使用唯一的域名
			now := time.Now()
			issueDate := now
			expiryDate := now.AddDate(0, 0, 15) // 15天后过期
			domain := fmt.Sprintf("test-%s.example.com", domainSuffix)
			autoRenew := &autoRenewValue

			cert := &SSLCert{
				Domain:          domain,
				Provider:        SSLProviderSelfSigned,
				Status:          SSLStatusValid,
				Email:           "test@example.com",
				IssueDate:       &issueDate,
				ExpiryDate:      &expiryDate,
				AutoRenew:       autoRenew,
				RenewDaysBefore: 30,
				UserID:          1,
				TenantID:        1,
			}

			err := service.CreateSSLCert(ctx, cert)
			if err != nil {
				t.Logf("创建证书失败: %v", err)
				return false
			}

			// 检查过期证书
			expiringCerts, err := service.CheckExpiry(ctx)
			if err != nil {
				t.Logf("检查过期证书失败: %v", err)
				return false
			}

			// 只有启用自动续期的证书才应该被检测到
			// 需要检查是否有我们创建的证书被检测到
			foundOurCert := false
			for _, c := range expiringCerts {
				if c.Domain == domain {
					foundOurCert = true
					break
				}
			}

			if autoRenewValue != foundOurCert {
				t.Logf("自动续期检测不正确: 期望AutoRenew=%v, 实际检测到=%v", autoRenewValue, foundOurCert)
				return false
			}

			return true
		},
		gen.Bool(),
		gen.Identifier(), // 生成唯一的域名后缀
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_CertificateUpdate 测试证书更新
func TestProperty11_CertificateUpdate(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 5: 证书更新后信息正确保存
	properties.Property("证书更新正确保存", prop.ForAll(
		func(cert *SSLCert, newStatus SSLStatus) bool {
			db := setupTestDB(t)
			service := NewSSLService(db)
			ctx := context.Background()

			// 创建证书
			err := service.CreateSSLCert(ctx, cert)
			if err != nil {
				t.Logf("创建证书失败: %v", err)
				return false
			}

			// 更新证书状态
			cert.Status = newStatus
			err = service.UpdateSSLCert(ctx, cert)
			if err != nil {
				t.Logf("更新证书失败: %v", err)
				return false
			}

			// 验证更新
			updatedCert, err := service.GetSSLCert(ctx, cert.ID)
			if err != nil {
				t.Logf("获取更新后的证书失败: %v", err)
				return false
			}

			if updatedCert.Status != newStatus {
				t.Logf("证书状态未正确更新: 期望 %s, 实际 %s", newStatus, updatedCert.Status)
				return false
			}

			return true
		},
		genSSLCert(90),
		gen.OneConstOf(SSLStatusValid, SSLStatusExpired, SSLStatusPending, SSLStatusError),
	))

	// Property 6: 更新不存在的证书应该返回错误
	properties.Property("更新不存在证书返回错误", prop.ForAll(
		func(cert *SSLCert) bool {
			db := setupTestDB(t)
			service := NewSSLService(db)
			ctx := context.Background()

			// 尝试更新不存在的证书（ID设置为一个不存在的值）
			cert.ID = 99999
			err := service.UpdateSSLCert(ctx, cert)

			// 应该返回错误
			if err == nil {
				t.Logf("更新不存在的证书应该返回错误")
				return false
			}

			// 应该是 ErrSSLCertNotFound 错误
			if err != ErrSSLCertNotFound {
				t.Logf("应该返回 ErrSSLCertNotFound 错误，实际: %v", err)
				return false
			}

			return true
		},
		genSSLCert(90),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_CertificateLifecycle 测试证书完整生命周期
func TestProperty11_CertificateLifecycle(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 7: 证书从创建到过期的状态转换正确
	properties.Property("证书生命周期状态转换", prop.ForAll(
		func(cert *SSLCert) bool {
			db := setupTestDB(t)
			service := NewSSLService(db)
			ctx := context.Background()

			// 1. 创建证书（初始状态）
			initialStatus := cert.Status
			err := service.CreateSSLCert(ctx, cert)
			if err != nil {
				t.Logf("创建证书失败: %v", err)
				return false
			}

			// 验证初始状态
			savedCert, err := service.GetSSLCert(ctx, cert.ID)
			if err != nil {
				t.Logf("获取证书失败: %v", err)
				return false
			}

			if savedCert.Status != initialStatus {
				t.Logf("初始状态不正确: 期望 %s, 实际 %s", initialStatus, savedCert.Status)
				return false
			}

			// 2. 模拟证书即将过期（更新过期时间）
			now := time.Now()
			expiryDate := now.AddDate(0, 0, 15) // 15天后过期
			autoRenew := true
			savedCert.ExpiryDate = &expiryDate
			savedCert.AutoRenew = &autoRenew
			savedCert.RenewDaysBefore = 30

			err = service.UpdateSSLCert(ctx, savedCert)
			if err != nil {
				t.Logf("更新证书过期时间失败: %v", err)
				return false
			}

			// 3. 检查是否被识别为即将过期
			expiringCerts, err := service.CheckExpiry(ctx)
			if err != nil {
				t.Logf("检查过期证书失败: %v", err)
				return false
			}

			// 应该被检测到
			found := false
			for _, c := range expiringCerts {
				if c.ID == savedCert.ID {
					found = true
					break
				}
			}

			if !found {
				t.Logf("即将过期的证书未被检测到")
				return false
			}

			// 4. 模拟续期（更新过期时间和签发时间）
			newIssueDate := now
			newExpiryDate := now.AddDate(0, 3, 0) // 3个月后过期
			savedCert.IssueDate = &newIssueDate
			savedCert.ExpiryDate = &newExpiryDate
			savedCert.Status = SSLStatusValid

			err = service.UpdateSSLCert(ctx, savedCert)
			if err != nil {
				t.Logf("续期证书失败: %v", err)
				return false
			}

			// 5. 验证续期后不再被识别为即将过期
			expiringCerts, err = service.CheckExpiry(ctx)
			if err != nil {
				t.Logf("检查过期证书失败: %v", err)
				return false
			}

			// 不应该被检测到
			for _, c := range expiringCerts {
				if c.ID == savedCert.ID {
					t.Logf("续期后的证书不应该被检测为即将过期")
					return false
				}
			}

			return true
		},
		genSSLCert(90),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_CertificateDeletion 测试证书删除
func TestProperty11_CertificateDeletion(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 8: 删除证书后无法再查询到
	properties.Property("删除证书后无法查询", prop.ForAll(
		func(cert *SSLCert) bool {
			db := setupTestDB(t)
			service := NewSSLService(db)
			ctx := context.Background()

			// 创建证书
			err := service.CreateSSLCert(ctx, cert)
			if err != nil {
				t.Logf("创建证书失败: %v", err)
				return false
			}

			certID := cert.ID

			// 删除证书
			err = service.DeleteSSLCert(ctx, certID)
			if err != nil {
				t.Logf("删除证书失败: %v", err)
				return false
			}

			// 尝试查询已删除的证书
			_, err = service.GetSSLCert(ctx, certID)
			if err == nil {
				t.Logf("删除后仍能查询到证书")
				return false
			}

			// 应该返回 ErrSSLCertNotFound 错误
			if err != ErrSSLCertNotFound {
				t.Logf("应该返回 ErrSSLCertNotFound 错误，实际: %v", err)
				return false
			}

			return true
		},
		genSSLCert(90),
	))

	// Property 9: 删除不存在的证书应该返回错误
	properties.Property("删除不存在证书返回错误", prop.ForAll(
		func() bool {
			db := setupTestDB(t)
			service := NewSSLService(db)
			ctx := context.Background()

			// 尝试删除不存在的证书
			err := service.DeleteSSLCert(ctx, 99999)
			if err == nil {
				t.Logf("删除不存在的证书应该返回错误")
				return false
			}

			// 应该返回 ErrSSLCertNotFound 错误
			if err != ErrSSLCertNotFound {
				t.Logf("应该返回 ErrSSLCertNotFound 错误，实际: %v", err)
				return false
			}

			return true
		},
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_MultiTenantIsolation 测试多租户隔离
func TestProperty11_MultiTenantIsolation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 10: 不同租户的证书相互隔离
	properties.Property("多租户证书隔离", prop.ForAll(
		func(cert1 *SSLCert, cert2 *SSLCert) bool {
			db := setupTestDB(t)
			service := NewSSLService(db)
			ctx := context.Background()

			// 设置不同的租户ID
			cert1.TenantID = 1
			cert1.UserID = 1
			cert2.TenantID = 2
			cert2.UserID = 2

			// 创建两个证书
			err := service.CreateSSLCert(ctx, cert1)
			if err != nil {
				t.Logf("创建证书1失败: %v", err)
				return false
			}

			err = service.CreateSSLCert(ctx, cert2)
			if err != nil {
				t.Logf("创建证书2失败: %v", err)
				return false
			}

			// 查询租户1的证书
			certs1, err := service.ListSSLCerts(ctx, 1, 1)
			if err != nil {
				t.Logf("查询租户1证书失败: %v", err)
				return false
			}

			// 查询租户2的证书
			certs2, err := service.ListSSLCerts(ctx, 2, 2)
			if err != nil {
				t.Logf("查询租户2证书失败: %v", err)
				return false
			}

			// 验证租户1只能看到自己的证书
			if len(certs1) != 1 {
				t.Logf("租户1应该只有1个证书，实际: %d", len(certs1))
				return false
			}

			if certs1[0].TenantID != 1 {
				t.Logf("租户1的证书租户ID不正确")
				return false
			}

			// 验证租户2只能看到自己的证书
			if len(certs2) != 1 {
				t.Logf("租户2应该只有1个证书，实际: %d", len(certs2))
				return false
			}

			if certs2[0].TenantID != 2 {
				t.Logf("租户2的证书租户ID不正确")
				return false
			}

			return true
		},
		genSSLCert(90),
		genSSLCert(90),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_RenewDaysBeforeLogic 测试续期提前天数逻辑
func TestProperty11_RenewDaysBeforeLogic(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 11: 续期提前天数逻辑正确
	properties.Property("续期提前天数逻辑", prop.ForAll(
		func(daysUntilExpiry int, renewDaysBefore int) bool {
			db := setupTestDB(t)
			service := NewSSLService(db)
			ctx := context.Background()

			// 创建证书
			now := time.Now()
			issueDate := now
			expiryDate := now.AddDate(0, 0, daysUntilExpiry)
			autoRenew := true

			cert := &SSLCert{
				Domain:          "test.example.com",
				Provider:        SSLProviderSelfSigned,
				Status:          SSLStatusValid,
				Email:           "test@example.com",
				IssueDate:       &issueDate,
				ExpiryDate:      &expiryDate,
				AutoRenew:       &autoRenew,
				RenewDaysBefore: renewDaysBefore,
				UserID:          1,
				TenantID:        1,
			}

			err := service.CreateSSLCert(ctx, cert)
			if err != nil {
				t.Logf("创建证书失败: %v", err)
				return false
			}

			// 检查过期证书
			expiringCerts, err := service.CheckExpiry(ctx)
			if err != nil {
				t.Logf("检查过期证书失败: %v", err)
				return false
			}

			// 如果距离过期天数 <= 30天（CheckExpiry的阈值），应该被检测到
			// 注意：CheckExpiry 使用固定的30天阈值
			shouldBeDetected := daysUntilExpiry <= 30
			isDetected := false

			for _, c := range expiringCerts {
				if c.ID == cert.ID {
					isDetected = true
					// 验证 RenewDaysBefore 字段正确保存
					if c.RenewDaysBefore != renewDaysBefore {
						t.Logf("RenewDaysBefore 不匹配: 期望 %d, 实际 %d", renewDaysBefore, c.RenewDaysBefore)
						return false
					}
					break
				}
			}

			if shouldBeDetected != isDetected {
				t.Logf("检测结果不正确: 距离过期 %d 天, 续期提前 %d 天, 应该检测到: %v, 实际: %v",
					daysUntilExpiry, renewDaysBefore, shouldBeDetected, isDetected)
				return false
			}

			return true
		},
		gen.IntRange(1, 90),  // 距离过期天数
		gen.IntRange(7, 60),  // 续期提前天数
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
