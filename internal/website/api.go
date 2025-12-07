package website

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// APIHandler 网站管理 API 处理器
// 聚合了网站、代理、SSL、DNS 和 AI 优化等服务
type APIHandler struct {
	websiteService WebsiteService        // 网站管理服务
	proxyService   ProxyService          // 反向代理服务
	sslService     SSLService            // SSL 证书服务
	dnsService     DNSService            // DNS 管理服务
	aiService      AIOptimizationService // AI 优化服务
}

// NewAPIHandler 创建 API 处理器
// 初始化所有相关服务并返回处理器实例
func NewAPIHandler(db *gorm.DB) *APIHandler {
	websiteService := NewWebsiteService(db)
	proxyService := NewProxyService(db)
	sslService := NewSSLService(db)
	dnsService := NewDNSService(db)
	aiService := NewAIOptimizationService(db, websiteService, proxyService)

	return &APIHandler{
		websiteService: websiteService,
		proxyService:   proxyService,
		sslService:     sslService,
		dnsService:     dnsService,
		aiService:      aiService,
	}
}

// RegisterRoutes 注册路由
func (h *APIHandler) RegisterRoutes(router *mux.Router) {
	// 网站管理路由
	router.HandleFunc("/api/v1/websites", h.ListWebsites).Methods("GET")
	router.HandleFunc("/api/v1/websites", h.CreateWebsite).Methods("POST")
	router.HandleFunc("/api/v1/websites/{id}", h.GetWebsite).Methods("GET")
	router.HandleFunc("/api/v1/websites/{id}", h.UpdateWebsite).Methods("PUT")
	router.HandleFunc("/api/v1/websites/{id}", h.DeleteWebsite).Methods("DELETE")
	router.HandleFunc("/api/v1/websites/{id}/ssl/enable", h.EnableSSL).Methods("POST")
	router.HandleFunc("/api/v1/websites/{id}/ssl/disable", h.DisableSSL).Methods("POST")

	// SSL 证书管理路由
	router.HandleFunc("/api/v1/ssl/certs", h.ListSSLCerts).Methods("GET")
	router.HandleFunc("/api/v1/ssl/certs", h.CreateSSLCert).Methods("POST")
	router.HandleFunc("/api/v1/ssl/certs/{id}", h.GetSSLCert).Methods("GET")
	router.HandleFunc("/api/v1/ssl/certs/{id}", h.UpdateSSLCert).Methods("PUT")
	router.HandleFunc("/api/v1/ssl/certs/{id}", h.DeleteSSLCert).Methods("DELETE")
	router.HandleFunc("/api/v1/ssl/certs/request", h.RequestCertificate).Methods("POST")
	router.HandleFunc("/api/v1/ssl/certs/{id}/renew", h.RenewCertificate).Methods("POST")
	router.HandleFunc("/api/v1/ssl/certs/check-expiry", h.CheckExpiry).Methods("GET")

	// 反向代理配置路由
	router.HandleFunc("/api/v1/proxy/configs", h.ListProxyConfigs).Methods("GET")
	router.HandleFunc("/api/v1/proxy/configs", h.CreateProxyConfig).Methods("POST")
	router.HandleFunc("/api/v1/proxy/configs/{id}", h.GetProxyConfig).Methods("GET")
	router.HandleFunc("/api/v1/proxy/configs/{id}", h.UpdateProxyConfig).Methods("PUT")
	router.HandleFunc("/api/v1/proxy/configs/{id}", h.DeleteProxyConfig).Methods("DELETE")
	router.HandleFunc("/api/v1/proxy/nginx/reload", h.ReloadNginx).Methods("POST")

	// DNS 管理路由
	router.HandleFunc("/api/v1/dns/records", h.ListDNSRecords).Methods("GET")
	router.HandleFunc("/api/v1/dns/records", h.CreateDNSRecord).Methods("POST")
	router.HandleFunc("/api/v1/dns/records/{id}", h.GetDNSRecord).Methods("GET")
	router.HandleFunc("/api/v1/dns/records/{id}", h.UpdateDNSRecord).Methods("PUT")
	router.HandleFunc("/api/v1/dns/records/{id}", h.DeleteDNSRecord).Methods("DELETE")
	router.HandleFunc("/api/v1/dns/verify", h.VerifyDNS).Methods("POST")
	router.HandleFunc("/api/v1/dns/sync", h.SyncWithProvider).Methods("POST")

	// AI 优化路由
	router.HandleFunc("/api/v1/websites/{id}/analyze", h.AnalyzeWebsiteConfig).Methods("GET")
	router.HandleFunc("/api/v1/websites/{id}/optimize", h.GenerateOptimizations).Methods("GET")
	router.HandleFunc("/api/v1/websites/{id}/autofix", h.AutoFixIssues).Methods("POST")
	router.HandleFunc("/api/v1/websites/{id}/performance", h.AnalyzePerformance).Methods("GET")

	// 健康检查
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
}

