// Package server 提供 qwq Ops 的 Web 服务器功能
// 包括前端资源服务、API 接口、WebSocket 连接、文件管理等核心功能
package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"qwq/internal/deployment"
	"qwq/internal/logger"
	"qwq/internal/monitor"
	"qwq/internal/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 前端静态资源嵌入
// 使用 Go embed 将前端构建产物打包到二进制文件中
// 这样可以将前端文件直接嵌入到 Go 二进制文件中，无需单独部署前端资源
// 
// 嵌入策略说明：
// - dist/* : 嵌入 dist 目录下的所有文件（包括 index.html 等）
// - dist/assets/* : 明确嵌入 assets 目录下的所有资源文件
// 这种双重指定确保所有前端文件都被正确包含，避免 404 错误
//go:embed dist/*
//go:embed dist/assets/*
var frontendDist embed.FS

var (
	// Use the upgrader's default same-origin policy for browser connections.
	upgrader = websocket.Upgrader{}
	
	// 外部回调函数，由主程序注入
	TriggerPatrolFunc func() // 触发系统巡检的回调函数
	TriggerStatusFunc func() // 触发状态推送的回调函数
	
	// 部署集成服务实例
	deploymentService *deployment.IntegrationService
	
	// 监控数据缓存
	// 使用读写锁保护并发访问，存储最近的系统监控数据点
	statsCache struct {
		sync.RWMutex
		History []StatsPoint // 历史监控数据，最多保存 60 个数据点
	}
	
	// 网站配置存储
	// 使用读写锁保护并发访问，存储网站配置信息
	websitesStore struct {
		sync.RWMutex
		Websites []Website // 网站配置列表
		NextID   int       // 下一个可用的ID
	}
	
	// 用户数据存储
	usersStore struct {
		sync.RWMutex
		Users []User // 用户列表
		NextID int   // 下一个可用的ID
	}
	
	// 角色数据存储
	rolesStore struct {
		sync.RWMutex
		Roles []Role // 角色列表
		NextID int   // 下一个可用的ID
	}
	
	// 权限数据存储（只读，预定义）
	permissionsStore = []Permission{
		{ID: 1, Resource: "websites", Action: "read", Description: "查看网站列表"},
		{ID: 2, Resource: "websites", Action: "write", Description: "创建/编辑网站"},
		{ID: 3, Resource: "websites", Action: "delete", Description: "删除网站"},
		{ID: 4, Resource: "users", Action: "read", Description: "查看用户列表"},
		{ID: 5, Resource: "users", Action: "write", Description: "创建/编辑用户"},
		{ID: 6, Resource: "users", Action: "delete", Description: "删除用户"},
		{ID: 7, Resource: "roles", Action: "read", Description: "查看角色列表"},
		{ID: 8, Resource: "roles", Action: "write", Description: "创建/编辑角色"},
		{ID: 9, Resource: "roles", Action: "delete", Description: "删除角色"},
		{ID: 10, Resource: "containers", Action: "read", Description: "查看容器列表"},
		{ID: 11, Resource: "containers", Action: "write", Description: "管理容器"},
		{ID: 12, Resource: "files", Action: "read", Description: "查看文件"},
		{ID: 13, Resource: "files", Action: "write", Description: "编辑文件"},
		{ID: 14, Resource: "logs", Action: "read", Description: "查看日志"},
	}
)

// User 用户结构
type User struct {
	ID        int      `json:"id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Password  string   `json:"-"` // 密码不返回给前端
	Roles     []string `json:"roles"`
	Enabled   bool     `json:"enabled"`
	CreatedAt string   `json:"created_at"`
}

// Role 角色结构
type Role struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"created_at"`
}

