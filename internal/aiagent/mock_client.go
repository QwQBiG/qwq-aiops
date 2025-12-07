package aiagent

import (
	"context"
	"strings"
)

// MockOpenAIClient 模拟的 OpenAI 客户端，用于测试
type MockOpenAIClient struct{}

// MockChatCompletionRequest 模拟的聊天完成请求
type MockChatCompletionRequest struct {
	Model       string                      `json:"model"`
	Messages    []MockChatCompletionMessage `json:"messages"`
	Temperature float32                     `json:"temperature"`
}

// MockChatCompletionMessage 模拟的聊天消息
type MockChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MockChatCompletionResponse 模拟的聊天完成响应
type MockChatCompletionResponse struct {
	Choices []MockChoice `json:"choices"`
}

// MockChoice 模拟的选择
type MockChoice struct {
	Message MockChatCompletionMessage `json:"message"`
}

// CreateChatCompletion 模拟创建聊天完成
func (m *MockOpenAIClient) CreateChatCompletion(ctx context.Context, req MockChatCompletionRequest) (MockChatCompletionResponse, error) {
	// 简单的模拟响应逻辑
	var response string
	
	// 根据用户输入生成模拟响应
	if len(req.Messages) > 0 {
		userMessage := req.Messages[len(req.Messages)-1].Content
		
		// 简单的意图识别模拟
		if containsAny(userMessage, []string{"部署", "deploy", "安装", "install"}) {
			response = `{"intent": "deploy", "confidence": 0.8}`
		} else if containsAny(userMessage, []string{"查看", "show", "状态", "status"}) {
			response = `{"intent": "query", "confidence": 0.8}`
		} else if containsAny(userMessage, []string{"重启", "restart"}) {
			response = `{"intent": "restart", "confidence": 0.8}`
		} else if containsAny(userMessage, []string{"停止", "stop"}) {
			response = `{"intent": "stop", "confidence": 0.8}`
		} else {
			response = `{"intent": "unknown", "confidence": 0.3}`
		}
	} else {
		response = `{"intent": "unknown", "confidence": 0.0}`
	}
	
	return MockChatCompletionResponse{
		Choices: []MockChoice{
			{
				Message: MockChatCompletionMessage{
					Role:    "assistant",
					Content: response,
				},
			},
		},
	}, nil
}

// containsAny 检查字符串是否包含任何给定的子字符串
func containsAny(text string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(strings.ToLower(text), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}