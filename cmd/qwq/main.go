package main

import (
	"fmt"
	"os"
	"os/signal"
	"qwq/internal/agent"
	"qwq/internal/config"
	"qwq/internal/executor"
	"qwq/internal/gateway"
	"qwq/internal/logger"
	"qwq/internal/monitor"
	"qwq/internal/notify"
	"qwq/internal/security"
	"qwq/internal/server"
	"qwq/internal/utils"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/chzyer/readline"
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

var configPath string

func main() {
	rootCmd := &cobra.Command{
		Use:   "qwq",
		Short: "OpsPilot - Enterprise AIOps Agent",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Init(configPath); err != nil {
				return err
			}
			logger.Init("qwq.log", config.GlobalConfig.DebugMode)
			if config.GlobalConfig.DingTalkWebhook != "" {
				config.GlobalConfig.DingTalkWebhook = strings.ReplaceAll(config.GlobalConfig.DingTalkWebhook, "\\", "")
			}
			agent.InitClient()
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file")
	rootCmd.PersistentFlags().StringVar(&config.GlobalConfig.DingTalkWebhook, "webhook", "", "DingTalk Webhook URL")
	rootCmd.PersistentFlags().StringVar(&config.GlobalConfig.WebUser, "user", "", "Web Dashboard Username")
	rootCmd.PersistentFlags().StringVar(&config.GlobalConfig.WebPassword, "password", "", "Web Dashboard Password")
	rootCmd.PersistentFlags().StringVar(&config.GlobalConfig.KnowledgeFile, "knowledge", "", "Path to knowledge base file")

	rootCmd.AddCommand(&cobra.Command{Use: "chat", Short: "Interactive Mode", Run: runChatMode})
	rootCmd.AddCommand(&cobra.Command{Use: "patrol", Short: "Patrol Mode", Run: runPatrolMode})
	rootCmd.AddCommand(&cobra.Command{Use: "status", Short: "Send status", Run: runStatusMode})
	rootCmd.AddCommand(&cobra.Command{Use: "web", Short: "Web Dashboard", Run: runWebMode})
	rootCmd.AddCommand(&cobra.Command{Use: "gateway", Short: "API Gateway Mode", Run: runGatewayMode})
	
	rootCmd.AddCommand(&cobra.Command{
		Use:   "run [command]",
		Short: "Smart execution with auto-remediation",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fullCmd := strings.Join(args, " ")
			if !utils.ConfirmExecution(fullCmd) {
				fmt.Println("已取消")
				return
			}
			executor.SmartRun(fullCmd)
		},
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runWebMode 启动 Web 控制台模式
// 提供可视化界面和 API 服务，支持通过环境变量 PORT 自定义端口
func runWebMode(cmd *cobra.Command, args []string) {
	// 注册巡检和状态推送回调函数
	server.TriggerPatrolFunc = performPatrol
	server.TriggerStatusFunc = sendSystemStatus
	
	// 启动后台定时任务：每 8 小时执行一次巡检和日报
	go runPatrolLoop(8 * time.Hour)
	
	// 从环境变量读取服务端口，默认使用 8080
	// 可通过 docker-compose.yml 或 .env 文件配置
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	// 启动 HTTP 服务器
	server.Start(":" + port)
}

// runGatewayMode 启动 API 网关模式
// 提供统一的 API 入口，支持服务发现、负载均衡和路由转发
func runGatewayMode(cmd *cobra.Command, args []string) {
	logger.Info("🚀 启动增强版 API Gateway 模式")
	
	// 从环境变量读取网关端口，默认 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	// 从环境变量读取 Web UI 端口，默认 8899
	webUIPort := os.Getenv("WEB_UI_PORT")
	if webUIPort == "" {
		webUIPort = "8899"
	}
	
	// 创建增强版网关服务器
	gatewayServer := gateway.NewEnhancedGatewayServer(":" + port)
	
	// 添加文档路由
	gatewayServer.GetGateway().AddDocsRoutes()
	
	// 启动后台服务
	server.TriggerPatrolFunc = performPatrol
	server.TriggerStatusFunc = sendSystemStatus
	go runPatrolLoop(8 * time.Hour)
	
	// 启动原有Web服务（作为微服务之一）
	go func() {
		logger.Info("启动 Web UI 服务在端口 :%s", webUIPort)
		server.Start(":" + webUIPort)
	}()
	
	// 等待服务启动
	time.Sleep(2 * time.Second)
	
	// 启动增强版网关
	if err := gatewayServer.Start(); err != nil {
		logger.Info("增强版网关启动失败: %v", err)
		os.Exit(1)
	}
}

func runPatrolMode(cmd *cobra.Command, args []string) {
	logger.Info("巡检模式启动 (无 Web 面板)")
	go runPatrolLoop(8 * time.Hour)
	waitForShutdown()
}

func runStatusMode(cmd *cobra.Command, args []string) {
	if config.GlobalConfig.DingTalkWebhook == "" {
		fmt.Println("错误: 请提供 --webhook 或在配置文件中设置")
		return
	}
	sendSystemStatus()
}

func runChatMode(cmd *cobra.Command, args []string) {
	rl, _ := readline.NewEx(&readline.Config{Prompt: "\033[32mqwq > \033[0m", HistoryFile: "/tmp/qwq_history"})
	defer rl.Close()
	fmt.Printf("\033[36m(qwq) Agent Online. System: %s\033[0m\n", runtime.GOOS)
	
	messages := agent.GetBaseMessages()

	for {
		line, _ := rl.Readline()
		if line == "exit" { break }
		if line == "" { continue }
		
		// 1. 静态规则
		staticResp := agent.CheckStaticResponse(line)
		if staticResp != "" {
			r, _ := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(100))
			out, _ := r.Render(staticResp)
			fmt.Print(out)
			continue
		}

		// 2. 关键词速查
		quickCmd := agent.GetQuickCommand(line)
		if quickCmd != "" {
			fmt.Printf("\033[90m⚡ 快速执行: %s\033[0m\n", quickCmd)
			output := utils.ExecuteShell(quickCmd)
			if strings.TrimSpace(output) == "" { output = "(No output)" }
			fmt.Println(output)
			continue
		}
		
		safeInput := security.Redact(line)
		enhancedInput := safeInput + " (Context: Current Linux Server)"
		
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: enhancedInput})
		
		for i := 0; i < 5; i++ {
			respMsg, cont := agent.ProcessAgentStep(&messages)
			
			if respMsg.Content != "" && len(respMsg.ToolCalls) == 0 {
				r, _ := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(100))
				out, _ := r.Render(respMsg.Content)
				fmt.Print(out)
				
				agent.CheckAndSaveFile(respMsg.Content)
			}
			
			if !cont { break }
		}
	}
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\n正在关闭服务...")
}