// Permission 权限结构
type Permission struct {
	ID          int    `json:"id"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

// StatsPoint 系统监控数据点结构
// 包含系统资源使用情况的快照数据
type StatsPoint struct {
	Time      string      `json:"time"`       // 数据采集时间 (HH:MM:SS 格式)
	Load      string      `json:"load"`       // 系统负载 (1分钟,5分钟,15分钟平均值)
	MemPct    string      `json:"mem_pct"`    // 内存使用百分比
	MemUsed   string      `json:"mem_used"`   // 已使用内存大小 (MB)
	MemTotal  string      `json:"mem_total"`  // 系统总内存大小 (MB)
	DiskPct   string      `json:"disk_pct"`   // 根目录磁盘使用百分比
	DiskAvail string      `json:"disk_avail"` // 根目录可用磁盘空间
	TcpConn   string      `json:"tcp_conn"`   // 当前 TCP 连接数
	Services  interface{} `json:"services"`   // HTTP 服务健康检查状态
}

// DockerContainer Docker 容器信息结构
// 用于容器管理 API 的数据传输
type DockerContainer struct {
	ID      string `json:"id"`     // 容器唯一标识符
	Image   string `json:"image"`  // 容器使用的镜像名称
	Status  string `json:"status"` // 容器状态描述 (如 "Up 2 hours")
	Name    string `json:"name"`   // 容器名称
	State   string `json:"state"`  // 运行状态 (running/exited)
}

// Website 网站配置结构
// 用于网站管理 API 的数据传输
type Website struct {
	ID            int    `json:"id"`             // 网站唯一标识符
	Domain        string `json:"domain"`         // 域名
	BackendURL    string `json:"backend_url"`    // 后端服务地址
	SSLEnabled    bool   `json:"ssl_enabled"`    // 是否启用SSL
	Enabled       bool   `json:"enabled"`        // 网站是否启用
	LoadBalance   string `json:"load_balance"`   // 负载均衡策略
	SSLCertExpiry string `json:"ssl_cert_expiry,omitempty"` // SSL证书有效期
	CreatedAt     string `json:"created_at"`     // 创建时间
}

// ============================================
// 容器管理 API
// ============================================

// ============================================
// 监控数据采集
// ============================================

// collectStatsLoop 定时采集系统监控数据
func collectStatsLoopContext(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		point := collectOnePointContext(ctx)
		if ctx.Err() != nil {
			return
		}
		statsCache.Lock()
		statsCache.History = append(statsCache.History, point)
		if len(statsCache.History) > 60 { statsCache.History = statsCache.History[1:] }
		statsCache.Unlock()
	}
}

// collectOnePoint 采集一次系统监控数据
// 包括：系统负载、内存使用、磁盘使用、TCP 连接数、服务状态
func collectOnePointContext(ctx context.Context) StatsPoint {
	// 获取系统负载（1分钟、5分钟、15分钟平均值）
	load := strings.TrimSpace(utils.ExecuteShellContext(ctx, "uptime | awk -F'load average:' '{ print $2 }'"))
	
	// 获取内存使用情况（单位：MB）
	memRaw := utils.ExecuteShellContext(ctx, "free -m | awk 'NR==2{print $2,$3}'")
	var memTotal, memUsed float64
	fmt.Sscanf(memRaw, "%f %f", &memTotal, &memUsed)
	memPct := 0.0
	if memTotal > 0 { 
		memPct = (memUsed / memTotal) * 100 
	}
	
	// 获取根目录磁盘使用情况（仪表盘只显示根目录）
	diskRaw := utils.ExecuteShellContext(ctx, "df -h / | awk 'NR==2 {print $5,$4}'")
	diskParts := strings.Fields(diskRaw)
	diskPct := "0"
	diskAvail := "0G"
	if len(diskParts) >= 2 {
		diskPct = strings.TrimSuffix(diskParts[0], "%")
		diskAvail = diskParts[1]
	}
	
	// 获取 TCP 连接数（已建立的连接）
	tcpRaw := utils.ExecuteShellContext(ctx, "ss -s | grep 'TCP:' | grep -oE 'estab [0-9]+' | awk '{print $2}'")
	tcpConn := strings.TrimSpace(tcpRaw)
	if tcpConn == "" { 
		tcpConn = "0" 
	}
	
	// 执行 HTTP 服务健康检查
	httpStatus := monitor.RunChecksContext(ctx)
	
	return StatsPoint{
		Time:      time.Now().Format("15:04:05"),
		Load:      load,
		MemPct:    fmt.Sprintf("%.1f", memPct),
		MemUsed:   fmt.Sprintf("%.0f", memUsed),
		MemTotal:  fmt.Sprintf("%.0f", memTotal),
		DiskPct:   diskPct,
		DiskAvail: diskAvail,
		TcpConn:   tcpConn,
		Services:  httpStatus,
	}
}

// ============================================
// WebSocket 聊天接口
// ============================================

// ============================================
// 通用 API 处理器
// ============================================

// handleLogs 获取系统日志
func handleLogs(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(logger.GetWebLogs())
}

// handleStats 获取监控统计数据
// 返回最近 60 个数据点（2 分钟历史）
func handleStats(w http.ResponseWriter, r *http.Request) {
	statsCache.RLock()
	defer statsCache.RUnlock()
	
	if len(statsCache.History) == 0 {
		json.NewEncoder(w).Encode([]StatsPoint{})
		return
	}
	json.NewEncoder(w).Encode(statsCache.History)
}

// handleTrigger 手动触发巡检和状态推送
// 异步执行，立即返回响应
func handleTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if TriggerPatrolFunc != nil { 
		go TriggerPatrolFunc() 
	}
	if TriggerStatusFunc != nil { 
		go TriggerStatusFunc() 
	}
	w.Write([]byte("指令已发送：正在后台执行巡检和汇报..."))
}

// WebLog 记录 Web 日志（供外部调用）
func WebLog(msg string) {
	logger.Info("%s", msg)
}

// ============================================
// 文件管理 API 处理器（在 files.go 中实现）
// ============================================

// ============================================
// 网站管理 API
// ============================================

// handleWebsites 处理网站列表和创建请求
func handleWebsites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// 获取网站列表
		websitesStore.RLock()
		defer websitesStore.RUnlock()
		
		// 确保返回数组格式，即使为空也返回 []
		sites := websitesStore.Websites
		if sites == nil {
			sites = []Website{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sites)
		
	case http.MethodPost:
		// 创建新网站
		var form struct {
			Domain      string `json:"domain"`
			BackendURL  string `json:"backend_url"`
			SSLEnabled  bool   `json:"ssl_enabled"`
			LoadBalance string `json:"load_balance"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// 参数验证
		if form.Domain == "" {
			http.Error(w, "Domain is required", http.StatusBadRequest)
			return
		}
		
		websitesStore.Lock()
		defer websitesStore.Unlock()
		
		// 检查域名是否已存在
		for _, site := range websitesStore.Websites {
			if site.Domain == form.Domain {
				http.Error(w, "Domain already exists", http.StatusConflict)
				return
			}
		}
		
		// 创建新网站
		newWebsite := Website{
			ID:          websitesStore.NextID,
			Domain:      form.Domain,
			BackendURL:  form.BackendURL,
			SSLEnabled:  form.SSLEnabled,
			Enabled:     true,
			LoadBalance: form.LoadBalance,
			CreatedAt:   time.Now().Format(time.RFC3339),
		}
		
		websitesStore.NextID++
		websitesStore.Websites = append(websitesStore.Websites, newWebsite)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newWebsite)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleWebsiteDetail 处理单个网站的详情、更新、删除和SSL管理请求