// HealthCheck 健康检查
func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// ListWebsites 列出网站
// 支持分页查询，返回当前用户和租户下的所有网站
func (h *APIHandler) ListWebsites(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	tenantID := getTenantID(r)
	page := getQueryInt(r, "page", 1)
	pageSize := getQueryInt(r, "pageSize", 20)

	websites, total, err := h.websiteService.ListWebsites(r.Context(), userID, tenantID, page, pageSize)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"websites": websites,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// CreateWebsite 创建网站
func (h *APIHandler) CreateWebsite(w http.ResponseWriter, r *http.Request) {
	var website Website
	if err := json.NewDecoder(r.Body).Decode(&website); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	website.UserID = getUserID(r)
	website.TenantID = getTenantID(r)

	if err := h.websiteService.CreateWebsite(r.Context(), &website); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, website)
}

// GetWebsite 获取网站
func (h *APIHandler) GetWebsite(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	website, err := h.websiteService.GetWebsite(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, website)
}

// UpdateWebsite 更新网站
func (h *APIHandler) UpdateWebsite(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	var website Website
	if err := json.NewDecoder(r.Body).Decode(&website); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	website.ID = id
	if err := h.websiteService.UpdateWebsite(r.Context(), &website); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, website)
}

// DeleteWebsite 删除网站
func (h *APIHandler) DeleteWebsite(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	if err := h.websiteService.DeleteWebsite(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Website deleted successfully"})
}

// EnableSSL 启用 SSL
func (h *APIHandler) EnableSSL(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	var req struct {
		CertID uint `json:"cert_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.websiteService.EnableSSL(r.Context(), id, req.CertID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "SSL enabled successfully"})
}

// DisableSSL 禁用 SSL
func (h *APIHandler) DisableSSL(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	if err := h.websiteService.DisableSSL(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "SSL disabled successfully"})
}

// ListSSLCerts 列出 SSL 证书
func (h *APIHandler) ListSSLCerts(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	tenantID := getTenantID(r)

	certs, err := h.sslService.ListSSLCerts(r.Context(), userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, certs)
}

// CreateSSLCert 创建 SSL 证书记录
func (h *APIHandler) CreateSSLCert(w http.ResponseWriter, r *http.Request) {
	var cert SSLCert
	if err := json.NewDecoder(r.Body).Decode(&cert); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	cert.UserID = getUserID(r)
	cert.TenantID = getTenantID(r)

	if err := h.sslService.CreateSSLCert(r.Context(), &cert); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, cert)
}

// GetSSLCert 获取 SSL 证书
func (h *APIHandler) GetSSLCert(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	cert, err := h.sslService.GetSSLCert(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, cert)
}

// UpdateSSLCert 更新 SSL 证书
func (h *APIHandler) UpdateSSLCert(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	var cert SSLCert
	if err := json.NewDecoder(r.Body).Decode(&cert); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	cert.ID = id
	if err := h.sslService.UpdateSSLCert(r.Context(), &cert); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, cert)
}

// DeleteSSLCert 删除 SSL 证书
func (h *APIHandler) DeleteSSLCert(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	if err := h.sslService.DeleteSSLCert(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "SSL certificate deleted successfully"})
}

// RequestCertificate 申请证书
// 通过指定的提供商（如 Let's Encrypt）为域名申请 SSL 证书
func (h *APIHandler) RequestCertificate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain   string      `json:"domain"`
		Email    string      `json:"email"`
		Provider SSLProvider `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	cert, err := h.sslService.RequestCertificate(r.Context(), req.Domain, req.Email, req.Provider)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	cert.UserID = getUserID(r)
	cert.TenantID = getTenantID(r)

	respondJSON(w, http.StatusCreated, cert)
}

// RenewCertificate 续期证书
func (h *APIHandler) RenewCertificate(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	if err := h.sslService.RenewCertificate(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Certificate renewed successfully"})
}

// CheckExpiry 检查证书过期状态
func (h *APIHandler) CheckExpiry(w http.ResponseWriter, r *http.Request) {
	certs, err := h.sslService.CheckExpiry(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, certs)
}

// ListProxyConfigs 列出代理配置
func (h *APIHandler) ListProxyConfigs(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	tenantID := getTenantID(r)

	configs, err := h.proxyService.ListProxyConfigs(r.Context(), userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, configs)
}

// CreateProxyConfig 创建代理配置
func (h *APIHandler) CreateProxyConfig(w http.ResponseWriter, r *http.Request) {
	var config ProxyConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	config.UserID = getUserID(r)
	config.TenantID = getTenantID(r)

	if err := h.proxyService.CreateProxyConfig(r.Context(), &config); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, config)
}

// GetProxyConfig 获取代理配置
func (h *APIHandler) GetProxyConfig(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	config, err := h.proxyService.GetProxyConfig(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, config)
}

// UpdateProxyConfig 更新代理配置
func (h *APIHandler) UpdateProxyConfig(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	var config ProxyConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	config.ID = id
	if err := h.proxyService.UpdateProxyConfig(r.Context(), &config); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, config)
}

// DeleteProxyConfig 删除代理配置
func (h *APIHandler) DeleteProxyConfig(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	if err := h.proxyService.DeleteProxyConfig(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Proxy config deleted successfully"})
}

// ReloadNginx 重载 Nginx
func (h *APIHandler) ReloadNginx(w http.ResponseWriter, r *http.Request) {
	if err := h.proxyService.ReloadNginx(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Nginx reloaded successfully"})
}

// ListDNSRecords 列出 DNS 记录
func (h *APIHandler) ListDNSRecords(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	userID := getUserID(r)
	tenantID := getTenantID(r)

	records, err := h.dnsService.ListDNSRecords(r.Context(), domain, userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, records)
}

// CreateDNSRecord 创建 DNS 记录
func (h *APIHandler) CreateDNSRecord(w http.ResponseWriter, r *http.Request) {
	var record DNSRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	record.UserID = getUserID(r)
	record.TenantID = getTenantID(r)

	if err := h.dnsService.CreateDNSRecord(r.Context(), &record); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, record)
}

// GetDNSRecord 获取 DNS 记录
func (h *APIHandler) GetDNSRecord(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	record, err := h.dnsService.GetDNSRecord(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, record)
}

// UpdateDNSRecord 更新 DNS 记录
func (h *APIHandler) UpdateDNSRecord(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	var record DNSRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	record.ID = id
	if err := h.dnsService.UpdateDNSRecord(r.Context(), &record); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, record)
}

// DeleteDNSRecord 删除 DNS 记录
func (h *APIHandler) DeleteDNSRecord(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	if err := h.dnsService.DeleteDNSRecord(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "DNS record deleted successfully"})
}

