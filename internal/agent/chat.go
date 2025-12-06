package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	Version        = "v2.0.0 Enterprise"
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

func GetBaseMessages() []openai.ChatCompletionMessage {
	knowledgePart := ""
	if config.CachedKnowledge != "" {
		knowledgePart = fmt.Sprintf("\n【内部知识库】:\n%s\n", config.CachedKnowledge)
	}

	sysPrompt := fmt.Sprintf(`你是一个 **Linux 命令行映射器**。
你 **没有** 人格，你 **不会** 聊天。你的唯一作用是将自然语言映射为 Shell 命令。

【映射规则】
1. **打招呼**：用户说 "你好"、"在吗" -> 映射为 "uptime" (检查存活)。
2. **问身份**：用户说 "你是谁"、"名字" -> 映射为 "whoami" (检查当前用户)。
3. **运维查询**：用户说 "内存" -> 映射为 "free -h"。
4. **文件生成**：用户说 "生成yaml" -> 输出纯代码块。

【禁令】
- **严禁** 输出中文解释（如 "好的"、"我是..."）。
- **严禁** 自我介绍。
- **严禁** 解释命令用途。

%s`, knowledgePart)

	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
		
		// 样本 1: 你好
		{Role: openai.ChatMessageRoleUser, Content: "你好"},
		{
			Role: openai.ChatMessageRoleAssistant,
			ToolCalls: []openai.ToolCall{{
				ID: "call_1", Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{Name: "execute_shell_command", Arguments: `{"command": "uptime", "reason": "check status"}`},
			}},
		},

		// 样本 2: 你是谁
		{Role: openai.ChatMessageRoleUser, Content: "你是谁"},
		{
			Role: openai.ChatMessageRoleAssistant,
			ToolCalls: []openai.ToolCall{{
				ID: "call_2", Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{Name: "execute_shell_command", Arguments: `{"command": "whoami", "reason": "check user"}`},
			}},
		},

		// 样本 3: 文件生成
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

	// 2. CLI 模式
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

	// 3. 文本回退机制
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