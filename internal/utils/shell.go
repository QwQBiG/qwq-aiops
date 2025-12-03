package utils

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// ExecuteShell 执行 Shell 命令并返回结果
func ExecuteShell(c string) string {
	cmd := exec.Command("bash", "-c", c)
	// 移除 Setpgid 以兼容 Windows 编译，Linux/Docker 下依然正常
	cmd.SysProcAttr = &syscall.SysProcAttr{} 
	
	out, err := cmd.CombinedOutput()
	res := string(out)
	if err != nil {
		if len(res) > 0 {
			res += fmt.Sprintf("\n(Command failed: %v)", err)
		} else {
			res = fmt.Sprintf("(Command failed: %v)", err)
		}
	}
	// 截断过长的输出
	if len(res) > 4000 {
		res = res[:4000] + "\n...(Output truncated)"
	}
	return res
}

// IsCommandSafe 检查命令是否包含高危操作
func IsCommandSafe(c string) bool {
	dangerous := []string{"rm -rf", "mkfs", ":(){:|:&};:", "> /dev/sda", "dd if=/dev/zero"}
	for _, d := range dangerous {
		if strings.Contains(c, d) {
			return false
		}
	}
	return true
}

// IsReadOnlyCommand 检查是否为只读命令（允许自动执行）
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

// ConfirmExecution 请求用户确认
func ConfirmExecution() bool {
	fmt.Print("\033[33m[?] 这是一个修改操作，确认执行? (Y/n): \033[0m")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "" || input == "y" || input == "yes"
}

// GetHostname 获取主机名
func GetHostname() string {
	h, _ := os.Hostname()
	return h
}