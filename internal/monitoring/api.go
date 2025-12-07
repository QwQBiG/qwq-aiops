package monitoring

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// APIHandler 监控 API 处理器
type APIHandler struct {
	service MonitoringService
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(service MonitoringService) *APIHandler {
	return &APIHandler{
		service: service,
	}
}

// RegisterRoutes 注册路由
func (h *APIHandler) RegisterRoutes(router *mux.Router) {
	// 指标路由
	router.HandleFunc("/api/v1/monitoring/metrics", h.RecordMetric).Methods("POST")
	router.HandleFunc("/api/v1/monitoring/metrics/query", h.QueryMetrics).Methods("POST")
	router.HandleFunc("/api/v1/monitoring/metrics", h.ListMetrics).Methods("GET")
	
	// 告警规则路由
	router.HandleFunc("/api/v1/monitoring/alert-rules", h.ListAlertRules).Methods("GET")
	router.HandleFunc("/api/v1/monitoring/alert-rules", h.CreateAlertRule).Methods("POST")
	router.HandleFunc("/api/v1/monitoring/alert-rules/{id}", h.GetAlertRule).Methods("GET")
	router.HandleFunc("/api/v1/monitoring/alert-rules/{id}", h.UpdateAlertRule).Methods("PUT")
	router.HandleFunc("/api/v1/monitoring/alert-rules/{id}", h.DeleteAlertRule).Methods("DELETE")
	
	// 告警路由
	router.HandleFunc("/api/v1/monitoring/alerts", h.ListAlerts).Methods("GET")
	router.HandleFunc("/api/v1/monitoring/alerts/{id}/acknowledge", h.AcknowledgeAlert).Methods("POST")
	router.HandleFunc("/api/v1/monitoring/alerts/{id}/resolve", h.ResolveAlert).Methods("POST")
	
	// AI 分析路由
	router.HandleFunc("/api/v1/monitoring/predict", h.PredictIssues).Methods("POST")
	router.HandleFunc("/api/v1/monitoring/capacity", h.AnalyzeCapacity).Methods("POST")
	
	// 健康检查
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
}

// HealthCheck 健康检查
func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// RecordMetric 记录指标
func (h *APIHandler) RecordMetric(w http.ResponseWriter, r *http.Request) {
	var metric Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}
	
	metric.UserID = getUserID(r)
	metric.TenantID = getTenantID(r)
	
	if err := h.service.RecordMetric(r.Context(), &metric); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusCreated, map[string]string{"message": "Metric recorded successfully"})
}

// QueryMetrics 查询指标
func (h *APIHandler) QueryMetrics(w http.ResponseWriter, r *http.Request) {
	var query MetricQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	data, err := h.service.QueryMetrics(r.Context(), &query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, data)
}

// ListMetrics 列出指标
func (h *APIHandler) ListMetrics(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	tenantID := getTenantID(r)
	
	metrics, err := h.service.ListMetrics(r.Context(), userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, metrics)
}

// ListAlertRules 列出告警规则
func (h *APIHandler) ListAlertRules(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	tenantID := getTenantID(r)
	
	rules, err := h.service.ListAlertRules(r.Context(), userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, rules)
}

// CreateAlertRule 创建告警规则
func (h *APIHandler) CreateAlertRule(w http.ResponseWriter, r *http.Request) {
	var rule AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	rule.UserID = getUserID(r)
	rule.TenantID = getTenantID(r)
	
	if err := h.service.CreateAlertRule(r.Context(), &rule); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusCreated, rule)
}

// GetAlertRule 获取告警规则
func (h *APIHandler) GetAlertRule(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	rule, err := h.service.GetAlertRule(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, rule)
}

// UpdateAlertRule 更新告警规则
func (h *APIHandler) UpdateAlertRule(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	var rule AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if err := h.service.UpdateAlertRule(r.Context(), id, &rule); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, rule)
}

// DeleteAlertRule 删除告警规则
func (h *APIHandler) DeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	if err := h.service.DeleteAlertRule(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]string{"message": "Alert rule deleted successfully"})
}

// ListAlerts 列出告警
func (h *APIHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	filters := &AlertFilters{
		UserID:   getUserID(r),
		TenantID: getTenantID(r),
	}
	
	// 解析查询参数
	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = AlertStatus(status)
	}
	if severity := r.URL.Query().Get("severity"); severity != "" {
		filters.Severity = AlertSeverity(severity)
	}
	if ruleID := r.URL.Query().Get("rule_id"); ruleID != "" {
		id, _ := strconv.ParseUint(ruleID, 10, 32)
		filters.RuleID = uint(id)
	}
	
	alerts, err := h.service.ListAlerts(r.Context(), filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, alerts)
}

// AcknowledgeAlert 确认告警
func (h *APIHandler) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	userID := getUserID(r)
	
	if err := h.service.AcknowledgeAlert(r.Context(), id, userID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]string{"message": "Alert acknowledged successfully"})
}

// ResolveAlert 解决告警
func (h *APIHandler) ResolveAlert(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	userID := getUserID(r)
	
	if err := h.service.ResolveAlert(r.Context(), id, userID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]string{"message": "Alert resolved successfully"})
}

// PredictIssues AI 预测问题
func (h *APIHandler) PredictIssues(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	result, err := h.service.PredictIssues(r.Context(), req.ResourceType, req.ResourceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, result)
}

// AnalyzeCapacity 分析容量
func (h *APIHandler) AnalyzeCapacity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceType string `json:"resource_type"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	analysis, err := h.service.AnalyzeCapacity(r.Context(), req.ResourceType)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, analysis)
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

func getUserID(r *http.Request) uint {
	// TODO: 从认证上下文中获取用户ID
	return 1
}

func getTenantID(r *http.Request) uint {
	// TODO: 从认证上下文中获取租户ID
	return 1
}
