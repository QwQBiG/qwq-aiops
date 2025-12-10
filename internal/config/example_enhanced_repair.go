// Package config 增强自动修复功能示例
// 展示如何使用新的自动修复功能
package config

import (
	"fmt"
	"log"
)

// ExampleEnhancedAutoRepair 展示增强自动修复功能的使用
func ExampleEnhancedAutoRepair() {
	fmt.Println("=== qwq AIOps 平台 - 增强自动修复功能示例 ===")
	
	// 1. 创建增强自动修复器
	options := DefaultAutoFixOptions()
	options.Verbose = true
	options.DryRun = false // 设置为 true 可以预览修复操作而不实际执行
	
	autoFixer := NewEnhancedAutoFixer(options)
	
	// 2. 运行全面自动修复
	fmt.Println("\n🚀 开始全面自动修复...")
	if err := autoFixer.RunComprehensiveRepair(); err != nil {
		log.Printf("自动修复失败: %v", err)
		return
	}
	
	// 3. 查看修复历史
	fmt.Println("\n📋 修复历史:")
	sessions, err := autoFixer.GetRepairHistory()
	if err != nil {
		log.Printf("获取修复历史失败: %v", err)
		return
	}
	
	for _, sessionID := range sessions {
		fmt.Printf("  - %s\n", sessionID)
	}
	
	// 4. 展示单独的诊断功能
	fmt.Println("\n🔍 运行配置诊断...")
	diagnostic := NewConfigDiagnostic()
	diagnostic.PrintDiagnosticReport()
	
	fmt.Println("\n✅ 示例完成")
}

// ExampleBasicAutoFix 展示基础自动修复功能
func ExampleBasicAutoFix() {
	fmt.Println("=== 基础自动修复功能示例 ===")
	
	diagnostic := NewConfigDiagnostic()
	
	// 运行诊断
	fmt.Println("🔍 运行诊断...")
	results := diagnostic.RunDiagnostics()
	
	hasIssues := false
	for _, result := range results {
		if result.Status == StatusError || result.Status == StatusWarning {
			hasIssues = true
			break
		}
	}
	
	if hasIssues {
		fmt.Println("⚠️  发现配置问题，开始自动修复...")
		if err := diagnostic.AutoFix(); err != nil {
			log.Printf("自动修复失败: %v", err)
		} else {
			fmt.Println("✅ 自动修复完成")
		}
	} else {
		fmt.Println("✅ 配置检查通过，无需修复")
	}
}

// ExampleRepairTracking 展示修复跟踪功能
func ExampleRepairTracking() {
	fmt.Println("=== 修复跟踪功能示例 ===")
	
	// 创建修复跟踪器
	tracker := NewRepairTracker("logs/repair_example.log")
	
	// 开始修复会话
	if err := tracker.StartSession(); err != nil {
		log.Printf("启动修复会话失败: %v", err)
		return
	}
	
	// 添加修复操作
	opID1 := tracker.AddOperation(RepairConfig, "修复配置文件", []string{"检查 .env", "创建默认配置"})
	opID2 := tracker.AddOperation(RepairFrontend, "修复前端资源", []string{"检查前端构建", "重建资源"})
	
	// 执行操作
	tracker.StartOperation(opID1)
	tracker.CompleteOperation(opID1, "配置文件检查完成", nil)
	
	tracker.StartOperation(opID2)
	tracker.CompleteOperation(opID2, "前端资源检查完成", nil)
	
	// 结束会话
	validationResult := &DeploymentValidationResult{
		Valid: true,
		ComponentsChecked: []string{"config", "frontend"},
		HealthyComponents: []string{"config", "frontend"},
	}
	
	if err := tracker.EndSession(validationResult); err != nil {
		log.Printf("结束修复会话失败: %v", err)
		return
	}
	
	// 打印摘要
	tracker.PrintSessionSummary()
}

// ExampleDryRunMode 展示预览模式
func ExampleDryRunMode() {
	fmt.Println("=== 预览模式示例 ===")
	
	options := DefaultAutoFixOptions()
	options.DryRun = true // 启用预览模式
	options.Verbose = true
	
	autoFixer := NewEnhancedAutoFixer(options)
	
	fmt.Println("🔍 预览修复操作（不会实际执行）...")
	if err := autoFixer.RunComprehensiveRepair(); err != nil {
		log.Printf("预览失败: %v", err)
		return
	}
	
	fmt.Println("✅ 预览完成，可以设置 DryRun=false 来实际执行修复")
}