package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/chzyer/readline"
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

// --- 配置 ---
const (
	DefaultModel   = "Qwen/Qwen2.5-7B-Instruct"
	DefaultBaseURL = "https://api.siliconflow.cn/v1"
	WebServerPort  = ":8899"
)

var (
	client           *openai.Client
	renderer         *glamour.TermRenderer
	dingTalkWebhook  string
	debugMode        bool
	ErrMissingAPIKey = errors.New("critical: OPENAI_API_KEY environment variable is not set")

	logBuffer []string
	logMutex  sync.Mutex
)

// --- 工具定义 ---
var tools = []openai.Tool{
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "execute_shell_command",
			Description: "Execute a shell command on the local Linux/MacOS system.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": { "type": "string", "description": "The shell command" },
					"reason": { "type": "string", "description": "The reason (in Chinese)" }
				},
				"required": ["command", "reason"]
			}`),
		},
	},
}

func main() {
	var err error
	renderer, err = glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(100))
	if err != nil { fmt.Println("Renderer init failed:", err) }

	rootCmd := &cobra.Command{
		Use:   "qwq",
		Short: "Advanced AIOps Agent",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if dingTalkWebhook != "" {
				dingTalkWebhook = strings.ReplaceAll(dingTalkWebhook, "\\", "")
			}
			return initClient()
		},
	}

	rootCmd.PersistentFlags().StringVar(&dingTalkWebhook, "webhook", "", "DingTalk Webhook URL")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logs")

	rootCmd.AddCommand(&cobra.Command{Use: "chat", Short: "Interactive Mode", Run: runChatMode})
	rootCmd.AddCommand(&cobra.Command{Use: "patrol", Short: "Patrol Mode", Run: runPatrolMode})
	rootCmd.AddCommand(&cobra.Command{Use: "status", Short: "Send status immediately", Run: runStatusMode})
	rootCmd.AddCommand(&cobra.Command{Use: "web", Short: "Start Web Dashboard", Run: runWebMode})

	if err := rootCmd.Execute(); err != nil { os.Exit(1) }
}

func initClient() error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" { return ErrMissingAPIKey }
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = DefaultBaseURL
	client = openai.NewClientWithConfig(config)
	return nil
}

// ==========================================
// Web Dashboard
// ==========================================

func runWebMode(cmd *cobra.Command, args []string) {
	go runPatrolLoop(8 * time.Hour)
	go sendSystemStatus()

	http.HandleFunc("/", handleWebIndex)
	http.HandleFunc("/api/logs", handleWebLogs)
	http.HandleFunc("/api/stats", handleWebStats)
	http.HandleFunc("/api/trigger", handleWebTrigger)

	webLog("🚀 qwq Dashboard started at http://localhost" + WebServerPort)
	webLog("守护进程运行中... (巡检周期: 5m, 汇报周期: 8h)")

	server := &http.Server{Addr: WebServerPort}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Web Server Error: %v\n", err)
		}
	}()

	waitForShutdown()
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\n正在关闭服务...")
}

func webLog(msg string) {
	logMutex.Lock()
	defer logMutex.Unlock()
	ts := time.Now().Format("15:04:05")
	logEntry := fmt.Sprintf("[%s] %s", ts, msg)
	fmt.Println(logEntry)
	logBuffer = append(logBuffer, logEntry)
	if len(logBuffer) > 100 { logBuffer = logBuffer[1:] }
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>qwq AIOps 控制台</title>
    <style>
        :root { --bg: #0f172a; --card-bg: #1e293b; --text: #e2e8f0; --accent: #38bdf8; --success: #4ade80; --danger: #f87171; }
        body { background-color: var(--bg); color: var(--text); font-family: 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; margin: 0; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 30px; padding-bottom: 20px; border-bottom: 1px solid #334155; }
        .header h1 { margin: 0; font-size: 24px; display: flex; align-items: center; gap: 10px; }
        .status-dot { width: 10px; height: 10px; background-color: var(--success); border-radius: 50%; box-shadow: 0 0 10px var(--success); }
        .btn { background: linear-gradient(135deg, #3b82f6, #2563eb); color: white; border: none; padding: 10px 20px; border-radius: 8px; cursor: pointer; font-weight: bold; transition: transform 0.1s; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1); }
        .btn:hover { transform: translateY(-2px); box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1); }
        .btn:active { transform: translateY(0); }
        .card-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .card { background: var(--card-bg); border-radius: 12px; padding: 20px; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1); border: 1px solid #334155; }
        .card h3 { margin-top: 0; color: #94a3b8; font-size: 14px; text-transform: uppercase; letter-spacing: 1px; }
        .metric { font-size: 28px; font-weight: bold; color: var(--accent); margin: 10px 0; }
        .sub-text { font-size: 12px; color: #64748b; }
        .progress-bg { background: #334155; height: 8px; border-radius: 4px; overflow: hidden; margin-top: 10px; }
        .progress-fill { height: 100%; background: var(--accent); width: 0%; transition: width 0.5s ease; }
        .progress-fill.high { background: var(--danger); }
        .log-window { background: #000; border: 1px solid #334155; border-radius: 8px; height: 400px; overflow-y: auto; padding: 15px; font-family: 'Consolas', 'Monaco', monospace; font-size: 13px; color: #a5b4fc; box-shadow: inset 0 2px 4px 0 rgba(0, 0, 0, 0.06); }
        .log-entry { margin-bottom: 6px; border-bottom: 1px solid #1e293b; padding-bottom: 4px; }
        .log-time { color: #64748b; margin-right: 10px; }
        ::-webkit-scrollbar { width: 8px; }
        ::-webkit-scrollbar-track { background: #0f172a; }
        ::-webkit-scrollbar-thumb { background: #334155; border-radius: 4px; }
        ::-webkit-scrollbar-thumb:hover { background: #475569; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1><div class="status-dot"></div> qwq AIOps 控制台</h1>
            <button class="btn" onclick="triggerPatrol()">⚡ 立即触发巡检</button>
        </div>
        <div class="card-grid">
            <div class="card">
                <h3>CPU 负载</h3>
                <div class="metric" id="cpu-load">--</div>
                <div class="sub-text">1min / 5min / 15min</div>
            </div>
            <div class="card">
                <h3>内存使用</h3>
                <div class="metric" id="mem-text">--</div>
                <div class="progress-bg"><div class="progress-fill" id="mem-bar"></div></div>
                <div class="sub-text" id="mem-detail">正在获取...</div>
            </div>
            <div class="card">
                <h3>系统磁盘 (/)</h3>
                <div class="metric" id="disk-text">--</div>
                <div class="progress-bg"><div class="progress-fill" id="disk-bar"></div></div>
                <div class="sub-text" id="disk-detail">正在获取...</div>
            </div>
        </div>
        <h3>📜 实时运行日志</h3>
        <div class="log-window" id="log-box"></div>
    </div>
    <script>
        function updateStats() {
            fetch('/api/stats').then(r => r.json()).then(data => {
                document.getElementById('cpu-load').innerText = data.load;
                document.getElementById('mem-text').innerText = data.mem_pct + '%';
                document.getElementById('mem-detail').innerText = data.mem_used + 'M / ' + data.mem_total + 'M';
                const memBar = document.getElementById('mem-bar');
                memBar.style.width = data.mem_pct + '%';
                if(data.mem_pct > 80) memBar.classList.add('high'); else memBar.classList.remove('high');
                document.getElementById('disk-text').innerText = data.disk_pct + '%';
                document.getElementById('disk-detail').innerText = '剩余 ' + data.disk_avail;
                const diskBar = document.getElementById('disk-bar');
                diskBar.style.width = data.disk_pct + '%';
                if(data.disk_pct > 85) diskBar.classList.add('high'); else diskBar.classList.remove('high');
            });
        }
        function updateLogs() {
            fetch('/api/logs').then(r => r.json()).then(logs => {
                const box = document.getElementById('log-box');
                const html = logs.map(l => {
                    const parts = l.split('] ');
                    const time = parts[0] + ']';
                    const msg = parts.slice(1).join('] ');
                    return '<div class="log-entry"><span class="log-time">' + time + '</span>' + msg + '</div>';
                }).join('');
                if (box.innerHTML !== html) {
                    box.innerHTML = html;
                    box.scrollTop = box.scrollHeight;
                }
            });
        }
        function triggerPatrol() {
            const btn = document.querySelector('.btn');
            btn.innerText = '⏳ 请求中...';
            btn.disabled = true;
            fetch('/api/trigger').then(r => r.text()).then(msg => {
                alert(msg);
                btn.innerText = '⚡ 立即触发巡检';
                btn.disabled = false;
            });
        }
        setInterval(updateStats, 2000);
        setInterval(updateLogs, 2000);
        updateStats(); updateLogs();
    </script>
</body>
</html>
`

