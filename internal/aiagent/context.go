package aiagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ContextManager 上下文管理器
type ContextManager struct {
	contexts     map[string]*ConversationContext
	mutex        sync.RWMutex
	persistPath  string
	maxHistory   int
	sessionTTL   time.Duration
}

// NewContextManager 创建新的上下文管理器
func NewContextManager(persistPath string) *ContextManager {
	return &ContextManager{
		contexts:    make(map[string]*ConversationContext),
		persistPath: persistPath,
		maxHistory:  50,  // 最多保存50轮对话
		sessionTTL:  24 * time.Hour, // 会话24小时过期
	}
}

// UpdateContext 更新对话上下文
func (s *NLUServiceImpl) UpdateContext(ctx context.Context, sessionID string, turn *ConversationTurn) error {
	s.contextMutex.Lock()
	defer s.contextMutex.Unlock()
	
	context, exists := s.contexts[sessionID]
	if !exists {
		context = &ConversationContext{
			SessionID: sessionID,
			Variables: make(map[string]interface{}),
			CreatedAt: time.Now(),
		}
		s.contexts[sessionID] = context
	}
	
	// 更新上下文信息
	context.LastIntent = turn.Intent
	context.LastEntities = turn.Entities
	context.UpdatedAt = time.Now()
	
	// 添加到历史记录
	context.History = append(context.History, *turn)
	
	// 限制历史记录长度
	if len(context.History) > 50 {
		context.History = context.History[len(context.History)-50:]
	}
	
	// 持久化上下文（可选）
	if s.persistPath != "" {
		go s.persistContext(sessionID, context)
	}
	
	return nil
}

// GetContext 获取对话上下文
func (s *NLUServiceImpl) GetContext(ctx context.Context, sessionID string) (*ConversationContext, error) {
	s.contextMutex.RLock()
	defer s.contextMutex.RUnlock()
	
	context, exists := s.contexts[sessionID]
	if !exists {
		// 尝试从持久化存储加载
		if s.persistPath != "" {
			loadedContext, err := s.loadContext(sessionID)
			if err == nil {
				s.contexts[sessionID] = loadedContext
				return loadedContext, nil
			}
		}
		return nil, fmt.Errorf("会话不存在: %s", sessionID)
	}
	
	// 检查会话是否过期
	if time.Since(context.UpdatedAt) > 24*time.Hour {
		delete(s.contexts, sessionID)
		return nil, fmt.Errorf("会话已过期: %s", sessionID)
	}
	
	return context, nil
}

// persistContext 持久化上下文到文件
func (s *NLUServiceImpl) persistContext(sessionID string, context *ConversationContext) {
	if s.persistPath == "" {
		return
	}
	
	// 确保目录存在
	if err := os.MkdirAll(s.persistPath, 0755); err != nil {
		return
	}
	
	// 序列化上下文
	data, err := json.Marshal(context)
	if err != nil {
		return
	}
	
	// 写入文件
	filename := filepath.Join(s.persistPath, fmt.Sprintf("session_%s.json", sessionID))
	os.WriteFile(filename, data, 0644)
}

// loadContext 从文件加载上下文
func (s *NLUServiceImpl) loadContext(sessionID string) (*ConversationContext, error) {
	if s.persistPath == "" {
		return nil, fmt.Errorf("未配置持久化路径")
	}
	
	filename := filepath.Join(s.persistPath, fmt.Sprintf("session_%s.json", sessionID))
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	var context ConversationContext
	if err := json.Unmarshal(data, &context); err != nil {
		return nil, err
	}
	
	return &context, nil
}

// CleanupExpiredSessions 清理过期会话
func (s *NLUServiceImpl) CleanupExpiredSessions() {
	s.contextMutex.Lock()
	defer s.contextMutex.Unlock()
	
	now := time.Now()
	for sessionID, context := range s.contexts {
		if now.Sub(context.UpdatedAt) > 24*time.Hour {
			delete(s.contexts, sessionID)
			
			// 删除持久化文件
			if s.persistPath != "" {
				filename := filepath.Join(s.persistPath, fmt.Sprintf("session_%s.json", sessionID))
				os.Remove(filename)
			}
		}
	}
}

