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
	Version        = "v3.3.0 Enterprise"
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

func GetQuickCommand(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	
	// --- 1. Kubernetes 专区 ---
	// 节点状态 (详细信息)
	if strings.Contains(input, "node") || strings.Contains(input, "节点") {
		return "kubectl get nodes -o wide"
	}
	// Pod 状态 (所有命名空间 + IP信息)
	if strings.Contains(input, "pod") || strings.Contains(input, "容器组") {
		return "kubectl get pods -A -o wide --sort-by=.metadata.creationTimestamp"
	}
	// Service 服务
	if strings.Contains(input, "svc") || strings.Contains(input, "service") || strings.Contains(input, "服务") {
		return "kubectl get svc -A"
	}
	// Deployment 部署
	if strings.Contains(input, "deploy") || strings.Contains(input, "部署") {
		return "kubectl get deploy -A"
	}
	// Events 事件 (排查 K8s 报错，按时间倒序)
	if strings.Contains(input, "event") || strings.Contains(input, "事件") || strings.Contains(input, "k8s报错") {
		return "kubectl get events -A --sort-by='.lastTimestamp' | tail -n 20"
	}
	// 集群信息
	if input == "k8s" || strings.Contains(input, "集群信息") || strings.Contains(input, "cluster") {
		return "kubectl cluster-info"
	}
	// 资源使用情况 (Top)
	if strings.Contains(input, "k8s资源") || strings.Contains(input, "pod内存") || strings.Contains(input, "pod cpu") {
		return "kubectl top pods -A --sort-by=cpu | head -n 15"
	}

	// --- 2. Docker 专区 ---
	// 容器列表 (包含退出的)
	if input == "docker" || input == "看看docker" || input == "docker容器" {
		return "docker ps -a"
	}
	// 镜像列表
	if strings.Contains(input, "镜像") || strings.Contains(input, "image") {
		return "docker images"
	}
	// 容器资源统计 (实时)
	if strings.Contains(input, "docker资源") || strings.Contains(input, "容器内存") {
		return "docker stats --no-stream --format 'table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}'"
	}

	// --- 3. Linux 基础资源 ---
	// 内存
	if strings.Contains(input, "内存") || strings.Contains(input, "memory") {
		return "free -h"
	}
	// 磁盘
	if strings.Contains(input, "磁盘") || strings.Contains(input, "硬盘") || strings.Contains(input, "disk") {
		return "df -hT | grep -v tmpfs" // 排除 tmpfs 干扰
	}
	// 负载/CPU
	if strings.Contains(input, "负载") || strings.Contains(input, "cpu") || strings.Contains(input, "load") {
		return "top -b -n 1 | head -n 15" // 只看前15行
	}
	// 进程 (按 CPU 排序)
	if strings.Contains(input, "进程") && !strings.Contains(input, "杀") {
		return "ps aux --sort=-%cpu | head -n 15"
	}

	// --- 4. 网络与系统信息 ---
	// 端口监听
	if strings.Contains(input, "端口") || strings.Contains(input, "port") || strings.Contains(input, "监听") {
		return "netstat -tulpn"
	}
	// 网络连接数统计
	if strings.Contains(input, "连接数") || strings.Contains(input, "并发") {
		return "netstat -ant | awk '{print $6}' | sort | uniq -c | sort -rn"
	}
	// IP 地址
	if input == "ip" || strings.Contains(input, "ip地址") {
		return "ip -4 a | grep inet | grep -v 127.0.0.1"
	}
	// 系统版本
	if strings.Contains(input, "系统") || strings.Contains(input, "os") || strings.Contains(input, "发行版") {
		return "cat /etc/os-release"
	}
	// 内核版本
	if strings.Contains(input, "内核") || strings.Contains(input, "kernel") {
		return "uname -sr"
	}
	// 登录用户
	if strings.Contains(input, "用户") || strings.Contains(input, "who") {
		return "w"
	}
	
	return ""
}

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

【决策逻辑】
1. **查询系统状态**：
   - **必须**调用 execute_shell_command。
   - **严禁**生成 Python/Shell 脚本来查询，直接用系统命令。

2. **生成文件/代码**：
   - 只有当用户明确说 "写一个..."、"生成..."、"代码" 时。
   - 输出 Markdown 代码块。
   - **严禁**输出 echo 命令，只输出文件内容。

3. **禁止废话**：
   - 不要解释命令，不要说 "你可以使用..."。

%s`, knowledgePart)

	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
		
		// 样本 1: 运维查询
		{Role: openai.ChatMessageRoleUser, Content: "分析一下 nginx 为什么挂了"},
		{
			Role: openai.ChatMessageRoleAssistant,
			ToolCalls: []openai.ToolCall{{
				ID: "call_1", Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{Name: "execute_shell_command", Arguments: `{"command": "systemctl status nginx || docker logs nginx", "reason": "check nginx status"}`},
			}},
		},

		// 样本 2: 代码生成
		{Role: openai.ChatMessageRoleUser, Content: "写一个清理日志的脚本"},
		{
			Role: openai.ChatMessageRoleAssistant,
			Content: "```bash\n#!/bin/bash\nfind /var/log -name \"*.log\" -mtime +7 -delete\n```",
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