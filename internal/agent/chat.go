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

	sysPrompt := fmt.Sprintf(`你是一个 **Linux Shell 命令翻译器**。
你的唯一任务是将用户的自然语言翻译成 **Linux Shell 命令**。

【负面约束 (绝对禁止)】
1. **禁止** 编写 Python、Java、C++ 代码。
2. **禁止** 编写 Dockerfile 或配置文件。
3. **禁止** 提供教程、解释或步骤说明。
4. **禁止** 使用 markdown 标题（如 ###）。

【输出规则】
1. 如果用户想查询状态，直接输出查询命令（如 free -m, docker ps）。
2. 如果用户想操作，直接输出操作命令。
3. 必须将命令包裹在 `+"```bash"+` 代码块中。
4. 对话要说中文。

%s`, knowledgePart)

	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
		
		// 强力样本 1
		{Role: openai.ChatMessageRoleUser, Content: "看看内存"},
		{Role: openai.ChatMessageRoleAssistant, Content: "```bash\nfree -h\n```"},

		// 强力样本 2
		{Role: openai.ChatMessageRoleUser, Content: "查看docker镜像"},
		{Role: openai.ChatMessageRoleAssistant, Content: "```bash\ndocker images\n```"},

		// 强力样本 3
		{Role: openai.ChatMessageRoleUser, Content: "系统负载高吗"},
		{Role: openai.ChatMessageRoleAssistant, Content: "```bash\nuptime\n```"},
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
		return "AI 连接失败: " + err.Error()
	}
	return resp.Choices[0].Message.Content
}

func ProcessAgentStep(msgs *[]openai.ChatCompletionMessage) (openai.ChatCompletionMessage, bool) {
	return ProcessAgentStepForWeb(msgs, func(log string) {
		fmt.Println(log)
	})
}

func ProcessAgentStepForWeb(msgs *[]openai.ChatCompletionMessage, logCallback func(string)) (openai.ChatCompletionMessage, bool) {
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


	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			handleToolCall(toolCall, msgs, logCallback)
		}
		return msg, true
	}


	cmd := extractCommandFromText(msg.Content)
	if cmd != "" {

		if isSafeAutoCommand(cmd) {
			logCallback(fmt.Sprintf("⚡ (自动捕获命令): %s", cmd))
			output := utils.ExecuteShell(cmd)
			if strings.TrimSpace(output) == "" { output = "(No output)" }
			
			finalOutput := fmt.Sprintf("```\n%s\n```", output)
			return openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleAssistant,
				Content: finalOutput,
			}, false
		} else {

			return msg, false
		}
	}

	return msg, true
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