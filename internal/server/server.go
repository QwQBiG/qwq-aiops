// Package server 提供 qwq AIOps 平台的 Web 服务器功能
// 包括前端资源服务、API 接口、WebSocket 连接、文件管理等核心功能
package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"qwq/internal/agent"
	"qwq/internal/config"
	"qwq/internal/deployment"
	"qwq/internal/logger"
	"qwq/internal/monitor"
	"qwq/internal/utils"
	"qwq/internal/notify"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	openai "github.com/sashabaranov/go-openai"
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
	
	// 日志文件句柄，用于写入操作日志
	logFile *os.File
	
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

// Start 启动 Web 服务器
// 初始化日志文件、启动监控数据采集、注册路由处理器并启动 HTTP 服务
func Start(port string) {
	// 初始化日志文件
	// 以追加模式打开日志文件，用于记录 Web 操作日志
	var err error
	logFile, err = os.OpenFile("qwq.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("无法创建日志文件: %v\n", err)
	}

	// 初始化部署集成服务，注入前端管理器适配器
	deploymentService = deployment.NewIntegrationService(GetDefaultFrontendManagerAdapter())
	logger.Info("🔧 部署集成服务已初始化")

	// 启动后台监控数据采集协程
	// 每 2 秒采集一次系统监控数据，保存到内存缓存中
	go collectStatsLoop()

	// 注册核心 API 路由
	http.HandleFunc("/health", handleLiveness)
	http.HandleFunc("/api/logs", basicAuth(handleLogs))                         // 获取系统日志
	http.HandleFunc("/api/stats", basicAuth(handleStats))                       // 获取监控统计数据
	http.HandleFunc("/api/trigger", basicAuth(handleTrigger))                   // 手动触发巡检
	http.HandleFunc("/api/containers", basicAuth(handleContainers))             // 获取容器列表
	http.HandleFunc("/api/container/action", basicAuth(handleContainerAction))  // 容器操作 (启动/停止/重启)
	
	// 网站管理 API 路由
	// 注意：更具体的路由需要先注册，确保路径匹配正确
	http.HandleFunc("/api/websites/", basicAuth(handleWebsiteDetail))            // 网站详情、更新、删除、SSL管理
	http.HandleFunc("/api/websites", basicAuth(handleWebsites))                  // 网站列表和创建
	
	// 用户管理 API 路由（返回空数组，避免前端报错）
	http.HandleFunc("/api/users/", basicAuth(handleUserDetail))                  // 用户详情、更新、删除、权限管理
	http.HandleFunc("/api/users", basicAuth(handleUsers))                        // 用户列表和创建
	http.HandleFunc("/api/roles/", basicAuth(handleRoleDetail))                 // 角色详情、更新、删除
	http.HandleFunc("/api/roles", basicAuth(handleRoles))                       // 角色列表和创建
	http.HandleFunc("/api/permissions", basicAuth(handlePermissions))           // 权限列表

	// 文件管理 API 路由
	http.HandleFunc("/api/files/list", basicAuth(handleFileList))       // 浏览文件目录
	http.HandleFunc("/api/files/content", basicAuth(handleFileContent)) // 读取文件内容
	http.HandleFunc("/api/files/save", basicAuth(handleFileSave))       // 保存文件内容
	http.HandleFunc("/api/files/action", basicAuth(handleFileAction))   // 文件操作 (删除/重命名/创建目录)
	
	// 应用商店 API 路由
	http.HandleFunc("/api/appstore/templates", basicAuth(handleAppStoreTemplates)) // 获取应用模板列表
	http.HandleFunc("/api/appstore/instances", basicAuth(handleAppStoreInstances)) // 获取/创建应用实例
	
	// 数据库管理 API 路由
	http.HandleFunc("/api/databases/connections", basicAuth(handleDatabaseConnections)) // 数据库连接管理

	// 部署验证和修复 API 路由
	http.HandleFunc("/api/deployment/validate", basicAuth(handleDeploymentValidation))   // 部署验证
	http.HandleFunc("/api/deployment/repair", basicAuth(handleDeploymentRepair))       // 自动修复
	http.HandleFunc("/api/deployment/status", basicAuth(handleDeploymentStatus))       // 部署状态
	http.HandleFunc("/api/deployment/workflow", basicAuth(handleDeploymentWorkflow))   // 部署工作流
	http.HandleFunc("/api/health", basicAuth(handleHealthCheck))                       // 健康检查

	// WebSocket 实时通信接口
	http.HandleFunc("/ws/chat", basicAuth(handleWSChat)) // AI 聊天 WebSocket 连接

	// 前端静态资源服务配置
	// 注意：必须在所有 API 路由之后注册，确保 API 路由优先匹配
	
	// 调试模式：检查 Go embed 文件系统内容
	// 用于排查前端资源是否正确嵌入到二进制文件中
	logger.Info("🔍 检查 embed 文件系统内容:")
	if entries, err := fs.ReadDir(frontendDist, "."); err == nil {
		// 遍历并记录所有嵌入的文件和目录
		for _, entry := range entries {
			logger.Info("  - %s (目录: %v)", entry.Name(), entry.IsDir())
		}
	} else {
		// 如果读取失败，记录错误信息用于调试
		logger.Info("  读取 embed 文件系统失败: %v", err)
	}
	
	// 创建前端资源文件系统
	// 尝试从 embed FS 的 "dist" 子目录创建文件系统
	distFS, err := fs.Sub(frontendDist, "dist")
	if err != nil {
		// 降级处理：如果 dist 子目录不存在，直接使用根文件系统
		// 这种情况可能发生在 Docker 构建时前端文件结构异常
		logger.Info("⚠️ 无法创建 dist 子文件系统: %v，使用根文件系统", err)
		distFS = frontendDist
	}
	
	// 验证前端资源文件系统内容
	// 确保前端构建产物已正确嵌入
	logger.Info("🔍 检查前端资源文件系统:")
	var distEntries []fs.DirEntry
	if entries, err := fs.ReadDir(distFS, "."); err == nil {
		// 保存目录条目列表，用于后续判断前端资源是否为空
		distEntries = entries
		// 记录所有前端资源文件，便于调试前端 404 问题
		for _, entry := range entries {
			logger.Info("  - %s (目录: %v)", entry.Name(), entry.IsDir())
			
			// 特别处理 assets 目录：详细列出其内容
			// 这对于调试前端资源加载问题非常重要，可以确认关键的 JS/CSS 文件是否存在
			if entry.Name() == "assets" && entry.IsDir() {
				if assetEntries, err := fs.ReadDir(distFS, "assets"); err == nil {
					logger.Info("    Assets 目录内容 (%d 个文件):", len(assetEntries))
					// 显示所有 assets 文件，用于调试前端 404 问题
					// 包括 Vue 插件文件、CSS 样式文件、JS 模块文件等
					for _, asset := range assetEntries {
						logger.Info("      * %s", asset.Name())
					}
				}
			}
		}
	} else {
		// 读取前端文件系统失败，这通常表示前端构建有问题
		logger.Info("  读取前端文件系统失败: %v", err)
	}
	
	// 前端资源服务处理
	if len(distEntries) == 0 {
		// 前端资源为空的错误处理
		logger.Info("⚠️ 前端资源为空，可能是构建失败")
		http.HandleFunc("/", basicAuth(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "前端资源为空，请检查构建是否成功", http.StatusNotFound)
		}))
	} else {
		// 创建 SPA (单页应用) 处理器
		// 支持 Vue Router 的 HTML5 History 模式路由
		spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 获取请求路径，去除前导斜杠
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			
			// 特殊处理 favicon.ico - 如果不存在，返回 204 No Content，避免浏览器报错
			if path == "favicon.ico" {
				content, err := fs.ReadFile(distFS, path)
				if err != nil {
					// favicon.ico 不存在时返回 204，不记录错误日志
					w.WriteHeader(http.StatusNoContent)
					return
				}
				w.Header().Set("Content-Type", "image/x-icon")
				w.Write(content)
				return
			}
			
			// 检查是否是静态资源请求
			isStaticResource := strings.HasPrefix(path, "assets/") || 
			                   strings.HasSuffix(path, ".js") || 
			                   strings.HasSuffix(path, ".css") || 
			                   strings.HasSuffix(path, ".png") || 
			                   strings.HasSuffix(path, ".jpg") || 
			                   strings.HasSuffix(path, ".jpeg") || 
			                   strings.HasSuffix(path, ".svg") || 
			                   strings.HasSuffix(path, ".json") ||
			                   strings.HasSuffix(path, ".woff") ||
			                   strings.HasSuffix(path, ".woff2") ||
			                   strings.HasSuffix(path, ".ttf") ||
			                   strings.HasSuffix(path, ".map")
			
			// 尝试直接读取文件
			content, err := fs.ReadFile(distFS, path)
			if err != nil {
				if isStaticResource {
					// 静态资源不存在，记录日志并返回 404
					logger.Info("静态资源未找到: %s (错误: %v)", path, err)
					http.NotFound(w, r)
					return
				}
				// 非静态资源（页面路由），返回 index.html（SPA fallback）
				// 这样 Vue Router 可以处理所有前端路由
				path = "index.html"
				content, err = fs.ReadFile(distFS, path)
				if err != nil {
					http.Error(w, "index.html 读取失败: "+err.Error(), http.StatusInternalServerError)
					return
				}
			} else if !isStaticResource && !strings.HasSuffix(path, ".html") {
				// 如果文件存在但不是静态资源也不是 HTML，可能是目录或其他
				// 为了安全，也返回 index.html
				path = "index.html"
				content, err = fs.ReadFile(distFS, path)
				if err != nil {
					http.Error(w, "index.html 读取失败: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}
			
			// 根据文件扩展名设置正确的 Content-Type
			// 确保浏览器正确解析文件类型
			contentType := "text/html; charset=utf-8"
			if strings.HasSuffix(path, ".js") {
				contentType = "application/javascript; charset=utf-8"
			} else if strings.HasSuffix(path, ".css") {
				contentType = "text/css; charset=utf-8"
			} else if strings.HasSuffix(path, ".json") {
				contentType = "application/json; charset=utf-8"
			} else if strings.HasSuffix(path, ".png") {
				contentType = "image/png"
			} else if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") {
				contentType = "image/jpeg"
			} else if strings.HasSuffix(path, ".svg") {
				contentType = "image/svg+xml"
			} else if strings.HasSuffix(path, ".ico") {
				contentType = "image/x-icon"
			} else if strings.HasSuffix(path, ".woff") || strings.HasSuffix(path, ".woff2") {
				contentType = "font/woff2"
			} else if strings.HasSuffix(path, ".ttf") {
				contentType = "font/ttf"
			}
			
			// 设置响应头并返回文件内容
			w.Header().Set("Content-Type", contentType)
			w.Write(content)
		})
		
		// 注册根路径处理器，应用身份验证中间件
		http.HandleFunc("/", basicAuth(spaHandler))
	}

	// 获取实际端口号（去掉冒号）
	displayPort := strings.TrimPrefix(port, ":")
	logger.Info("🚀 qwq Dashboard started at http://localhost:%s", displayPort)
	if config.GlobalConfig.WebUser != "" {
		logger.Info("🔒 安全模式已开启 (Basic Auth)")
	}

	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Printf("Web Server Error: %v\n", err)
	}
}

