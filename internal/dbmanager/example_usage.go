package dbmanager

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// ExampleUsage 展示数据库管理服务的使用示例
func ExampleUsage(db *gorm.DB) {
	ctx := context.Background()
	
	// 1. 创建数据库管理服务
	fmt.Println("=== 1. 创建数据库管理服务 ===")
	encryptionKey := "qwq-aiops-encryption-key-32b"
	dbService := NewDatabaseService(db, encryptionKey)
	fmt.Println("✓ 数据库管理服务创建成功")
	
	// 2. 创建MySQL数据库连接
	fmt.Println("\n=== 2. 创建MySQL数据库连接 ===")
	mysqlConn := &DatabaseConnection{
		Name:     "测试MySQL数据库",
		Type:     DatabaseTypeMySQL,
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "testdb",
		UserID:   1,
		TenantID: 1,
	}
	
	if err := dbService.CreateConnection(ctx, mysqlConn); err != nil {
		log.Printf("创建连接失败: %v", err)
	} else {
		fmt.Printf("✓ MySQL连接创建成功 (ID: %d)\n", mysqlConn.ID)
	}
	
	// 3. 测试数据库连接
	fmt.Println("\n=== 3. 测试数据库连接 ===")
	if err := dbService.TestConnection(ctx, mysqlConn); err != nil {
		log.Printf("连接测试失败: %v", err)
	} else {
		fmt.Println("✓ 数据库连接测试成功")
	}
	
	// 4. 列出所有数据库
	fmt.Println("\n=== 4. 列出所有数据库 ===")
	databases, err := dbService.ListDatabases(ctx, mysqlConn.ID)
	if err != nil {
		log.Printf("列出数据库失败: %v", err)
	} else {
		fmt.Printf("找到 %d 个数据库:\n", len(databases))
		for _, db := range databases {
			fmt.Printf("  - %s (大小: %d MB, 表数: %d)\n", 
				db.Name, db.Size/1024/1024, db.Tables)
		}
	}
	
	// 5. 执行SQL查询
	fmt.Println("\n=== 5. 执行SQL查询 ===")
	queryReq := &QueryRequest{
		ConnectionID: mysqlConn.ID,
		SQL:          "SELECT * FROM users LIMIT 10",
		Timeout:      30,
		MaxRows:      10,
	}
	
	result, err := dbService.ExecuteQuery(ctx, queryReq)
	if err != nil {
		log.Printf("查询执行失败: %v", err)
	} else {
		fmt.Printf("✓ 查询执行成功\n")
		fmt.Printf("  执行时间: %.2f ms\n", result.ExecutionTime)
		fmt.Printf("  返回行数: %d\n", len(result.Rows))
		fmt.Printf("  列名: %v\n", result.Columns)
	}
	
	// 6. AI查询优化
	fmt.Println("\n=== 6. AI查询优化 ===")
	sql := "SELECT * FROM orders WHERE user_id = 123 ORDER BY created_at DESC"
	optimization, err := dbService.OptimizeQuery(ctx, mysqlConn.ID, sql)
	if err != nil {
		log.Printf("查询优化失败: %v", err)
	} else {
		fmt.Println("✓ 查询优化分析完成")
		fmt.Printf("  原始SQL: %s\n", optimization.OriginalSQL)
		fmt.Printf("  优化后SQL: %s\n", optimization.OptimizedSQL)
		fmt.Printf("  预计性能提升: %.1f%%\n", optimization.EstimatedImprovement)
		
		fmt.Println("\n  优化建议:")
		for i, suggestion := range optimization.Suggestions {
			fmt.Printf("    %d. %s\n", i+1, suggestion)
		}
		
		if len(optimization.IndexRecommendations) > 0 {
			fmt.Println("\n  索引推荐:")
			for i, idx := range optimization.IndexRecommendations {
				fmt.Printf("    %d. 表: %s, 列: %v\n", i+1, idx.Table, idx.Columns)
				fmt.Printf("       原因: %s\n", idx.Reason)
				fmt.Printf("       SQL: %s\n", idx.CreateSQL)
			}
		}
	}
	
	// 7. 获取执行计划
	fmt.Println("\n=== 7. 获取SQL执行计划 ===")
	plan, err := dbService.GetExecutionPlan(ctx, mysqlConn.ID, sql)
	if err != nil {
		log.Printf("获取执行计划失败: %v", err)
	} else {
		fmt.Println("✓ 执行计划获取成功")
		for i, step := range plan.Steps {
			fmt.Printf("  步骤 %d:\n", i+1)
			fmt.Printf("    表: %s\n", step.Table)
			fmt.Printf("    类型: %s\n", step.Type)
			fmt.Printf("    扫描行数: %d\n", step.Rows)
			if step.Key != "" {
				fmt.Printf("    使用索引: %s\n", step.Key)
			}
		}
	}
	
	// 8. 创建备份配置
	fmt.Println("\n=== 8. 创建备份配置 ===")
	backupConfig := &BackupConfig{
		ConnectionID: mysqlConn.ID,
		Name:         "每日自动备份",
		Schedule:     "0 2 * * *", // 每天凌晨2点
		Enabled:      true,
		BackupPath:   "./backups",
		Compression:  true,
		Retention:    7, // 保留7天
		UserID:       1,
		TenantID:     1,
	}
	
	if err := dbService.CreateBackupConfig(ctx, backupConfig); err != nil {
		log.Printf("创建备份配置失败: %v", err)
	} else {
		fmt.Printf("✓ 备份配置创建成功 (ID: %d)\n", backupConfig.ID)
		fmt.Printf("  备份计划: %s\n", backupConfig.Schedule)
		fmt.Printf("  保留天数: %d\n", backupConfig.Retention)
	}
	
	// 9. 执行备份
	fmt.Println("\n=== 9. 执行数据库备份 ===")
	if err := dbService.ExecuteBackup(ctx, backupConfig.ID); err != nil {
		log.Printf("执行备份失败: %v", err)
	} else {
		fmt.Println("✓ 备份任务已启动")
	}
	
	// 10. 列出备份记录
	fmt.Println("\n=== 10. 列出备份记录 ===")
	records, err := dbService.ListBackupRecords(ctx, backupConfig.ID)
	if err != nil {
		log.Printf("列出备份记录失败: %v", err)
	} else {
		fmt.Printf("找到 %d 条备份记录:\n", len(records))
		for i, record := range records {
			fmt.Printf("  %d. 状态: %s, 文件: %s\n", 
				i+1, record.Status, record.FilePath)
			if record.EndTime != nil {
				duration := record.EndTime.Sub(record.StartTime)
				fmt.Printf("     耗时: %v, 大小: %d MB\n", 
					duration, record.FileSize/1024/1024)
			}
		}
	}
	
	// 11. 使用查询引擎
	fmt.Println("\n=== 11. 使用查询引擎 ===")
	queryEngine := NewQueryEngine(dbService)
	
	// 分页查询
	paginatedResult, err := queryEngine.ExecuteWithPagination(ctx, queryReq, 1, 10)
	if err != nil {
		log.Printf("分页查询失败: %v", err)
	} else {
		fmt.Println("✓ 分页查询成功")
		fmt.Printf("  当前页: %d/%d\n", paginatedResult.Page, paginatedResult.TotalPages)
		fmt.Printf("  总记录数: %d\n", paginatedResult.Total)
	}
	
	// 12. 使用性能分析器
	fmt.Println("\n=== 12. 使用性能分析器 ===")
	analyzer := NewPerformanceAnalyzer(dbService)
	
	report, err := analyzer.AnalyzeQuery(ctx, mysqlConn.ID, sql)
	if err != nil {
		log.Printf("性能分析失败: %v", err)
	} else {
		fmt.Println("✓ 性能分析完成")
		fmt.Printf("  执行时间: %.2f ms\n", report.ExecutionTime)
		fmt.Printf("  扫描行数: %d\n", report.RowsScanned)
		fmt.Printf("  返回行数: %d\n", report.RowsReturned)
		fmt.Printf("  使用索引: %v\n", report.IndexUsed)
		
		if len(report.Issues) > 0 {
			fmt.Println("\n  性能问题:")
			for i, issue := range report.Issues {
				fmt.Printf("    %d. %s\n", i+1, issue)
			}
		}
		
		if len(report.Recommendations) > 0 {
			fmt.Println("\n  优化建议:")
			for i, rec := range report.Recommendations {
				fmt.Printf("    %d. %s\n", i+1, rec)
			}
		}
	}
	
	// 13. 使用备份管理器
	fmt.Println("\n=== 13. 使用备份管理器 ===")
	backupManager := NewBackupManager(dbService, "./backups")
	
	// 启动定时备份
	backupManager.Start()
	fmt.Println("✓ 备份调度器已启动")
	
	// 调度备份任务
	if err := backupManager.ScheduleBackup(ctx, backupConfig); err != nil {
		log.Printf("调度备份任务失败: %v", err)
	} else {
		fmt.Println("✓ 备份任务已调度")
	}
	
	// 等待一段时间后停止
	time.Sleep(2 * time.Second)
	backupManager.Stop()
	fmt.Println("✓ 备份调度器已停止")
	
	fmt.Println("\n=== 示例完成 ===")
}

// ExampleAPIIntegration 展示如何集成API
func ExampleAPIIntegration(db *gorm.DB) {
	// 创建数据库管理服务
	encryptionKey := "qwq-aiops-encryption-key-32b"
	dbService := NewDatabaseService(db, encryptionKey)
	
	// 创建API
	_ = NewAPI(dbService)
	
	// 注册路由（需要Gin路由器）
	// router := gin.Default()
	// apiGroup := router.Group("/api")
	// api.RegisterRoutes(apiGroup)
	
	fmt.Println("数据库管理API已准备就绪")
	fmt.Println("可用的API端点:")
	fmt.Println("  POST   /api/database/connections")
	fmt.Println("  GET    /api/database/connections")
	fmt.Println("  POST   /api/database/query")
	fmt.Println("  POST   /api/database/connections/:id/optimize")
	fmt.Println("  POST   /api/database/backup/configs")
	fmt.Println("  ... 更多端点请参考README.md")
}
