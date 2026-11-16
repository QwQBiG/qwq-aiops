package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"qwq/internal/agent"
	"qwq/internal/logger"
	"qwq/internal/security"
	"qwq/internal/utils"
	"time"

	"github.com/charmbracelet/glamour"
)

func SmartRun(cmdStr string) {
	_ = SmartRunContext(context.Background(), cmdStr)
}

// SmartRunContext executes once and uses that same attempt's output for diagnostics.
func SmartRunContext(ctx context.Context, cmdStr string) error {
	fmt.Printf("🚀 执行命令: %s\n", cmdStr)
	output, err := runCommand(ctx, cmdStr, os.Stdout, os.Stderr)

	if err == nil {
		logger.Info("✅ 命令执行成功")
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	fmt.Println("\n❌ 命令执行失败，正在分析原因...")

	prompt := fmt.Sprintf(`我执行了命令 "%s" 失败了。
报错信息如下：
%s

请分析原因，并直接给出修复后的正确命令。
格式要求：
原因：...
建议命令：...`, security.Redact(cmdStr), security.Redact(output))

	suggestion := agent.AnalyzeWithAIContext(ctx, prompt)

	r, renderErr := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(100))
	if renderErr == nil {
		if rendered, renderErr := r.Render(suggestion); renderErr == nil {
			fmt.Println(rendered)
			return err
		}
	}
	fmt.Println(suggestion)
	return err
}

func runCommand(ctx context.Context, command string, stdout, stderr io.Writer) (string, error) {
	output := &utils.CommandOutput{}
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.WaitDelay = time.Second
	cmd.Stdout = io.MultiWriter(stdout, output)
	cmd.Stderr = io.MultiWriter(stderr, output)
	err := cmd.Run()
	return output.String(), err
}