// ============================================
// 容器管理 API
// ============================================

// handleContainers 获取 Docker 容器列表
func handleContainers(w http.ResponseWriter, r *http.Request) {
	cmd := `docker ps -a --format "{{.ID}}|{{.Image}}|{{.Status}}|{{.Names}}"`
	output := utils.ExecuteShell(cmd)
	
	var containers []DockerContainer
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" { continue }
		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			state := "exited"
			if strings.Contains(parts[2], "Up") {
				state = "running"
			}
			containers = append(containers, DockerContainer{
				ID:     parts[0],
				Image:  parts[1],
				Status: parts[2],
				Name:   parts[3],
				State:  state,
			})
		}
	}
	json.NewEncoder(w).Encode(containers)
}

// ============================================
// 监控数据采集
// ============================================

// collectStatsLoop 定时采集系统监控数据
func collectStatsLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		point := collectOnePoint()
		statsCache.Lock()
		statsCache.History = append(statsCache.History, point)
		if len(statsCache.History) > 60 { statsCache.History = statsCache.History[1:] }
		statsCache.Unlock()
	}
}

// collectOnePoint 采集一次系统监控数据
// 包括：系统负载、内存使用、磁盘使用、TCP 连接数、服务状态
func collectOnePoint() StatsPoint {
	// 获取系统负载（1分钟、5分钟、15分钟平均值）
	load := strings.TrimSpace(utils.ExecuteShell("uptime | awk -F'load average:' '{ print $2 }'"))
	
	// 获取内存使用情况（单位：MB）
	memRaw := utils.ExecuteShell("free -m | awk 'NR==2{print $2,$3}'")
	var memTotal, memUsed float64
	fmt.Sscanf(memRaw, "%f %f", &memTotal, &memUsed)
	memPct := 0.0
	if memTotal > 0 { 
		memPct = (memUsed / memTotal) * 100 
	}
	
	// 获取根目录磁盘使用情况（仪表盘只显示根目录）
	diskRaw := utils.ExecuteShell("df -h / | awk 'NR==2 {print $5,$4}'")
	diskParts := strings.Fields(diskRaw)
	diskPct := "0"
	diskAvail := "0G"
	if len(diskParts) >= 2 {
		diskPct = strings.TrimSuffix(diskParts[0], "%")
		diskAvail = diskParts[1]
	}
	
	// 获取 TCP 连接数（已建立的连接）
	tcpRaw := utils.ExecuteShell("ss -s | grep 'TCP:' | grep -oE 'estab [0-9]+' | awk '{print $2}'")
	tcpConn := strings.TrimSpace(tcpRaw)
	if tcpConn == "" { 
		tcpConn = "0" 
	}
	
	// 执行 HTTP 服务健康检查
	httpStatus := monitor.RunChecks()
	
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

// handleWSChat 处理 WebSocket 聊天连接
// 支持三种处理模式：
// 1. 静态响应 - 快速回答常见问题
// 2. 快速命令 - 直接执行预定义命令
// 3. AI 对话 - 调用 AI 进行智能分析
func handleWSChat(w http.ResponseWriter, r *http.Request) {
	// 升级 HTTP 连接为 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil { 
		logger.Info("WS Upgrade Error: %v", err)
		return 
	}
	defer conn.Close()
	
	// 初始化对话上下文
	messages := agent.GetBaseMessages()
	
	// 持续监听客户端消息
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil { break }
		
		input := string(msg)
		
		// 1. 尝试静态响应（最快）
		staticResp := agent.CheckStaticResponse(input)
		if staticResp != "" {
			conn.WriteJSON(map[string]string{"type": "answer", "content": staticResp})
			conn.WriteJSON(map[string]string{"type": "status", "content": "等待指令..."})
			continue
		}
		
		// 2. 尝试快速命令执行
		quickCmd := agent.GetQuickCommand(input)
		if quickCmd != "" {
			conn.WriteJSON(map[string]string{"type": "status", "content": "⚡ 快速执行: " + quickCmd})
			output := utils.ExecuteShell(quickCmd)
			if strings.TrimSpace(output) == "" { output = "(No output)" }
			finalOutput := fmt.Sprintf("```\n%s\n```", output)
			conn.WriteJSON(map[string]string{"type": "answer", "content": finalOutput})
			conn.WriteJSON(map[string]string{"type": "status", "content": "等待指令..."})
			continue
		}
		
		// 3. AI 智能对话（最慢但最强大）
		enhancedInput := input + " (Context: Current Linux Server)"
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: enhancedInput})
		
		// 最多执行 5 轮对话（防止无限循环）
		for i := 0; i < 5; i++ {
			conn.WriteJSON(map[string]string{"type": "status", "content": "🤖 思考中..."})
			
			// 处理 AI 响应，实时推送日志
			respMsg, cont := agent.ProcessAgentStepForWeb(&messages, func(log string) {
				conn.WriteJSON(map[string]string{"type": "log", "content": log})
			})
			
			if respMsg.Content != "" {
				conn.WriteJSON(map[string]string{"type": "answer", "content": respMsg.Content})
			}
			
			// 如果 AI 表示完成，退出循环
			if !cont { break }
		}
		
		conn.WriteJSON(map[string]string{"type": "status", "content": "等待指令..."})
	}
}

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
	logger.Info(msg)
}

