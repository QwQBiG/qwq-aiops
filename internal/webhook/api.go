package webhook

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// APIHandler Webhook API 处理器
type APIHandler struct {
	service WebhookService
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(service WebhookService) *APIHandler {
	return &APIHandler{
		service: service,
	}
}

// RegisterRoutes 注册路由
func (h *APIHandler) RegisterRoutes(router *mux.Router) {
	// Webhook 管理路由
	router.HandleFunc("/api/v1/webhooks", h.ListWebhooks).Methods("GET")
	router.HandleFunc("/api/v1/webhooks", h.CreateWebhook).Methods("POST")
	router.HandleFunc("/api/v1/webhooks/{id}", h.GetWebhook).Methods("GET")
	router.HandleFunc("/api/v1/webhooks/{id}", h.UpdateWebhook).Methods("PUT")
	router.HandleFunc("/api/v1/webhooks/{id}", h.DeleteWebhook).Methods("DELETE")
	router.HandleFunc("/api/v1/webhooks/{id}/events", h.ListEvents).Methods("GET")
	
	// 事件触发路由（内部使用）
	router.HandleFunc("/api/v1/webhooks/trigger", h.TriggerEvent).Methods("POST")
	
	// 健康检查
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
}

// HealthCheck 健康检查
func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// ListWebhooks 列出 Webhooks
func (h *APIHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	tenantID := getTenantID(r)
	
	webhooks, err := h.service.ListWebhooks(r.Context(), userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, webhooks)
}

// CreateWebhook 创建 Webhook
func (h *APIHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	var webhook Webhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	webhook.UserID = getUserID(r)
	webhook.TenantID = getTenantID(r)
	
	if err := h.service.CreateWebhook(r.Context(), &webhook); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusCreated, webhook)
}

// GetWebhook 获取 Webhook
func (h *APIHandler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	webhook, err := h.service.GetWebhook(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, webhook)
}

// UpdateWebhook 更新 Webhook
func (h *APIHandler) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	var webhook Webhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if err := h.service.UpdateWebhook(r.Context(), id, &webhook); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, webhook)
}

// DeleteWebhook 删除 Webhook
func (h *APIHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	if err := h.service.DeleteWebhook(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]string{"message": "Webhook deleted successfully"})
}

// ListEvents 列出事件日志
func (h *APIHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	events, err := h.service.ListEvents(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, events)
}

// TriggerEvent 触发事件（内部使用）
func (h *APIHandler) TriggerEvent(w http.ResponseWriter, r *http.Request) {
	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if err := h.service.TriggerEvent(r.Context(), &event); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusAccepted, map[string]string{"message": "Event triggered successfully"})
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
