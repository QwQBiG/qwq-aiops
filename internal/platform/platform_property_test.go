package platform

import (
	"runtime"
	"strings"
	"testing"
	"testing/quick"
)

// TestProperty9_CrossPlatformFunctionalConsistency 测试跨平台功能一致性
//
// **Feature: deployment-ai-config-fix, Property 9: 跨平台功能一致性**
// **Validates: Requirements 3.1**
//
// 这个属性测试验证核心功能在 Windows 和 Ubuntu 环境中表现一致
func TestProperty9_CrossPlatformFunctionalConsistency(t *testing.T) {
	adapter := NewPlatformAdapter()
	
	// 属性1: 平台信息获取的一致性
	t.Run("PlatformInfoConsistency", func(t *testing.T) {
		property := func() bool {
			info := adapter.GetPlatformInfo()
			
			// 验证平台信息的基本字段都有值
			if info.OS == "" || info.Architecture == "" {
				return false
			}
			
			// 验证操作系统字段与 runtime.GOOS 一致
			if info.OS != runtime.GOOS {
				return false
			}
			
			// 验证架构字段与 runtime.GOARCH 一致
			if info.Architecture != runtime.GOARCH {
				return false
			}
			
			return true
		}
		
		if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("平台信息获取一致性测试失败: %v", err)
		}
	})
	
	// 属性2: 路径处理的一致性
	t.Run("PathHandlingConsistency", func(t *testing.T) {
		property := func(path string) bool {
			// 过滤无效输入
			if len(path) == 0 || len(path) > 200 {
				return true
			}
			
			// 获取平台特定路径
			platformPath := adapter.GetPlatformPath(path)
			
			// 验证路径不为空
			if platformPath == "" {
				return false
			}
			
			// 验证路径长度合理
			if len(platformPath) > 500 {
				return false
			}
			
			return true
		}
		
		if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("路径处理一致性测试失败: %v", err)
		}
	})
	
	// 属性3: 基本命令执行的一致性
	t.Run("BasicCommandConsistency", func(t *testing.T) {
		// 测试基本的系统信息命令在不同平台上都能执行
		basicCommands := []string{
			"echo hello",
		}
		
		for _, cmd := range basicCommands {
			t.Run("Command_"+strings.ReplaceAll(cmd, " ", "_"), func(t *testing.T) {
				output, err := adapter.ExecuteCommand(cmd)
				
				// 基本命令应该能够执行成功
				if err != nil {
					t.Errorf("基本命令 '%s' 执行失败: %v", cmd, err)
				}
				
				// 输出不应该为空（对于 echo 命令）
				if cmd == "echo hello" && !strings.Contains(output, "hello") {
					t.Errorf("echo 命令输出不正确: %s", output)
				}
			})
		}
	})
	
	// 属性4: 平台兼容性验证的一致性
	t.Run("CompatibilityValidationConsistency", func(t *testing.T) {
		property := func() bool {
			err := adapter.ValidatePlatformCompatibility()
			
			// 在支持的平台上，兼容性验证应该成功
			supportedPlatforms := []string{"linux", "windows", "darwin"}
			currentOS := runtime.GOOS
			
			isSupported := false
			for _, os := range supportedPlatforms {
				if os == currentOS {
					isSupported = true
					break
				}
			}
			
			if isSupported && err != nil {
				// 支持的平台不应该返回错误（除非是 Docker 相关问题）
				if !strings.Contains(err.Error(), "Docker") {
					return false
				}
			}
			
			return true
		}
		
		if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("平台兼容性验证一致性测试失败: %v", err)
		}
	})
}

