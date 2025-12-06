package server

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"qwq/internal/agent"
	"qwq/internal/config"
	"qwq/internal/logger"
	"qwq/internal/monitor"
	"qwq/internal/utils"
	"qwq/internal/notify"
	"sort"
	"strconv"

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

type DockerContainer struct {
	ID      string `json:"id"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	Name    string `json:"name"`
	State   string `json:"state"`
}

// --- 巡检逻辑 ---
func performPatrol() {
	logger.Info("正在执行系统巡检...")
	var anomalies []string

	// 1. 磁盘检查
	diskOut := utils.ExecuteShell("df -h")
	diskLines := strings.Split(diskOut, "\n")
	
	for _, line := range diskLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Filesystem") {
			continue
		}

		// 过滤
		if strings.Contains(line, "/dev/loop") || 
		   strings.Contains(line, "/snap") || 
		   strings.Contains(line, "tmpfs") || 
		   strings.Contains(line, "overlay") || 
		   strings.Contains(line, "cdrom") {
			continue
		}

		// 解析使用率 (df -h 输出的第5列通常是 Use%)
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			useStr := strings.TrimSuffix(fields[4], "%")
			usePct, err := strconv.Atoi(useStr)
			if err == nil && usePct > 85 {
				// 只有非 loop 设备且使用率 > 85% 才报警
				anomalies = append(anomalies, fmt.Sprintf("**磁盘告警 (%s)**:\n```\n%s\n```", fields[0], line))
			}
		}
	}

	// 2. 负载
	if out := utils.ExecuteShell("uptime | awk -F'load average:' '{ print $2 }' | awk '{ if ($1 > 4.0) print $0 }'"); strings.TrimSpace(out) != "" && !strings.Contains(out, "exit status") {
		anomalies = append(anomalies, "**高负载**:\n```\n"+strings.TrimSpace(out)+"\n```")
	}
	
	// 3. OOM
	dmesgOut := utils.ExecuteShell("dmesg | grep -i 'out of memory' | tail -n 5")
	if !strings.Contains(dmesgOut, "Operation not permitted") && !strings.Contains(dmesgOut, "不允许的操作") && strings.TrimSpace(dmesgOut) != "" && !strings.Contains(dmesgOut, "exit status") {
		anomalies = append(anomalies, "**OOM日志**:\n```\n"+strings.TrimSpace(dmesgOut)+"\n```")
	}
	
	// 4. 僵尸进程
	rawZombies := utils.ExecuteShell("ps -A -o stat,ppid,pid,cmd | awk '$1 ~ /^[Zz]/'")
	if strings.TrimSpace(rawZombies) != "" && !strings.Contains(rawZombies, "exit status") {
		detailZombie := "STAT    PPID     PID CMD\n" + rawZombies
		anomalies = append(anomalies, "**僵尸进程**:\n```\n"+strings.TrimSpace(detailZombie)+"\n```")
	}

	// 5. 自定义规则
	for _, rule := range config.GlobalConfig.PatrolRules {
		out := utils.ExecuteShell(rule.Command)
		if strings.TrimSpace(out) != "" && !strings.Contains(out, "exit status") {
			logger.Info(fmt.Sprintf("⚠️ 触发自定义规则: %s", rule.Name))
			anomalies = append(anomalies, fmt.Sprintf("**%s**:\n```\n%s\n```", rule.Name, strings.TrimSpace(out)))
		}
	}

	// 6. HTTP 监控
	httpResults := monitor.RunChecks()
	for _, res := range httpResults {
		if !res.Success {
			logger.Info(fmt.Sprintf("⚠️ HTTP 监控失败: %s", res.Name))
			anomalies = append(anomalies, fmt.Sprintf("**HTTP异常 (%s)**:\n%s", res.Name, res.Error))
		}
	}

	if len(anomalies) > 0 {
		report := strings.Join(anomalies, "\n")
		logger.Info("🚨 发现异常，正在请求 AI 分析...")
		analysis := agent.AnalyzeWithAI(report)
		alertMsg := fmt.Sprintf("🚨 **系统告警** [%s]\n\n%s\n\n💡 **处理建议**:\n%s", utils.GetHostname(), report, analysis)
		notify.Send("系统告警", alertMsg)
		logger.Info("告警已推送")
	} else {
		logger.Info("✔ 系统健康")
	}
}

// --- 其他 Handlers ---

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

func handleContainerAction(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	action := r.URL.Query().Get("action")
	if id == "" || action == "" { http.Error(w, "Missing params", 400); return }
	if action != "start" && action != "stop" && action != "restart" { http.Error(w, "Invalid action", 400); return }
	cmd := fmt.Sprintf("docker %s %s", action, id)
	logger.Info("Web操作容器: %s", cmd)
	utils.ExecuteShell(cmd)
	w.Write([]byte("success"))
}

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

func handleWSChat(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil { logger.Info("WS Upgrade Error: %v", err); return }
	defer conn.Close()
	messages := agent.GetBaseMessages()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil { break }
		input := string(msg)
		staticResp := agent.CheckStaticResponse(input)
		if staticResp != "" {
			conn.WriteJSON(map[string]string{"type": "answer", "content": staticResp})
			conn.WriteJSON(map[string]string{"type": "status", "content": "等待指令..."})
			continue
		}
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

func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userCfg := config.GlobalConfig.WebUser
		passCfg := config.GlobalConfig.WebPassword
		if userCfg == "" || passCfg == "" { next(w, r); return }
		user, pass, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(userCfg)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(passCfg)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func WebLog(msg string) { logger.Info(msg) }
