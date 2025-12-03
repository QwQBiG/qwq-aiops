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
	// [修改] 使用 GlobalConfig
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
		Model: DefaultModel, Messages: []openai.ChatCompletionMessage{{Role: "system", Content: sysPrompt}, {Role: "user", Content: issue}}, Temperature: 0.1,
	})
	if err != nil {
		return "AI 连接失败: " + err.Error()
	}
	return resp.Choices[0].Message.Content
}

// ProcessAgentStep 处理 Chat 模式的单步逻辑
func ProcessAgentStep(msgs *[]openai.ChatCompletionMessage) (openai.ChatCompletionMessage, bool) {
	ctx := context.Background()
	fmt.Print("\033[33m🤖 思考中...\033[0m\r")
	resp, err := Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{Model: DefaultModel, Messages: *msgs, Tools: Tools, Temperature: 0.1})
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
				json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				cmdStr := strings.TrimSpace(args["command"])
				reason := args["reason"]
				if cmdStr == "" { continue }

				fmt.Printf("\n\033[36m⚡ 意图: %s\033[0m\n", reason)
				fmt.Printf("\033[33m👉 命令: \033[1m%s\033[0m\n", cmdStr)

				if !utils.IsCommandSafe(cmdStr) {
					fmt.Println("\033[31m[拦截] 高危命令\033[0m")
					addToolOutput(msgs, toolCall.ID, "Error: Blocked.")
					continue
				}

				if utils.IsReadOnlyCommand(cmdStr) {
					fmt.Println("\033[90m(自动执行查询命令...)\033[0m")
				} else {
					if !utils.ConfirmExecution() {
						fmt.Println("\033[90m已跳过\033[0m")
						addToolOutput(msgs, toolCall.ID, "User denied.")
						continue
					}
				}

				fmt.Print("\033[90m执行中...\033[0m")
				output := utils.ExecuteShell(cmdStr)
				if strings.TrimSpace(output) == "" { output = "(No output)" }
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