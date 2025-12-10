package notify

import (
	"context"
	"fmt"
	"qwq/internal/config"
	"time"
)

// ExampleDingTalkNotification 演示钉钉通知服务的使用
func ExampleDingTalkNotification() {
	// 1. 设置配置
	config.GlobalConfig.DingTalkWebhook = "https://oapi.dingtalk.com/robot/send?access_token=your_token_here"
	
	// 2. 初始化通知服务
	InitNotificationService()
	
	// 3. 发送基本告警
	err := GetNotificationService().SendAlert("系统告警", "服务器 CPU 使用率过高")
	if err != nil {
		fmt.Printf("发送告警失败: %v\n", err)
	}
	
	// 4. 发送状态报告
	err = SendStatusReport("系统运行正常，所有服务状态良好")
	if err != nil {
		fmt.Printf("发送状态报告失败: %v\n", err)
	}
	
	// 5. 测试连接
	err = TestNotificationConnection()
	if err != nil {
		fmt.Printf("连接测试失败: %v\n", err)
	}
	
	// 6. 验证配置
	err = ValidateNotificationConfig()
	if err != nil {
		fmt.Printf("配置验证失败: %v\n", err)
	}
}

// ExampleContainerNotification 演示容器通知的使用
func ExampleContainerNotification() {
	// 1. 创建容器通知适配器
	adapter := NewContainerNotificationAdapter()
	
	// 2. 创建告警信息
	alert := &ContainerAlert{
		Level:       "error",
		Title:       "容器启动失败",
		Message:     "Docker 容器 web-service 无法启动，端口冲突",
		ContainerID: "container_12345",
		ServiceName: "web-service",
		ProjectName: "my-project",
		Timestamp:   time.Now(),
		Details: map[string]interface{}{
			"port":  8080,
			"image": "nginx:latest",
		},
	}
	
	// 3. 发送容器告警
	err := adapter.SendAlert(context.Background(), alert)
	if err != nil {
		fmt.Printf("发送容器告警失败: %v\n", err)
	}
}

// ExampleBackwardCompatibility 演示向后兼容性
func ExampleBackwardCompatibility() {
	// 原有的 Send 函数仍然可以使用
	Send("兼容性测试", "这是使用原有 Send 函数发送的消息")
	
	fmt.Println("向后兼容性测试完成")
}

// ExampleNotificationWithoutConfig 演示未配置通知服务的处理
func ExampleNotificationWithoutConfig() {
	// 清空配置
	originalWebhook := config.GlobalConfig.DingTalkWebhook
	config.GlobalConfig.DingTalkWebhook = ""
	defer func() {
		config.GlobalConfig.DingTalkWebhook = originalWebhook
	}()
	
	// 重新初始化
	InitNotificationService()
	
	// 尝试发送消息
	err := GetNotificationService().SendAlert("测试", "这条消息不会发送成功")
	if err != nil {
		fmt.Printf("预期的错误: %v\n", err)
	}
}