func handleWebIndex(w http.ResponseWriter, r *http.Request) {
	t, _ := template.New("index").Parse(htmlTemplate)
	t.Execute(w, nil)
}

func handleWebLogs(w http.ResponseWriter, r *http.Request) {
	logMutex.Lock()
	defer logMutex.Unlock()
	json.NewEncoder(w).Encode(logBuffer)
}

func handleWebStats(w http.ResponseWriter, r *http.Request) {
	load := strings.TrimSpace(executeShell("uptime | awk -F'load average:' '{ print $2 }'"))
	memRaw := executeShell("free -m | awk 'NR==2{print $2,$3}'")
	var memTotal, memUsed float64
	fmt.Sscanf(memRaw, "%f %f", &memTotal, &memUsed)
	memPct := 0.0
	if memTotal > 0 { memPct = (memUsed / memTotal) * 100 }
	diskRaw := executeShell("df -h / | awk 'NR==2 {print $5,$4}'")
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

func handleWebTrigger(w http.ResponseWriter, r *http.Request) {
	go performPatrol()
	go sendSystemStatus()
	w.Write([]byte("指令已发送：正在后台执行巡检和汇报..."))
}

// ==========================================
// 核心逻辑
// ==========================================

func runStatusMode(cmd *cobra.Command, args []string) {
	if dingTalkWebhook == "" { fmt.Println("错误: 请提供 --webhook"); return }
	sendSystemStatus()
}

func sendSystemStatus() {
	hostname := getHostname()
	// [修复] 使用 ip route 获取真实 IP，兼容 Alpine/Docker
	ip := strings.TrimSpace(executeShell("ip route get 1 | awk '{print $7; exit}'"))
	uptime := strings.TrimSpace(executeShell("uptime -p"))
	memInfo := strings.TrimSpace(executeShell("free -m | awk 'NR==2{printf \"%.1f%% (已用 %sM / 总计 %sM)\", $3/$2*100, $3, $2}'"))
	diskInfo := strings.TrimSpace(executeShell("df -h / | awk 'NR==2 {print $5 \" (剩余 \" $4 \")\"}'"))
	loadInfo := strings.TrimSpace(executeShell("uptime | awk -F'load average:' '{ print $2 }'"))

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
		strings.TrimSpace(executeShell("netstat -ant | grep ESTABLISHED | wc -l")))

	sendDingTalk(report, "服务器状态日报")
	webLog("✅ 健康日报已发送")
}

