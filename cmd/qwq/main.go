// hello world
package main

import (
	"fmt"
	"os"
	"os/signal"
	"qwq/internal/agent"
	"qwq/internal/config"
	"qwq/internal/executor"
	"qwq/internal/logger"
	"qwq/internal/monitor"
	"qwq/internal/notify"
	"qwq/internal/security"
	"qwq/internal/server"
	"qwq/internal/utils"
	"runtime"
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

func runWebMode(cmd *cobra.Command, args []string) {
	server.TriggerPatrolFunc = performPatrol
	server.TriggerStatusFunc = sendSystemStatus
	go runPatrolLoop(8 * time.Hour)
	go sendSystemStatus()
	server.Start(":8899")
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
	
	knowledgePart := ""
	if config.CachedKnowledge != "" {
		knowledgePart = fmt.Sprintf("\n【内部知识库】:\n%s\n", config.CachedKnowledge)
	}

	sysPrompt := fmt.Sprintf(`你是一个资深运维专家助手(qwq)。
规则：
1. 请用中文回答。
2. **分步执行**：先获取信息，再执行下一步。
3. **严禁编造**：如果命令返回 "exit status 1" 或空，说明进程不存在或命令失败。
4. 如果是查询类命令（如 get, describe, logs, top, ps），请放心执行。
%s`, knowledgePart)

	messages := []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: sysPrompt}}
	for {
		line, _ := rl.Readline()
		if line == "exit" { break }
		if line == "" { continue }
		
		// [关键修复] 使用 security 包进行脱敏，并真正使用 safeInput
		safeInput := security.Redact(line)
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: safeInput})
		
		for i := 0; i < 5; i++ {
			respMsg, cont := agent.ProcessAgentStep(&messages)
			if !cont { break }
			if respMsg.Content != "" && len(respMsg.ToolCalls) == 0 {
				r, _ := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(100))
				out, _ := r.Render(respMsg.Content)
				fmt.Print(out)
				break
			}
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
	performPatrol()
	for {
		select {
		case <-checkTicker.C: performPatrol()
		case <-reportTicker.C: sendSystemStatus()
		}
	}
}

func performPatrol() {
	logger.Info("正在执行系统巡检...")
	var anomalies []string

	if out := utils.ExecuteShell("df -h | grep -vE '^Filesystem|tmpfs|cdrom|efivarfs|overlay' | awk 'int($5) > 85 {print $0}'"); strings.TrimSpace(out) != "" && !strings.Contains(out, "exit status") {
		anomalies = append(anomalies, "**磁盘告警**:\n```\n"+strings.TrimSpace(out)+"\n```")
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