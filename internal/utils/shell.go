package utils

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)


const CommandTimeout = 60 * time.Second

// ExecuteShell 执行 Shell 命令 (带超时保护)
func ExecuteShell(c string) string {
	// 使用 Context 控制超时
	ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", c)
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	out, err := cmd.CombinedOutput()
	res := string(out)

	// 区分是超时还是普通错误
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

func IsCommandSafe(c string) bool {
	dangerous := []string{"rm -rf", "mkfs", ":(){:|:&};:", "> /dev/sda", "dd if=/dev/zero"}
	for _, d := range dangerous {
		if strings.Contains(c, d) {
			return false
		}
	}
	return true
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

func ConfirmExecution() bool {
	fmt.Print("\033[33m[?] 这是一个修改操作，确认执行? (Y/n): \033[0m")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "" || input == "y" || input == "yes"
}

func GetHostname() string {
	h, _ := os.Hostname()
	return h
}