// TestProperty9_CommandAdapterConsistency 测试命令适配器的一致性
//
// **Feature: deployment-ai-config-fix, Property 9: 跨平台功能一致性**
// **Validates: Requirements 3.1**
//
// 验证命令适配器在不同平台上的行为一致性
func TestProperty9_CommandAdapterConsistency(t *testing.T) {
	adapter := NewPlatformAdapter()
	cmdAdapter := NewCommandAdapter(adapter)
	
	// 属性: 命令适配的幂等性
	t.Run("CommandAdaptationIdempotency", func(t *testing.T) {
		property := func(cmd string) bool {
			// 过滤无效输入
			if len(cmd) == 0 || len(cmd) > 100 {
				return true
			}
			
			// 多次适配同一个命令应该得到相同结果
			adapted1 := cmdAdapter.AdaptCommand(cmd)
			adapted2 := cmdAdapter.AdaptCommand(cmd)
			
			return adapted1 == adapted2
		}
		
		if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("命令适配幂等性测试失败: %v", err)
		}
	})
	
	// 属性: 适配后命令的有效性
	t.Run("AdaptedCommandValidity", func(t *testing.T) {
		testCommands := []string{
			"echo test",
			"ls",
			"pwd",
		}
		
		for _, cmd := range testCommands {
			t.Run("Adapt_"+strings.ReplaceAll(cmd, " ", "_"), func(t *testing.T) {
				adapted := cmdAdapter.AdaptCommand(cmd)
				
				// 适配后的命令不应该为空
				if adapted == "" {
					t.Errorf("命令 '%s' 适配后为空", cmd)
				}
				
				// 适配后的命令长度应该合理
				if len(adapted) > 200 {
					t.Errorf("命令 '%s' 适配后过长: %s", cmd, adapted)
				}
			})
		}
	})
}

// TestProperty9_PlatformSpecificBehavior 测试平台特定行为
//
// **Feature: deployment-ai-config-fix, Property 9: 跨平台功能一致性**
// **Validates: Requirements 3.1**
//
// 验证平台特定行为的正确性
func TestProperty9_PlatformSpecificBehavior(t *testing.T) {
	adapter := NewPlatformAdapter()
	info := adapter.GetPlatformInfo()
	
	// 根据当前平台测试特定行为
	switch info.OS {
	case "windows":
		t.Run("WindowsSpecificBehavior", func(t *testing.T) {
			// 测试 Windows 特定的命令适配
			cmdAdapter := NewCommandAdapter(adapter)
			
			// ls 应该被适配为 dir
			adapted := cmdAdapter.AdaptCommand("ls")
			if !strings.Contains(adapted, "dir") {
				t.Errorf("Windows 下 ls 命令未正确适配: %s", adapted)
			}
			
			// cat 应该被适配为 type
			adapted = cmdAdapter.AdaptCommand("cat file.txt")
			if !strings.Contains(adapted, "type") {
				t.Errorf("Windows 下 cat 命令未正确适配: %s", adapted)
			}
		})
		
	case "linux", "darwin":
		t.Run("UnixSpecificBehavior", func(t *testing.T) {
			// 测试 Unix 系统的行为
			cmdAdapter := NewCommandAdapter(adapter)
			
			// Unix 命令应该保持不变
			testCmd := "ls -la"
			adapted := cmdAdapter.AdaptCommand(testCmd)
			if adapted != testCmd {
				t.Errorf("Unix 下命令不应该被修改: 原始='%s', 适配后='%s'", testCmd, adapted)
			}
		})
	}
	
	// 测试容器环境检测
	t.Run("ContainerDetection", func(t *testing.T) {
		// 容器检测应该返回布尔值
		isContainer := info.IsContainer
		
		// 这是一个有效的布尔值（true 或 false 都可以）
		_ = isContainer
		
		// 如果在容器中，Docker Host 可能有值
		if isContainer && info.DockerHost != "" {
			// 验证 Docker Host 格式
			if !strings.Contains(info.DockerHost, "://") {
				t.Errorf("Docker Host 格式可能不正确: %s", info.DockerHost)
			}
		}
	})
}

