package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
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
)

var (
	client           *openai.Client
	renderer         *glamour.TermRenderer
	dingTalkWebhook  string
	debugMode        bool
	ErrMissingAPIKey = errors.New("critical: OPENAI_API_KEY environment variable is not set")
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
	if err != nil {
		fmt.Println("Renderer init failed:", err)
	}

	rootCmd := &cobra.Command{
		Use:   "qwq",
		Short: "Advanced AIOps Agent",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initClient()
		},
	}

	rootCmd.PersistentFlags().StringVar(&dingTalkWebhook, "webhook", "", "DingTalk Webhook URL")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logs")

	rootCmd.AddCommand(&cobra.Command{Use: "chat", Short: "Interactive Mode", Run: runChatMode})
	rootCmd.AddCommand(&cobra.Command{Use: "patrol", Short: "Patrol Mode", Run: runPatrolMode})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initClient() error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return ErrMissingAPIKey
	}
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = DefaultBaseURL
	client = openai.NewClientWithConfig(config)
	return nil
}

// ==========================================
// 模式 1: Patrol Mode (巡检)
// ==========================================

func runPatrolMode(cmd *cobra.Command, args []string) {
	printSystemMessage("巡检模式启动 (周期: 5m)")
	if dingTalkWebhook == "" {
		fmt.Println("\033[33m[警告] 未配置 Webhook，仅本地打印。\033[0m")
	}
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	performPatrol()
	
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		os.Exit(0)
	}()
	
	for range ticker.C {
		performPatrol()
	}
}

func performPatrol() {
	fmt.Printf("\n[%s] 巡检中...\n", time.Now().Format("15:04:05"))
	var anomalies []string

	// 1. 磁盘
	if out := executeShell("df -h | grep -vE '^Filesystem|tmpfs|cdrom|efivarfs|overlay' | awk 'int($5) > 85 {print $0}'"); strings.TrimSpace(out) != "" && !strings.Contains(out, "exit status") {
		anomalies = append(anomalies, "**磁盘告警**:\n```\n"+strings.TrimSpace(out)+"\n```")
	}
	// 2. 负载
	if out := executeShell("uptime | awk -F'load average:' '{ print $2 }' | awk '{ if ($1 > 4.0) print $0 }'"); strings.TrimSpace(out) != "" && !strings.Contains(out, "exit status") {
		anomalies = append(anomalies, "**高负载**:\n```\n"+strings.TrimSpace(out)+"\n```")
	}
	// 3. OOM
	dmesgOut := executeShell("dmesg | grep -i 'out of memory' | tail -n 5")
	if !strings.Contains(dmesgOut, "Operation not permitted") && !strings.Contains(dmesgOut, "不允许的操作") && strings.TrimSpace(dmesgOut) != "" && !strings.Contains(dmesgOut, "exit status") {
		anomalies = append(anomalies, "**OOM日志**:\n```\n"+strings.TrimSpace(dmesgOut)+"\n```")
	}
	// 4. 僵尸进程
	checkZombie := executeShell("ps -A -o stat,ppid,pid,cmd | awk '$1 ~ /^[Zz]/'")
	if strings.TrimSpace(checkZombie) != "" && !strings.Contains(checkZombie, "exit status") {
		detailZombie := executeShell("ps -A -o stat,ppid,pid,cmd | head -n 1; ps -A -o stat,ppid,pid,cmd | awk '$1 ~ /^[Zz]/'")
		anomalies = append(anomalies, "**僵尸进程**:\n```\n"+strings.TrimSpace(detailZombie)+"\n```")
	}

	if len(anomalies) > 0 {
		report := strings.Join(anomalies, "\n")
		analysis := analyzeWithAI(report)
		alertMsg := fmt.Sprintf("🚨 **系统告警** [%s]\n\n%s\n\n💡 **处理建议**:\n%s", getHostname(), report, analysis)
		fmt.Println(alertMsg)
		sendDingTalk(alertMsg)
	} else {
		fmt.Println("\033[32m✔ 系统健康\033[0m")
	}
}

func analyzeWithAI(issue string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sysPrompt := `你是一个紧急故障响应专家。
规则：
1. **极度简练**：只输出核心原因和一条修复命令。
2. **拒绝废话**：不要解释原理。
3. **格式固定**：
   原因：<一句话原因>
   修复：<一条核心命令>
4. **僵尸进程特判**：
   - 输入数据包含表头：STAT PPID PID CMD
   - **PPID (第二列)** 是父进程 ID。
   - 修复命令格式：kill -9 <PPID>`

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: DefaultModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: issue},
		},
		Temperature: 0.1,
	})
	if err != nil {
		return "AI 连接失败"
	}
	return resp.Choices[0].Message.Content
}

func sendDingTalk(msg string) {
	if dingTalkWebhook == "" {
		return
	}
	payload := map[string]interface{}{"msgtype": "markdown", "markdown": map[string]string{"title": "系统告警", "text": msg}}
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(dingTalkWebhook, "application/json", bytes.NewBuffer(jsonData))
	if err == nil {
		defer resp.Body.Close()
		io.ReadAll(resp.Body)
	}
}

// ==========================================
// 模式 2: Chat Mode (智能交互)
// ==========================================

