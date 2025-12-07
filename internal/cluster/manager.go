package cluster

import (
	"context"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
)

var (
	// ErrNodeNotFound 节点不存在
	ErrNodeNotFound = errors.New("节点不存在")
	
	// ErrClusterNotReady 集群未就绪
	ErrClusterNotReady = errors.New("集群未就绪")
)

// ClusterManager 集群管理器
type ClusterManager struct {
	db          *gorm.DB
	nodes       map[string]*Node
	mu          sync.RWMutex
	healthCheck *HealthChecker
}

// NodeStatus 节点状态
type NodeStatus string

const (
	NodeStatusHealthy   NodeStatus = "healthy"
	NodeStatusUnhealthy NodeStatus = "unhealthy"
	NodeStatusDraining  NodeStatus = "draining"
	NodeStatusOffline   NodeStatus = "offline"
)

// NodeRole 节点角色
type NodeRole string

const (
	NodeRoleMaster NodeRole = "master"
	NodeRoleWorker NodeRole = "worker"
)

// Node 集群节点
type Node struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"uniqueIndex;not null"`
	Address     string     `json:"address" gorm:"not null"`
	Port        int        `json:"port" gorm:"not null"`
	Role        NodeRole   `json:"role" gorm:"not null"`
	Status      NodeStatus `json:"status" gorm:"index"`
	Version     string     `json:"version"`
	
	// 资源信息
	CPUCores    int     `json:"cpu_cores"`
	MemoryGB    float64 `json:"memory_gb"`
	DiskGB      float64 `json:"disk_gb"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	
	// 负载信息
	ActiveConnections int     `json:"active_connections"`
	RequestsPerSecond float64 `json:"requests_per_second"`
	
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NewClusterManager 创建集群管理器
func NewClusterManager(db *gorm.DB) *ClusterManager {
	cm := &ClusterManager{
		db:    db,
		nodes: make(map[string]*Node),
	}
	
	cm.healthCheck = NewHealthChecker(cm)
	
	// 加载现有节点
	cm.loadNodes()
	
	// 启动健康检查
	go cm.healthCheck.Start(context.Background())
	
	return cm
}

// loadNodes 加载节点
func (cm *ClusterManager) loadNodes() {
	var nodes []*Node
	cm.db.Find(&nodes)
	
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	for _, node := range nodes {
		cm.nodes[node.Name] = node
	}
}

// RegisterNode 注册节点
func (cm *ClusterManager) RegisterNode(ctx context.Context, node *Node) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// 保存到数据库
	if err := cm.db.WithContext(ctx).Create(node).Error; err != nil {
		return err
	}
	
	// 添加到内存
	cm.nodes[node.Name] = node
	
	return nil
}

// UnregisterNode 注销节点
func (cm *ClusterManager) UnregisterNode(ctx context.Context, nodeName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	node, exists := cm.nodes[nodeName]
	if !exists {
		return ErrNodeNotFound
	}
	
	// 从数据库删除
	if err := cm.db.WithContext(ctx).Delete(node).Error; err != nil {
		return err
	}
	
	// 从内存删除
	delete(cm.nodes, nodeName)
	
	return nil
}

// GetNode 获取节点
func (cm *ClusterManager) GetNode(nodeName string) (*Node, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	node, exists := cm.nodes[nodeName]
	if !exists {
		return nil, ErrNodeNotFound
	}
	
	return node, nil
}

// ListNodes 列出所有节点
func (cm *ClusterManager) ListNodes() []*Node {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	nodes := make([]*Node, 0, len(cm.nodes))
	for _, node := range cm.nodes {
		nodes = append(nodes, node)
	}
	
	return nodes
}

// GetHealthyNodes 获取健康节点
func (cm *ClusterManager) GetHealthyNodes() []*Node {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	var healthy []*Node
	for _, node := range cm.nodes {
		if node.Status == NodeStatusHealthy {
			healthy = append(healthy, node)
		}
	}
	
	return healthy
}

// UpdateNodeStatus 更新节点状态
func (cm *ClusterManager) UpdateNodeStatus(ctx context.Context, nodeName string, status NodeStatus) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	node, exists := cm.nodes[nodeName]
	if !exists {
		return ErrNodeNotFound
	}
	
	node.Status = status
	node.LastHeartbeat = time.Now()
	
	// 更新数据库
	return cm.db.WithContext(ctx).Model(node).Updates(map[string]interface{}{
		"status":         status,
		"last_heartbeat": node.LastHeartbeat,
	}).Error
}