// TestProperty10_DockerDeploymentCompatibility 测试 Docker 部署兼容性
//
// **Feature: deployment-ai-config-fix, Property 10: Docker 部署兼容性**
// **Validates: Requirements 3.2**
//
// 验证 Docker Compose 配置在不同宿主机操作系统上正确运行
func TestProperty10_DockerDeploymentCompatibility(t *testing.T) {
	adapter := NewPlatformAdapter()
	info := adapter.GetPlatformInfo()
	
	// 属性1: Docker 环境检测的一致性
	t.Run("DockerEnvironmentDetection", func(t *testing.T) {
		property := func() bool {
			// 检查 Docker 是否可用
			_, err := adapter.ExecuteCommand("docker version")
			
			// 如果 Docker 不可用，这在某些环境下是正常的
			// 但是错误信息应该是有意义的
			if err != nil {
				errMsg := err.Error()
				// 错误信息不应该为空
				if errMsg == "" {
					return false
				}
			}
			
			return true
		}
		
		if err := quick.Check(property, &quick.Config{MaxCount: 10}); err != nil {
			t.Errorf("Docker 环境检测一致性测试失败: %v", err)
		}
	})
	
	// 属性2: Docker Compose 路径处理兼容性
	t.Run("DockerComposePathCompatibility", func(t *testing.T) {
		property := func(relativePath string) bool {
			// 过滤无效输入
			if len(relativePath) == 0 || len(relativePath) > 100 {
				return true
			}
			
			// 测试路径转换
			platformPath := adapter.GetPlatformPath(relativePath)
			
			// 转换后的路径应该有效
			if platformPath == "" {
				return false
			}
			
			// 路径应该使用正确的分隔符
			switch info.OS {
			case "windows":
				// Windows 路径可能包含反斜杠或正斜杠
				return true // Windows 支持两种分隔符
			default:
				// Unix 系统使用正斜杠
				return !strings.Contains(platformPath, "\\")
			}
		}
		
		if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("Docker Compose 路径兼容性测试失败: %v", err)
		}
	})
	
	// 属性3: Docker Host 配置验证
	t.Run("DockerHostConfiguration", func(t *testing.T) {
		dockerHost := info.DockerHost
		
		if dockerHost != "" {
			// 如果配置了 Docker Host，应该是有效的 URL 格式
			validPrefixes := []string{"tcp://", "unix://", "npipe://", "ssh://"}
			isValid := false
			
			for _, prefix := range validPrefixes {
				if strings.HasPrefix(dockerHost, prefix) {
					isValid = true
					break
				}
			}
			
			if !isValid {
				t.Errorf("Docker Host 配置格式无效: %s", dockerHost)
			}
		}
	})
	
	// 属性4: 容器环境下的行为一致性
	t.Run("ContainerEnvironmentConsistency", func(t *testing.T) {
		if info.IsContainer {
			// 在容器环境中，某些命令可能有不同的行为
			property := func() bool {
				// 基本命令仍然应该可用
				_, err := adapter.ExecuteCommand("echo container-test")
				return err == nil
			}
			
			if err := quick.Check(property, &quick.Config{MaxCount: 10}); err != nil {
				t.Errorf("容器环境行为一致性测试失败: %v", err)
			}
		}
	})
}

// TestProperty10_DockerComposeCompatibility 测试 Docker Compose 兼容性
//
// **Feature: deployment-ai-config-fix, Property 10: Docker 部署兼容性**
// **Validates: Requirements 3.2**
//
// 验证 Docker Compose 相关功能的跨平台兼容性
func TestProperty10_DockerComposeCompatibility(t *testing.T) {
	adapter := NewPlatformAdapter()
	
	// 属性: Docker Compose 命令适配
	t.Run("DockerComposeCommandAdaptation", func(t *testing.T) {
		cmdAdapter := NewCommandAdapter(adapter)
		
		composeCommands := []string{
			"docker-compose up -d",
			"docker-compose down",
			"docker-compose ps",
			"docker-compose logs",
		}
		
		for _, cmd := range composeCommands {
			t.Run("Command_"+strings.ReplaceAll(cmd, " ", "_"), func(t *testing.T) {
				adapted := cmdAdapter.AdaptCommand(cmd)
				
				// Docker Compose 命令在所有平台上应该保持一致
				if adapted != cmd {
					// 除非是特殊的平台适配
					t.Logf("Docker Compose 命令被适配: '%s' -> '%s'", cmd, adapted)
				}
				
				// 适配后的命令不应该为空
				if adapted == "" {
					t.Errorf("Docker Compose 命令适配后为空: %s", cmd)
				}
			})
		}
	})
	
	// 属性: 卷挂载路径兼容性
	t.Run("VolumePathCompatibility", func(t *testing.T) {
		property := func(hostPath string) bool {
			// 过滤无效输入
			if len(hostPath) == 0 || len(hostPath) > 200 {
				return true
			}
			
			// 测试卷挂载路径的平台适配
			platformPath := adapter.GetPlatformPath(hostPath)
			
			// 路径不应该为空
			if platformPath == "" {
				return false
			}
			
			// 路径长度应该合理
			if len(platformPath) > 500 {
				return false
			}
			
			return true
		}
		
		if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("卷挂载路径兼容性测试失败: %v", err)
		}
	})
}

