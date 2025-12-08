package gateway

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

// GatewayServer API网关服务器
type GatewayServer struct {
	gateway *Gateway
	server  *http.Server
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewGatewayServer 创建新的网关服务器
func NewGatewayServer(port string) *GatewayServer {
	gateway := NewGateway()
	
	// 添加中间件
	gateway.AddMiddleware(LoggingMiddleware())
	gateway.AddMiddleware(AuthMiddleware())
	gateway.AddMiddleware(RateLimitMiddleware(100)) // 每分钟100个请求
	gateway.AddMiddleware(HealthCheckMiddleware())

	ctx, cancel := context.WithCancel(context.Background())

	server := &http.Server{
		Addr:         port,
		Handler:      gateway,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &GatewayServer{
		gateway: gateway,
		server:  server,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start 启动网关服务器
func (gs *GatewayServer) Start() error {
	// 注册默认服务和路由
	gs.registerDefaultServices()
	
	// 启动健康检查器
	go gs.gateway.StartHealthChecker(gs.ctx)

	// API Gateway starting on port: gs.server.Addr
	
	if err := gs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("网关服务器启动失败: %v", err)
	}
	
	return nil
}

// Stop 停止网关服务器
func (gs *GatewayServer) Stop() error {
	gs.cancel()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return gs.server.Shutdown(ctx)
}

// GetGateway 获取网关实例
func (gs *GatewayServer) GetGateway() *Gateway {
	return gs.gateway
}

// registerDefaultServices 注册默认服务
func (gs *GatewayServer) registerDefaultServices() {
	// 从环境变量读取主服务端口，默认 8080
	// Docker 环境中通过 PORT 环境变量统一配置
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	baseURL := fmt.Sprintf("http://localhost:%s", port)
	
	// 注册 Web UI 服务（主服务）
	gs.gateway.RegisterService("web-ui", baseURL, baseURL+"/health", "1.0")
	
	// 注册微服务（预留端口，实际部署时可能不启用）
	gs.gateway.RegisterService("ai-agent", "http://localhost:8900", "http://localhost:8900/health", "1.0")
	gs.gateway.RegisterService("app-store", "http://localhost:8901", "http://localhost:8901/health", "1.0")
	gs.gateway.RegisterService("container-service", "http://localhost:8902", "http://localhost:8902/health", "1.0")
	gs.gateway.RegisterService("website-manager", "http://localhost:8903", "http://localhost:8903/health", "1.0")
	gs.gateway.RegisterService("database-manager", "http://localhost:8904", "http://localhost:8904/health", "1.0")
	gs.gateway.RegisterService("backup-service", "http://localhost:8905", "http://localhost:8905/health", "1.0")
	gs.gateway.RegisterService("monitoring", "http://localhost:8906", "http://localhost:8906/health", "1.0")

	// 配置 API 路由规则（路径前缀 -> 服务映射）
	gs.gateway.AddRoute("/api/v1/ai/", "ai-agent", []string{"GET", "POST"})
	gs.gateway.AddRoute("/api/v1/apps/", "app-store", []string{"GET", "POST", "PUT", "DELETE"})
	gs.gateway.AddRoute("/api/v1/containers/", "container-service", []string{"GET", "POST", "PUT", "DELETE"})
	gs.gateway.AddRoute("/api/v1/websites/", "website-manager", []string{"GET", "POST", "PUT", "DELETE"})
	gs.gateway.AddRoute("/api/v1/databases/", "database-manager", []string{"GET", "POST", "PUT", "DELETE"})
	gs.gateway.AddRoute("/api/v1/backups/", "backup-service", []string{"GET", "POST", "PUT", "DELETE"})
	gs.gateway.AddRoute("/api/v1/monitoring/", "monitoring", []string{"GET", "POST"})
	
	// 兼容现有 API 路径（转发到主服务）
	gs.gateway.AddRoute("/api/", "web-ui", []string{"GET", "POST", "PUT", "DELETE"})
	gs.gateway.AddRoute("/ws/", "web-ui", []string{"GET"})        // WebSocket 连接
	gs.gateway.AddRoute("/assets/", "web-ui", []string{"GET"})    // 静态资源
	gs.gateway.AddRoute("/", "web-ui", []string{"GET"})           // 首页
}

// RegisterService 注册新服务的便捷方法
func (gs *GatewayServer) RegisterService(name, url, health, version string) error {
	return gs.gateway.RegisterService(name, url, health, version)
}

// AddRoute 添加新路由的便捷方法
func (gs *GatewayServer) AddRoute(path, serviceName string, methods []string) {
	gs.gateway.AddRoute(path, serviceName, methods)
}

// GetServiceStatus 获取服务状态
func (gs *GatewayServer) GetServiceStatus() map[string]*ServiceInfo {
	return gs.gateway.ListServices()
}