func runPatrolLoop(interval time.Duration) {
	checkTicker := time.NewTicker(5 * time.Minute)
	reportTicker := time.NewTicker(interval)
	defer checkTicker.Stop()
	defer reportTicker.Stop()
	
	// 启动时立即执行一次巡检
	performPatrol()
	
	// 启动时延迟一小段时间后发送第一次日报（避免和立即发送的冲突）
	go func() {
		time.Sleep(30 * time.Second)
		sendSystemStatus()
	}()
	
	logger.Info("📅 定时任务已启动: 巡检每5分钟, 日报每%v", interval)
	
	for {
		select {
		case <-checkTicker.C:
			logger.Info("⏰ 定时巡检触发")
			performPatrol()
		case <-reportTicker.C:
			logger.Info("⏰ 定时日报触发")
			sendSystemStatus()
		}
	}
}

func performPatrol() {
	logger.Info("正在执行系统巡检...")
	var anomalies []string

	// 磁盘检查：在代码中解析和过滤，确保可靠过滤 loop、snap 等设备
	diskOut := utils.ExecuteShell("df -h")
	diskLines := strings.Split(diskOut, "\n")
	var diskAlerts []string
	
	for _, line := range diskLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Filesystem") {
			continue
		}
		
		// 严格过滤：检查设备名和挂载点
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		
		device := fields[0]
		mountPoint := fields[len(fields)-1]
		
		// 过滤所有 loop 设备、snap 相关、虚拟文件系统
		if strings.Contains(device, "/dev/loop") ||
		   strings.Contains(device, "loop") ||
		   strings.Contains(mountPoint, "/snap") ||
		   strings.Contains(mountPoint, "snap/") ||
		   strings.Contains(mountPoint, "/hostfs") ||
		   strings.Contains(line, "tmpfs") ||
		   strings.Contains(line, "overlay") ||
		   strings.Contains(line, "cdrom") ||
		   strings.Contains(line, "efivarfs") {
			continue
		}
		
		// 解析使用率
		useStr := strings.TrimSuffix(fields[4], "%")
		usePct, err := strconv.Atoi(useStr)
		if err == nil && usePct > 85 {
			diskAlerts = append(diskAlerts, line)
		}
	}
	
	if len(diskAlerts) > 0 {
		anomalies = append(anomalies, "**磁盘告警**:\n```\n"+strings.Join(diskAlerts, "\n")+"\n```")
	}
	if out := utils.ExecuteShell("uptime | awk -F'load average:' '{ print $2 }' | awk '{ if ($1 > 4.0) print $0 }'"); strings.TrimSpace(out) != "" && !strings.Contains(out, "exit status") {
		anomalies = append(anomalies, "**高负载**:\n```\n"+strings.TrimSpace(out)+"\n```")
	}
	dmesgOut := utils.ExecuteShell("dmesg | grep -i 'out of memory' | tail -n 5")
	if !strings.Contains(dmesgOut, "Operation not permitted") && !strings.Contains(dmesgOut, "不允许的操作") && strings.TrimSpace(dmesgOut) != "" && !strings.Contains(dmesgOut, "exit status") {
		anomalies = append(anomalies, "**OOM日志**:\n```\n"+strings.TrimSpace(dmesgOut)+"\n```")
	}
	rawZombies := utils.ExecuteShell("ps -A -o stat,ppid,pid,cmd | awk '$1 ~ /^[Zz]/'")
	if strings.TrimSpace(rawZombies) != "" && !strings.Contains(rawZombies, "exit status") {
		detailZombie := "STAT    PPID     PID CMD\n" + rawZombies
		anomalies = append(anomalies, "**僵尸进程**:\n```\n"+strings.TrimSpace(detailZombie)+"\n```")
	}

	for _, rule := range config.GlobalConfig.PatrolRules {
		out := utils.ExecuteShell(rule.Command)
		if strings.TrimSpace(out) != "" && !strings.Contains(out, "exit status") {
			logger.Info(fmt.Sprintf("⚠️ 触发自定义规则: %s", rule.Name))
			anomalies = append(anomalies, fmt.Sprintf("**%s**:\n```\n%s\n```", rule.Name, strings.TrimSpace(out)))
		}
	}

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

