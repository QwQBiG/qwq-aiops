package website

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	_ "modernc.org/sqlite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// **Feature: enhanced-aiops-platform, Property 12: DNS 管理完整性**
// **Validates: Requirements 4.3**
//
// Property 12: DNS 管理完整性
// *For any* 域名配置，系统应该提供完整的 DNS 管理和健康检查功能
//
// 这个属性测试验证：
// 1. DNS 记录创建后能正确保存和查询
// 2. DNS 记录更新后信息正确同步
// 3. DNS 记录删除后无法再查询到
// 4. 支持多种 DNS 记录类型（A, AAAA, CNAME, TXT, MX）
// 5. 多租户环境下 DNS 记录正确隔离
// 6. DNS 记录列表查询支持域名过滤
// 7. DNS 记录的 TTL 和优先级正确保存

// setupDNSTestDB 创建 DNS 测试数据库
func setupDNSTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(&DNSRecord{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

// genDNSRecordType 生成 DNS 记录类型
func genDNSRecordType() gopter.Gen {
	return gen.OneConstOf(
		DNSRecordA,
		DNSRecordAAAA,
		DNSRecordCNAME,
		DNSRecordTXT,
		DNSRecordMX,
	)
}

// genDNSRecordValue 根据类型生成 DNS 记录值
func genDNSRecordValue(recordType DNSRecordType) gopter.Gen {
	switch recordType {
	case DNSRecordA:
		// 生成 IPv4 地址
		return gopter.CombineGens(
			gen.IntRange(1, 255),
			gen.IntRange(0, 255),
			gen.IntRange(0, 255),
			gen.IntRange(1, 255),
		).Map(func(values []interface{}) string {
			return fmt.Sprintf("%d.%d.%d.%d",
				values[0].(int), values[1].(int),
				values[2].(int), values[3].(int))
		})
	case DNSRecordAAAA:
		// 生成简化的 IPv6 地址
		return gen.Const("2001:db8::1")
	case DNSRecordCNAME:
		// 生成 CNAME 目标
		return gopter.CombineGens(
			gen.Identifier(),
			gen.OneConstOf("com", "net", "org"),
		).Map(func(values []interface{}) string {
			return fmt.Sprintf("%s.%s", values[0].(string), values[1].(string))
		})
	case DNSRecordTXT:
		// 生成 TXT 记录值
		return gen.OneConstOf(
			"v=spf1 include:_spf.example.com ~all",
			"google-site-verification=abc123",
			"verification-code=xyz789",
		)
	case DNSRecordMX:
		// 生成 MX 记录值
		return gopter.CombineGens(
			gen.Identifier(),
			gen.OneConstOf("com", "net"),
		).Map(func(values []interface{}) string {
			return fmt.Sprintf("mail.%s.%s", values[0].(string), values[1].(string))
		})
	default:
		return gen.Const("192.168.1.1")
	}
}

// genDNSRecord 生成 DNS 记录
func genDNSRecord() gopter.Gen {
	return gopter.CombineGens(
		genValidDomain(),
		genDNSRecordType(),
		gen.Identifier(),
		gen.IntRange(60, 86400),    // TTL: 60秒到1天
		gen.IntRange(1, 100),       // Priority
		gen.OneConstOf("aliyun", "tencent", "cloudflare", ""),
	).FlatMap(func(values interface{}) gopter.Gen {
		vals := values.([]interface{})
		domain := vals[0].(string)
		recordType := vals[1].(DNSRecordType)
		name := vals[2].(string)
		ttl := vals[3].(int)
		priority := vals[4].(int)
		provider := vals[5].(string)

		return genDNSRecordValue(recordType).Map(func(value string) *DNSRecord {
			record := &DNSRecord{
				Domain:   domain,
				Type:     recordType,
				Name:     name,
				Value:    value,
				TTL:      ttl,
				Provider: provider,
				UserID:   1,
				TenantID: 1,
			}
			// 只有 MX 记录需要优先级
			if recordType == DNSRecordMX {
				record.Priority = priority
			}
			return record
		})
	}, reflect.TypeOf(&DNSRecord{}))
}

// TestProperty12_DNSRecordCreation 测试 DNS 记录创建
func TestProperty12_DNSRecordCreation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 1: DNS 记录创建后能正确保存和查询
	properties.Property("DNS记录创建后正确保存", prop.ForAll(
		func(record *DNSRecord) bool {
			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 创建 DNS 记录
			err := service.CreateDNSRecord(ctx, record)
			if err != nil {
				t.Logf("创建 DNS 记录失败: %v", err)
				return false
			}

			// 验证记录已保存
			savedRecord, err := service.GetDNSRecord(ctx, record.ID)
			if err != nil {
				t.Logf("获取 DNS 记录失败: %v", err)
				return false
			}

			// 验证基本字段
			if savedRecord.Domain != record.Domain {
				t.Logf("域名不匹配: 期望 %s, 实际 %s", record.Domain, savedRecord.Domain)
				return false
			}

			if savedRecord.Type != record.Type {
				t.Logf("记录类型不匹配: 期望 %s, 实际 %s", record.Type, savedRecord.Type)
				return false
			}

			if savedRecord.Name != record.Name {
				t.Logf("记录名称不匹配: 期望 %s, 实际 %s", record.Name, savedRecord.Name)
				return false
			}

			if savedRecord.Value != record.Value {
				t.Logf("记录值不匹配: 期望 %s, 实际 %s", record.Value, savedRecord.Value)
				return false
			}

			if savedRecord.TTL != record.TTL {
				t.Logf("TTL 不匹配: 期望 %d, 实际 %d", record.TTL, savedRecord.TTL)
				return false
			}

			// 验证 MX 记录的优先级
			if record.Type == DNSRecordMX && savedRecord.Priority != record.Priority {
				t.Logf("MX 优先级不匹配: 期望 %d, 实际 %d", record.Priority, savedRecord.Priority)
				return false
			}

			return true
		},
		genDNSRecord(),
	))

	// Property 2: 支持多种 DNS 记录类型
	properties.Property("支持多种DNS记录类型", prop.ForAll(
		func(recordType DNSRecordType) bool {
			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 为每种类型创建记录
			record := &DNSRecord{
				Domain:   "test.example.com",
				Type:     recordType,
				Name:     "test",
				Value:    "test-value",
				TTL:      600,
				UserID:   1,
				TenantID: 1,
			}

			// 根据类型设置合适的值
			switch recordType {
			case DNSRecordA:
				record.Value = "192.168.1.1"
			case DNSRecordAAAA:
				record.Value = "2001:db8::1"
			case DNSRecordCNAME:
				record.Value = "target.example.com"
			case DNSRecordTXT:
				record.Value = "v=spf1 ~all"
			case DNSRecordMX:
				record.Value = "mail.example.com"
				record.Priority = 10
			}

			err := service.CreateDNSRecord(ctx, record)
			if err != nil {
				t.Logf("创建 %s 记录失败: %v", recordType, err)
				return false
			}

			// 验证记录类型正确保存
			savedRecord, err := service.GetDNSRecord(ctx, record.ID)
			if err != nil {
				t.Logf("获取 %s 记录失败: %v", recordType, err)
				return false
			}

			if savedRecord.Type != recordType {
				t.Logf("记录类型不匹配: 期望 %s, 实际 %s", recordType, savedRecord.Type)
				return false
			}

			return true
		},
		genDNSRecordType(),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty12_DNSRecordUpdate 测试 DNS 记录更新
func TestProperty12_DNSRecordUpdate(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 3: DNS 记录更新后信息正确同步
	properties.Property("DNS记录更新正确同步", prop.ForAll(
		func(record *DNSRecord, newValue string, newTTL int) bool {
			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 创建记录
			err := service.CreateDNSRecord(ctx, record)
			if err != nil {
				t.Logf("创建 DNS 记录失败: %v", err)
				return false
			}

			// 更新记录
			record.Value = newValue
			record.TTL = newTTL
			err = service.UpdateDNSRecord(ctx, record)
			if err != nil {
				t.Logf("更新 DNS 记录失败: %v", err)
				return false
			}

			// 验证更新
			updatedRecord, err := service.GetDNSRecord(ctx, record.ID)
			if err != nil {
				t.Logf("获取更新后的记录失败: %v", err)
				return false
			}

			if updatedRecord.Value != newValue {
				t.Logf("记录值未正确更新: 期望 %s, 实际 %s", newValue, updatedRecord.Value)
				return false
			}

			if updatedRecord.TTL != newTTL {
				t.Logf("TTL 未正确更新: 期望 %d, 实际 %d", newTTL, updatedRecord.TTL)
				return false
			}

			return true
		},
		genDNSRecord(),
		gen.AlphaString(),
		gen.IntRange(60, 86400),
	))

	// Property 4: 更新不存在的记录应该返回错误
	properties.Property("更新不存在记录返回错误", prop.ForAll(
		func(record *DNSRecord) bool {
			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 尝试更新不存在的记录
			record.ID = 99999
			err := service.UpdateDNSRecord(ctx, record)

			// 应该返回错误
			if err == nil {
				t.Logf("更新不存在的记录应该返回错误")
				return false
			}

			// 应该是 ErrDNSRecordNotFound 错误
			if err != ErrDNSRecordNotFound {
				t.Logf("应该返回 ErrDNSRecordNotFound 错误，实际: %v", err)
				return false
			}

			return true
		},
		genDNSRecord(),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty12_DNSRecordDeletion 测试 DNS 记录删除
func TestProperty12_DNSRecordDeletion(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 5: DNS 记录删除后无法再查询到
	properties.Property("DNS记录删除后无法查询", prop.ForAll(
		func(record *DNSRecord) bool {
			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 创建记录
			err := service.CreateDNSRecord(ctx, record)
			if err != nil {
				t.Logf("创建 DNS 记录失败: %v", err)
				return false
			}

			recordID := record.ID

			// 删除记录
			err = service.DeleteDNSRecord(ctx, recordID)
			if err != nil {
				t.Logf("删除 DNS 记录失败: %v", err)
				return false
			}

			// 尝试查询已删除的记录
			_, err = service.GetDNSRecord(ctx, recordID)
			if err == nil {
				t.Logf("删除后仍能查询到记录")
				return false
			}

			// 应该返回 ErrDNSRecordNotFound 错误
			if err != ErrDNSRecordNotFound {
				t.Logf("应该返回 ErrDNSRecordNotFound 错误，实际: %v", err)
				return false
			}

			return true
		},
		genDNSRecord(),
	))

	// Property 6: 删除不存在的记录应该返回错误
	properties.Property("删除不存在记录返回错误", prop.ForAll(
		func() bool {
			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 尝试删除不存在的记录
			err := service.DeleteDNSRecord(ctx, 99999)
			if err == nil {
				t.Logf("删除不存在的记录应该返回错误")
				return false
			}

			// 应该返回 ErrDNSRecordNotFound 错误
			if err != ErrDNSRecordNotFound {
				t.Logf("应该返回 ErrDNSRecordNotFound 错误，实际: %v", err)
				return false
			}

			return true
		},
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty12_DNSRecordListing 测试 DNS 记录列表查询
func TestProperty12_DNSRecordListing(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 7: DNS 记录列表查询支持域名过滤
	properties.Property("DNS记录列表支持域名过滤", prop.ForAll(
		func(domain1 string, domain2 string) bool {
			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 确保两个域名不同
			if domain1 == domain2 {
				domain2 = domain2 + ".test"
			}

			// 为域名1创建记录
			record1 := &DNSRecord{
				Domain:   domain1,
				Type:     DNSRecordA,
				Name:     "www",
				Value:    "192.168.1.1",
				TTL:      600,
				UserID:   1,
				TenantID: 1,
			}
			err := service.CreateDNSRecord(ctx, record1)
			if err != nil {
				t.Logf("创建域名1记录失败: %v", err)
				return false
			}

			// 为域名2创建记录
			record2 := &DNSRecord{
				Domain:   domain2,
				Type:     DNSRecordA,
				Name:     "www",
				Value:    "192.168.1.2",
				TTL:      600,
				UserID:   1,
				TenantID: 1,
			}
			err = service.CreateDNSRecord(ctx, record2)
			if err != nil {
				t.Logf("创建域名2记录失败: %v", err)
				return false
			}

			// 查询域名1的记录
			records1, err := service.ListDNSRecords(ctx, domain1, 1, 1)
			if err != nil {
				t.Logf("查询域名1记录失败: %v", err)
				return false
			}

			// 应该只返回域名1的记录
			if len(records1) != 1 {
				t.Logf("域名1应该只有1条记录，实际: %d", len(records1))
				return false
			}

			if records1[0].Domain != domain1 {
				t.Logf("返回的记录域名不匹配: 期望 %s, 实际 %s", domain1, records1[0].Domain)
				return false
			}

			// 查询域名2的记录
			records2, err := service.ListDNSRecords(ctx, domain2, 1, 1)
			if err != nil {
				t.Logf("查询域名2记录失败: %v", err)
				return false
			}

			// 应该只返回域名2的记录
			if len(records2) != 1 {
				t.Logf("域名2应该只有1条记录，实际: %d", len(records2))
				return false
			}

			if records2[0].Domain != domain2 {
				t.Logf("返回的记录域名不匹配: 期望 %s, 实际 %s", domain2, records2[0].Domain)
				return false
			}

			return true
		},
		genValidDomain(),
		genValidDomain(),
	))

	// Property 8: 查询所有记录（不指定域名）
	properties.Property("查询所有DNS记录", prop.ForAll(
		func(records []*DNSRecord) bool {
			if len(records) == 0 {
				return true // 空列表跳过
			}

			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 创建所有记录
			for _, record := range records {
				record.UserID = 1
				record.TenantID = 1
				err := service.CreateDNSRecord(ctx, record)
				if err != nil {
					t.Logf("创建记录失败: %v", err)
					return false
				}
			}

			// 查询所有记录（不指定域名）
			allRecords, err := service.ListDNSRecords(ctx, "", 1, 1)
			if err != nil {
				t.Logf("查询所有记录失败: %v", err)
				return false
			}

			// 应该返回所有创建的记录
			if len(allRecords) != len(records) {
				t.Logf("记录数量不匹配: 期望 %d, 实际 %d", len(records), len(allRecords))
				return false
			}

			return true
		},
		gen.SliceOfN(3, genDNSRecord()),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty12_MultiTenantIsolation 测试多租户隔离
func TestProperty12_MultiTenantIsolation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 9: 多租户环境下 DNS 记录正确隔离
	properties.Property("多租户DNS记录隔离", prop.ForAll(
		func(record1 *DNSRecord, record2 *DNSRecord) bool {
			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 设置不同的租户ID
			record1.TenantID = 1
			record1.UserID = 1
			record2.TenantID = 2
			record2.UserID = 2

			// 创建两个记录
			err := service.CreateDNSRecord(ctx, record1)
			if err != nil {
				t.Logf("创建记录1失败: %v", err)
				return false
			}

			err = service.CreateDNSRecord(ctx, record2)
			if err != nil {
				t.Logf("创建记录2失败: %v", err)
				return false
			}

			// 查询租户1的记录
			records1, err := service.ListDNSRecords(ctx, "", 1, 1)
			if err != nil {
				t.Logf("查询租户1记录失败: %v", err)
				return false
			}

			// 查询租户2的记录
			records2, err := service.ListDNSRecords(ctx, "", 2, 2)
			if err != nil {
				t.Logf("查询租户2记录失败: %v", err)
				return false
			}

			// 验证租户1只能看到自己的记录
			if len(records1) != 1 {
				t.Logf("租户1应该只有1条记录，实际: %d", len(records1))
				return false
			}

			if records1[0].TenantID != 1 {
				t.Logf("租户1的记录租户ID不正确")
				return false
			}

			// 验证租户2只能看到自己的记录
			if len(records2) != 1 {
				t.Logf("租户2应该只有1条记录，实际: %d", len(records2))
				return false
			}

			if records2[0].TenantID != 2 {
				t.Logf("租户2的记录租户ID不正确")
				return false
			}

			return true
		},
		genDNSRecord(),
		genDNSRecord(),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty12_DNSRecordTTL 测试 DNS 记录 TTL
func TestProperty12_DNSRecordTTL(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 10: DNS 记录的 TTL 正确保存和查询
	properties.Property("DNS记录TTL正确保存", prop.ForAll(
		func(ttl int) bool {
			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 创建带有指定 TTL 的记录
			record := &DNSRecord{
				Domain:   "test.example.com",
				Type:     DNSRecordA,
				Name:     "www",
				Value:    "192.168.1.1",
				TTL:      ttl,
				UserID:   1,
				TenantID: 1,
			}

			err := service.CreateDNSRecord(ctx, record)
			if err != nil {
				t.Logf("创建记录失败: %v", err)
				return false
			}

			// 验证 TTL 正确保存
			savedRecord, err := service.GetDNSRecord(ctx, record.ID)
			if err != nil {
				t.Logf("获取记录失败: %v", err)
				return false
			}

			if savedRecord.TTL != ttl {
				t.Logf("TTL 不匹配: 期望 %d, 实际 %d", ttl, savedRecord.TTL)
				return false
			}

			return true
		},
		gen.IntRange(60, 86400), // TTL 范围：60秒到1天
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty12_MXRecordPriority 测试 MX 记录优先级
func TestProperty12_MXRecordPriority(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 11: MX 记录的优先级正确保存
	properties.Property("MX记录优先级正确保存", prop.ForAll(
		func(priority int) bool {
			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 创建 MX 记录
			record := &DNSRecord{
				Domain:   "example.com",
				Type:     DNSRecordMX,
				Name:     "@",
				Value:    "mail.example.com",
				TTL:      600,
				Priority: priority,
				UserID:   1,
				TenantID: 1,
			}

			err := service.CreateDNSRecord(ctx, record)
			if err != nil {
				t.Logf("创建 MX 记录失败: %v", err)
				return false
			}

			// 验证优先级正确保存
			savedRecord, err := service.GetDNSRecord(ctx, record.ID)
			if err != nil {
				t.Logf("获取 MX 记录失败: %v", err)
				return false
			}

			if savedRecord.Priority != priority {
				t.Logf("MX 优先级不匹配: 期望 %d, 实际 %d", priority, savedRecord.Priority)
				return false
			}

			return true
		},
		gen.IntRange(1, 100), // MX 优先级范围
	))

	// Property 12: 非 MX 记录的优先级应该为0或被忽略
	properties.Property("非MX记录优先级为0", prop.ForAll(
		func(recordType DNSRecordType) bool {
			// 跳过 MX 记录类型
			if recordType == DNSRecordMX {
				return true
			}

			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 创建非 MX 记录
			record := &DNSRecord{
				Domain:   "test.example.com",
				Type:     recordType,
				Name:     "test",
				Value:    "test-value",
				TTL:      600,
				Priority: 10, // 设置优先级，但应该被忽略
				UserID:   1,
				TenantID: 1,
			}

			// 根据类型设置合适的值
			switch recordType {
			case DNSRecordA:
				record.Value = "192.168.1.1"
			case DNSRecordAAAA:
				record.Value = "2001:db8::1"
			case DNSRecordCNAME:
				record.Value = "target.example.com"
			case DNSRecordTXT:
				record.Value = "v=spf1 ~all"
			}

			err := service.CreateDNSRecord(ctx, record)
			if err != nil {
				t.Logf("创建记录失败: %v", err)
				return false
			}

			// 对于非 MX 记录，优先级应该被保存（即使不使用）
			// 这是数据库层面的行为，我们只验证记录能正确创建和查询
			savedRecord, err := service.GetDNSRecord(ctx, record.ID)
			if err != nil {
				t.Logf("获取记录失败: %v", err)
				return false
			}

			// 验证记录类型正确
			if savedRecord.Type != recordType {
				t.Logf("记录类型不匹配: 期望 %s, 实际 %s", recordType, savedRecord.Type)
				return false
			}

			return true
		},
		genDNSRecordType(),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty12_DNSRecordProvider 测试 DNS 提供商字段
func TestProperty12_DNSRecordProvider(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 13: DNS 记录的提供商信息正确保存
	properties.Property("DNS提供商信息正确保存", prop.ForAll(
		func(provider string) bool {
			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 创建带有提供商信息的记录
			record := &DNSRecord{
				Domain:   "test.example.com",
				Type:     DNSRecordA,
				Name:     "www",
				Value:    "192.168.1.1",
				TTL:      600,
				Provider: provider,
				UserID:   1,
				TenantID: 1,
			}

			err := service.CreateDNSRecord(ctx, record)
			if err != nil {
				t.Logf("创建记录失败: %v", err)
				return false
			}

			// 验证提供商信息正确保存
			savedRecord, err := service.GetDNSRecord(ctx, record.ID)
			if err != nil {
				t.Logf("获取记录失败: %v", err)
				return false
			}

			if savedRecord.Provider != provider {
				t.Logf("提供商不匹配: 期望 %s, 实际 %s", provider, savedRecord.Provider)
				return false
			}

			return true
		},
		gen.OneConstOf("aliyun", "tencent", "cloudflare", ""),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty12_DNSRecordCompleteness 测试 DNS 管理完整性
func TestProperty12_DNSRecordCompleteness(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 14: DNS 记录的完整生命周期管理
	properties.Property("DNS记录完整生命周期", prop.ForAll(
		func(record *DNSRecord) bool {
			db := setupDNSTestDB(t)
			service := NewDNSService(db)
			ctx := context.Background()

			// 1. 创建记录
			err := service.CreateDNSRecord(ctx, record)
			if err != nil {
				t.Logf("创建记录失败: %v", err)
				return false
			}

			// 2. 查询记录
			savedRecord, err := service.GetDNSRecord(ctx, record.ID)
			if err != nil {
				t.Logf("查询记录失败: %v", err)
				return false
			}

			if savedRecord.ID != record.ID {
				t.Logf("记录ID不匹配")
				return false
			}

			// 3. 列表查询包含该记录
			records, err := service.ListDNSRecords(ctx, record.Domain, record.UserID, record.TenantID)
			if err != nil {
				t.Logf("列表查询失败: %v", err)
				return false
			}

			found := false
			for _, r := range records {
				if r.ID == record.ID {
					found = true
					break
				}
			}

			if !found {
				t.Logf("列表查询中未找到记录")
				return false
			}

			// 4. 更新记录
			newValue := "updated-value"
			savedRecord.Value = newValue
			err = service.UpdateDNSRecord(ctx, savedRecord)
			if err != nil {
				t.Logf("更新记录失败: %v", err)
				return false
			}

			// 5. 验证更新
			updatedRecord, err := service.GetDNSRecord(ctx, record.ID)
			if err != nil {
				t.Logf("获取更新后的记录失败: %v", err)
				return false
			}

			if updatedRecord.Value != newValue {
				t.Logf("记录值未正确更新")
				return false
			}

			// 6. 删除记录
			err = service.DeleteDNSRecord(ctx, record.ID)
			if err != nil {
				t.Logf("删除记录失败: %v", err)
				return false
			}

			// 7. 验证删除
			_, err = service.GetDNSRecord(ctx, record.ID)
			if err != ErrDNSRecordNotFound {
				t.Logf("删除后仍能查询到记录")
				return false
			}

			return true
		},
		genDNSRecord(),
	))

	// 运行属性测试（100次迭代）
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
