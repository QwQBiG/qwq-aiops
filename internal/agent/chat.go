package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"qwq/internal/config"
	"qwq/internal/utils"
	"regexp"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	DefaultModel   = "Qwen/Qwen2.5-7B-Instruct"
	DefaultBaseURL = "https://api.siliconflow.cn/v1"
	Version        = "v3.1.0 Enterprise"
)

var Client *openai.Client

func InitClient() {
	cfg := openai.DefaultConfig(config.GlobalConfig.ApiKey)
	if config.GlobalConfig.BaseURL != "" {
		cfg.BaseURL = config.GlobalConfig.BaseURL
	} else {
		cfg.BaseURL = DefaultBaseURL
	}
	Client = openai.NewClientWithConfig(cfg)
}

var Tools = []openai.Tool{
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "execute_shell_command",
			Description: "Execute a shell command on the local Linux/MacOS system.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": { "type": "string", "description": "The shell command" },
					"reason": { "type": "string", "description": "The reason" }
				},
				"required": ["command", "reason"]
			}`),
		},
	},
}

// 拦截器
func CheckStaticResponse(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	
	// 1. 身份/版本类
	if input == "你好" || input == "你是谁" || input == "版本" || input == "version" || input == "whoami" || strings.Contains(input, "介绍") {
		return fmt.Sprintf(`**qwq-aiops %s**
--------------------------------
我是您的私有化智能运维专家。

**核心能力：**
1. 🛠️ **自动巡检**：监控系统负载、Docker、K8s 状态。
2. ⚡ **命令执行**：直接执行 "看看内存"、"查负载"。
3. 📝 **配置生成**：生成 YAML、Python 脚本。
4. 🔒 **安全风控**：高危命令自动拦截。

*请直接下达运维指令，例如：“看看内存” 或 “生成 nginx yaml”。*`, Version)
	}

	// 2. 帮助类
	if input == "help" || input == "帮助" || input == "能做什么" {
		return `**可用指令示例：**
- 🔍 **查询**：看看内存、查负载、看Docker容器、看K8s Pod
- ⚙️ **操作**：重启 nginx (需确认)、清理磁盘
- 📄 **生成**：写一个 busybox yaml、生成 python hello world
- 📊 **报表**：生成系统状态日报`
	}

	return ""
}

func GetBaseMessages() []openai.ChatCompletionMessage {
	knowledgePart := ""
	if config.CachedKnowledge != "" {
		knowledgePart = fmt.Sprintf("\n【内部知识库】:\n%s\n", config.CachedKnowledge)
	}

	sysPrompt := fmt.Sprintf(`你是一个 **Linux 运维终端**。
当前环境：**Linux Server**。
用户身份：**Root 管理员**。

【最高指令】
1. **查询即执行**：用户问 "内存"、"负载"、"Docker"，**必须**调用 execute_shell_command。
2. **文件生成**：
   - 用户问 "写个yaml"、"生成配置"，**只输出文件内容**。
   - **必须**使用 Markdown 代码块包裹 (e.g., `+"```yaml ... ```"+`)。
   - **严禁**输出任何解释文字（如 "好的"、"这是文件"）。
   - **严禁**输出 echo 命令。

