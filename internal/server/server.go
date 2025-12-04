package server

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
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
    "strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/gorilla/websocket"
	openai "github.com/sashabaranov/go-openai"
)

//go:embed static/index.html
var content embed.FS

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	TriggerPatrolFunc func()
	TriggerStatusFunc func()
	logFile           *os.File
	
	// 统计数据缓存与历史记录
	statsCache struct {
		sync.RWMutex
		History []StatsPoint
	}
)

// StatsPoint 单个时间点的数据
type StatsPoint struct {
	Time      string      `json:"time"`
	Load      string      `json:"load"`
	MemPct    string      `json:"mem_pct"`
	MemUsed   string      `json:"mem_used"`
	MemTotal  string      `json:"mem_total"`
	DiskPct   string      `json:"disk_pct"`
	DiskAvail string      `json:"disk_avail"`
	Services  interface{} `json:"services"`
}

func Start(port string) {
	var err error
	logFile, err = os.OpenFile("qwq.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("无法创建日志文件: %v\n", err)
	}

	http.Handle("/metrics", promhttp.Handler())

	// 启动后台采集协程
	go collectStatsLoop()

	http.HandleFunc("/", basicAuth(handleIndex))
	http.HandleFunc("/api/logs", basicAuth(handleLogs))
	http.HandleFunc("/api/stats", basicAuth(handleStats))
	http.HandleFunc("/api/trigger", basicAuth(handleTrigger))
	http.HandleFunc("/ws/chat", basicAuth(handleWSChat))

	logger.Info("🚀 qwq Dashboard started at http://localhost" + port)
	if config.GlobalConfig.WebUser != "" {
		logger.Info("🔒 安全模式已开启 (Basic Auth)")
	}

	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Printf("Web Server Error: %v\n", err)
	}
}

// 后台采集循环：每2秒采集一次，存入历史记录
func collectStatsLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		point := collectOnePoint()
		
		statsCache.Lock()
		// 保留最近 60 个点 (约2分钟的高频数据，或者你可以改成更长)
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

	// 注意：HTTP 检查比较耗时，这里每2秒跑一次可能太频繁
	// 生产环境建议把 HTTP 检查单独开一个低频 Ticker
	httpStatus := monitor.RunChecks()

	loadFloat, _ := strconv.ParseFloat(load, 64)
    diskPctFloat, _ := strconv.ParseFloat(diskPct, 64)
    tcpStr := strings.TrimSpace(utils.ExecuteShell("netstat -ant | grep ESTABLISHED | wc -l"))
    tcpFloat, _ := strconv.ParseFloat(tcpStr, 64)

    monitor.UpdatePrometheusMetrics(loadFloat, memPct, diskPctFloat, tcpFloat)
    
    monitor.UpdateAppMetrics(httpStatus)

	return StatsPoint{
		Time:      time.Now().Format("15:04:05"),
		Load:      load,
		MemPct:    fmt.Sprintf("%.1f", memPct),
		MemUsed:   fmt.Sprintf("%.0f", memUsed),
		MemTotal:  fmt.Sprintf("%.0f", memTotal),
		DiskPct:   diskPct,
		DiskAvail: diskAvail,
		Services:  httpStatus,
	}
}

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

func handleWSChat(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Info("WS Upgrade Error: %v", err)
		return
	}
	defer conn.Close()

	knowledgePart := ""
	if config.CachedKnowledge != "" {
		knowledgePart = fmt.Sprintf("\n【内部知识库】:\n%s\n", config.CachedKnowledge)
	}
	sysPrompt := fmt.Sprintf(`你是一个资深运维专家助手(qwq)。
规则：
1. 请用中文回答。
2. **分步执行**：先获取信息，再执行下一步。
3. **Web模式**：你现在运行在 Web 终端中。
4. 如果是查询类命令（如 get, describe, logs, top, ps），请放心执行。
%s`, knowledgePart)

	messages := []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: sysPrompt}}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil { break }
		input := string(msg)
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: input})

		for i := 0; i < 5; i++ {
			conn.WriteJSON(map[string]string{"type": "status", "content": "🤖 思考中..."})
			respMsg, cont := agent.ProcessAgentStepForWeb(&messages, func(log string) {
				conn.WriteJSON(map[string]string{"type": "log", "content": log})
			})
			if !cont { break }
			if respMsg.Content != "" && len(respMsg.ToolCalls) == 0 {
				conn.WriteJSON(map[string]string{"type": "answer", "content": respMsg.Content})
				break
			}
		}
		conn.WriteJSON(map[string]string{"type": "status", "content": "等待指令..."})
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data, _ := content.ReadFile("static/index.html")
	w.Write(data)
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(logger.GetWebLogs())
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	statsCache.RLock()
	defer statsCache.RUnlock()
	
	// 返回整个历史记录，前端可以一次性渲染出曲线
	// 如果历史记录为空（刚启动），返回空数组
	if len(statsCache.History) == 0 {
		json.NewEncoder(w).Encode([]StatsPoint{})
		return
	}
	
	// 返回最近的数据
	json.NewEncoder(w).Encode(statsCache.History)
}

func handleTrigger(w http.ResponseWriter, r *http.Request) {
	if TriggerPatrolFunc != nil { go TriggerPatrolFunc() }
	if TriggerStatusFunc != nil { go TriggerStatusFunc() }
	w.Write([]byte("指令已发送：正在后台执行巡检和汇报..."))
}