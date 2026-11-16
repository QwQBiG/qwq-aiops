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
	Version        = "development"
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
			Description: "Execute an allowed read-only command on the local system.",
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

func GetQuickCommand(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	// Content requests must reach the model even when their subject is a metric
	// or container. Shortcuts are only useful for direct diagnostic queries.
	if strings.Contains(input, "生成") || strings.Contains(input, "写一个") || strings.Contains(input, "写个") {
		return ""
	}
	for _, word := range strings.Fields(input) {
		if word == "write" || word == "generate" || word == "create" {
			return ""
		}
	}
	if strings.Contains(input, "qwq") || strings.Contains(input, "ops") {
		if strings.Contains(input, "镜像") {
			return "docker images | grep qwq"
		}
		return "docker ps -a | grep qwq || ps aux | grep qwq"
	}

	// 1. 状态/运行类
	if strings.Contains(input, "运行") || strings.Contains(input, "状态") || strings.Contains(input, "活") || strings.Contains(input, "挂") {
		if strings.Contains(input, "docker") || strings.Contains(input, "容器") {
			return "docker ps -a"
		}
		if strings.Contains(input, "pod") {
			return "kubectl get pods -A"
		}
		// 默认看负载
		return "top -b -n 1 | head -15"
	}

	// 2. 资源类
	if strings.Contains(input, "内存") || strings.Contains(input, "memory") {
		return "free -h"
	}
	if strings.Contains(input, "磁盘") || strings.Contains(input, "硬盘") || strings.Contains(input, "disk") {
		return "df -hT | grep -v tmpfs"
	}
	if strings.Contains(input, "负载") || strings.Contains(input, "cpu") {
		return "uptime && top -b -n 1 | head -15"
	}

	// 3. Docker/K8s 概览
	if input == "docker" || input == "容器" {
		return "docker ps -a"
	}
	if strings.Contains(input, "镜像") || strings.Contains(input, "image") {
		return "docker images"
	}
	
	return ""
}

func CheckStaticResponse(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	
	// 1. 身份/版本类
	if input == "你好" || input == "你是谁" || input == "版本" || input == "version" || input == "whoami" || input == "介绍" || input == "介绍一下自己" {
		return fmt.Sprintf(`**qwq Ops %s**
--------------------------------
我可以查询服务进程所在机器的系统状态，并协助整理配置示例。

**核心能力：**
1. 🛠️ **自动巡检**：监控系统负载、Docker、K8s 状态。
2. ⚡ **命令执行**：直接执行 "看看内存"、"查负载"。
3. 📝 **配置生成**：生成 YAML、Python 脚本。
4. 🔒 **命令边界**：仅自动执行允许的只读查询。

*请直接下达运维指令，例如：“看看内存” 或 “生成 nginx yaml”。*`, Version)
	}

	// 2. 帮助类
	if input == "help" || input == "帮助" || input == "能做什么" {
		return `**可用指令示例：**
- 🔍 **查询**：看看内存、查负载、看Docker容器、看K8s Pod
- ⚙️ **修改操作**：请在独立终端中确认后执行
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
操作权限以当前服务进程的用户权限为准。

【决策逻辑】
1. **模糊名词处理**：
   - 如果用户只说一个名词（如 "nginx", "qwq-ops", "mysql"），**默认意图是查询其运行状态**。
   - 使用只读命令，如 'ps aux' 或 'docker ps --filter name=xxx'。
   - **严禁**生成清理日志、备份等维护脚本，除非用户明确要求。

2. **查询系统状态**：
   - **必须**调用 execute_shell_command。
   - **严禁**生成 Python/Shell 脚本来查询。

3. **生成文件/代码**：
   - 只有当用户明确说 "写一个..."、"生成..."、"代码" 时。
   - 输出 Markdown 代码块。
   - **严禁**输出 echo 命令，只输出文件内容。

4. **禁止废话**：
   - 不要解释命令，不要说 "你可以使用..."。

%s`, knowledgePart)

	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
	}
}

func AnalyzeWithAI(issue string) string {
	return AnalyzeWithAIContext(context.Background(), issue)
}

func AnalyzeWithAIContext(ctx context.Context, issue string) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	msgs := GetBaseMessages()
	msgs = append(msgs, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: issue})

	resp, err := Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: getModelName(),
		Messages: msgs,
		Temperature: 0.0,
	})
	if err != nil {
		return "Model request failed: " + err.Error()
	}
	if len(resp.Choices) == 0 {
		return "Model request failed: response contained no choices"
	}
	return resp.Choices[0].Message.Content
}

func ProcessAgentStep(msgs *[]openai.ChatCompletionMessage) (openai.ChatCompletionMessage, bool) {
	return ProcessAgentStepForWeb(msgs, func(log string) {
		// CLI 模式静默
	}, true)
}

func ProcessAgentStepForWeb(msgs *[]openai.ChatCompletionMessage, logCallback func(string), isCLI ...bool) (openai.ChatCompletionMessage, bool) {
	return ProcessAgentStepForWebContext(context.Background(), msgs, logCallback, isCLI...)
}

func ProcessAgentStepForWebContext(ctx context.Context, msgs *[]openai.ChatCompletionMessage, logCallback func(string), isCLI ...bool) (openai.ChatCompletionMessage, bool) {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	
	resp, err := Client.CreateChatCompletion(reqCtx, openai.ChatCompletionRequest{
		Model: getModelName(),
		Messages: *msgs, 
		Tools: Tools, 
		Temperature: 0.0,
	})
	
	if err != nil {
		return openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleAssistant, Content: "Model request failed: " + err.Error(),
		}, false
	}
	if len(resp.Choices) == 0 {
		return openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleAssistant, Content: "Model request failed: response contained no choices",
		}, false
	}
	msg := resp.Choices[0].Message
	*msgs = append(*msgs, msg)

	// 1. 处理 Tool Calls
	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			handleToolCallContext(ctx, toolCall, msgs, logCallback)
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
			return msg, false
		}
	}

	// 3. 文本回退机制
	cmd := extractCommandFromText(msg.Content)
	if cmd != "" {
		if isSafeAutoCommand(cmd) {
			logCallback(fmt.Sprintf("⚡ (自动捕获命令): %s", cmd))
			output := utils.ExecuteShellContext(ctx, cmd)
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

	return msg, false
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
	handleToolCallContext(context.Background(), toolCall, msgs, logCallback)
}

func handleToolCallContext(ctx context.Context, toolCall openai.ToolCall, msgs *[]openai.ChatCompletionMessage, logCallback func(string)) {
	if toolCall.Function.Name != "execute_shell_command" {
		addToolOutput(msgs, toolCall.ID, "Error: unsupported tool.")
		return
	}
	if toolCall.Function.Name == "execute_shell_command" {
		var args map[string]string
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			addToolOutput(msgs, toolCall.ID, "Error: invalid command arguments.")
			return
		}
		cmdStr := strings.TrimSpace(args["command"])
		reason := args["reason"]
		if cmdStr == "" {
			addToolOutput(msgs, toolCall.ID, "Error: command is required.")
			return
		}

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

		output := utils.ExecuteShellContext(ctx, cmdStr)
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
	return utils.IsReadOnlyCommand(cmd)
}
