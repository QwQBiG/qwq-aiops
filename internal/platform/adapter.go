// Package platform reports process environment information without probing services.
package platform

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

type PlatformAdapter interface {
	GetPlatformInfo() PlatformInfo
	ValidatePlatformCompatibility() error
}

type PlatformInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	IsContainer  bool   `json:"is_container"`
	DockerHost   string `json:"docker_host"`
}

type DefaultAdapter struct{}

func NewPlatformAdapter() PlatformAdapter { return &DefaultAdapter{} }

func (a *DefaultAdapter) GetPlatformInfo() PlatformInfo {
	return PlatformInfo{
		OS: runtime.GOOS, Architecture: runtime.GOARCH,
		IsContainer: isRunningInContainer(), DockerHost: os.Getenv("DOCKER_HOST"),
	}
}

func (a *DefaultAdapter) ValidatePlatformCompatibility() error {
	switch runtime.GOOS {
	case "linux", "windows", "darwin":
		return nil
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func isRunningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		return strings.Contains(content, "docker") || strings.Contains(content, "containerd")
	}
	return false
}