// VerifyDNS 验证 DNS 解析
// 检查域名的 DNS 记录是否已正确解析到期望值
func (h *APIHandler) VerifyDNS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain        string `json:"domain"`
		RecordType    string `json:"record_type"`
		ExpectedValue string `json:"expected_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	verified, err := h.dnsService.VerifyDNS(r.Context(), req.Domain, req.RecordType, req.ExpectedValue)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{"verified": verified})
}

// SyncWithProvider 与 DNS 提供商同步
// 从云服务商（阿里云、腾讯云等）同步 DNS 记录到本地数据库
func (h *APIHandler) SyncWithProvider(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain   string `json:"domain"`
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.dnsService.SyncWithProvider(r.Context(), req.Domain, req.Provider); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "DNS records synced successfully"})
}

// AnalyzeWebsiteConfig 分析网站配置
func (h *APIHandler) AnalyzeWebsiteConfig(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	analysis, err := h.aiService.AnalyzeWebsiteConfig(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, analysis)
}

// GenerateOptimizations 生成优化建议
func (h *APIHandler) GenerateOptimizations(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	suggestions, err := h.aiService.GenerateOptimizationSuggestions(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, suggestions)
}

// AutoFixIssues 自动修复问题
func (h *APIHandler) AutoFixIssues(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	result, err := h.aiService.AutoFixCommonIssues(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// AnalyzePerformance 分析性能
func (h *APIHandler) AnalyzePerformance(w http.ResponseWriter, r *http.Request) {
	id := getIDFromPath(r)
	analysis, err := h.aiService.AnalyzePerformance(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, analysis)
}

// 辅助函数

// respondJSON 返回 JSON 格式响应
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError 返回错误响应
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// getIDFromPath 从路径参数中提取 ID
func getIDFromPath(r *http.Request) uint {
	vars := mux.Vars(r)
	id, _ := strconv.ParseUint(vars["id"], 10, 32)
	return uint(id)
}

// getUserID 获取当前用户 ID
// TODO: 从认证上下文中获取用户ID
func getUserID(r *http.Request) uint {
	return 1
}

// getTenantID 获取当前租户 ID
// TODO: 从认证上下文中获取租户ID
func getTenantID(r *http.Request) uint {
	return 1
}

// getQueryInt 从查询参数中获取整数值，支持默认值
func getQueryInt(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