func handleWebsiteDetail(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取网站ID和可能的操作类型
	path := strings.TrimPrefix(r.URL.Path, "/api/websites/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Website ID is required", http.StatusBadRequest)
		return
	}
	
	idStr := parts[0]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid website ID", http.StatusBadRequest)
		return
	}
	
	// 检查是否是SSL操作
	if len(parts) >= 3 && parts[1] == "ssl" {
		handleWebsiteSSL(w, r, id, parts[2])
		return
	}
	
	websitesStore.Lock()
	defer websitesStore.Unlock()
	
	// 查找网站
	index := -1
	for i, site := range websitesStore.Websites {
		if site.ID == id {
			index = i
			break
		}
	}
	
	if index == -1 {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}
	
	switch r.Method {
	case http.MethodGet:
		// 获取网站详情
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(websitesStore.Websites[index])
		
	case http.MethodPut:
		// 更新网站
		var form struct {
			Enabled     *bool   `json:"enabled"`
			BackendURL  *string `json:"backend_url"`
			SSLEnabled  *bool   `json:"ssl_enabled"`
			LoadBalance *string `json:"load_balance"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// 更新字段
		if form.Enabled != nil {
			websitesStore.Websites[index].Enabled = *form.Enabled
		}
		if form.BackendURL != nil {
			websitesStore.Websites[index].BackendURL = *form.BackendURL
		}
		if form.SSLEnabled != nil {
			websitesStore.Websites[index].SSLEnabled = *form.SSLEnabled
		}
		if form.LoadBalance != nil {
			websitesStore.Websites[index].LoadBalance = *form.LoadBalance
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(websitesStore.Websites[index])
		
	case http.MethodDelete:
		// 删除网站
		websitesStore.Websites = append(websitesStore.Websites[:index], websitesStore.Websites[index+1:]...)
		w.WriteHeader(http.StatusNoContent)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleWebsiteSSL 处理SSL证书管理请求
func handleWebsiteSSL(w http.ResponseWriter, r *http.Request, id int, action string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if action != "apply" && action != "renew" {
		http.Error(w, "Invalid SSL action", http.StatusBadRequest)
		return
	}
	websitesStore.RLock()
	defer websitesStore.RUnlock()
	for _, site := range websitesStore.Websites {
		if site.ID == id {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotImplemented)
			json.NewEncoder(w).Encode(map[string]string{"error": "Certificate management is not connected to the dashboard"})
			return
		}
	}
	http.Error(w, "Website not found", http.StatusNotFound)
}

// ============================================
// 用户管理 API
// ============================================

// handleUsers 处理用户列表和创建请求
func handleUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.Method {
	case http.MethodGet:
		// 获取用户列表
		usersStore.RLock()
		defer usersStore.RUnlock()
		
		// 创建返回列表，排除密码字段
		users := make([]map[string]interface{}, len(usersStore.Users))
		for i, user := range usersStore.Users {
			users[i] = map[string]interface{}{
				"id":         user.ID,
				"username":   user.Username,
				"email":      user.Email,
				"roles":      user.Roles,
				"enabled":    user.Enabled,
				"created_at": user.CreatedAt,
			}
		}
		json.NewEncoder(w).Encode(users)
		
	case http.MethodPost:
		// 创建新用户
		var form struct {
			Username string   `json:"username"`
			Email    string   `json:"email"`
			Password string   `json:"password"`
			Roles    []string `json:"roles"`
			Enabled  bool     `json:"enabled"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// 参数验证
		if form.Username == "" {
			http.Error(w, "Username is required", http.StatusBadRequest)
			return
		}
		if form.Email == "" {
			http.Error(w, "Email is required", http.StatusBadRequest)
			return
		}
		
		usersStore.Lock()
		defer usersStore.Unlock()
		
		// 检查用户名是否已存在
		for _, u := range usersStore.Users {
			if u.Username == form.Username {
				http.Error(w, "Username already exists", http.StatusConflict)
				return
			}
		}
		
		// 创建新用户
		newUser := User{
			ID:        usersStore.NextID,
			Username:  form.Username,
			Email:     form.Email,
			Password:  form.Password, // 实际应用中应该加密
			Roles:     form.Roles,
			Enabled:   form.Enabled,
			CreatedAt: time.Now().Format(time.RFC3339),
		}
		
		usersStore.NextID++
		usersStore.Users = append(usersStore.Users, newUser)
		
		// 返回用户（不包含密码）
		response := map[string]interface{}{
			"id":         newUser.ID,
			"username":   newUser.Username,
			"email":      newUser.Email,
			"roles":      newUser.Roles,
			"enabled":    newUser.Enabled,
			"created_at": newUser.CreatedAt,
		}
		json.NewEncoder(w).Encode(response)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleUserDetail 处理单个用户的详情、更新、删除和权限管理请求
func handleUserDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/users/")
	parts := strings.Split(path, "/")
	
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}
	
	idStr := parts[0]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	
	// 检查是否是权限管理操作
	if len(parts) >= 2 && parts[1] == "permissions" {
		handleUserPermissions(w, r, id)
		return
	}
	
	usersStore.Lock()
	defer usersStore.Unlock()
	
	// 查找用户
	index := -1
	for i, user := range usersStore.Users {
		if user.ID == id {
			index = i
			break
		}
	}
	
	if index == -1 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	
	switch r.Method {
	case http.MethodGet:
		// 获取用户详情
		user := usersStore.Users[index]
		response := map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"roles":      user.Roles,
			"enabled":    user.Enabled,
			"created_at": user.CreatedAt,
		}
		json.NewEncoder(w).Encode(response)
		
	case http.MethodPut:
		// 更新用户
		var form struct {
			Username *string   `json:"username"`
			Email    *string   `json:"email"`
			Password *string   `json:"password"`
			Roles    *[]string `json:"roles"`
			Enabled  *bool     `json:"enabled"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// 更新字段
		if form.Username != nil {
			usersStore.Users[index].Username = *form.Username
		}
		if form.Email != nil {
			usersStore.Users[index].Email = *form.Email
		}
		if form.Password != nil {
			usersStore.Users[index].Password = *form.Password
		}
		if form.Roles != nil {
			usersStore.Users[index].Roles = *form.Roles
		}
		if form.Enabled != nil {
			usersStore.Users[index].Enabled = *form.Enabled
		}
		
		user := usersStore.Users[index]
		response := map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"roles":      user.Roles,
			"enabled":    user.Enabled,
			"created_at": user.CreatedAt,
		}
		json.NewEncoder(w).Encode(response)
		
	case http.MethodDelete:
		// 删除用户
		usersStore.Users = append(usersStore.Users[:index], usersStore.Users[index+1:]...)
		w.WriteHeader(http.StatusNoContent)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleUserPermissions 处理用户权限管理
func handleUserPermissions(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		w.Header().Set("Allow", "GET, PUT")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	usersStore.RLock()
	defer usersStore.RUnlock()
	for _, user := range usersStore.Users {
		if user.ID == id {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotImplemented)
			json.NewEncoder(w).Encode(map[string]string{"error": "Per-user authorization is not connected to the dashboard"})
			return
		}
	}
	http.Error(w, "User not found", http.StatusNotFound)
}

// ============================================
// 角色管理 API
// ============================================

// handleRoles 处理角色列表和创建请求
func handleRoles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.Method {
	case http.MethodGet:
		// 获取角色列表
		rolesStore.RLock()
		defer rolesStore.RUnlock()
		
		json.NewEncoder(w).Encode(rolesStore.Roles)
		
	case http.MethodPost:
		// 创建新角色
		var form struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Permissions []string `json:"permissions"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// 参数验证
		if form.Name == "" {
			http.Error(w, "Role name is required", http.StatusBadRequest)
			return
		}
		
		rolesStore.Lock()
		defer rolesStore.Unlock()
		
		// 检查角色名是否已存在
		for _, role := range rolesStore.Roles {
			if role.Name == form.Name {
				http.Error(w, "Role name already exists", http.StatusConflict)
				return
			}
		}
		
		// 创建新角色
		newRole := Role{
			ID:          rolesStore.NextID,
			Name:        form.Name,
			Description: form.Description,
			Permissions: form.Permissions,
			CreatedAt:   time.Now().Format(time.RFC3339),
		}
		
		rolesStore.NextID++
		rolesStore.Roles = append(rolesStore.Roles, newRole)
		
		json.NewEncoder(w).Encode(newRole)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRoleDetail 处理单个角色的详情、更新和删除请求
func handleRoleDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/roles/")
	parts := strings.Split(path, "/")
	
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Role ID is required", http.StatusBadRequest)
		return
	}
	
	idStr := parts[0]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}
	
	rolesStore.Lock()
	defer rolesStore.Unlock()
	
	// 查找角色
	index := -1
	for i, role := range rolesStore.Roles {
		if role.ID == id {
			index = i
			break
		}
	}
	
	if index == -1 {
		http.Error(w, "Role not found", http.StatusNotFound)
		return
	}
	
	switch r.Method {
	case http.MethodGet:
		// 获取角色详情
		json.NewEncoder(w).Encode(rolesStore.Roles[index])
		
	case http.MethodPut:
		// 更新角色
		var form struct {
			Name        *string   `json:"name"`
			Description *string   `json:"description"`
			Permissions *[]string `json:"permissions"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// 更新字段
		if form.Name != nil {
			rolesStore.Roles[index].Name = *form.Name
		}
		if form.Description != nil {
			rolesStore.Roles[index].Description = *form.Description
		}
		if form.Permissions != nil {
			rolesStore.Roles[index].Permissions = *form.Permissions
		}
		
		json.NewEncoder(w).Encode(rolesStore.Roles[index])
		
	case http.MethodDelete:
		// 删除角色
		rolesStore.Roles = append(rolesStore.Roles[:index], rolesStore.Roles[index+1:]...)
		w.WriteHeader(http.StatusNoContent)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ============================================
// 权限管理 API
// ============================================

// handlePermissions 处理权限列表请求
func handlePermissions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(permissionsStore)
}

// ============================================
// 应用商店 API
// ============================================

// handleAppStoreTemplates 处理应用模板列表请求
func handleAppStoreTemplates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// 返回空数组，避免前端报错
	json.NewEncoder(w).Encode([]interface{}{})
}

// handleAppStoreInstances 处理应用实例请求
func handleAppStoreInstances(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.Method {
	case http.MethodGet:
		// 获取应用实例列表
		json.NewEncoder(w).Encode([]interface{}{})
		
	case http.MethodPost, http.MethodDelete:
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{"error": "Application instance management is not connected to the dashboard"})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ============================================
// 数据库管理 API
// ============================================

// handleDatabaseConnections 处理数据库连接请求
func handleDatabaseConnections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.Method {
	case http.MethodGet:
		// 获取数据库连接列表
		json.NewEncoder(w).Encode([]interface{}{})
		
	case http.MethodPost, http.MethodPut, http.MethodDelete:
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database connection management is not connected to the dashboard"})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
// ============================================
// 部署验证和修复 API
// ============================================

// handleDeploymentValidation 处理部署验证请求
func handleDeploymentValidation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logger.Info("🔍 开始部署验证...")
	
	// 运行全面的部署验证
	status := deploymentService.RunComprehensiveValidation()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
	
	logger.Info("✅ 部署验证完成，状态: %s", status.Overall)
}

// handleDeploymentRepair 处理自动修复请求
func handleDeploymentRepair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Runtime repair is not connected; rebuild and restart the program using the deployment guide"})
}

// handleDeploymentStatus 处理部署状态查询请求
func handleDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取部署状态
	status := deploymentService.RunComprehensiveValidation()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleDeploymentWorkflow 处理部署工作流请求
func handleDeploymentWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取完整的部署工作流
	workflow := deploymentService.GetDeploymentWorkflow()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflow)
}

// handleHealthCheck 处理健康检查请求
func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取健康状态
	health := deploymentService.GetHealthStatus()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}
