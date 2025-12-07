package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"qwq/internal/registry"
)

// EnhancedGatewayServer 增强版API网关服务器（集成服务注册与发现）
type EnhancedGatewayServer struct {
	gateway         *Gateway
	server          *http.Server
	registry        *registry.ServiceRegistry
	discoveryClient *registry.ServiceDiscoveryClient
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewEnhancedGatewayServer 创建增强版网关服务器
func NewEnhancedGatewayServer(port string) *EnhancedGatewayServer {
	// 创建服务注册中心
	serviceRegistry := registry.NewServiceRegistry()
	
	// 创建服务发现客户端
	discoveryClient := registry.NewServiceDiscoveryClient(serviceRegistry)
	
	// 创建网关
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

	return &EnhancedGatewayServer{
		gateway:         gateway,
		server:          server,
		registry:        serviceRegistry,
		discoveryClient: discoveryClient,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start 启动增强版网关服务器
func (egs *EnhancedGatewayServer) Start() error {
	// 注册默认服务
	egs.registerDefaultServices()
	
	// 启动健康检查器
	go egs.gateway.StartHealthChecker(egs.ctx)
	
	// 启动服务注册中心API服务器
	go egs.startRegistryAPI()
	
	// 设置动态路由更新
	egs.setupDynamicRouting()

	// API Gateway starting on port: egs.server.Addr
	
	if err := egs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("增强版网关服务器启动失败: %v", err)
	}
	
	return nil
}

// Stop 停止增强版网关服务器
func (egs *EnhancedGatewayServer) Stop() error {
	egs.cancel()
	egs.registry.Stop()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return egs.server.Shutdown(ctx)
}

// registerDefaultServices 注册默认服务到服务注册中心
func (egs *EnhancedGatewayServer) registerDefaultServices() {
	defaultServices := []struct {
		name    string
		address string
		port    int
		health  string
		version string
		tags    []string
	}{
		{"web-ui", "localhost", 8899, "/health", "1.0", []string{"ui", "frontend"}},
		{"ai-agent", "localhost", 8900, "/health", "1.0", []string{"ai", "backend"}},
		{"app-store", "localhost", 8901, "/health", "1.0", []string{"apps", "backend"}},
		{"container-service", "localhost", 8902, "/health", "1.0", []string{"containers", "backend"}},
		{"website-manager", "localhost", 8903, "/health", "1.0", []string{"websites", "backend"}},
		{"database-manager", "localhost", 8904, "/health", "1.0", []string{"database", "backend"}},
		{"backup-service", "localhost", 8905, "/health", "1.0", []string{"backup", "backend"}},
		{"monitoring", "localhost", 8906, "/health", "1.0", []string{"monitoring", "backend"}},
	}
	
	for _, service := range defaultServices {
		req := &registry.RegistrationRequest{
			Name:     service.name,
			Address:  service.address,
			Port:     service.port,
			Health:   fmt.Sprintf("http://%s:%d%s", service.address, service.port, service.health),
			Version:  service.version,
			Tags:     service.tags,
			Weight:   100,
			MaxFails: 3,
		}
		
		_, err := egs.registry.Register(req)
		if err != nil {
			// Service registration failed: service.name - err
			continue
		}
		
		// 同时注册到网关的路由表
		egs.registerServiceRoutes(service.name)
	}
}

// registerServiceRoutes 为服务注册路由
func (egs *EnhancedGatewayServer) registerServiceRoutes(serviceName string) {
	switch serviceName {
	case "ai-agent":
		egs.gateway.AddRoute("/api/v1/ai/", serviceName, []string{"GET", "POST"})
	case "app-store":
		egs.gateway.AddRoute("/api/v1/apps/", serviceName, []string{"GET", "POST", "PUT", "DELETE"})
	case "container-service":
		egs.gateway.AddRoute("/api/v1/containers/", serviceName, []string{"GET", "POST", "PUT", "DELETE"})
	case "website-manager":
		egs.gateway.AddRoute("/api/v1/websites/", serviceName, []string{"GET", "POST", "PUT", "DELETE"})
	case "database-manager":
		egs.gateway.AddRoute("/api/v1/databases/", serviceName, []string{"GET", "POST", "PUT", "DELETE"})
	case "backup-service":
		egs.gateway.AddRoute("/api/v1/backups/", serviceName, []string{"GET", "POST", "PUT", "DELETE"})
	case "monitoring":
		egs.gateway.AddRoute("/api/v1/monitoring/", serviceName, []string{"GET", "POST"})
	case "web-ui":
		// 兼容现有API路径
		egs.gateway.AddRoute("/api/", serviceName, []string{"GET", "POST", "PUT", "DELETE"})
		egs.gateway.AddRoute("/ws/", serviceName, []string{"GET"})
		egs.gateway.AddRoute("/assets/", serviceName, []string{"GET"})
		egs.gateway.AddRoute("/", serviceName, []string{"GET"})
	}
}

// startRegistryAPI 启动服务注册中心API
func (egs *EnhancedGatewayServer) startRegistryAPI() {
	// 从网关端口号推导注册中心端口号
	gatewayPort := strings.TrimPrefix(egs.server.Addr, ":")
	port, err := strconv.Atoi(gatewayPort)
	if err != nil {
		port = 8080
	}
	registryPort := port + 1000 // 注册中心端口 = 网关端口 + 1000
	
	err = registry.StartRegistryServer(registryPort, egs.registry, egs.discoveryClient)
	if err != nil {
		// Registry API server failed to start: err
	}
}

// setupDynamicRouting 设置动态路由更新
func (egs *EnhancedGatewayServer) setupDynamicRouting() {
	// 监听所有服务的变化
	healthyServices := egs.registry.GetHealthyServices()
	for serviceName := range healthyServices {
		egs.discoveryClient.WatchService(serviceName, egs.onServiceChange)
	}
}

// onServiceChange 服务变化回调
func (egs *EnhancedGatewayServer) onServiceChange(serviceName string, instances []*registry.ServiceInstance) {
	// 更新网关中的服务信息
	egs.updateGatewayServices(serviceName, instances)
}

// updateGatewayServices 更新网关中的服务信息
func (egs *EnhancedGatewayServer) updateGatewayServices(serviceName string, instances []*registry.ServiceInstance) {
	// 清除旧的服务信息
	egs.gateway.UnregisterService(serviceName)
	
	// 注册新的服务实例
	for _, instance := range instances {
		if instance.Status == registry.StatusHealthy {
			serviceURL := fmt.Sprintf("http://%s:%d", instance.Address, instance.Port)
			healthURL := instance.Health
			
			err := egs.gateway.RegisterService(instance.ID, serviceURL, healthURL, instance.Version)
			if err != nil {
				// Failed to register service instance: instance.ID - err
				continue
			}
		}
	}
}

// RegisterService 注册新服务
func (egs *EnhancedGatewayServer) RegisterService(req *registry.RegistrationRequest) (*registry.ServiceInstance, error) {
	// 注册到服务注册中心
	instance, err := egs.registry.Register(req)
	if err != nil {
		return nil, err
	}
	
	// 注册路由
	egs.registerServiceRoutes(req.Name)
	
	return instance, nil
}

// DeregisterService 注销服务
func (egs *EnhancedGatewayServer) DeregisterService(serviceID string) error {
	// 从服务注册中心注销
	err := egs.registry.Deregister(serviceID)
	if err != nil {
		return err
	}
	
	// 从网关注销
	return egs.gateway.UnregisterService(serviceID)
}

// DiscoverService 发现服务
func (egs *EnhancedGatewayServer) DiscoverService(serviceName string) ([]*registry.ServiceInstance, error) {
	return egs.discoveryClient.GetInstances(serviceName)
}

// SelectServiceInstance 选择服务实例
func (egs *EnhancedGatewayServer) SelectServiceInstance(serviceName, key string) (*registry.ServiceInstance, error) {
	return egs.discoveryClient.SelectInstance(serviceName, key)
}

// SetLoadBalancer 设置负载均衡器
func (egs *EnhancedGatewayServer) SetLoadBalancer(balancerName string) error {
	return egs.discoveryClient.SetLoadBalancer(balancerName)
}

// GetServiceStats 获取服务统计信息
func (egs *EnhancedGatewayServer) GetServiceStats() map[string]interface{} {
	return egs.registry.GetServiceStats()
}

// GetRegistry 获取服务注册中心
func (egs *EnhancedGatewayServer) GetRegistry() *registry.ServiceRegistry {
	return egs.registry
}

// GetDiscoveryClient 获取服务发现客户端
func (egs *EnhancedGatewayServer) GetDiscoveryClient() *registry.ServiceDiscoveryClient {
	return egs.discoveryClient
}

// GetGateway 获取网关实例
func (egs *EnhancedGatewayServer) GetGateway() *Gateway {
	return egs.gateway
}