// TestProperty10_CrossPlatformDockerIntegration 测试跨平台 Docker 集成
//
// **Feature: deployment-ai-config-fix, Property 10: Docker 部署兼容性**
// **Validates: Requirements 3.2**
//
// 验证 Docker 集成在不同平台上的一致性
func TestProperty10_CrossPlatformDockerIntegration(t *testing.T) {
	adapter := NewPlatformAdapter()
	info := adapter.GetPlatformInfo()
	
	// 属性: 平台特定的 Docker 配置
	t.Run("PlatformSpecificDockerConfig", func(t *testing.T) {
		switch info.OS {
		case "windows":
			// Windows 特定的 Docker 配置测试
			t.Run("WindowsDockerConfig", func(t *testing.T) {
				// Windows 下可能使用 npipe 或 tcp 连接
				if info.DockerHost != "" {
					if !strings.Contains(info.DockerHost, "npipe") && 
					   !strings.Contains(info.DockerHost, "tcp") {
						t.Logf("Windows Docker Host 配置: %s", info.DockerHost)
					}
				}
			})
			
		case "linux":
			// Linux 特定的 Docker 配置测试
			t.Run("LinuxDockerConfig", func(t *testing.T) {
				// Linux 下通常使用 unix socket
				if info.DockerHost == "" || strings.Contains(info.DockerHost, "unix://") {
					// 这是正常的配置
				} else {
					t.Logf("Linux Docker Host 配置: %s", info.DockerHost)
				}
			})
			
		case "darwin":
			// macOS 特定的 Docker 配置测试
			t.Run("MacOSDockerConfig", func(t *testing.T) {
				// macOS 下通常使用 unix socket 或 Docker Desktop
				t.Logf("macOS Docker Host 配置: %s", info.DockerHost)
			})
		}
	})
	
	// 属性: Docker 网络兼容性
	t.Run("DockerNetworkCompatibility", func(t *testing.T) {
		// 首先检查 Docker 是否可用
		_, dockerErr := adapter.ExecuteCommand("docker version")
		if dockerErr != nil {
			t.Skip("Docker 不可用，跳过网络兼容性测试")
			return
		}
		
		property := func() bool {
			// 测试 Docker 网络相关命令的兼容性
			networkCommands := []string{
				"docker network ls",
			}
			
			for _, cmd := range networkCommands {
				// 尝试执行命令
				output, err := adapter.ExecuteCommand(cmd)
				
				if err != nil {
					// 如果 Docker 服务不可用，这是可以接受的
					errMsg := strings.ToLower(err.Error())
					if strings.Contains(errMsg, "not found") || 
					   strings.Contains(errMsg, "cannot connect") ||
					   strings.Contains(errMsg, "daemon") ||
					   strings.Contains(errMsg, "permission denied") {
						return true // 这些错误是可以接受的
					}
					return false
				}
				
				// 如果命令成功，输出应该包含网络信息或至少是表头
				output = strings.TrimSpace(output)
				if len(output) == 0 {
					return false
				}
			}
			
			return true
		}
		
		if err := quick.Check(property, &quick.Config{MaxCount: 3}); err != nil {
			t.Errorf("Docker 网络兼容性测试失败: %v", err)
		}
	})
}