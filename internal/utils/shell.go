package utils

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"qwq/internal/security"
	"strings"
	"syscall"
	"time"
)

const CommandTimeout = 60 * time.Second

func ExecuteShell(c string) string {
	if strings.HasPrefix(strings.TrimSpace(c), "kubectl") {
		if !CheckK8sConnection() {
			return "❌ Error: Kubernetes cluster is unreachable. Please check ~/.kube/config mount."
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", c)
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	out, err := cmd.CombinedOutput()
	res := string(out)

	if ctx.Err() == context.DeadlineExceeded {
		res += "\n(Command timed out after 60s)"
	} else if err != nil {
		if len(res) > 0 {
			res += fmt.Sprintf("\n(Command failed: %v)", err)
		} else {
			res = fmt.Sprintf("(Command failed: %v)", err)
		}
	}
	if len(res) > 4000 {
		res = res[:4000] + "\n...(Output truncated)"
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

func IsReadOnlyCommand(cmd string) bool {
	safeKeywords := []string{
		"ls", "cat", "head", "tail", "grep", "find", "pwd", "echo", "whoami", "id",
		"ps", "top", "uptime", "free", "df", "du", "netstat", "ss", "lsof",
		"kubectl get", "kubectl describe", "kubectl logs", "kubectl top", "kubectl cluster-info",
		"docker ps", "docker logs", "docker stats", "ip", "hostname",
	}
	c := strings.ToLower(cmd)
	for _, kw := range safeKeywords {
		if strings.HasPrefix(c, kw) || strings.Contains(c, " "+kw) {
			if !strings.Contains(c, ">") && !strings.Contains(c, "rm ") && !strings.Contains(c, "kill") && !strings.Contains(c, "delete") {
				return true
			}
		}
	}
	return false
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