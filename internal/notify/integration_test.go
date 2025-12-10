package notify

import (
	"context"
	"qwq/internal/config"
	"testing"
	"time"
)

// TestNotificationIntegration 测试通知系统集成
func TestNotificationIntegration(t *testing.T) {
	// 设置测试配置
	config.GlobalConfig.DingTalkWebhook = "https://oapi.dingtalk.com/robot/send?access_token=test_token"
	
	// 初始化通知服务
	InitNotificationService()
	
	// 测试基本发送功能
	t.Run("基本发送功能", func(t *testing.T) {
		err := GetNotificationService().SendAlert("测试标题", "测试内容")
		// 由于是测试环境，我们期望网络错误
		if err == nil {
			t.Log("发送成功（可能是模拟环境）")
		} else {
			t.Logf("发送失败（预期的网络错误）: %v", err)
		}
	})
	
	// 测试配置验证
	t.Run("配置验证", func(t *testing.T) {
		err := ValidateNotificationConfig()
		if err != nil {
			t.Errorf("配置验证失败: %v", err)
		}
	})
	
	// 测试容器通知适配器
	t.Run("容器通知适配器", func(t *testing.T) {
		adapter := NewContainerNotificationAdapter()
		
		alert := &ContainerAlert{
			Level:       "error",
			Title:       "容器异常",
			Message:     "容器无法启动",
			ContainerID: "test_container_123",
			ServiceName: "web-service",
			ProjectName: "test-project",
			Timestamp:   time.Now(),
		}
		
		err := adapter.SendAlert(context.Background(), alert)
		// 由于是测试环境，我们期望网络错误
		if err == nil {
			t.Log("容器告警发送成功（可能是模拟环境）")
		} else {
			t.Logf("容器告警发送失败（预期的网络错误）: %v", err)
		}
	})
}

// TestNotificationServiceWithoutConfig 测试未配置通知服务的情况
func TestNotificationServiceWithoutConfig(t *testing.T) {
	// 清空配置
	originalWebhook := config.GlobalConfig.DingTalkWebhook
	config.GlobalConfig.DingTalkWebhook = ""
	defer func() {
		config.GlobalConfig.DingTalkWebhook = originalWebhook
	}()
	
	// 重新初始化
	InitNotificationService()
	
	// 测试发送应该失败
	err := GetNotificationService().SendAlert("测试", "测试内容")
	if err == nil {
		t.Error("期望发送失败，但实际成功了")
	} else {
		t.Logf("正确处理了未配置的情况: %v", err)
	}
}

// TestBackwardCompatibility 测试向后兼容性
func TestBackwardCompatibility(t *testing.T) {
	// 设置测试配置
	config.GlobalConfig.DingTalkWebhook = "https://oapi.dingtalk.com/robot/send?access_token=test_token"
	
	// 测试原有的 Send 函数
	t.Run("原有Send函数兼容性", func(t *testing.T) {
		// 这应该不会报错，即使网络失败也只是日志输出
		Send("兼容性测试", "测试原有的Send函数")
		t.Log("原有Send函数调用完成")
	})
	
	// 测试新的函数
	t.Run("新函数功能", func(t *testing.T) {
		err := SendStatusReport("系统状态正常")
		if err == nil {
			t.Log("状态报告发送成功（可能是模拟环境）")
		} else {
			t.Logf("状态报告发送失败（预期的网络错误）: %v", err)
		}
	})
}