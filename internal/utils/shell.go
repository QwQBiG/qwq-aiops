package utils

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"qwq/internal/security"
	"strings"
	"time"
)

const CommandTimeout = 60 * time.Second

func ExecuteShell(c string) string {
	return ExecuteShellContext(context.Background(), c)
}

func ExecuteShellContext(parent context.Context, c string) string {
	ctx, cancel := context.WithTimeout(parent, CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", c)
	cmd.WaitDelay = time.Second
	output := &commandOutput{}
	cmd.Stdout, cmd.Stderr = output, output
	err := cmd.Run()
	res := output.String()

	if ctx.Err() == context.DeadlineExceeded {
		res += "\n(Command timed out)"
	} else if ctx.Err() == context.Canceled {
		res += "\n(Command canceled)"
	} else if err != nil {
		if len(res) > 0 {
			res += fmt.Sprintf("\n(Command failed: %v)", err)
		} else {
			res = fmt.Sprintf("(Command failed: %v)", err)
		}
	}
	return res
}

func CheckK8sConnection() bool {
	cmd := exec.Command("kubectl", "cluster-info")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func IsCommandSafe(c string) bool {
	return security.CheckRisk(c) != security.RiskCritical
}

func ConfirmExecution(cmd string) bool {
	risk := security.CheckRisk(cmd)
	switch risk {
	case security.RiskLow:
		return true
	case security.RiskMedium:
		fmt.Printf("\n\033[33m⚠️  [中风险] 这是一个修改操作: %s\033[0m\n", cmd)
		fmt.Print("确认执行? (y/N): ")
	case security.RiskHigh:
		fmt.Printf("\n\033[31m🔥 [高风险] 这是一个危险操作: %s\033[0m\n", cmd)
		fmt.Print("确认执行? (输入 'yes' 确认): ")
	case security.RiskCritical:
		code := security.GenerateVerifyCode()
		fmt.Printf("\n\033[41;37m💀 [极高风险] 毁灭性操作警告: %s \033[0m\n", cmd)
		fmt.Printf("请输入验证码 \033[1;33m%s\033[0m 以确认: ", code)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		return strings.TrimSpace(input) == code
	}

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	if risk == security.RiskHigh {
		return input == "yes"
	}
	return input == "y" || input == "yes"
}

func GetHostname() string {
	h, _ := os.Hostname()
	return h
}