// UpdateNodeMetrics 更新节点指标
func (cm *ClusterManager) UpdateNodeMetrics(ctx context.Context, nodeName string, metrics *NodeMetrics) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	node, exists := cm.nodes[nodeName]
	if !exists {
		return ErrNodeNotFound
	}
	
	node.CPUUsage = metrics.CPUUsage
	node.MemoryUsage = metrics.MemoryUsage
	node.DiskUsage = metrics.DiskUsage
	node.ActiveConnections = metrics.ActiveConnections
	node.RequestsPerSecond = metrics.RequestsPerSecond
	node.LastHeartbeat = time.Now()
	
	// 更新数据库
	return cm.db.WithContext(ctx).Model(node).Updates(map[string]interface{}{
		"cpu_usage":          metrics.CPUUsage,
		"memory_usage":       metrics.MemoryUsage,
		"disk_usage":         metrics.DiskUsage,
		"active_connections": metrics.ActiveConnections,
		"requests_per_second": metrics.RequestsPerSecond,
		"last_heartbeat":     node.LastHeartbeat,
	}).Error
}

// SelectNode 选择节点（负载均衡）
func (cm *ClusterManager) SelectNode(strategy string) (*Node, error) {
	healthyNodes := cm.GetHealthyNodes()
	if len(healthyNodes) == 0 {
		return nil, ErrClusterNotReady
	}
	
	switch strategy {
	case "least_connections":
		return cm.selectLeastConnections(healthyNodes), nil
	case "least_cpu":
		return cm.selectLeastCPU(healthyNodes), nil
	case "round_robin":
		fallthrough
	default:
		return healthyNodes[0], nil // 简化实现
	}
}

// selectLeastConnections 选择连接数最少的节点
func (cm *ClusterManager) selectLeastConnections(nodes []*Node) *Node {
	if len(nodes) == 0 {
		return nil
	}
	
	selected := nodes[0]
	for _, node := range nodes[1:] {
		if node.ActiveConnections < selected.ActiveConnections {
			selected = node
		}
	}
	
	return selected
}

// selectLeastCPU 选择 CPU 使用率最低的节点
func (cm *ClusterManager) selectLeastCPU(nodes []*Node) *Node {
	if len(nodes) == 0 {
		return nil
	}
	
	selected := nodes[0]
	for _, node := range nodes[1:] {
		if node.CPUUsage < selected.CPUUsage {
			selected = node
		}
	}
	
	return selected
}

// DrainNode 排空节点（准备下线）
func (cm *ClusterManager) DrainNode(ctx context.Context, nodeName string) error {
	return cm.UpdateNodeStatus(ctx, nodeName, NodeStatusDraining)
}

// GetClusterStats 获取集群统计信息
func (cm *ClusterManager) GetClusterStats() *ClusterStats {
	nodes := cm.ListNodes()
	
	stats := &ClusterStats{
		TotalNodes:   len(nodes),
		HealthyNodes: 0,
		TotalCPU:     0,
		TotalMemory:  0,
		TotalDisk:    0,
		UsedCPU:      0,
		UsedMemory:   0,
		UsedDisk:     0,
	}
	
	for _, node := range nodes {
		if node.Status == NodeStatusHealthy {
			stats.HealthyNodes++
		}
		
		stats.TotalCPU += float64(node.CPUCores)
		stats.TotalMemory += node.MemoryGB
		stats.TotalDisk += node.DiskGB
		stats.UsedCPU += node.CPUUsage * float64(node.CPUCores) / 100
		stats.UsedMemory += node.MemoryUsage * node.MemoryGB / 100
		stats.UsedDisk += node.DiskUsage * node.DiskGB / 100
	}
	
	return stats
}

// NodeMetrics 节点指标
type NodeMetrics struct {
	CPUUsage          float64
	MemoryUsage       float64
	DiskUsage         float64
	ActiveConnections int
	RequestsPerSecond float64
}

// ClusterStats 集群统计信息
type ClusterStats struct {
	TotalNodes   int     `json:"total_nodes"`
	HealthyNodes int     `json:"healthy_nodes"`
	TotalCPU     float64 `json:"total_cpu"`
	TotalMemory  float64 `json:"total_memory"`
	TotalDisk    float64 `json:"total_disk"`
	UsedCPU      float64 `json:"used_cpu"`
	UsedMemory   float64 `json:"used_memory"`
	UsedDisk     float64 `json:"used_disk"`
}

// TableName 指定表名
func (Node) TableName() string {
	return "cluster_nodes"
}
