package server

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"qwq/internal/agent"
	"qwq/internal/config"
	"qwq/internal/logger"
	"qwq/internal/monitor"
	"qwq/internal/utils"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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

// --- 文件管理相关结构 ---
const MountPoint = "/hostfs"

var BlockList = []string{
	"/proc",
	"/sys",
	"/dev",
	"/boot",
}

type FileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"mod_time"`
	IsDir   bool   `json:"is_dir"`
	IsLink  bool   `json:"is_link"`
}

type FileResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func Start(port string) {
	var err error
	logFile, err = os.OpenFile("qwq.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("无法创建日志文件: %v\n", err)
	}

	go collectStatsLoop()

	distFS, err := fs.Sub(frontendDist, "dist")
	if err != nil {
		logger.Info("⚠️ 前端资源加载异常: %v", err)
	} else {
		fileServer := http.FileServer(http.FS(distFS))
		http.Handle("/assets/", fileServer)
		http.HandleFunc("/", basicAuth(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
				return 
			}
			fileServer.ServeHTTP(w, r)
		}))
	}

	http.HandleFunc("/api/logs", basicAuth(handleLogs))
	http.HandleFunc("/api/stats", basicAuth(handleStats))
	http.HandleFunc("/api/trigger", basicAuth(handleTrigger))
	http.HandleFunc("/api/containers", basicAuth(handleContainers))
	http.HandleFunc("/api/container/action", basicAuth(handleContainerAction))
	
	// 文件管理路由
	http.HandleFunc("/api/files/list", basicAuth(handleFileList))
	http.HandleFunc("/api/files/content", basicAuth(handleFileContent))
	http.HandleFunc("/api/files/save", basicAuth(handleFileSave))
	http.HandleFunc("/api/files/action", basicAuth(handleFileAction))

	http.HandleFunc("/ws/chat", basicAuth(handleWSChat))

	logger.Info("🚀 qwq Dashboard started at http://localhost" + port)
	if config.GlobalConfig.WebUser != "" {
		logger.Info("🔒 安全模式已开启 (Basic Auth)")
	}

	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Printf("Web Server Error: %v\n", err)
	}
}

// --- 巡检逻辑  ---
func performPatrol() {
	logger.Info("正在执行系统巡检...")
	var anomalies []string

	// 1. 磁盘检查：不再依赖 grep，改用 Go 代码逐行过滤
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

// --- 文件管理逻辑 ---

func jsonResponse(w http.ResponseWriter, code int, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FileResponse{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

func resolveSafePath(userPath string) (string, error) {
	cleanPath := filepath.Clean(userPath)
	for _, blocked := range BlockList {
		if strings.HasPrefix(cleanPath, blocked) {
			return "", fmt.Errorf("access denied: path '%s' is in blocklist", cleanPath)
		}
	}
	realPath := filepath.Join(MountPoint, cleanPath)
	if !strings.HasPrefix(realPath, MountPoint) {
		return "", fmt.Errorf("access denied: path escape detected")
	}
	return realPath, nil
}

func handleFileList(w http.ResponseWriter, r *http.Request) {
	userPath := r.URL.Query().Get("path")
	if userPath == "" { userPath = "/" }

	realPath, err := resolveSafePath(userPath)
	if err != nil {
		logger.Info("[AUDIT] 🚨 非法访问尝试: %s | Error: %v", userPath, err)
		jsonResponse(w, 403, err.Error(), nil)
		return
	}

	entries, err := os.ReadDir(realPath)
	if err != nil {
		logger.Info("读取目录失败: %s | Error: %v", realPath, err)
		jsonResponse(w, 500, fmt.Sprintf("无法读取目录: %v", err), nil)
		return
	}

	files := make([]FileInfo, 0)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil { continue }
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			IsDir:   entry.IsDir(),
			IsLink:  info.Mode()&os.ModeSymlink != 0,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})

	jsonResponse(w, 200, "success", map[string]interface{}{
		"path":  userPath,
		"files": files,
	})
}

func handleFileContent(w http.ResponseWriter, r *http.Request) {
	userPath := r.URL.Query().Get("path")
	realPath, err := resolveSafePath(userPath)
	if err != nil {
		jsonResponse(w, 403, err.Error(), nil)
		return
	}

	info, err := os.Stat(realPath)
	if err != nil {
		jsonResponse(w, 404, "文件不存在", nil)
		return
	}
	if info.Size() > 2*1024*1024 {
		jsonResponse(w, 400, "文件过大 (>2MB)，不支持在线编辑", nil)
		return
	}

	content, err := os.ReadFile(realPath)
	if err != nil {
		jsonResponse(w, 500, "读取失败", nil)
		return
	}

	if !utf8.Valid(content) {
		jsonResponse(w, 400, "检测到二进制文件，不支持编辑", nil)
		return
	}

	w.Write(content)
}

func handleFileSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonResponse(w, 405, "Method not allowed", nil)
		return
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, "Invalid JSON", nil)
		return
	}
	realPath, err := resolveSafePath(req.Path)
	if err != nil {
		logger.Info("[AUDIT] 🚨 非法写入尝试: %s", req.Path)
		jsonResponse(w, 403, err.Error(), nil)
		return
	}
	if err := atomicWriteFile(realPath, []byte(req.Content), 0644); err != nil {
		logger.Info("[AUDIT] ❌ 文件保存失败: %s | Error: %v", req.Path, err)
		jsonResponse(w, 500, fmt.Sprintf("保存失败: %v", err), nil)
		return
	}
	logger.Info("[AUDIT] 📝 文件已修改: %s (Size: %d bytes)", req.Path, len(req.Content))
	jsonResponse(w, 200, "success", nil)
}

func handleFileAction(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("type")
	userPath := r.URL.Query().Get("path")
	realPath, err := resolveSafePath(userPath)
	if err != nil {
		jsonResponse(w, 403, err.Error(), nil)
		return
	}
	var errOp error
	switch action {
	case "delete":
		if userPath == "/" || realPath == MountPoint {
			jsonResponse(w, 403, "禁止删除根目录", nil)
			return
		}
		errOp = os.RemoveAll(realPath)
		if errOp == nil { logger.Info("[AUDIT] 🗑️ 文件/目录已删除: %s", userPath) }
	case "mkdir":
		errOp = os.MkdirAll(realPath, 0755)
		if errOp == nil { logger.Info("[AUDIT] 📂 目录已创建: %s", userPath) }
	default:
		jsonResponse(w, 400, "Unknown action", nil)
		return
	}
	if errOp != nil {
		jsonResponse(w, 500, fmt.Sprintf("操作失败: %v", errOp), nil)
		return
	}
	jsonResponse(w, 200, "success", nil)
}

func atomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, "qwq_tmp_*")
	if err != nil { return err }
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)
	if _, err := tmpFile.Write(data); err != nil { tmpFile.Close(); return err }
	if err := tmpFile.Sync(); err != nil { tmpFile.Close(); return err }
	if err := tmpFile.Close(); err != nil { return err }
	return os.Rename(tmpName, filename)
}

func sendSystemStatus() {
	hostname := utils.GetHostname()
	ip := strings.TrimSpace(utils.ExecuteShell("ip route get 1 | awk '{print $7; exit}'"))
	uptime := strings.TrimSpace(utils.ExecuteShell("uptime -p"))
	memInfo := strings.TrimSpace(utils.ExecuteShell("free -m | awk 'NR==2{printf \"%.1f%% (已用 %sM / 总计 %sM)\", $3/$2*100, $3, $2}'"))
	diskInfo := strings.TrimSpace(utils.ExecuteShell("df -h / | awk 'NR==2 {print $5 \" (剩余 \" $4 \")\"}'"))
	loadInfo := strings.TrimSpace(utils.ExecuteShell("uptime | awk -F'load average:' '{ print $2 }'"))
	report := fmt.Sprintf(`### 📊 服务器状态日报 [%s]

> **IP**: %s
> **运行**: %s

---

| 指标 | 状态 |
| :--- | :--- |
| **CPU负载** | %s |
| **内存使用** | %s |
| **系统磁盘** | %s |
| **TCP连接** | %s |

---
*qwq AIOps 自动监控*
`, hostname, ip, uptime, loadInfo, memInfo, diskInfo,
		strings.TrimSpace(utils.ExecuteShell("netstat -ant | grep ESTABLISHED | wc -l")))
	notify.Send("服务器状态日报", report)
	logger.Info("✅ 健康日报已发送")
}