func runPatrolLoop(reportInterval time.Duration) {
	checkTicker := time.NewTicker(5 * time.Minute)
	reportTicker := time.NewTicker(reportInterval)
	defer checkTicker.Stop()
	defer reportTicker.Stop()

	performPatrol()

	for {
		select {
		case <-checkTicker.C:
			performPatrol()
		case <-reportTicker.C:
			sendSystemStatus()
		}
	}
}

func runPatrolMode(cmd *cobra.Command, args []string) {
	webLog("巡检模式启动 (无 Web 面板)")
	if dingTalkWebhook == "" { fmt.Println("警告: 未配置 Webhook") }
	go runPatrolLoop(8 * time.Hour)
	waitForShutdown()
}

func performPatrol() {
	webLog("正在执行系统巡检...")
	var anomalies []string

	if out := executeShell("df -h | grep -vE '^Filesystem|tmpfs|cdrom|efivarfs|overlay' | awk 'int($5) > 85 {print $0}'"); strings.TrimSpace(out) != "" && !strings.Contains(out, "exit status") {
		anomalies = append(anomalies, "**磁盘告警**:\n```\n"+strings.TrimSpace(out)+"\n```")
	}
	if out := executeShell("uptime | awk -F'load average:' '{ print $2 }' | awk '{ if ($1 > 4.0) print $0 }'"); strings.TrimSpace(out) != "" && !strings.Contains(out, "exit status") {
		anomalies = append(anomalies, "**高负载**:\n```\n"+strings.TrimSpace(out)+"\n```")
	}
	dmesgOut := executeShell("dmesg | grep -i 'out of memory' | tail -n 5")
	if !strings.Contains(dmesgOut, "Operation not permitted") && !strings.Contains(dmesgOut, "不允许的操作") && strings.TrimSpace(dmesgOut) != "" && !strings.Contains(dmesgOut, "exit status") {
		anomalies = append(anomalies, "**OOM日志**:\n```\n"+strings.TrimSpace(dmesgOut)+"\n```")
	}
	
	rawZombies := executeShell("ps -A -o stat,ppid,pid,cmd | awk '$1 ~ /^[Zz]/'")
	if strings.TrimSpace(rawZombies) != "" && !strings.Contains(rawZombies, "exit status") {
		detailZombie := "STAT    PPID     PID CMD\n" + rawZombies
		anomalies = append(anomalies, "**僵尸进程**:\n```\n"+strings.TrimSpace(detailZombie)+"\n```")
	}

	if len(anomalies) > 0 {
		report := strings.Join(anomalies, "\n")
		webLog("🚨 发现异常，正在请求 AI 分析...")
		analysis := analyzeWithAI(report)
		alertMsg := fmt.Sprintf("🚨 **系统告警** [%s]\n\n%s\n\n💡 **处理建议**:\n%s", getHostname(), report, analysis)
		sendDingTalk(alertMsg, "系统告警")
		webLog("告警已推送")
	} else {
		webLog("✔ 系统健康")
	}
}

