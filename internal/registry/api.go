package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// RegistryAPI 服务注册API
type RegistryAPI struct {
	registry *ServiceRegistry
	client   *ServiceDiscoveryClient
}

// NewRegistryAPI 创建注册API
func NewRegistryAPI(registry *ServiceRegistry, client *ServiceDiscoveryClient) *RegistryAPI {
	return &RegistryAPI{
		registry: registry,
		client:   client,
	}
}

// RegisterRoutes 注册路由
func (api *RegistryAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/registry/services", api.handleServices)
	mux.HandleFunc("/api/v1/registry/services/", api.handleServiceDetail)
	mux.HandleFunc("/api/v1/registry/discover/", api.handleDiscover)
	mux.HandleFunc("/api/v1/registry/health", api.handleHealth)
	mux.HandleFunc("/api/v1/registry/stats", api.handleStats)
}

// handleServices 处理服务列表请求
func (api *RegistryAPI) handleServices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.listServices(w, r)
	case http.MethodPost:
		api.registerService(w, r)
	default:
		api.writeError(w, "方法不支持", http.StatusMethodNotAllowed)
	}
}

// handleServiceDetail 处理单个服务请求
func (api *RegistryAPI) handleServiceDetail(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取服务ID
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/registry/services/")
	serviceID := strings.Split(path, "/")[0]
	
	if serviceID == "" {
		api.writeError(w, "服务ID不能为空", http.StatusBadRequest)
		return
	}
	
	switch r.Method {
	case http.MethodGet:
		api.getService(w, r, serviceID)
	case http.MethodDelete:
		api.deregisterService(w, r, serviceID)
	case http.MethodPut:
		api.updateService(w, r, serviceID)
	default:
		api.writeError(w, "方法不支持", http.StatusMethodNotAllowed)
	}
}

// handleDiscover 处理服务发现请求
func (api *RegistryAPI) handleDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, "方法不支持", http.StatusMethodNotAllowed)
		return
	}
	
	// 从URL路径中提取服务名称
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/registry/discover/")
	serviceName := strings.Split(path, "/")[0]
	
	if serviceName == "" {
		api.writeError(w, "服务名称不能为空", http.StatusBadRequest)
		return
	}
	
	api.discoverService(w, r, serviceName)
}

// handleHealth 处理健康检查请求
func (api *RegistryAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, "方法不支持", http.StatusMethodNotAllowed)
		return
	}
	
	stats := api.registry.GetServiceStats()
	
	response := map[string]interface{}{
		"status": "healthy",
		"stats":  stats,
	}
	
	api.writeSuccess(w, response)
}

// handleStats 处理统计信息请求
func (api *RegistryAPI) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, "方法不支持", http.StatusMethodNotAllowed)
		return
	}
	
	stats := api.registry.GetServiceStats()
	api.writeSuccess(w, stats)
}

// listServices 列出所有服务
func (api *RegistryAPI) listServices(w http.ResponseWriter, r *http.Request) {
	services := api.registry.ListServices()
	
	// 按服务名称分组
	servicesByName := make(map[string][]*ServiceInstance)
	for _, instance := range services {
		servicesByName[instance.Name] = append(servicesByName[instance.Name], instance)
	}
	
	api.writeSuccess(w, servicesByName)
}

