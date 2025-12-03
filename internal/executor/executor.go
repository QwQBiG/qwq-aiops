package executor

import (
	"fmt"
	"os"
	"os/exec"
	"qwq/internal/agent"
	"qwq/internal/logger"

	"github.com/charmbracelet/glamour"
)

func SmartRun(cmdStr string) {
	fmt.Printf("🚀 执行命令: %s\n", cmdStr)
	
	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	
	if err == nil {
		logger.Info("✅ 命令执行成功")
		return
	}

	fmt.Println("\n❌ 命令执行失败，正在请求 AI 分析原因...")
	
	out, _ := exec.Command("bash", "-c", cmdStr).CombinedOutput()
	errorLog := string(out)

	prompt := fmt.Sprintf(`我执行了命令 "%s" 失败了。
报错信息如下：
%s

请分析原因，并直接给出修复后的正确命令。
格式要求：
原因：...
建议命令：...`, cmdStr, errorLog)

	suggestion := agent.AnalyzeWithAI(prompt)

	r, _ := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(100))
	rendered, _ := r.Render(suggestion)
	fmt.Println(rendered)
}