func analyzeWithAI(issue string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sysPrompt := `你是一个紧急故障响应专家。
规则：
1. **极度简练**：只输出核心原因和一条修复命令。
2. **拒绝废话**：不要解释原理。
3. **空数据防御**：如果输入只包含表头（如 STAT PPID PID CMD）而没有数据行，或者数据为空，请回答“误报，无异常”，不要给出任何修复建议。
4. **僵尸进程特判**：
   - 输入数据包含表头：STAT PPID PID CMD
   - **PPID (第二列)** 是父进程 ID。
   - 修复命令格式：kill -9 <PPID>`
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: DefaultModel, Messages: []openai.ChatCompletionMessage{{Role: "system", Content: sysPrompt}, {Role: "user", Content: issue}}, Temperature: 0.1,
	})
	if err != nil { return "AI 连接失败" }
	return resp.Choices[0].Message.Content
}

func sendDingTalk(msg string, title string) {
	if dingTalkWebhook == "" { return }
	payload := map[string]interface{}{"msgtype": "markdown", "markdown": map[string]string{"title": title, "text": msg}}
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(dingTalkWebhook, "application/json", bytes.NewBuffer(jsonData))
	if err == nil { defer resp.Body.Close(); io.ReadAll(resp.Body) }
}

// ==========================================
// Chat Mode
// ==========================================
func runChatMode(cmd *cobra.Command, args []string) {
	rl, err := readline.NewEx(&readline.Config{Prompt: "\033[32mqwq > \033[0m", HistoryFile: "/tmp/qwq_history"})
	if err != nil { panic(err) }
	defer rl.Close()
	printSystemMessage("Agent Online. System: " + runtime.GOOS)
	sysPrompt := `你是一个资深运维专家助手(qwq)。
规则：
1. 请用中文回答。
2. **分步执行**：先获取信息，再执行下一步。
3. **严禁编造**：如果命令返回 "exit status 1" 或空，说明进程不存在或命令失败，请直接告诉用户“未找到”或“失败”，**绝对不要捏造输出结果**。
4. 如果是查询类命令（如 get, describe, logs, top, ps），请放心执行。`
	messages := []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: sysPrompt}}
	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt { if len(line)==0 {break}; continue }
		if err == io.EOF { break }
		input := strings.TrimSpace(line)
		if input == "exit" || input == "quit" { break }
		if input == "" { continue }
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: input})
		for i := 0; i < 5; i++ {
			respMsg, shouldContinue := processAgentStep(&messages)
			if !shouldContinue { break }
			if respMsg.Content != "" && len(respMsg.ToolCalls) == 0 { renderMarkdown(respMsg.Content); break }
		}
	}
}

