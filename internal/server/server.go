package server

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"qwq/internal/agent"
	"qwq/internal/config"
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

//go:embed dist
var frontendDist embed.FS

var (
	// WebSocket 升级器，允许所有来源的连接
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	// 触发巡检的回调函数
	TriggerPatrolFunc func()
	// 触发状态推送的回调函数
	TriggerStatusFunc func()
	// 日志文件句柄
	logFile           *os.File
	
	// 统计数据缓存，用于存储历史监控数据
	statsCache struct {
		sync.RWMutex
		History []StatsPoint
	}
)

// StatsPoint 系统监控数据点
type StatsPoint struct {
	Time      string      `json:"time"`       // 采集时间
	Load      string      `json:"load"`       // 系统负载
	MemPct    string      `json:"mem_pct"`    // 内存使用百分比
	MemUsed   string      `json:"mem_used"`   // 已使用内存(MB)
	MemTotal  string      `json:"mem_total"`  // 总内存(MB)
	DiskPct   string      `json:"disk_pct"`   // 磁盘使用百分比
	DiskAvail string      `json:"disk_avail"` // 可用磁盘空间
	TcpConn   string      `json:"tcp_conn"`   // TCP 连接数
	Services  interface{} `json:"services"`   // 服务状态
}

// DockerContainer Docker 容器信息
type DockerContainer struct {
	ID      string `json:"id"`     // 容器 ID
	Image   string `json:"image"`  // 镜像名称
	Status  string `json:"status"` // 状态描述
	Name    string `json:"name"`   // 容器名称
	State   string `json:"state"`  // 运行状态(running/exited)
}

// Start 启动 Web 服务器
func Start(port string) {
	// 打开日志文件
	var err error
	logFile, err = os.OpenFile("qwq.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("无法创建日志文件: %v\n", err)
	}

	// 启动后台监控数据采集
	go collectStatsLoop()

	// 注册 API 路由
	http.HandleFunc("/api/logs", basicAuth(handleLogs))                     // 日志查询
	http.HandleFunc("/api/stats", basicAuth(handleStats))                   // 监控数据
	http.HandleFunc("/api/trigger", basicAuth(handleTrigger))               // 触发巡检
	http.HandleFunc("/api/containers", basicAuth(handleContainers))         // 容器列表
	http.HandleFunc("/api/container/action", basicAuth(handleContainerAction)) // 容器操作

	// 文件管理 API
	http.HandleFunc("/api/files/list", basicAuth(handleFileList))       // 文件列表
	http.HandleFunc("/api/files/content", basicAuth(handleFileContent)) // 文件内容
	http.HandleFunc("/api/files/save", basicAuth(handleFileSave))       // 保存文件
	http.HandleFunc("/api/files/action", basicAuth(handleFileAction))   // 文件操作

	// WebSocket 聊天接口
	http.HandleFunc("/ws/chat", basicAuth(handleWSChat))

	// 加载前端静态资源（Vue 3 SPA 构建产物）
	// 注意：必须在所有 API 路由之后注册，确保 API 路由优先匹配
	distFS, err := fs.Sub(frontendDist, "dist")
	if err != nil {
		// 前端资源加载失败，返回错误提示页面
		logger.Info("⚠️ 前端资源加载异常: %v", err)
		http.HandleFunc("/", basicAuth(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "前端资源未找到，请检查构建是否成功", http.StatusNotFound)
		}))
	} else {
		// 创建 SPA 单页应用处理器
		// 支持 Vue Router 的 HTML5 History 模式
		spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 获取请求路径，去除前导斜杠
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			
			// 检查文件是否存在
			_, err := distFS.Open(path)
			if err != nil {
				// 文件不存在时返回 index.html，支持前端路由
				// 这样 /dashboard、/containers 等路由都会返回 index.html
				// 由 Vue Router 在客户端处理路由
				path = "index.html"
			}
			
			// 读取文件内容
			content, err := fs.ReadFile(distFS, path)
			if err != nil {
				http.Error(w, "文件读取失败: "+err.Error(), http.StatusInternalServerError)
				return
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

// handleContainerAction 执行容器操作（启动/停止/重启）
func handleContainerAction(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	action := r.URL.Query().Get("action")
	
	// 参数验证
	if id == "" || action == "" { 
		http.Error(w, "Missing params", 400)
		return 
	}
	if action != "start" && action != "stop" && action != "restart" { 
		http.Error(w, "Invalid action", 400)
		return 
	}
	
	// 执行 Docker 命令
	cmd := fmt.Sprintf("docker %s %s", action, id)
	logger.Info("Web操作容器: %s", cmd)
	utils.ExecuteShell(cmd)
	w.Write([]byte("success"))
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
// 认证中间件
// ============================================

// basicAuth HTTP 基础认证中间件
// 如果配置了用户名和密码，则要求客户端提供认证信息
// 使用 constant time 比较防止时序攻击
func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userCfg := config.GlobalConfig.WebUser
		passCfg := config.GlobalConfig.WebPassword
		
		// 未配置认证，直接放行
		if userCfg == "" || passCfg == "" {
			next(w, r)
			return
		}
		
		// 验证认证信息
		user, pass, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(userCfg)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(passCfg)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
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