// DetectLanguage 检测语言
func (s *NLUServiceImpl) DetectLanguage(ctx context.Context, text string) (string, error) {
	// 简单的语言检测逻辑
	chineseCount := 0
	englishCount := 0
	
	for _, char := range text {
		if char >= 0x4e00 && char <= 0x9fff {
			chineseCount++
		} else if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			englishCount++
		}
	}
	
	if chineseCount > englishCount {
		return "zh", nil
	} else if englishCount > 0 {
		return "en", nil
	}
	
	// 默认返回中文
	return "zh", nil
}

// TranslateIntent 翻译意图
func (s *NLUServiceImpl) TranslateIntent(ctx context.Context, intent Intent, language string) (string, error) {
	translations := map[Intent]map[string]string{
		IntentDeploy: {
			"zh": "部署",
			"en": "deploy",
		},
		IntentQuery: {
			"zh": "查询",
			"en": "query",
		},
		IntentStart: {
			"zh": "启动",
			"en": "start",
		},
		IntentStop: {
			"zh": "停止",
			"en": "stop",
		},
		IntentRestart: {
			"zh": "重启",
			"en": "restart",
		},
		IntentUpdate: {
			"zh": "更新",
			"en": "update",
		},
		IntentDelete: {
			"zh": "删除",
			"en": "delete",
		},
		IntentDiagnose: {
			"zh": "诊断",
			"en": "diagnose",
		},
		IntentConfigure: {
			"zh": "配置",
			"en": "configure",
		},
		IntentGenerate: {
			"zh": "生成",
			"en": "generate",
		},
	}
	
	if langMap, exists := translations[intent]; exists {
		if translation, exists := langMap[language]; exists {
			return translation, nil
		}
	}
	
	return string(intent), nil
}

// GetContextSummary 获取上下文摘要
func (s *NLUServiceImpl) GetContextSummary(sessionID string) string {
	s.contextMutex.RLock()
	defer s.contextMutex.RUnlock()
	
	context, exists := s.contexts[sessionID]
	if !exists {
		return "无上下文信息"
	}
	
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("会话ID: %s\n", context.SessionID))
	summary.WriteString(fmt.Sprintf("用户ID: %s\n", context.UserID))
	summary.WriteString(fmt.Sprintf("最后意图: %s\n", context.LastIntent))
	summary.WriteString(fmt.Sprintf("对话轮数: %d\n", len(context.History)))
	summary.WriteString(fmt.Sprintf("创建时间: %s\n", context.CreatedAt.Format("2006-01-02 15:04:05")))
	summary.WriteString(fmt.Sprintf("更新时间: %s\n", context.UpdatedAt.Format("2006-01-02 15:04:05")))
	
	if len(context.LastEntities) > 0 {
		summary.WriteString("最后提取的实体:\n")
		for _, entity := range context.LastEntities {
			summary.WriteString(fmt.Sprintf("  - %s: %s (%.2f)\n", entity.Type, entity.Value, entity.Confidence))
		}
	}
	
	return summary.String()
}

// SetContextVariable 设置上下文变量
func (s *NLUServiceImpl) SetContextVariable(sessionID, key string, value interface{}) error {
	s.contextMutex.Lock()
	defer s.contextMutex.Unlock()
	
	context, exists := s.contexts[sessionID]
	if !exists {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}
	
	if context.Variables == nil {
		context.Variables = make(map[string]interface{})
	}
	
	context.Variables[key] = value
	context.UpdatedAt = time.Now()
	
	return nil
}

// GetContextVariable 获取上下文变量
func (s *NLUServiceImpl) GetContextVariable(sessionID, key string) (interface{}, bool) {
	s.contextMutex.RLock()
	defer s.contextMutex.RUnlock()
	
	context, exists := s.contexts[sessionID]
	if !exists {
		return nil, false
	}
	
	if context.Variables == nil {
		return nil, false
	}
	
	value, exists := context.Variables[key]
	return value, exists
}