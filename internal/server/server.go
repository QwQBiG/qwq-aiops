package server

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"qwq/internal/config"
	"qwq/internal/logger"
	"qwq/internal/monitor"
	"qwq/internal/notify"
	"qwq/internal/utils"
	"strings"
	"sync"
	"time"
)

//go:embed static/index.html
var content embed.FS

var (
	LogBuffer         []string
	LogMutex          sync.Mutex
	TriggerPatrolFunc func()
	TriggerStatusFunc func()
	logFile           *os.File
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

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data, _ := content.ReadFile("static/index.html")
	w.Write(data)
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(logger.GetWebLogs())
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

	// 获取 HTTP 监控状态
	httpStatus := monitor.RunChecks()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"load":       load,
		"mem_pct":    fmt.Sprintf("%.1f", memPct),
		"mem_used":   fmt.Sprintf("%.0f", memUsed),
		"mem_total":  fmt.Sprintf("%.0f", memTotal),
		"disk_pct":   diskPct,
		"disk_avail": diskAvail,
		"services":   httpStatus, // 传给前端
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