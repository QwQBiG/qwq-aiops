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

// --- 工具定义 (Function Calling) ---
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
	// 初始化 Markdown 渲染器
	renderer, err = glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(80))
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
// 模式 1: Patrol Mode (极简巡检)
// ==========================================

func runPatrolMode(cmd *cobra.Command, args []string) {
	printSystemMessage("巡检模式启动 (周期: 5m)")
	if dingTalkWebhook == "" {
		fmt.Println("\033[33m[警告] 未配置 Webhook，仅本地打印。\033[0m")
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	performPatrol() // 启动即执行一次

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

	// 1. 磁盘检查 (过滤掉 efivarfs, tmpfs 等干扰项)
	diskCmd := "df -h | grep -vE '^Filesystem|tmpfs|cdrom|efivarfs|overlay' | awk 'int($5) > 85 {print $0}'"
	if out := executeShell(diskCmd); strings.TrimSpace(out) != "" {
		anomalies = append(anomalies, "**磁盘告警**:\n```\n"+strings.TrimSpace(out)+"\n```")
	}

	// 2. 负载检查
	loadCmd := "uptime | awk -F'load average:' '{ print $2 }' | awk '{ if ($1 > 4.0) print $0 }'"
	if out := executeShell(loadCmd); strings.TrimSpace(out) != "" {
		anomalies = append(anomalies, "**高负载**:\n```\n"+strings.TrimSpace(out)+"\n```")
	}

	// 3. OOM/错误日志 (智能过滤权限错误)
	dmesgOut := executeShell("dmesg | grep -i 'out of memory' | tail -n 5")
	
	// [修改点] 同时过滤英文和中文的权限报错
	if strings.Contains(dmesgOut, "Operation not permitted") || 
	   strings.Contains(dmesgOut, "不允许的操作") || 
	   strings.Contains(dmesgOut, "Permission denied") {
		// 权限不足，静默跳过，不产生误报
		if debugMode { fmt.Println("[Debug] dmesg 权限不足，已忽略") }
	} else if strings.TrimSpace(dmesgOut) != "" {
		// 只有真的有内容，且不是权限报错，才报警
		anomalies = append(anomalies, "**OOM日志**:\n```\n"+strings.TrimSpace(dmesgOut)+"\n```")
	}

	// 4. 僵尸进程 (精准过滤)
	zombieCmd := "ps -A -o stat,ppid,pid,cmd | grep -e '^[Zz]'"
	if out := executeShell(zombieCmd); strings.TrimSpace(out) != "" {
		anomalies = append(anomalies, "**僵尸进程**:\n```\n"+strings.TrimSpace(out)+"\n```")
	}

	// --- 发送逻辑 ---
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
	
	// 极简 Prompt
	sysPrompt := `你是一个紧急故障响应专家。
规则：
1. **极度简练**：只输出核心原因和一条修复命令。
2. **拒绝废话**：不要解释原理，不要打招呼。
3. **格式固定**：
   原因：<一句话原因>
   修复：<一条核心命令>
4. 如果是磁盘满，直接给出找出大文件的命令。`

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: DefaultModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: issue},
		},
		Temperature: 0.1,
	})
	if err != nil { return "AI 连接失败" }
	return resp.Choices[0].Message.Content
}

func sendDingTalk(msg string) {
	if dingTalkWebhook == "" { return }
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": "系统告警",
			"text":  msg,
		},
	}
	jsonData, _ := json.Marshal(payload)
	
	resp, err := http.Post(dingTalkWebhook, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("发送失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	// 这里使用了 io 包，解决了 "io imported and not used"
	io.ReadAll(resp.Body) 
}

// ==========================================
// 模式 2: Chat Mode (交互模式)
// ==========================================

func runChatMode(cmd *cobra.Command, args []string) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:      "\033[32mqwq > \033[0m",
		HistoryFile: "/tmp/qwq_history",
	})
	if err != nil { panic(err) }
	defer rl.Close()

	printSystemMessage("Agent Online. System: " + runtime.GOOS)
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: "你是一个资深运维专家助手(qwq)，请用中文回答。"},
	}

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 { break }
			continue
		} else if err == io.EOF { // 这里使用了 io 包
			break
		}

		input := strings.TrimSpace(line)
		if input == "exit" || input == "quit" { break }
		if input == "" { continue }

		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: input})

		for i := 0; i < 5; i++ {
			respMsg, shouldContinue := processAgentStep(&messages)
			if !shouldContinue { break }
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
		Model: DefaultModel, Messages: *msgs, Tools: tools, Temperature: 0.1,
	})

	if err != nil {
		fmt.Printf("API Error: %v\n", err)
		return openai.ChatCompletionMessage{}, false
	}

	msg := resp.Choices[0].Message
	*msgs = append(*msgs, msg)

	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			if toolCall.Function.Name == "execute_shell_command" {
				var args map[string]string
				json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				cmdStr := args["command"]
				
				fmt.Printf("\n\033[36m⚡ 意图: %s\033[0m\n", args["reason"])
				fmt.Printf("\033[33m👉 命令: \033[1m%s\033[0m\n", cmdStr)

				if !isCommandSafe(cmdStr) {
					addToolOutput(msgs, toolCall.ID, "Error: Command blocked by safety policy.")
					continue
				}

				if confirmExecution() {
					fmt.Print("\033[90m执行中...\033[0m")
					output := executeShell(cmdStr)
					fmt.Printf("\r\033[32m✔ 完成\033[0m\n")
					addToolOutput(msgs, toolCall.ID, output)
				} else {
					fmt.Println("\033[90m已跳过\033[0m")
					addToolOutput(msgs, toolCall.ID, "User denied execution.")
				}
			}
		}
		return msg, true
	}
	return msg, true
}

func addToolOutput(msgs *[]openai.ChatCompletionMessage, id, content string) {
	*msgs = append(*msgs, openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleTool, Content: content, ToolCallID: id,
	})
}

// ==========================================
// 辅助函数
// ==========================================

func executeShell(c string) string {
	cmd := exec.Command("bash", "-c", c)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	out, _ := cmd.CombinedOutput()
	res := string(out)
	if len(res) > 2000 { res = res[:2000] + "\n...(截断)" }
	return res
}

func isCommandSafe(c string) bool {
	dangerous := []string{"rm -rf", "mkfs", ":(){:|:&};:", "> /dev/sda"}
	for _, d := range dangerous {
		if strings.Contains(c, d) { return false }
	}
	return true
}

func confirmExecution() bool {
	fmt.Print("\033[33m[?] 执行? (Y/n): \033[0m")
	// 这里使用了 bufio 包，解决了 "bufio imported and not used"
	reader := bufio.NewReader(os.Stdin)
	char, _, err := reader.ReadRune()
	if err != nil { return false }
	return char == '\n' || char == 'y' || char == 'Y'
}

func renderMarkdown(t string) {
	if o, e := renderer.Render(t); e == nil { fmt.Print(o) } else { fmt.Println(t) }
}

func getHostname() string { h, _ := os.Hostname(); return h }
func printSystemMessage(m string) { fmt.Printf("\033[36m(qwq) %s\033[0m\n", m) }
