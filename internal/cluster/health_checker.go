package cluster

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	manager *ClusterManager
	ticker  *time.Ticker
	client  *http.Client
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(manager *ClusterManager) *HealthChecker {
	return &HealthChecker{
		manager: manager,
		ticker:  time.NewTicker(30 * time.Second),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Start 启动健康检查
func (hc *HealthChecker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			hc.ticker.Stop()
			return
		case <-hc.ticker.C:
			hc.checkAllNodes(ctx)
		}
	}
}

// checkAllNodes 检查所有节点
func (hc *HealthChecker) checkAllNodes(ctx context.Context) {
	nodes := hc.manager.ListNodes()
	
	for _, node := range nodes {
		// 跳过已下线的节点
		if node.Status == NodeStatusOffline {
			continue
		}
		
		// 执行健康检查
		healthy := hc.checkNode(node)
		
		// 更新状态
		var newStatus NodeStatus
		if healthy {
			newStatus = NodeStatusHealthy
		} else {
			newStatus = NodeStatusUnhealthy
		}
		
		if node.Status != newStatus {
			hc.manager.UpdateNodeStatus(ctx, node.Name, newStatus)
		}
	}
}

// checkNode 检查单个节点
func (hc *HealthChecker) checkNode(node *Node) bool {
	// 检查心跳超时
	if time.Since(node.LastHeartbeat) > 2*time.Minute {
		return false
	}
	
	// HTTP 健康检查
	url := fmt.Sprintf("http://%s:%d/health", node.Address, node.Port)
	resp, err := hc.client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}
