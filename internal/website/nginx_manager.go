package website

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// NginxConfigDir Nginx 配置目录
	NginxConfigDir = "/etc/nginx/sites-available"
	// NginxEnabledDir Nginx 启用的配置目录
	NginxEnabledDir = "/etc/nginx/sites-enabled"
	// NginxBinary Nginx 二进制文件路径
	NginxBinary = "/usr/sbin/nginx"
)

// validateNginxConfig 验证 Nginx 配置
func validateNginxConfig(config string) error {
	// 创建临时配置文件
	tmpFile, err := os.CreateTemp("", "nginx-config-*.conf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// 写入配置
	if _, err := tmpFile.WriteString(config); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	tmpFile.Close()

	// 使用 nginx -t 验证配置
	cmd := exec.Command(NginxBinary, "-t", "-c", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("config validation failed: %s", string(output))
	}

	return nil
}

// reloadNginx 重载 Nginx 配置
func reloadNginx() error {
	// 首先测试配置
	cmd := exec.Command(NginxBinary, "-t")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("config test failed: %s", string(output))
	}

	// 重载 Nginx
	cmd = exec.Command(NginxBinary, "-s", "reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("reload failed: %s", string(output))
	}

	return nil
}

// WriteNginxConfig 写入 Nginx 配置文件
func WriteNginxConfig(domain, config string) error {
	// 清理域名作为文件名
	filename := sanitizeName(domain) + ".conf"
	configPath := filepath.Join(NginxConfigDir, filename)

	// 确保目录存在
	if err := os.MkdirAll(NginxConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 写入配置文件
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// EnableNginxSite 启用 Nginx 站点
func EnableNginxSite(domain string) error {
	filename := sanitizeName(domain) + ".conf"
	availablePath := filepath.Join(NginxConfigDir, filename)
	enabledPath := filepath.Join(NginxEnabledDir, filename)

	// 确保启用目录存在
	if err := os.MkdirAll(NginxEnabledDir, 0755); err != nil {
		return fmt.Errorf("failed to create enabled directory: %w", err)
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(availablePath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", availablePath)
	}

	// 创建符号链接
	if err := os.Symlink(availablePath, enabledPath); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create symlink: %w", err)
		}
	}

	return nil
}

// DisableNginxSite 禁用 Nginx 站点
func DisableNginxSite(domain string) error {
	filename := sanitizeName(domain) + ".conf"
	enabledPath := filepath.Join(NginxEnabledDir, filename)

	// 删除符号链接
	if err := os.Remove(enabledPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove symlink: %w", err)
		}
	}

	return nil
}

// RemoveNginxConfig 删除 Nginx 配置
func RemoveNginxConfig(domain string) error {
	filename := sanitizeName(domain) + ".conf"
	
	// 先禁用站点
	if err := DisableNginxSite(domain); err != nil {
		return err
	}

	// 删除配置文件
	configPath := filepath.Join(NginxConfigDir, filename)
	if err := os.Remove(configPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove config file: %w", err)
		}
	}

	return nil
}

// TestNginxConfig 测试 Nginx 配置
func TestNginxConfig() error {
	cmd := exec.Command(NginxBinary, "-t")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("config test failed: %s", string(output))
	}
	return nil
}

// GetNginxVersion 获取 Nginx 版本
func GetNginxVersion() (string, error) {
	cmd := exec.Command(NginxBinary, "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get nginx version: %w", err)
	}
	
	// 解析版本信息
	version := strings.TrimSpace(string(output))
	version = strings.TrimPrefix(version, "nginx version: ")
	
	return version, nil
}

// IsNginxRunning 检查 Nginx 是否运行
func IsNginxRunning() bool {
	cmd := exec.Command("pgrep", "-x", "nginx")
	err := cmd.Run()
	return err == nil
}

// StartNginx 启动 Nginx
func StartNginx() error {
	cmd := exec.Command("systemctl", "start", "nginx")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start nginx: %s", string(output))
	}
	return nil
}

// StopNginx 停止 Nginx
func StopNginx() error {
	cmd := exec.Command("systemctl", "stop", "nginx")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop nginx: %s", string(output))
	}
	return nil
}

// RestartNginx 重启 Nginx
func RestartNginx() error {
	cmd := exec.Command("systemctl", "restart", "nginx")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restart nginx: %s", string(output))
	}
	return nil
}
