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

	"github.com/gorilla/websocket"
	openai "github.com/sashabaranov/go-openai"
)

//go:embed static/index.html
var content embed.FS

var (
	LogBuffer         []string
	LogMutex          sync.Mutex
	TriggerPatrolFunc func()
	TriggerStatusFunc func()
	logFile           *os.File

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func Start(port string) {
	var err error
	logFile, err = os.OpenFile("qwq.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("无法创建日志文件: %v\n", err)
	}

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

func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userCfg := config.GlobalConfig.WebUser
		passCfg := config.GlobalConfig.WebPassword

		// WebSocket 认证特殊处理 (浏览器 JS 无法直接带 Auth 头，这里简化处理，或者通过 URL Token，这里暂复用 Basic Auth)
		// 注意：部分浏览器 WebSocket 不支持 Basic Auth 弹窗，生产环境建议用 Cookie 或 Token
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

// 处理 WebSocket 聊天
func handleWSChat(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Info("WS Upgrade Error: %v", err)
		return
	}
	defer conn.Close()

	// 初始化对话上下文
	knowledgePart := ""
	if config.CachedKnowledge != "" {
		knowledgePart = fmt.Sprintf("\n【内部知识库】:\n%s\n", config.CachedKnowledge)
	}
	sysPrompt := fmt.Sprintf(`你是一个资深运维专家助手(qwq)。
规则：
1. 请用中文回答。
2. **分步执行**：先获取信息，再执行下一步。
3. **Web模式**：你现在运行在 Web 终端中，用户可以直接看到你的回复。
4. 如果是查询类命令（如 get, describe, logs, top, ps），请放心执行。
%s`, knowledgePart)

	messages := []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: sysPrompt}}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		input := string(msg)
		

		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: input})

		for i := 0; i < 5; i++ {
			conn.WriteJSON(map[string]string{"type": "status", "content": "🤖 思考中..."})
			
			respMsg, cont := agent.ProcessAgentStepForWeb(&messages, func(log string) {
				conn.WriteJSON(map[string]string{"type": "log", "content": log})
			})
			
			if !cont { break }
			
			if respMsg.Content != "" && len(respMsg.ToolCalls) == 0 {
				// 最终回复
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
	LogMutex.Lock()
	defer LogMutex.Unlock()
	json.NewEncoder(w).Encode(LogBuffer)
}

func handleStats(w http.ResponseWriter, r *http.Request) {
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

	httpStatus := monitor.RunChecks()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"load":       load,
		"mem_pct":    fmt.Sprintf("%.1f", memPct),
		"mem_used":   fmt.Sprintf("%.0f", memUsed),
		"mem_total":  fmt.Sprintf("%.0f", memTotal),
		"disk_pct":   diskPct,
		"disk_avail": diskAvail,
		"services":   httpStatus,
		"time":       time.Now().Format("15:04:05"),
	})
}

func handleTrigger(w http.ResponseWriter, r *http.Request) {
	if TriggerPatrolFunc != nil { go TriggerPatrolFunc() }
	if TriggerStatusFunc != nil { go TriggerStatusFunc() }
	w.Write([]byte("指令已发送：正在后台执行巡检和汇报..."))
}

func WebLog(msg string) {
	logger.Info(msg)
}