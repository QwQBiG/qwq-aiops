package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"qwq/internal/config"
	"qwq/internal/security"
	"qwq/internal/utils"
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
	cfg.BaseURL = DefaultBaseURL
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
					"reason": { "type": "string", "description": "The reason (in Chinese)" }
				},
				"required": ["command", "reason"]
			}`),
		},
	},
}

func AnalyzeWithAI(issue string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	safeIssue := security.Redact(issue)
	knowledgePart := ""
	if config.CachedKnowledge != "" {
		knowledgePart = fmt.Sprintf("\n【内部知识库】:\n%s\n", config.CachedKnowledge)
	}

	sysPrompt := fmt.Sprintf(`你是一个紧急故障响应专家。
规则：
1. **极度简练**：只输出核心原因和一条修复命令。
2. **拒绝废话**：不要解释原理。
3. **空数据防御**：如果输入只包含表头而没有数据，回答“误报”。
4. **僵尸进程特判**：必须杀掉父进程(PPID)。
%s`, knowledgePart)

	resp, err := Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: DefaultModel, Messages: []openai.ChatCompletionMessage{{Role: "system", Content: sysPrompt}, {Role: "user", Content: safeIssue}}, Temperature: 0.1,
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
	resp, err := Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{Model: DefaultModel, Messages: *msgs, Tools: Tools, Temperature: 0.1})
	if err != nil {
		logCallback(fmt.Sprintf("API Error: %v", err))
		return openai.ChatCompletionMessage{}, false
	}
	msg := resp.Choices[0].Message
	*msgs = append(*msgs, msg)

	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			if toolCall.Function.Name == "execute_shell_command" {
				var args map[string]string
				json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				cmdStr := strings.TrimSpace(args["command"])
				reason := args["reason"]
				if cmdStr == "" { continue }

				logCallback(fmt.Sprintf("⚡ 意图: %s", reason))
				logCallback(fmt.Sprintf("👉 命令: %s", cmdStr))

				if !utils.IsCommandSafe(cmdStr) {
					logCallback("❌ [拦截] 高危命令")
					addToolOutput(msgs, toolCall.ID, "Error: Blocked.")
					continue
				}

				if utils.IsReadOnlyCommand(cmdStr) {
					// Auto run
				} else {
					// Web mode default deny for write ops unless interactive
					logCallback("⚠️ Web模式暂不支持交互式修改命令，已跳过")
					addToolOutput(msgs, toolCall.ID, "User denied (Web mode safe guard).")
					continue
				}

				output := utils.ExecuteShell(cmdStr)
				if strings.TrimSpace(output) == "" { output = "(No output)" }
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