package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"qwq/internal/config"
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
					"command": { "type": "string", "description": "The shell command (e.g., 'ls -la', 'free -m')" },
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

	sysPrompt := fmt.Sprintf(`你是一个 Linux Shell Agent。
你运行在服务器内部。你的唯一作用是执行命令。

【绝对规则】
1. 当用户要求查询系统状态时，**必须**调用 execute_shell_command。
2. **禁止**回答“你可以使用xx命令”，而是**直接执行**该命令。
3. **禁止**列出 Windows/Mac 的操作方法。
4. 不要废话，直接干活。

%s`, knowledgePart)

	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
		
		// --- 伪造示例 1: 查磁盘 ---
		{Role: openai.ChatMessageRoleUser, Content: "磁盘空间够吗"},
		{
			Role: openai.ChatMessageRoleAssistant,
			ToolCalls: []openai.ToolCall{
				{
					ID: "call_1",
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name: "execute_shell_command",
						Arguments: `{"command": "df -h", "reason": "check disk usage"}`,
					},
				},
			},
		},
		{
			Role: openai.ChatMessageRoleTool,
			ToolCallID: "call_1",
			Content: "Filesystem Size Used Avail Use% Mounted on\n/dev/sda1 50G 10G 40G 20% /",
		},
		{Role: openai.ChatMessageRoleAssistant, Content: "磁盘空间充足，根目录使用率为 20%。"},

		// --- 伪造示例 2: 查负载 ---
		{Role: openai.ChatMessageRoleUser, Content: "看看负载"},
		{
			Role: openai.ChatMessageRoleAssistant,
			ToolCalls: []openai.ToolCall{
				{
					ID: "call_2",
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name: "execute_shell_command",
						Arguments: `{"command": "uptime", "reason": "check load average"}`,
					},
				},
			},
		},
	}
}

func AnalyzeWithAI(issue string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 使用带示例的消息列表
	msgs := GetBaseMessages()
	msgs = append(msgs, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: issue})

	resp, err := Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: getModelName(),
		Messages: msgs,
		Temperature: 0.1,
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
		Temperature: 0.1,
	})
	
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

func getModelName() string {
	if config.GlobalConfig.Model != "" {
		return config.GlobalConfig.Model
	}
	return DefaultModel
}