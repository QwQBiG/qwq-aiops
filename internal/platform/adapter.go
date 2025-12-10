// Package platform 跨平台适配器模块
// 提供 Windows、Linux、macOS 等不同操作系统的兼容性支持
// 主要解决命令执行、路径处理、Docker 环境等跨平台差异问题
package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// PlatformAdapter 跨平台适配器接口
// 定义了处理不同操作系统差异的标准方法
type PlatformAdapter interface {
	// GetPlatformInfo 获取当前运行平台的详细信息
	// 包括操作系统类型、架构、是否在容器中运行等
	GetPlatformInfo() PlatformInfo
	
	// ExecuteCommand 执行平台特定的系统命令
	// 自动选择合适的 shell（Windows 用 cmd，Unix 用 bash）
	ExecuteCommand(cmd string) (string, error)
	
	// GetPlatformPath 将路径转换为当前平台的格式
	// 处理不同操作系统的路径分隔符差异
	GetPlatformPath(path string) string
	
	// ValidatePlatformCompatibility 验证当前平台是否支持
	// 检查操作系统兼容性和必要的依赖环境
	ValidatePlatformCompatibility() error
}

// PlatformInfo 平台信息结构体
// 包含运行环境的详细信息，用于平台适配决策
type PlatformInfo struct {
	OS           string `json:"os"`           // 操作系统类型（linux, windows, darwin）
	Architecture string `json:"architecture"` // 系统架构（amd64, arm64 等）
	IsContainer  bool   `json:"is_container"` // 是否在容器中运行
	DockerHost   string `json:"docker_host"`  // Docker 守护进程地址
}

// DefaultAdapter 默认平台适配器实现
// 提供基础的跨平台功能支持
type DefaultAdapter struct{}

// NewPlatformAdapter 创建新的平台适配器实例
// 返回默认的平台适配器实现
func NewPlatformAdapter() PlatformAdapter {
	return &DefaultAdapter{}
}

// GetPlatformInfo 获取当前平台的详细信息
// 通过 runtime 包和环境检测获取系统信息
func (a *DefaultAdapter) GetPlatformInfo() PlatformInfo {
	info := PlatformInfo{
		OS:           runtime.GOOS,           // 获取操作系统类型
		Architecture: runtime.GOARCH,        // 获取系统架构
		IsContainer:  isRunningInContainer(), // 检测是否在容器中运行
		DockerHost:   os.Getenv("DOCKER_HOST"), // 获取 Docker 主机地址
	}
	
	return info
}

// ExecuteCommand 执行平台特定的系统命令
// 根据操作系统类型选择合适的命令执行器
func (a *DefaultAdapter) ExecuteCommand(cmd string) (string, error) {
	var execCmd *exec.Cmd
	
	// 根据操作系统选择合适的 shell
	switch runtime.GOOS {
	case "windows":
		// Windows 系统使用 cmd.exe
		execCmd = exec.Command("cmd", "/C", cmd)
	default:
		// Unix 系统（Linux, macOS）使用 bash
		execCmd = exec.Command("bash", "-c", cmd)
	}
	
	// 执行命令并获取输出（包括标准输出和错误输出）
	output, err := execCmd.CombinedOutput()
	return string(output), err
}

// GetPlatformPath 获取平台特定的路径格式
// 将通用路径转换为当前操作系统的路径格式
func (a *DefaultAdapter) GetPlatformPath(path string) string {
	// 使用 filepath.FromSlash 将正斜杠路径转换为当前平台的路径分隔符
	return filepath.FromSlash(path)
}

// ValidatePlatformCompatibility 验证平台兼容性
// 检查当前操作系统是否受支持，以及必要的依赖是否可用
func (a *DefaultAdapter) ValidatePlatformCompatibility() error {
	info := a.GetPlatformInfo()
	
	// 检查支持的操作系统列表
	supportedOS := []string{"linux", "windows", "darwin"}
	if !contains(supportedOS, info.OS) {
		return fmt.Errorf("不支持的操作系统: %s", info.OS)
	}
	
	// Windows 环境下的特殊检查
	if info.OS == "windows" && info.DockerHost == "" {
		// 检查 Docker Desktop 是否可用
		_, err := a.ExecuteCommand("docker version")
		if err != nil {
			return fmt.Errorf("Windows 环境下 Docker 不可用: %v", err)
		}
	}
	
	return nil
}

// isRunningInContainer 检查当前程序是否在容器中运行
// 通过检查容器特有的文件和环境来判断
func isRunningInContainer() bool {
	// 方法1：检查 Docker 容器标识文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	
	// 方法2：检查 cgroup 信息（Linux 系统）
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		// 检查是否包含容器运行时标识
		return strings.Contains(content, "docker") || strings.Contains(content, "containerd")
	}
	
	return false
}

// contains 检查字符串切片是否包含指定元素
// 用于验证操作系统是否在支持列表中
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// CommandAdapter 命令适配器
// 负责将通用命令转换为特定平台的命令格式
type CommandAdapter struct {
	adapter PlatformAdapter // 底层平台适配器
}

// NewCommandAdapter 创建新的命令适配器
// 基于给定的平台适配器创建命令转换器
func NewCommandAdapter(adapter PlatformAdapter) *CommandAdapter {
	return &CommandAdapter{adapter: adapter}
}

// AdaptCommand 将命令适配到当前平台
// 根据操作系统类型选择相应的命令转换策略
func (c *CommandAdapter) AdaptCommand(cmd string) string {
	info := c.adapter.GetPlatformInfo()
	
	switch info.OS {
	case "windows":
		return c.adaptWindowsCommand(cmd)
	default:
		return c.adaptUnixCommand(cmd)
	}
}

// adaptWindowsCommand 将 Unix 命令适配为 Windows 命令
// 处理常见的 Unix 命令到 Windows 命令的转换
func (c *CommandAdapter) adaptWindowsCommand(cmd string) string {
	// 常见的 Unix 命令到 Windows 命令的映射表
	replacements := map[string]string{
		"ls":     "dir",              // 列出目录内容
		"cat":    "type",             // 显示文件内容
		"grep":   "findstr",          // 文本搜索
		"ps":     "tasklist",         // 进程列表
		"kill":   "taskkill /PID",    // 终止进程
		"which":  "where",            // 查找命令位置
		"pwd":    "cd",               // 显示当前目录
		"rm -rf": "rmdir /s /q",      // 递归删除目录
		"cp":     "copy",             // 复制文件
		"mv":     "move",             // 移动文件
	}
	
	// 遍历映射表，替换匹配的命令
	for unix, windows := range replacements {
		if strings.HasPrefix(cmd, unix+" ") || cmd == unix {
			cmd = strings.Replace(cmd, unix, windows, 1)
		}
	}
	
	return cmd
}

// adaptUnixCommand 适配 Unix 系统命令
// Unix 系统通常不需要特殊适配，但可以在这里处理特殊情况
func (c *CommandAdapter) adaptUnixCommand(cmd string) string {
	// Unix 系统（Linux, macOS）通常不需要命令转换
	// 但可以在这里添加特定的优化或修正
	return cmd
}

// ExecuteAdaptedCommand 执行适配后的命令
// 先将命令适配到当前平台，然后执行
func (c *CommandAdapter) ExecuteAdaptedCommand(cmd string) (string, error) {
	// 首先适配命令到当前平台
	adaptedCmd := c.AdaptCommand(cmd)
	// 然后执行适配后的命令
	return c.adapter.ExecuteCommand(adaptedCmd)
}