// registerService 注册服务
func (api *RegistryAPI) registerService(w http.ResponseWriter, r *http.Request) {
	var req RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, "请求格式错误: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// 验证必填字段
	if req.Name == "" {
		api.writeError(w, "服务名称不能为空", http.StatusBadRequest)
		return
	}
	
	if req.Address == "" {
		api.writeError(w, "服务地址不能为空", http.StatusBadRequest)
		return
	}
	
	if req.Port <= 0 {
		api.writeError(w, "服务端口必须大于0", http.StatusBadRequest)
		return
	}
	
	instance, err := api.registry.Register(&req)
	if err != nil {
		api.writeError(w, "注册服务失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	api.writeSuccess(w, instance)
}

// getService 获取服务详情
func (api *RegistryAPI) getService(w http.ResponseWriter, r *http.Request, serviceID string) {
	instance, err := api.registry.GetService(serviceID)
	if err != nil {
		api.writeError(w, err.Error(), http.StatusNotFound)
		return
	}
	
	api.writeSuccess(w, instance)
}

// deregisterService 注销服务
func (api *RegistryAPI) deregisterService(w http.ResponseWriter, r *http.Request, serviceID string) {
	err := api.registry.Deregister(serviceID)
	if err != nil {
		api.writeError(w, err.Error(), http.StatusNotFound)
		return
	}
	
	api.writeSuccess(w, map[string]string{"message": "服务注销成功"})
}

// updateService 更新服务状态
func (api *RegistryAPI) updateService(w http.ResponseWriter, r *http.Request, serviceID string) {
	var updateReq struct {
		Status   string            `json:"status"`
		Metadata map[string]string `json:"metadata"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		api.writeError(w, "请求格式错误: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// 更新服务状态
	if updateReq.Status != "" {
		status := ServiceStatus(updateReq.Status)
		if status != StatusHealthy && status != StatusUnhealthy && status != StatusDraining {
			api.writeError(w, "无效的服务状态", http.StatusBadRequest)
			return
		}
		
		err := api.registry.UpdateServiceStatus(serviceID, status)
		if err != nil {
			api.writeError(w, err.Error(), http.StatusNotFound)
			return
		}
	}
	
	// 获取更新后的服务信息
	instance, err := api.registry.GetService(serviceID)
	if err != nil {
		api.writeError(w, err.Error(), http.StatusNotFound)
		return
	}
	
	api.writeSuccess(w, instance)
}

// discoverService 发现服务
func (api *RegistryAPI) discoverService(w http.ResponseWriter, r *http.Request, serviceName string) {
	// 解析查询参数
	tags := r.URL.Query()["tags"]
	key := r.URL.Query().Get("key")
	selectOne := r.URL.Query().Get("select") == "one"
	balancer := r.URL.Query().Get("balancer")
	
	var instances []*ServiceInstance
	var err error
	
	// 根据是否有标签选择不同的发现方法
	if len(tags) > 0 {
		instances, err = api.client.GetInstancesWithTags(serviceName, tags)
	} else {
		instances, err = api.client.GetInstances(serviceName)
	}
	
	if err != nil {
		api.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 如果需要选择单个实例
	if selectOne {
		if len(instances) == 0 {
			api.writeError(w, "没有可用的服务实例", http.StatusNotFound)
			return
		}
		
		// 设置负载均衡器
		if balancer != "" {
			if err := api.client.SetLoadBalancer(balancer); err != nil {
				api.writeError(w, "无效的负载均衡器: "+balancer, http.StatusBadRequest)
				return
			}
		}
		
		var selectedInstance *ServiceInstance
		if len(tags) > 0 {
			selectedInstance, err = api.client.SelectInstanceWithTags(serviceName, tags, key)
		} else {
			selectedInstance, err = api.client.SelectInstance(serviceName, key)
		}
		
		if err != nil {
			api.writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		api.writeSuccess(w, selectedInstance)
		return
	}
	
	// 返回所有实例
	response := map[string]interface{}{
		"service_name": serviceName,
		"instances":    instances,
		"count":        len(instances),
		"tags":         tags,
	}
	
	api.writeSuccess(w, response)
}

// heartbeat 处理心跳请求
func (api *RegistryAPI) heartbeat(w http.ResponseWriter, r *http.Request, serviceID string) {
	if r.Method != http.MethodPost {
		api.writeError(w, "方法不支持", http.StatusMethodNotAllowed)
		return
	}
	
	err := api.registry.Heartbeat(serviceID)
	if err != nil {
		api.writeError(w, err.Error(), http.StatusNotFound)
		return
	}
	
	api.writeSuccess(w, map[string]string{"message": "心跳成功"})
}

// writeSuccess 写入成功响应
func (api *RegistryAPI) writeSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]interface{}{
		"success": true,
		"data":    data,
		"code":    http.StatusOK,
	}
	
	json.NewEncoder(w).Encode(response)
}

// writeError 写入错误响应
func (api *RegistryAPI) writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	response := map[string]interface{}{
		"success": false,
		"error":   message,
		"code":    code,
	}
	
	json.NewEncoder(w).Encode(response)
}

// StartRegistryServer 启动注册中心服务器
func StartRegistryServer(port int, registry *ServiceRegistry, client *ServiceDiscoveryClient) error {
	api := NewRegistryAPI(registry, client)
	
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)
	
	// 添加CORS支持
	handler := corsMiddleware(mux)
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}
	
	return server.ListenAndServe()
}

// corsMiddleware CORS中间件
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}