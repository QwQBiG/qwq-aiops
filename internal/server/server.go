package server

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"qwq/internal/config"
	"qwq/internal/utils"
	"strings"
	"sync"
	"time"
)

//go:embed static/index.html
var content embed.FS

var (
	LogBuffer []string
	LogMutex  sync.Mutex
	// 外部注入的回调函数
	TriggerPatrolFunc func()
	TriggerStatusFunc func()
)

func Start(port string) {
	http.HandleFunc("/", basicAuth(handleIndex))
	http.HandleFunc("/api/logs", basicAuth(handleLogs))
	http.HandleFunc("/api/stats", basicAuth(handleStats))
	http.HandleFunc("/api/trigger", basicAuth(handleTrigger))

	WebLog("🚀 qwq Dashboard started at http://localhost" + port)
	// [修改] 使用 GlobalConfig
	if config.GlobalConfig.WebUser != "" {
		WebLog("🔒 安全模式已开启 (Basic Auth)")
	}

	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Printf("Web Server Error: %v\n", err)
	}
}

func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// [修改] 使用 GlobalConfig
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

	json.NewEncoder(w).Encode(map[string]interface{}{
		"load":       load,
		"mem_pct":    fmt.Sprintf("%.1f", memPct),
		"mem_used":   fmt.Sprintf("%.0f", memUsed),
		"mem_total":  fmt.Sprintf("%.0f", memTotal),
		"disk_pct":   diskPct,
		"disk_avail": diskAvail,
	})
}

func handleTrigger(w http.ResponseWriter, r *http.Request) {
	if TriggerPatrolFunc != nil { go TriggerPatrolFunc() }
	if TriggerStatusFunc != nil { go TriggerStatusFunc() }
	w.Write([]byte("指令已发送：正在后台执行巡检和汇报..."))
}

func WebLog(msg string) {
	LogMutex.Lock()
	defer LogMutex.Unlock()
	ts := time.Now().Format("15:04:05")
	logEntry := fmt.Sprintf("[%s] %s", ts, msg)
	fmt.Println(logEntry)
	LogBuffer = append(LogBuffer, logEntry)
	if len(LogBuffer) > 100 {
		LogBuffer = LogBuffer[1:]
	}
}