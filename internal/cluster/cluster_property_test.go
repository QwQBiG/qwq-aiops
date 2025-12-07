package cluster

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	
	_ "modernc.org/sqlite"
)

// **Feature: enhanced-aiops-platform, Property 27: 系统扩展性保证**
// **Validates: Requirements 10.2**
//
// Property 27: 系统扩展性保证
// *For any* 大量并发请求，系统应该能通过水平扩展维持性能和可用性
//
// 这个属性测试验证：
// 1. 集群能够注册多个节点
// 2. 负载均衡器能够在多个健康节点之间分配请求
// 3. 不同的负载均衡策略能够正确工作
// 4. 节点资源信息被正确跟踪
func TestProperty27_SystemScalabilityGuarantee(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("集群应该支持多节点水平扩展", prop.ForAll(
		func(nodeCount int, strategy string) bool {
			// 限制节点数量在合理范围内
			if nodeCount < 2 {
				nodeCount = 2
			}
			if nodeCount > 10 {
				nodeCount = 10
			}

			// 设置测试环境
			db := setupClusterTestDB(t)
			manager := NewClusterManager(db)

			ctx := context.Background()

			// 注册多个节点
			for i := 0; i < nodeCount; i++ {
				node := &Node{
					Name:              fmt.Sprintf("node-%d", i),
					Address:           fmt.Sprintf("192.168.1.%d", i+1),
					Port:              8080,
					Role:              NodeRoleWorker,
					Status:            NodeStatusHealthy,
					Version:           "1.0.0",
					CPUCores:          4,
					MemoryGB:          16.0,
					DiskGB:            100.0,
					CPUUsage:          float64(10 + i*5),  // 不同的负载
					MemoryUsage:       float64(20 + i*5),
					DiskUsage:         30.0,
					ActiveConnections: 10 + i*10,
					RequestsPerSecond: 100.0,
					LastHeartbeat:     time.Now(),
				}

				if err := manager.RegisterNode(ctx, node); err != nil {
					t.Logf("注册节点失败: %v", err)
					return false
				}
			}

			// 验证：所有节点都应该被注册
			nodes := manager.ListNodes()
			if len(nodes) != nodeCount {
				t.Logf("期望 %d 个节点，实际 %d 个", nodeCount, len(nodes))
				return false
			}

			// 验证：健康节点数量应该正确
			healthyNodes := manager.GetHealthyNodes()
			if len(healthyNodes) != nodeCount {
				t.Logf("期望 %d 个健康节点，实际 %d 个", nodeCount, len(healthyNodes))
				return false
			}

			// 验证：负载均衡应该能选择节点
			selectedNode, err := manager.SelectNode(strategy)
			if err != nil {
				t.Logf("选择节点失败: %v", err)
				return false
			}

			if selectedNode == nil {
				t.Logf("未选择到节点")
				return false
			}

			// 验证：选择的节点应该是健康的
			if selectedNode.Status != NodeStatusHealthy {
				t.Logf("选择的节点不健康")
				return false
			}

			// 验证：根据策略选择的节点应该符合预期
			switch strategy {
			case "least_connections":
				// 应该选择连接数最少的节点
				for _, node := range healthyNodes {
					if node.ActiveConnections < selectedNode.ActiveConnections {
						t.Logf("least_connections 策略应该选择连接数最少的节点")
						return false
					}
				}
			case "least_cpu":
				// 应该选择 CPU 使用率最低的节点
				for _, node := range healthyNodes {
					if node.CPUUsage < selectedNode.CPUUsage {
						t.Logf("least_cpu 策略应该选择 CPU 使用率最低的节点")
						return false
					}
				}
			}

			return true
		},
		gen.IntRange(2, 10),                                           // 节点数量
		gen.OneConstOf("round_robin", "least_connections", "least_cpu"), // 负载均衡策略
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty27_ClusterResourceTracking 测试集群资源跟踪
// **Feature: enhanced-aiops-platform, Property 27: 系统扩展性保证**
// **Validates: Requirements 10.2**
func TestProperty27_ClusterResourceTracking(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("集群应该正确跟踪资源使用情况", prop.ForAll(
		func(cpuCores int, memoryGB float64, diskGB float64) bool {
			// 限制资源在合理范围内
			if cpuCores < 1 {
				cpuCores = 1
			}
			if cpuCores > 64 {
				cpuCores = 64
			}
			if memoryGB < 1 {
				memoryGB = 1
			}
			if memoryGB > 256 {
				memoryGB = 256
			}
			if diskGB < 10 {
				diskGB = 10
			}
			if diskGB > 10000 {
				diskGB = 10000
			}

			// 设置测试环境
			db := setupClusterTestDB(t)
			manager := NewClusterManager(db)

			ctx := context.Background()

			// 注册节点
			node := &Node{
				Name:          "test-node",
				Address:       "192.168.1.1",
				Port:          8080,
				Role:          NodeRoleWorker,
				Status:        NodeStatusHealthy,
				CPUCores:      cpuCores,
				MemoryGB:      memoryGB,
				DiskGB:        diskGB,
				CPUUsage:      50.0,
				MemoryUsage:   60.0,
				DiskUsage:     40.0,
				LastHeartbeat: time.Now(),
			}

			if err := manager.RegisterNode(ctx, node); err != nil {
				t.Logf("注册节点失败: %v", err)
				return false
			}

			// 获取集群统计信息
			stats := manager.GetClusterStats()

			// 验证：总资源应该等于节点资源
			if stats.TotalCPU != float64(cpuCores) {
				t.Logf("总 CPU 不匹配: 期望 %.0f, 实际 %.0f", float64(cpuCores), stats.TotalCPU)
				return false
			}

			if stats.TotalMemory != memoryGB {
				t.Logf("总内存不匹配: 期望 %.2f, 实际 %.2f", memoryGB, stats.TotalMemory)
				return false
			}

			if stats.TotalDisk != diskGB {
				t.Logf("总磁盘不匹配: 期望 %.2f, 实际 %.2f", diskGB, stats.TotalDisk)
				return false
			}

			// 验证：已用资源应该根据使用率正确计算
			expectedUsedCPU := 50.0 * float64(cpuCores) / 100.0
			if stats.UsedCPU != expectedUsedCPU {
				t.Logf("已用 CPU 不匹配: 期望 %.2f, 实际 %.2f", expectedUsedCPU, stats.UsedCPU)
				return false
			}

			expectedUsedMemory := 60.0 * memoryGB / 100.0
			if stats.UsedMemory != expectedUsedMemory {
				t.Logf("已用内存不匹配: 期望 %.2f, 实际 %.2f", expectedUsedMemory, stats.UsedMemory)
				return false
			}

			return true
		},
		gen.IntRange(1, 64),        // CPU 核心数
		gen.Float64Range(1, 256),   // 内存 GB
		gen.Float64Range(10, 10000), // 磁盘 GB
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: enhanced-aiops-platform, Property 28: 零停机升级能力**
// **Validates: Requirements 10.5**
//
// Property 28: 零停机升级能力
// *For any* 系统版本升级，应该支持零停机升级和快速回滚机制
//
// 这个属性测试验证：
// 1. 节点可以被标记为排空状态（draining）
// 2. 排空状态的节点不会被负载均衡器选中
// 3. 节点状态转换正确
// 4. 支持节点的优雅下线和上线
func TestProperty28_ZeroDowntimeUpgradeCapability(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("节点排空应该支持零停机升级", prop.ForAll(
		func(totalNodes int, drainingNodes int) bool {
			// 限制参数范围
			if totalNodes < 3 {
				totalNodes = 3
			}
			if totalNodes > 10 {
				totalNodes = 10
			}
			if drainingNodes < 1 {
				drainingNodes = 1
			}
			if drainingNodes >= totalNodes {
				drainingNodes = totalNodes - 1 // 至少保留一个健康节点
			}

			// 设置测试环境
			db := setupClusterTestDB(t)
			manager := NewClusterManager(db)

			ctx := context.Background()

			// 注册多个节点
			nodeNames := make([]string, totalNodes)
			for i := 0; i < totalNodes; i++ {
				nodeName := fmt.Sprintf("node-%d", i)
				nodeNames[i] = nodeName

				node := &Node{
					Name:              nodeName,
					Address:           fmt.Sprintf("192.168.1.%d", i+1),
					Port:              8080,
					Role:              NodeRoleWorker,
					Status:            NodeStatusHealthy,
					CPUCores:          4,
					MemoryGB:          16.0,
					DiskGB:            100.0,
					CPUUsage:          30.0,
					MemoryUsage:       40.0,
					ActiveConnections: 50,
					LastHeartbeat:     time.Now(),
				}

				if err := manager.RegisterNode(ctx, node); err != nil {
					t.Logf("注册节点失败: %v", err)
					return false
				}
			}

			// 将部分节点标记为排空状态（模拟升级）
			for i := 0; i < drainingNodes; i++ {
				if err := manager.DrainNode(ctx, nodeNames[i]); err != nil {
					t.Logf("排空节点失败: %v", err)
					return false
				}
			}

			// 验证：排空的节点状态应该正确
			for i := 0; i < drainingNodes; i++ {
				node, err := manager.GetNode(nodeNames[i])
				if err != nil {
					t.Logf("获取节点失败: %v", err)
					return false
				}

				if node.Status != NodeStatusDraining {
					t.Logf("节点 %s 应该处于排空状态，实际状态: %s", nodeNames[i], node.Status)
					return false
				}
			}

			// 验证：健康节点数量应该正确
			healthyNodes := manager.GetHealthyNodes()
			expectedHealthy := totalNodes - drainingNodes
			if len(healthyNodes) != expectedHealthy {
				t.Logf("期望 %d 个健康节点，实际 %d 个", expectedHealthy, len(healthyNodes))
				return false
			}

			// 验证：负载均衡不应该选择排空状态的节点
			for i := 0; i < 10; i++ {
				selectedNode, err := manager.SelectNode("round_robin")
				if err != nil {
					t.Logf("选择节点失败: %v", err)
					return false
				}

				if selectedNode.Status == NodeStatusDraining {
					t.Logf("负载均衡不应该选择排空状态的节点")
					return false
				}
			}

			// 模拟升级完成，恢复节点
			for i := 0; i < drainingNodes; i++ {
				if err := manager.UpdateNodeStatus(ctx, nodeNames[i], NodeStatusHealthy); err != nil {
					t.Logf("恢复节点失败: %v", err)
					return false
				}
			}

			// 验证：所有节点都应该恢复健康
			healthyNodes = manager.GetHealthyNodes()
			if len(healthyNodes) != totalNodes {
				t.Logf("升级完成后，期望 %d 个健康节点，实际 %d 个", totalNodes, len(healthyNodes))
				return false
			}

			return true
		},
		gen.IntRange(3, 10),  // 总节点数
		gen.IntRange(1, 5),   // 排空节点数
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty28_NodeMetricsUpdate 测试节点指标更新
// **Feature: enhanced-aiops-platform, Property 28: 零停机升级能力**
// **Validates: Requirements 10.5**
func TestProperty28_NodeMetricsUpdate(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("节点指标应该能够动态更新", prop.ForAll(
		func(cpuUsage float64, memoryUsage float64, connections int) bool {
			// 限制参数范围
			if cpuUsage < 0 {
				cpuUsage = 0
			}
			if cpuUsage > 100 {
				cpuUsage = 100
			}
			if memoryUsage < 0 {
				memoryUsage = 0
			}
			if memoryUsage > 100 {
				memoryUsage = 100
			}
			if connections < 0 {
				connections = 0
			}
			if connections > 10000 {
				connections = 10000
			}

			// 设置测试环境
			db := setupClusterTestDB(t)
			manager := NewClusterManager(db)

			ctx := context.Background()

			// 注册节点
			nodeName := "test-node"
			node := &Node{
				Name:              nodeName,
				Address:           "192.168.1.1",
				Port:              8080,
				Role:              NodeRoleWorker,
				Status:            NodeStatusHealthy,
				CPUCores:          4,
				MemoryGB:          16.0,
				DiskGB:            100.0,
				CPUUsage:          0,
				MemoryUsage:       0,
				ActiveConnections: 0,
				LastHeartbeat:     time.Now(),
			}

			if err := manager.RegisterNode(ctx, node); err != nil {
				t.Logf("注册节点失败: %v", err)
				return false
			}

			// 更新节点指标
			metrics := &NodeMetrics{
				CPUUsage:          cpuUsage,
				MemoryUsage:       memoryUsage,
				DiskUsage:         50.0,
				ActiveConnections: connections,
				RequestsPerSecond: 100.0,
			}

			if err := manager.UpdateNodeMetrics(ctx, nodeName, metrics); err != nil {
				t.Logf("更新节点指标失败: %v", err)
				return false
			}

			// 验证：指标应该被正确更新
			updatedNode, err := manager.GetNode(nodeName)
			if err != nil {
				t.Logf("获取节点失败: %v", err)
				return false
			}

			if updatedNode.CPUUsage != cpuUsage {
				t.Logf("CPU 使用率不匹配: 期望 %.2f, 实际 %.2f", cpuUsage, updatedNode.CPUUsage)
				return false
			}

			if updatedNode.MemoryUsage != memoryUsage {
				t.Logf("内存使用率不匹配: 期望 %.2f, 实际 %.2f", memoryUsage, updatedNode.MemoryUsage)
				return false
			}

			if updatedNode.ActiveConnections != connections {
				t.Logf("活跃连接数不匹配: 期望 %d, 实际 %d", connections, updatedNode.ActiveConnections)
				return false
			}

			// 验证：LastHeartbeat 应该被更新
			if updatedNode.LastHeartbeat.Before(node.LastHeartbeat) {
				t.Logf("LastHeartbeat 应该被更新")
				return false
			}

			return true
		},
		gen.Float64Range(0, 100),  // CPU 使用率
		gen.Float64Range(0, 100),  // 内存使用率
		gen.IntRange(0, 10000),    // 活跃连接数
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// setupClusterTestDB 为集群测试设置数据库
func setupClusterTestDB(t *testing.T) *gorm.DB {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("打开 SQL 数据库失败: %v", err)
	}

	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 GORM 数据库失败: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&Node{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}