// ============================================
// 文件管理 API 处理器（在 files.go 中实现）
// ============================================

// ============================================
// 系统巡检功能
// ============================================

// performPatrol 执行系统巡检
// 检查项目：磁盘使用、系统负载、OOM 日志、僵尸进程、自定义规则、HTTP 服务
// 发现异常时调用 AI 分析并推送告警
func performPatrol() {
	logger.Info("正在执行系统巡检...")
	var anomalies []string

	// 1. 磁盘使用率检查（过滤虚拟设备）
	diskOut := utils.ExecuteShell("df -h")
	diskLines := strings.Split(diskOut, "\n")

	for _, line := range diskLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Filesystem") {
			continue
		}

		// 解析字段
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		
		device := fields[0]
		mountPoint := fields[len(fields)-1]
		
		// 严格过滤：检查设备名和挂载点
		if isIgnoredDisk(line, device, mountPoint) {
			continue
		}

		// 解析使用率
		useStr := strings.TrimSuffix(fields[4], "%")
		usePct, err := strconv.Atoi(useStr)
		if err == nil && usePct > 85 {
			// 只有非 loop 设备且使用率 > 85% 才报警
			anomalies = append(anomalies, fmt.Sprintf("**磁盘告警 (%s)**:\n```\n%s\n```", fields[0], line))
		}
	}

	// 2. 系统负载检查（1分钟负载 > 4.0 时告警）
	if out := utils.ExecuteShell("uptime | awk -F'load average:' '{ print $2 }' | awk '{ if ($1 > 4.0) print $0 }'"); strings.TrimSpace(out) != "" && !strings.Contains(out, "exit status") {
		anomalies = append(anomalies, "**高负载**:\n```\n"+strings.TrimSpace(out)+"\n```")
	}

	// 3. OOM（内存溢出）日志检查
	dmesgOut := utils.ExecuteShell("dmesg | grep -i 'out of memory' | tail -n 5")
	if !strings.Contains(dmesgOut, "Operation not permitted") && !strings.Contains(dmesgOut, "不允许的操作") && strings.TrimSpace(dmesgOut) != "" && !strings.Contains(dmesgOut, "exit status") {
		anomalies = append(anomalies, "**OOM日志**:\n```\n"+strings.TrimSpace(dmesgOut)+"\n```")
	}

	// 4. 僵尸进程检查（状态为 Z 的进程）
	rawZombies := utils.ExecuteShell("ps -A -o stat,ppid,pid,cmd | awk '$1 ~ /^[Zz]/'")
	if strings.TrimSpace(rawZombies) != "" && !strings.Contains(rawZombies, "exit status") {
		detailZombie := "STAT    PPID     PID CMD\n" + rawZombies
		anomalies = append(anomalies, "**僵尸进程**:\n```\n"+strings.TrimSpace(detailZombie)+"\n```")
	}

	// 5. 自定义巡检规则（从配置文件读取）
	for _, rule := range config.GlobalConfig.PatrolRules {
		out := utils.ExecuteShell(rule.Command)
		if strings.TrimSpace(out) != "" && !strings.Contains(out, "exit status") {
			logger.Info(fmt.Sprintf("⚠️ 触发自定义规则: %s", rule.Name))
			anomalies = append(anomalies, fmt.Sprintf("**%s**:\n```\n%s\n```", rule.Name, strings.TrimSpace(out)))
		}
	}

	// 6. HTTP 服务健康检查
	httpResults := monitor.RunChecks()
	for _, res := range httpResults {
		if !res.Success {
			logger.Info(fmt.Sprintf("⚠️ HTTP 监控失败: %s", res.Name))
			anomalies = append(anomalies, fmt.Sprintf("**HTTP异常 (%s)**:\n%s", res.Name, res.Error))
		}
	}

	// 过滤掉虚拟设备相关的告警（避免误报）
	var cleanedAnomalies []string
	for _, anomaly := range anomalies {
		if !strings.Contains(anomaly, "/dev/loop") && 
		   !strings.Contains(anomaly, "/snap") && 
		   !strings.Contains(anomaly, "snap/") &&
		   !strings.Contains(anomaly, "/hostfs") &&
		   !strings.Contains(anomaly, "overlay") &&
		   !strings.Contains(anomaly, "tmpfs") {
			cleanedAnomalies = append(cleanedAnomalies, anomaly)
		}
	}

	// 如果发现异常，调用 AI 分析并推送告警
	if len(cleanedAnomalies) > 0 {
		report := strings.Join(cleanedAnomalies, "\n")
		logger.Info("🚨 发现异常，正在请求 AI 分析...")
		
		// 调用 AI 分析异常原因和解决方案
		analysis := agent.AnalyzeWithAI(report)
		analysis = cleanAIAnalysis(analysis)

		// 组装告警消息并推送
		alertMsg := fmt.Sprintf("🚨 **系统告警** [%s]\n\n%s\n\n💡 **处理建议**:\n%s", utils.GetHostname(), report, analysis)
		notify.Send("系统告警", alertMsg)
		logger.Info("告警已推送")
	} else {
		logger.Info("✔ 系统健康")
	}
}

