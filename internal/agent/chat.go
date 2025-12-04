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

func getModelName() string {
	if config.GlobalConfig.Model != "" {
		return config.GlobalConfig.Model
	}
	return DefaultModel
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

func AnalyzeWithAI(issue string) string {
	// 保持 5 分钟超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	knowledgePart := ""
	if config.CachedKnowledge != "" {
		knowledgePart = fmt.Sprintf("\n【内部知识库】:\n%s\n", config.CachedKnowledge)
	}

	sysPrompt := fmt.Sprintf(`你是一个 Linux 系统故障排查专家。
你收到的输入是系统自动巡检发现的异常日志。

【分析规则】
1. **高负载 (Load Average)**：输入格式通常为 "1.2, 3.4, 5.6"。这是 Linux Load Average (1min, 5min, 15min)。
   - **不要**把它当成百分比！
   - 如果数值超过 4.0 (假设4核CPU)，视为高负载。
   - 建议使用 'top -b -n 1 | head -20' 查看进程。
2. **僵尸进程**：输入包含 STAT 为 Z 的进程。
   - 必须建议杀掉父进程 (PPID)。
   - 修复命令：kill -9 <PPID>
3. **极简输出**：直接给出原因和一条核心修复/排查命令，不要写废话。

%s`, knowledgePart)

	resp, err := Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: getModelName(),
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: sysPrompt}, 
			{Role: "user", Content: issue},
		}, 
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