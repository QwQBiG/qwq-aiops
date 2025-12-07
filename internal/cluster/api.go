package cluster

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// APIHandler 集群 API 处理器
type APIHandler struct {
	manager *ClusterManager
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(manager *ClusterManager) *APIHandler {
	return &APIHandler{
		manager: manager,
	}
}

// RegisterRoutes 注册路由
func (h *APIHandler) RegisterRoutes(router *mux.Router) {
	// 节点管理路由
	router.HandleFunc("/api/v1/cluster/nodes", h.ListNodes).Methods("GET")
	router.HandleFunc("/api/v1/cluster/nodes", h.RegisterNode).Methods("POST")
	router.HandleFunc("/api/v1/cluster/nodes/{name}", h.GetNode).Methods("GET")
	router.HandleFunc("/api/v1/cluster/nodes/{name}", h.UnregisterNode).Methods("DELETE")
	router.HandleFunc("/api/v1/cluster/nodes/{name}/drain", h.DrainNode).Methods("POST")
	router.HandleFunc("/api/v1/cluster/nodes/{name}/metrics", h.UpdateNodeMetrics).Methods("POST")
	
	// 集群统计路由
	router.HandleFunc("/api/v1/cluster/stats", h.GetClusterStats).Methods("GET")
	
	// 健康检查
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
}

// HealthCheck 健康检查
func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// ListNodes 列出节点
func (h *APIHandler) ListNodes(w http.ResponseWriter, r *http.Request) {
	nodes := h.manager.ListNodes()
	respondJSON(w, http.StatusOK, nodes)
}

// RegisterNode 注册节点
func (h *APIHandler) RegisterNode(w http.ResponseWriter, r *http.Request) {
	var node Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if err := h.manager.RegisterNode(r.Context(), &node); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusCreated, node)
}

// GetNode 获取节点
func (h *APIHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	node, err := h.manager.GetNode(name)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, node)
}

// UnregisterNode 注销节点
func (h *APIHandler) UnregisterNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	if err := h.manager.UnregisterNode(r.Context(), name); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]string{"message": "Node unregistered successfully"})
}

// DrainNode 排空节点
func (h *APIHandler) DrainNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	if err := h.manager.DrainNode(r.Context(), name); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]string{"message": "Node draining started"})
}

// UpdateNodeMetrics 更新节点指标
func (h *APIHandler) UpdateNodeMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	var metrics NodeMetrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if err := h.manager.UpdateNodeMetrics(r.Context(), name, &metrics); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]string{"message": "Metrics updated successfully"})
}

// GetClusterStats 获取集群统计信息
func (h *APIHandler) GetClusterStats(w http.ResponseWriter, r *http.Request) {
	stats := h.manager.GetClusterStats()
	respondJSON(w, http.StatusOK, stats)
}

// 辅助函数

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func getIDFromPath(r *http.Request) uint {
	vars := mux.Vars(r)
	id, _ := strconv.ParseUint(vars["id"], 10, 32)
	return uint(id)
}