%s`, knowledgePart)

	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
		
		// 样本 1: 运维查询
		{Role: openai.ChatMessageRoleUser, Content: "看看内存"},
		{
			Role: openai.ChatMessageRoleAssistant,
			ToolCalls: []openai.ToolCall{{
				ID: "call_1", Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{Name: "execute_shell_command", Arguments: `{"command": "free -m", "reason": "check memory"}`},
			}},
		},

		// 样本 2: 文件生成
		{Role: openai.ChatMessageRoleUser, Content: "写一个 hello.py"},
		{
			Role: openai.ChatMessageRoleAssistant,
			Content: "```python\nprint('Hello World')\n```",
		},
	}
}

func AnalyzeWithAI(issue string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	msgs := GetBaseMessages()
	msgs = append(msgs, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: issue})

	resp, err := Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: getModelName(),
		Messages: msgs,
		Temperature: 0.0,
	})
	if err != nil {
		return "AI Error: " + err.Error()
	}
	return resp.Choices[0].Message.Content
}

func ProcessAgentStep(msgs *[]openai.ChatCompletionMessage) (openai.ChatCompletionMessage, bool) {
	return ProcessAgentStepForWeb(msgs, func(log string) {
		// CLI 模式静默
	}, true)
}

func ProcessAgentStepForWeb(msgs *[]openai.ChatCompletionMessage, logCallback func(string), isCLI ...bool) (openai.ChatCompletionMessage, bool) {
	ctx := context.Background()
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	
	resp, err := Client.CreateChatCompletion(reqCtx, openai.ChatCompletionRequest{
		Model: getModelName(),
		Messages: *msgs, 
		Tools: Tools, 
		Temperature: 0.0,
	})
	
	if err != nil {
		logCallback(fmt.Sprintf("API Error: %v", err))
		return openai.ChatCompletionMessage{}, false
	}
	msg := resp.Choices[0].Message
	*msgs = append(*msgs, msg)

	// 1. 处理 Tool Calls
	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			handleToolCall(toolCall, msgs, logCallback)
		}
		return msg, true
	}

	// 2. CLI 模式：检测代码块并询问保存
	if len(isCLI) > 0 && isCLI[0] {
		filename, content := extractCodeBlock(msg.Content)
		if filename != "" && content != "" {
			fmt.Printf("\n\033[36m💾 检测到配置文件，是否保存为 '%s'? (y/N): \033[0m", filename)
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "y" || input == "yes" {
				err := os.WriteFile(filename, []byte(content), 0644)
				if err == nil {
					fmt.Printf("\033[32m✔ 文件已保存: %s\033[0m\n", filename)
				} else {
					fmt.Printf("\033[31m❌ 保存失败: %v\033[0m\n", err)
				}
			}
			return msg, true
		}
	}

	// 3. 文本回退机制 (自动捕获命令)
	cmd := extractCommandFromText(msg.Content)
	if cmd != "" {
		if isSafeAutoCommand(cmd) {
			logCallback(fmt.Sprintf("⚡ (自动捕获命令): %s", cmd))
			output := utils.ExecuteShell(cmd)
			if strings.TrimSpace(output) == "" { output = "(No output)" }
			
			feedback := fmt.Sprintf("[System Output]:\n%s", output)
			*msgs = append(*msgs, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: feedback})

			finalOutput := fmt.Sprintf("```\n%s\n```", output)
			return openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleAssistant,
				Content: finalOutput,
			}, false
		}
	}

	return msg, true
}

// CLI 后处理
func CheckAndSaveFile(content string) {
	filename, fileContent := extractCodeBlock(content)
	if filename != "" && fileContent != "" {
		fmt.Printf("\n\033[36m💾 检测到配置文件，是否保存为 '%s'? (y/N): \033[0m", filename)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "y" || input == "yes" {
			err := os.WriteFile(filename, []byte(fileContent), 0644)
			if err == nil {
				fmt.Printf("\033[32m✔ 文件已保存: %s\033[0m\n", filename)
			} else {
				fmt.Printf("\033[31m❌ 保存失败: %v\033[0m\n", err)
			}
		}
	}
}

func handleToolCall(toolCall openai.ToolCall, msgs *[]openai.ChatCompletionMessage, logCallback func(string)) {
	if toolCall.Function.Name == "execute_shell_command" {
		var args map[string]string
		json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		cmdStr := strings.TrimSpace(args["command"])
		reason := args["reason"]
		if cmdStr == "" { return }

		logCallback(fmt.Sprintf("⚡ 意图: %s", reason))
		logCallback(fmt.Sprintf("👉 命令: %s", cmdStr))

		if !utils.IsCommandSafe(cmdStr) {
			logCallback("❌ [拦截] 高危命令")
			addToolOutput(msgs, toolCall.ID, "Error: Blocked.")
			return
		}

		if utils.IsReadOnlyCommand(cmdStr) {
			// Auto run
		} else {
			logCallback("⚠️ Web模式暂不支持交互式修改命令，已跳过")
			addToolOutput(msgs, toolCall.ID, "User denied.")
			return
		}

		output := utils.ExecuteShell(cmdStr)
		if strings.TrimSpace(output) == "" { output = "(No output)" }
		addToolOutput(msgs, toolCall.ID, output)
	}
}

func addToolOutput(msgs *[]openai.ChatCompletionMessage, id, content string) {
	*msgs = append(*msgs, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleTool, Content: content, ToolCallID: id})
}

func getModelName() string {
	if config.GlobalConfig.Model != "" {
		return config.GlobalConfig.Model
	}
	return DefaultModel
}

func extractCommandFromText(text string) string {
	re := regexp.MustCompile("(?s)```(?:bash|shell|sh)?\\n(.*?)\\n```")
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	reSingle := regexp.MustCompile("`([^`]+)`")
	matchesSingle := reSingle.FindStringSubmatch(text)
	if len(matchesSingle) > 1 {
		return strings.TrimSpace(matchesSingle[1])
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) == 1 && isSafeAutoCommand(lines[0]) {
		return lines[0]
	}
	return ""
}

func extractCodeBlock(text string) (string, string) {
	re := regexp.MustCompile("(?s)```([a-zA-Z0-9]+)?\\n(.*?)\\n```")
	matches := re.FindStringSubmatch(text)
	if len(matches) > 2 {
		lang := matches[1]
		content := matches[2]
		
		// 1. 垃圾过滤
		if strings.Contains(content, "PID") || strings.Contains(content, "REPOSITORY") || 
		   strings.Contains(content, "Mem:") || strings.Contains(content, "Error") || 
		   strings.Contains(content, "<html>") || strings.Contains(content, "Usage:") {
			return "", ""
		}

		// 2. 教程过滤
		if strings.Contains(content, "sudo ") || strings.Contains(content, "apt-get") || 
		   strings.Contains(content, "docker run") || strings.Contains(content, "kubectl apply") {
			return "", ""
		}

		// 3. 特征码匹配
		isConfig := false
		if strings.Contains(content, "apiVersion:") || strings.Contains(content, "kind:") { isConfig = true }
		if strings.Contains(content, "import ") || strings.Contains(content, "def ") { isConfig = true }
		if strings.Contains(content, "{") && strings.Contains(content, "}") && strings.Contains(content, ":") { isConfig = true }
		
		if !isConfig {
			return "", ""
		}

		filename := "output.txt"
		if lang == "yaml" || lang == "yml" { filename = "config.yaml" }
		if lang == "json" { filename = "config.json" }
		if lang == "python" || lang == "py" { filename = "script.py" }
		
		if strings.Contains(text, ".yaml") {
			reFile := regexp.MustCompile(`([a-zA-Z0-9_\-]+\.yaml)`)
			if m := reFile.FindStringSubmatch(text); len(m) > 1 { filename = m[1] }
		}
		
		return filename, content
	}
	return "", ""
}

func isSafeAutoCommand(cmd string) bool {
	parts := strings.Fields(cmd)
	if len(parts) == 0 { return false }
	mainCmd := parts[0]

	whitelist := []string{
		"ls", "pwd", "cat", "head", "tail", "grep", "find",
		"ps", "top", "htop", "free", "df", "du", "uptime", "w",
		"netstat", "ss", "lsof", "ip", "ifconfig", 
		"docker", "kubectl", "systemctl", "service", "journalctl",
		"whoami", "id", "uname", "date", "history",
		"hostname",
	}

	for _, c := range whitelist {
		if mainCmd == c {
			if strings.Contains(cmd, ">") || strings.Contains(cmd, "| bash") || strings.Contains(cmd, "| sh") {
				return false
			}
			return true
		}
	}
	return false
}