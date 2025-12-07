package backup

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// APIHandler 备份服务 API 处理器
type APIHandler struct {
	service BackupService
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(service BackupService) *APIHandler {
	return &APIHandler{
		service: service,
	}
}

// RegisterRoutes 注册路由
func (h *APIHandler) RegisterRoutes(router *mux.Router) {
	// 备份策略路由
	router.HandleFunc("/api/v1/backups/policies", h.ListPolicies).Methods("GET")
	router.HandleFunc("/api/v1/backups/policies", h.CreatePolicy).Methods("POST")
	router.HandleFunc("/api/v1/backups/policies/{id}", h.GetPolicy).Methods("GET")
	router.HandleFunc("/api/v1/backups/policies/{id}", h.UpdatePolicy).Methods("PUT")
	router.HandleFunc("/api/v1/backups/policies/{id}", h.DeletePolicy).Methods("DELETE")
	router.HandleFunc("/api/v1/backups/policies/{id}/health", h.CheckHealth).Methods("GET")
	
	// 备份任务路由
	router.HandleFunc("/api/v1/backups/policies/{id}/execute", h.ExecuteBackup).Methods("POST")
	router.HandleFunc("/api/v1/backups/policies/{id}/jobs", h.ListBackupJobs).Methods("GET")
	router.HandleFunc("/api/v1/backups/jobs/{id}", h.GetBackupJob).Methods("GET")
	router.HandleFunc("/api/v1/backups/jobs/{id}/validate", h.ValidateBackup).Methods("POST")
	
	// 恢复任务路由
	router.HandleFunc("/api/v1/backups/jobs/{id}/restore", h.RestoreBackup).Methods("POST")
	router.HandleFunc("/api/v1/backups/restores", h.ListRestoreJobs).Methods("GET")
	router.HandleFunc("/api/v1/backups/restores/{id}", h.GetRestoreJob).Methods("GET")
	
	// 健康检查
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
}

// HealthCheck 健康检查
func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// ListPolicies 列出备份策略
func (h *APIHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	tenantID := getTenantID(r)
	
	policies, err := h.service.ListPolicies(r.Context(), userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, policies)
}

// CreatePolicy 创建备份策略
func (h *APIHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var policy BackupPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	policy.UserID = getUserID(r)
	policy.TenantID = getTenantID(r)
	
	if err := h.service.CreatePolicy(r.Context(), &policy); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusCreated, policy)
}

// GetPolicy 获取备份策略
func (h *APIHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	policy, err := h.service.GetPolicy(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, policy)
}

// UpdatePolicy 更新备份策略
func (h *APIHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	var policy BackupPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if err := h.service.UpdatePolicy(r.Context(), id, &policy); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, policy)
}

// DeletePolicy 删除备份策略
func (h *APIHandler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	if err := h.service.DeletePolicy(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]string{"message": "Policy deleted successfully"})
}

// ExecuteBackup 执行备份
func (h *APIHandler) ExecuteBackup(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	job, err := h.service.ExecuteBackup(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusAccepted, job)
}

// ListBackupJobs 列出备份任务
func (h *APIHandler) ListBackupJobs(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	jobs, err := h.service.ListBackupJobs(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, jobs)
}

// GetBackupJob 获取备份任务
func (h *APIHandler) GetBackupJob(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	job, err := h.service.GetBackupJob(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, job)
}

// ValidateBackup 验证备份
func (h *APIHandler) ValidateBackup(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	result, err := h.service.ValidateBackup(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, result)
}

// RestoreBackup 恢复备份
func (h *APIHandler) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	var target RestoreTarget
	if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	job, err := h.service.RestoreBackup(r.Context(), id, &target)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusAccepted, job)
}

// ListRestoreJobs 列出恢复任务
func (h *APIHandler) ListRestoreJobs(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	tenantID := getTenantID(r)
	
	jobs, err := h.service.ListRestoreJobs(r.Context(), userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, jobs)
}

// GetRestoreJob 获取恢复任务
func (h *APIHandler) GetRestoreJob(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	job, err := h.service.GetRestoreJob(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, job)
}

// CheckHealth 检查备份健康状态
func (h *APIHandler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	report, err := h.service.CheckBackupHealth(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, report)
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