// isIgnoredDisk 判断是否应该忽略该磁盘设备
// 过滤虚拟设备和临时文件系统，避免误报
func isIgnoredDisk(line, device, mountPoint string) bool {
	// 检查设备名：过滤所有 loop 设备（虚拟块设备）
	if strings.Contains(device, "/dev/loop") || strings.Contains(device, "loop") {
		return true
	}
	
	// 检查挂载点：过滤 snap 相关路径（Ubuntu snap 包）
	if strings.Contains(mountPoint, "/snap") || 
	   strings.Contains(mountPoint, "snap/") ||
	   strings.Contains(mountPoint, "/hostfs") {
		return true
	}
	
	// 检查整行：过滤虚拟文件系统
	if strings.Contains(line, "tmpfs") ||      // 临时文件系统
	   strings.Contains(line, "overlay") ||    // Docker overlay 文件系统
	   strings.Contains(line, "cdrom") ||      // 光驱
	   strings.Contains(line, "efivarfs") {    // EFI 变量文件系统
		return true
	}
	
	return false
}

// cleanAIAnalysis 清理 AI 分析结果
// 标记已过滤的虚拟设备，避免用户混淆
func cleanAIAnalysis(analysis string) string {
	analysis = strings.Replace(analysis, "/dev/loop", "[排除] /dev/loop", -1)
	analysis = strings.Replace(analysis, "/snap", "[排除] /snap", -1)
	analysis = strings.Replace(analysis, "overlay", "[排除] overlay", -1)
	return analysis
}

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
		if websitesStore.Websites == nil {
			websitesStore.Websites = []Website{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(websitesStore.Websites)
		
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
	
	switch action {
	case "apply":
		// 申请SSL证书（模拟）
		websitesStore.Websites[index].SSLEnabled = true
		// 设置证书有效期（模拟：1年后过期）
		expiry := time.Now().AddDate(1, 0, 0)
		websitesStore.Websites[index].SSLCertExpiry = expiry.Format(time.RFC3339)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "SSL证书申请成功"})
		
	case "renew":
		// 续期SSL证书（模拟）
		if !websitesStore.Websites[index].SSLEnabled {
			http.Error(w, "SSL is not enabled for this website", http.StatusBadRequest)
			return
		}
		// 更新证书有效期（模拟：1年后过期）
		expiry := time.Now().AddDate(1, 0, 0)
		websitesStore.Websites[index].SSLCertExpiry = expiry.Format(time.RFC3339)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "SSL证书续期成功"})
		
	default:
		http.Error(w, "Invalid SSL action", http.StatusBadRequest)
	}
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
	switch r.Method {
	case http.MethodGet:
		// 获取用户权限列表
		usersStore.RLock()
		defer usersStore.RUnlock()
		
		// 查找用户
		var user *User
		for i := range usersStore.Users {
			if usersStore.Users[i].ID == id {
				user = &usersStore.Users[i]
				break
			}
		}
		
		if user == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		
		// 根据用户的角色，返回所有权限
		userPermissions := []map[string]interface{}{}
		for _, perm := range permissionsStore {
			// 简化处理：如果用户有角色，就返回所有权限
			// 实际应用中应该根据角色关联的权限来返回
			userPermissions = append(userPermissions, map[string]interface{}{
				"resource":    perm.Resource,
				"action":      perm.Action,
				"description": perm.Description,
			})
		}
		json.NewEncoder(w).Encode(userPermissions)
		
	case http.MethodPut:
		// 更新用户权限
		var form struct {
			Permissions []string `json:"permissions"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		usersStore.Lock()
		defer usersStore.Unlock()
		
		// 查找用户
		index := -1
		for i, u := range usersStore.Users {
			if u.ID == id {
				index = i
				break
			}
		}
		
		if index == -1 {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		
		// 这里简化处理，实际应该更新用户的权限关联
		// 可以将权限转换为角色，或者单独存储用户权限
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
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

	logger.Info("🔧 开始自动修复...")
	
	// 运行自动修复
	result := deploymentService.RunAutomaticRepair()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
	
	if result.Success {
		logger.Info("✅ 自动修复完成，修复了 %d 个问题", len(result.FixedIssues))
	} else {
		logger.Info("⚠️ 自动修复部分完成，剩余 %d 个问题", len(result.RemainingIssues))
	}
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
