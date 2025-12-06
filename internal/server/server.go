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
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	openai "github.com/sashabaranov/go-openai"
)

//go:embed dist
var frontendDist embed.FS

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	TriggerPatrolFunc func()
	TriggerStatusFunc func()
	logFile           *os.File
	
	statsCache struct {
		sync.RWMutex
		History []StatsPoint
	}
)

// StatsPoint 系统状态数据点
type StatsPoint struct {
	Time      string      `json:"time"`
	Load      string      `json:"load"`
	MemPct    string      `json:"mem_pct"`
	MemUsed   string      `json:"mem_used"`
	MemTotal  string      `json:"mem_total"`
	DiskPct   string      `json:"disk_pct"`
	DiskAvail string      `json:"disk_avail"`
	TcpConn   string      `json:"tcp_conn"`
	Services  interface{} `json:"services"`
}

// DockerContainer 容器信息结构体
type DockerContainer struct {
	ID      string `json:"id"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	Name    string `json:"name"`
	State   string `json:"state"` // running, exited
}

func Start(port string) {
	var err error
	// 日志持久化
	logFile, err = os.OpenFile("qwq.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("无法创建日志文件: %v\n", err)
	}

	// 启动后台采集循环
	go collectStatsLoop()

	// 前端静态资源服务
	distFS, err := fs.Sub(frontendDist, "dist")
	if err != nil {
		logger.Info("⚠️ 前端资源加载异常: %v", err)
	} else {
		fileServer := http.FileServer(http.FS(distFS))
		
		// 静态资源直接放行
		http.Handle("/assets/", fileServer)
		
		// 首页及其他路径走鉴权
		http.HandleFunc("/", basicAuth(func(w http.ResponseWriter, r *http.Request) {
			// API 和 WS 请求跳过文件服务
			if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
				return 
			}
			fileServer.ServeHTTP(w, r)
		}))
	}

	// 注册 API 路由
	http.HandleFunc("/api/logs", basicAuth(handleLogs))
	http.HandleFunc("/api/stats", basicAuth(handleStats))
	http.HandleFunc("/api/trigger", basicAuth(handleTrigger))
	
	// 容器管理 API
	http.HandleFunc("/api/containers", basicAuth(handleContainers))
	http.HandleFunc("/api/container/action", basicAuth(handleContainerAction))
	
	// WebSocket 聊天路由
	http.HandleFunc("/ws/chat", basicAuth(handleWSChat))

	logger.Info("🚀 qwq Dashboard started at http://localhost" + port)
	if config.GlobalConfig.WebUser != "" {
		logger.Info("🔒 安全模式已开启 (Basic Auth)")
	}

	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Printf("Web Server Error: %v\n", err)
	}
}

// --- 容器管理处理函数 ---

func handleContainers(w http.ResponseWriter, r *http.Request) {
	// 使用自定义格式获取：ID|Image|Status|Names
	cmd := `docker ps -a --format "{{.ID}}|{{.Image}}|{{.Status}}|{{.Names}}"`
	output := utils.ExecuteShell(cmd)
	
	var containers []DockerContainer
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" { continue }
		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			state := "exited"
			// 简单的状态判断逻辑
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

func handleContainerAction(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	action := r.URL.Query().Get("action") // start, stop, restart
	
	if id == "" || action == "" {
		http.Error(w, "Missing params", 400)
		return
	}

	// 安全检查：只允许特定命令
	if action != "start" && action != "stop" && action != "restart" {
		http.Error(w, "Invalid action", 400)
		return
	}

	cmd := fmt.Sprintf("docker %s %s", action, id)
	logger.Info("Web操作容器: %s", cmd)
	utils.ExecuteShell(cmd)
	
	w.Write([]byte("success"))
}

// --- 统计采集逻辑 ---

func collectStatsLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		point := collectOnePoint()
		statsCache.Lock()
		statsCache.History = append(statsCache.History, point)
		if len(statsCache.History) > 60 {
			statsCache.History = statsCache.History[1:]
		}
		statsCache.Unlock()
	}
}

func collectOnePoint() StatsPoint {
	load := strings.TrimSpace(utils.ExecuteShell("uptime | awk -F'load average:' '{ print $2 }'"))
	memRaw := utils.ExecuteShell("free -m | awk 'NR==2{print $2,$3}'")
	var memTotal, memUsed float64
	fmt.Sscanf(memRaw, "%f %f", &memTotal, &memUsed)
	memPct := 0.0
	if memTotal > 0 { memPct = (memUsed / memTotal) * 100 }
	
	diskRaw := utils.ExecuteShell("df -h / | awk 'NR==2 {print $5,$4}'")
	diskParts := strings.Fields(diskRaw)
	diskPct := "0"
	diskAvail := "0G"
	if len(diskParts) >= 2 {
		diskPct = strings.TrimSuffix(diskParts[0], "%")
		diskAvail = diskParts[1]
	}

	tcpRaw := utils.ExecuteShell("ss -s | grep 'TCP:' | grep -oE 'estab [0-9]+' | awk '{print $2}'")
	tcpConn := strings.TrimSpace(tcpRaw)
	if tcpConn == "" { tcpConn = "0" }

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

// --- 基础认证 ---

func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userCfg := config.GlobalConfig.WebUser
		passCfg := config.GlobalConfig.WebPassword
		if userCfg == "" || passCfg == "" {
			next(w, r)
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(userCfg)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(passCfg)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// --- WebSocket 聊天处理 ---

func handleWSChat(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Info("WS Upgrade Error: %v", err)
		return
	}
	defer conn.Close()

	messages := agent.GetBaseMessages()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil { break }
		input := string(msg)
		
		// 1. 静态规则拦截 (你是谁/版本)
		staticResp := agent.CheckStaticResponse(input)
		if staticResp != "" {
			conn.WriteJSON(map[string]string{"type": "answer", "content": staticResp})
			conn.WriteJSON(map[string]string{"type": "status", "content": "等待指令..."})
			continue
		}

		// 2. 关键词速查 (看看内存/Docker)
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

		// 3. AI 处理 (注入上下文)
		enhancedInput := input + " (Context: Current Linux Server)"
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: enhancedInput})

		for i := 0; i < 5; i++ {
			conn.WriteJSON(map[string]string{"type": "status", "content": "🤖 思考中..."})
			
			respMsg, cont := agent.ProcessAgentStepForWeb(&messages, func(log string) {
				conn.WriteJSON(map[string]string{"type": "log", "content": log})
			})
			
			if respMsg.Content != "" {
				conn.WriteJSON(map[string]string{"type": "answer", "content": respMsg.Content})
			}
			
			if !cont { break }
		}
		conn.WriteJSON(map[string]string{"type": "status", "content": "等待指令..."})
	}
}

// --- 其他 API Handlers ---

func handleLogs(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(logger.GetWebLogs())
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	statsCache.RLock()
	defer statsCache.RUnlock()
	if len(statsCache.History) == 0 {
		json.NewEncoder(w).Encode([]StatsPoint{})
		return
	}
	json.NewEncoder(w).Encode(statsCache.History)
}

func handleTrigger(w http.ResponseWriter, r *http.Request) {
	if TriggerPatrolFunc != nil { go TriggerPatrolFunc() }
	if TriggerStatusFunc != nil { go TriggerStatusFunc() }
	w.Write([]byte("指令已发送：正在后台执行巡检和汇报..."))
}

func WebLog(msg string) {
	logger.Info(msg)
}