func sendSystemStatus() {
	// 检查是否有配置通知渠道
	if config.GlobalConfig.DingTalkWebhook == "" && 
	   (config.GlobalConfig.TelegramToken == "" || config.GlobalConfig.TelegramChatID == "") {
		logger.Info("⚠️ 未配置通知渠道，跳过日报发送")
		return
	}
	
	hostname := utils.GetHostname()
	
	// 获取IP地址（多种方法尝试）
	ip := strings.TrimSpace(utils.ExecuteShell("ip route get 1 2>/dev/null | awk '{print $7; exit}' || hostname -I 2>/dev/null | awk '{print $1}' || echo 'N/A'"))
	if ip == "" || strings.Contains(ip, "exit status") {
		ip = "N/A"
	}
	
	// 获取运行时间
	uptime := strings.TrimSpace(utils.ExecuteShell("uptime -p 2>/dev/null || uptime | awk -F'up' '{print $2}' | awk '{print $1,$2,$3}'"))
	if uptime == "" || strings.Contains(uptime, "exit status") {
		uptime = "N/A"
	}
	
	// 获取内存信息
	memInfo := strings.TrimSpace(utils.ExecuteShell("free -m | awk 'NR==2{printf \"%.1f%% (已用 %sM / 总计 %sM)\", $3/$2*100, $3, $2}'"))
	if memInfo == "" || strings.Contains(memInfo, "exit status") {
		memInfo = "N/A"
	}
	
	// 获取磁盘信息（只检查根目录，过滤掉 loop 设备）
	diskInfo := strings.TrimSpace(utils.ExecuteShell("df -h / 2>/dev/null | awk 'NR==2 {print $5 \" (剩余 \" $4 \")\"}'"))
	if diskInfo == "" || strings.Contains(diskInfo, "exit status") {
		diskInfo = "N/A"
	}
	
	// 获取负载信息
	loadInfo := strings.TrimSpace(utils.ExecuteShell("uptime | awk -F'load average:' '{ print $2 }' | sed 's/^ *//'"))
	if loadInfo == "" || strings.Contains(loadInfo, "exit status") {
		loadInfo = "N/A"
	}
	
	// 获取TCP连接数（多种方法尝试）
	tcpConn := strings.TrimSpace(utils.ExecuteShell("ss -s 2>/dev/null | grep 'TCP:' | grep -oE 'estab [0-9]+' | awk '{print $2}' || netstat -ant 2>/dev/null | grep ESTABLISHED | wc -l || echo '0'"))
	if tcpConn == "" || strings.Contains(tcpConn, "exit status") {
		tcpConn = "0"
	}
	
	// 获取当前时间
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	
	report := fmt.Sprintf(`### 📊 服务器状态日报 [%s]

> **IP**: %s  
> **运行时间**: %s  
> **报告时间**: %s

---

| 指标 | 状态 |
| :--- | :--- |
| **CPU负载** | %s |
| **内存使用** | %s |
| **系统磁盘** | %s |
| **TCP连接** | %s |

---

*qwq AIOps 自动监控*
`, hostname, ip, uptime, currentTime, loadInfo, memInfo, diskInfo, tcpConn)
	
	notify.Send("服务器状态日报", report)
	logger.Info("✅ 健康日报已发送 [%s]", hostname)
}