func processAgentStep(msgs *[]openai.ChatCompletionMessage) (openai.ChatCompletionMessage, bool) {
	ctx := context.Background()
	fmt.Print("\033[33m🤖 思考中...\033[0m\r")
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{Model: DefaultModel, Messages: *msgs, Tools: tools, Temperature: 0.1})
	if err != nil { fmt.Printf("\nAPI Error: %v\n", err); return openai.ChatCompletionMessage{}, false }
	msg := resp.Choices[0].Message
	*msgs = append(*msgs, msg)
	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			if toolCall.Function.Name == "execute_shell_command" {
				var args map[string]string
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil { continue }
				cmdStr := strings.TrimSpace(args["command"])
				reason := args["reason"]
				if cmdStr == "" { continue }
				fmt.Printf("\n\033[36m⚡ 意图: %s\033[0m\n", reason)
				fmt.Printf("\033[33m👉 命令: \033[1m%s\033[0m\n", cmdStr)
				if !isCommandSafe(cmdStr) {
					fmt.Println("\033[31m[拦截] 高危命令\033[0m")
					addToolOutput(msgs, toolCall.ID, "Error: Blocked.")
					continue
				}
				shouldAutoRun := isReadOnlyCommand(cmdStr)
				if shouldAutoRun { fmt.Println("\033[90m(自动执行查询命令...)\033[0m") } else {
					if !confirmExecution() { fmt.Println("\033[90m已跳过\033[0m"); addToolOutput(msgs, toolCall.ID, "User denied."); continue }
				}
				fmt.Print("\033[90m执行中...\033[0m")
				output := executeShell(cmdStr)
				if strings.TrimSpace(output) == "" { output = "(No output)" }
				fmt.Printf("\r\033[32m✔ 完成\033[0m\n")
				addToolOutput(msgs, toolCall.ID, output)
			}
		}
		return msg, true
	}
	return msg, true
}

func addToolOutput(msgs *[]openai.ChatCompletionMessage, id, content string) {
	*msgs = append(*msgs, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleTool, Content: content, ToolCallID: id})
}

func isReadOnlyCommand(cmd string) bool {
	safeKeywords := []string{"ls", "cat", "head", "tail", "grep", "find", "pwd", "echo", "whoami", "id", "ps", "top", "uptime", "free", "df", "du", "netstat", "ss", "lsof", "kubectl get", "kubectl describe", "kubectl logs", "kubectl top", "kubectl cluster-info", "docker ps", "docker logs", "docker stats"}
	c := strings.ToLower(cmd)
	for _, kw := range safeKeywords {
		if strings.HasPrefix(c, kw) || strings.Contains(c, " "+kw) {
			if !strings.Contains(c, ">") && !strings.Contains(c, "rm ") && !strings.Contains(c, "kill") && !strings.Contains(c, "delete") { return true }
		}
	}
	return false
}

func executeShell(c string) string {
	cmd := exec.Command("bash", "-c", c)
	// 移除了 Setpgid 以兼容 Windows 编译，但在 Docker/Linux 中依然正常工作
	out, err := cmd.CombinedOutput()
	res := string(out)
	if err != nil { if len(res) > 0 { res += fmt.Sprintf("\n(Command failed: %v)", err) } else { res = fmt.Sprintf("(Command failed: %v)", err) } }
	if len(res) > 4000 { res = res[:4000] + "\n...(Output truncated)" }
	return res
}

func isCommandSafe(c string) bool {
	dangerous := []string{"rm -rf", "mkfs", ":(){:|:&};:", "> /dev/sda", "dd if=/dev/zero"}
	for _, d := range dangerous { if strings.Contains(c, d) { return false } }
	return true
}

func confirmExecution() bool {
	fmt.Print("\033[33m[?] 这是一个修改操作，确认执行? (Y/n): \033[0m")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "" || input == "y" || input == "yes"
}

func renderMarkdown(t string) { if o, e := renderer.Render(t); e == nil { fmt.Print(o) } else { fmt.Println(t) } }
func getHostname() string { h, _ := os.Hostname(); return h }
func printSystemMessage(m string) { fmt.Printf("\033[36m(qwq) %s\033[0m\n", m) }