func runChatMode(cmd *cobra.Command, args []string) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:      "\033[32mqwq > \033[0m",
		HistoryFile: "/tmp/qwq_history",
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	printSystemMessage("Agent Online. System: " + runtime.GOOS)
	
	// [核心优化] 增加防幻觉指令
	sysPrompt := `你是一个资深运维专家助手(qwq)。
规则：
1. 请用中文回答。
2. **分步执行**：先获取信息，再执行下一步。
3. **严禁编造**：如果命令返回 "exit status 1" 或空，说明进程不存在或命令失败，请直接告诉用户“未找到”或“失败”，**绝对不要捏造输出结果**。
4. 如果是查询类命令（如 get, describe, logs, top, ps），请放心执行。`

	messages := []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: sysPrompt}}

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			}
			continue
		}
		if err == io.EOF {
			break
		}
		input := strings.TrimSpace(line)
		if input == "exit" || input == "quit" {
			break
		}
		if input == "" {
			continue
		}

		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: input})

		for i := 0; i < 5; i++ {
			respMsg, shouldContinue := processAgentStep(&messages)
			if !shouldContinue {
				break
			}
			if respMsg.Content != "" && len(respMsg.ToolCalls) == 0 {
				renderMarkdown(respMsg.Content)
				break
			}
		}
	}
}

func processAgentStep(msgs *[]openai.ChatCompletionMessage) (openai.ChatCompletionMessage, bool) {
	ctx := context.Background()
	fmt.Print("\033[33m🤖 思考中...\033[0m\r")

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       DefaultModel,
		Messages:    *msgs,
		Tools:       tools,
		Temperature: 0.1,
	})

	if err != nil {
		fmt.Printf("\nAPI Error: %v\n", err)
		return openai.ChatCompletionMessage{}, false
	}
	msg := resp.Choices[0].Message
	*msgs = append(*msgs, msg)

	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			if toolCall.Function.Name == "execute_shell_command" {
				var args map[string]string
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					continue
				}
				cmdStr := strings.TrimSpace(args["command"])
				reason := args["reason"]
				if cmdStr == "" {
					continue
				}

				fmt.Printf("\n\033[36m⚡ 意图: %s\033[0m\n", reason)
				fmt.Printf("\033[33m👉 命令: \033[1m%s\033[0m\n", cmdStr)

				if !isCommandSafe(cmdStr) {
					fmt.Println("\033[31m[拦截] 高危命令\033[0m")
					addToolOutput(msgs, toolCall.ID, "Error: Blocked by safety policy.")
					continue
				}

				shouldAutoRun := isReadOnlyCommand(cmdStr)

				if shouldAutoRun {
					fmt.Println("\033[90m(自动执行查询命令...)\033[0m")
				} else {
					if !confirmExecution() {
						fmt.Println("\033[90m已跳过\033[0m")
						addToolOutput(msgs, toolCall.ID, "User denied.")
						continue
					}
				}

				fmt.Print("\033[90m执行中...\033[0m")
				output := executeShell(cmdStr)
				
				// [优化] 如果输出为空，明确告知 AI，防止它瞎编
				if strings.TrimSpace(output) == "" {
					output = "(Command returned no output)"
				}
				
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

// ==========================================
// 辅助函数
// ==========================================

func isReadOnlyCommand(cmd string) bool {
	safeKeywords := []string{
		"ls", "cat", "head", "tail", "grep", "find", "pwd", "echo", "whoami", "id",
		"ps", "top", "uptime", "free", "df", "du", "netstat", "ss", "lsof",
		"kubectl get", "kubectl describe", "kubectl logs", "kubectl top", "kubectl cluster-info",
		"docker ps", "docker logs", "docker stats",
	}

	c := strings.ToLower(cmd)
	for _, kw := range safeKeywords {
		if strings.HasPrefix(c, kw) || strings.Contains(c, " "+kw) {
			if !strings.Contains(c, ">") && !strings.Contains(c, "rm ") && !strings.Contains(c, "kill") && !strings.Contains(c, "delete") {
				return true
			}
		}
	}
	return false
}

// [核心修复] 捕获 Exit Code，防止 AI 幻觉
func executeShell(c string) string {
	cmd := exec.Command("bash", "-c", c)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	out, err := cmd.CombinedOutput()
	res := string(out)
	
	// 如果命令执行失败（比如 grep 没找到，ps 没找到 PID），把错误码传给 AI
	if err != nil {
		if len(res) > 0 {
			res += fmt.Sprintf("\n(Command failed: %v)", err)
		} else {
			res = fmt.Sprintf("(Command failed: %v)", err)
		}
	}
	
	if len(res) > 4000 {
		res = res[:4000] + "\n...(Output truncated)"
	}
	return res
}

func isCommandSafe(c string) bool {
	dangerous := []string{"rm -rf", "mkfs", ":(){:|:&};:", "> /dev/sda", "dd if=/dev/zero"}
	for _, d := range dangerous {
		if strings.Contains(c, d) {
			return false
		}
	}
	return true
}

func confirmExecution() bool {
	fmt.Print("\033[33m[?] 这是一个修改操作，确认执行? (Y/n): \033[0m")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "" || input == "y" || input == "yes"
}

func renderMarkdown(t string) {
	if o, e := renderer.Render(t); e == nil {
		fmt.Print(o)
	} else {
		fmt.Println(t)
	}
}
func getHostname() string { h, _ := os.Hostname(); return h }
func printSystemMessage(m string) { fmt.Printf("\033[36m(qwq) %s\033[0m\n", m) }
