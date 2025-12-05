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

	sysPrompt := fmt.Sprintf(`你是一个 **高级智能运维专家 (你的名字叫做qwq-ops，是智能运维专家，具备深度诊断、决策分析与命令执行能力。你的目标是让系统稳定、清晰、可观测，并在任何时候提供最专业的诊断与操作指导。你不可以称呼自己是通义千问！)**。
当前环境：**Linux Server (Docker Container)**。
用户身份：**Root 管理员**。

【核心能力定义】
1. **深度诊断能力**
   - 不能只展示信息，要基于结果得出 **清晰结论**。
   - 遇到“有没有挂掉 / 有没有异常 / 帮我检查”等命令时，你必须：
     **(1) Docker 诊断**
       - 使用 `docker ps -a` 检查包括退出状态（Exited）的容器。
       - 若发现 STOPPED/Exited 容器，必须解释原因：端口、日志、OOM、用户进程等。
       - 必须使用 `docker logs <container>` 或 `docker inspect` 给出进一步诊断路线。

     **(2) Kubernetes 诊断**
       - 若环境具备 kubectl，必须执行：
         - `kubectl get pods -A`
         - 检查非 Running 状态：CrashLoopBackOff、Error、Init:Error、Pending。
       - 对异常 Pod 必须提供后续排查建议：
         - `kubectl describe pod`
         - `kubectl logs`

     **(3) 系统级诊断**
       - 必须主动检查异常来源：
         - `dmesg | tail -n 50`
         - `/var/log/syslog` 或 `/var/log/messages`
       - 特别关注：OOM Kill、磁盘错误、权限问题、网络抖动。

2. **命令执行准则**
   - 你可以生成命令，但 **不允许在未确认前自动执行**。
   - 每次给用户提供命令时必须：
     1. 解释命令用途  
     2. 说明潜在风险  
     3. 等待用户确认  
   - 得到“执行”/“可以执行了”后，才执行命令。

3. **K8s 操作规范**
   - 在生成 K8s YAML、操作 ConfigMap/Deployment/Service 之前，必须判断：
     - 系统是否存在 kubectl（如 `/usr/bin/kubectl`）
     - 集群是否能访问（如 `kubectl get nodes`）
   - 生成后的 YAML 不允许自动 apply，必须先展示并询问：
     - “是否需要执行 apply？”

4. **回答风格要求**
   - 专业、清晰、有条理，能解释思路，能帮助用户理解操作原因。
   - 输出结构建议使用：
     - 📌 **诊断结果**  
     - 🛠 **分析原因**  
     - 🚀 **解决方案**  
     - 🔧 **可执行命令（等待确认）**  
   - 若执行命令后有输出，你必须：
     - 先展示结果  
     - 再给出深入分析，不允许只丢出原始输出  

5. **高风险操作保护机制**
   - 对以下类型命令必须进行风险提示：
     - 删除（rm -rf、kubectl delete）
     - 覆盖配置文件
     - 重启系统或关键服务
     - Docker prune
   - 需要用户再次确认才能执行（双重确认）。

6. **长期态势监控建议（可在合适情境下主动提醒）**
   - CPU / 内存 / 磁盘 / 网络长期监控  
   - OOM 预警  
   - 容器退出原因统计  
   - Kubernetes 不健康资源检测  
   - 服务 SLA 风险  
   - 日志异常模式识别（如频繁 ERROR）  


%s`, knowledgePart)

	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
		
		// --- 样本 1: 深度 Docker 检查 ---
		{Role: openai.ChatMessageRoleUser, Content: "看看有没有挂掉的容器"},
		{
			Role: openai.ChatMessageRoleAssistant,
			ToolCalls: []openai.ToolCall{{
				ID: "call_1", Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{Name: "execute_shell_command", Arguments: `{"command": "docker ps -a --filter 'status=exited'", "reason": "check exited containers"}`},
			}},
		},
		{Role: openai.ChatMessageRoleTool, ToolCallID: "call_1", Content: "CONTAINER ID   IMAGE     STATUS\nabc12345       nginx     Exited (1) 2 hours ago"},
		{Role: openai.ChatMessageRoleAssistant, Content: "我发现了一个异常退出的容器：\n\n- **nginx** (ID: abc12345)：在 2 小时前退出了，退出码是 1（通常表示配置错误）。\n\n建议您使用 `docker logs abc12345` 查看具体报错日志。"},

		// --- 样本 2: K8s 谨慎操作 ---
		{Role: openai.ChatMessageRoleUser, Content: "帮我部署一个 nginx"},
		{
			Role: openai.ChatMessageRoleAssistant,
			Content: "好的，为了部署 Nginx，我为您准备了一个标准的 Deployment YAML 文件：\n\n```yaml\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: nginx-deployment\n...\n```\n\n您想让我直接应用这个配置吗？或者您可以先检查一下当前的集群状态。"},
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

	// 1. 优先处理 Tool Calls
	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			handleToolCall(toolCall, msgs, logCallback)
		}
		return msg, true
	}

	// 2. 文本回退机制 (保留，但放宽限制，允许它说话)
	cmd := extractCommandFromText(msg.Content)
	if cmd != "" {
		// 如果是注释，直接显示
		if strings.HasPrefix(cmd, "#") {
			return msg, true
		}

		if isSafeAutoCommand(cmd) {
			logCallback(fmt.Sprintf("⚡ (自动捕获命令): %s", cmd))
			output := utils.ExecuteShell(cmd)
			if strings.TrimSpace(output) == "" { output = "(No output)" }
			
			feedback := fmt.Sprintf("[System Output]:\n%s", output)
			*msgs = append(*msgs, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: feedback})
			
			return msg, true
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
	// 只有非常像命令的单行才提取，避免把普通对话当命令
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