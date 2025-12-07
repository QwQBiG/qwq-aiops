package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
)

var (
	// ErrWebhookNotFound Webhook 不存在
	ErrWebhookNotFound = errors.New("webhook not found")
	
	// ErrInvalidSignature 签名无效
	ErrInvalidSignature = errors.New("invalid signature")
)

// WebhookService Webhook 服务接口
type WebhookService interface {
	// Webhook 管理
	CreateWebhook(ctx context.Context, webhook *Webhook) error
	UpdateWebhook(ctx context.Context, id uint, webhook *Webhook) error
	DeleteWebhook(ctx context.Context, id uint) error
	GetWebhook(ctx context.Context, id uint) (*Webhook, error)
	ListWebhooks(ctx context.Context, userID, tenantID uint) ([]*Webhook, error)
	
	// 事件触发
	TriggerEvent(ctx context.Context, event *Event) error
	ListEvents(ctx context.Context, webhookID uint) ([]*EventLog, error)
}

// EventType 事件类型
type EventType string

const (
	EventAppInstalled      EventType = "app.installed"
	EventAppUninstalled    EventType = "app.uninstalled"
	EventContainerStarted  EventType = "container.started"
	EventContainerStopped  EventType = "container.stopped"
	EventBackupCompleted   EventType = "backup.completed"
	EventBackupFailed      EventType = "backup.failed"
	EventWebsiteCreated    EventType = "website.created"
	EventWebsiteUpdated    EventType = "website.updated"
	EventSSLRenewed        EventType = "ssl.renewed"
	EventDatabaseConnected EventType = "database.connected"
)

// Webhook Webhook 配置
type Webhook struct {
	ID          uint        `json:"id" gorm:"primaryKey"`
	Name        string      `json:"name" gorm:"not null"`
	URL         string      `json:"url" gorm:"not null"`
	Secret      string      `json:"secret,omitempty"`
	Events      []EventType `json:"events" gorm:"type:jsonb"`
	Enabled     bool        `json:"enabled" gorm:"default:true"`
	RetryCount  int         `json:"retry_count" gorm:"default:3"`
	Timeout     int         `json:"timeout" gorm:"default:30"` // 秒
	UserID      uint        `json:"user_id" gorm:"index"`
	TenantID    uint        `json:"tenant_id" gorm:"index"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Event 事件
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	UserID    uint                   `json:"user_id"`
	TenantID  uint                   `json:"tenant_id"`
}

// EventLog 事件日志
type EventLog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	WebhookID    uint      `json:"webhook_id" gorm:"index"`
	EventType    EventType `json:"event_type" gorm:"index"`
	Payload      string    `json:"payload" gorm:"type:text"`
	StatusCode   int       `json:"status_code"`
	Response     string    `json:"response" gorm:"type:text"`
	Success      bool      `json:"success"`
	RetryCount   int       `json:"retry_count"`
	ErrorMessage string    `json:"error_message,omitempty" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at"`
}

// WebhookServiceImpl Webhook 服务实现
type WebhookServiceImpl struct {
	db     *gorm.DB
	client *http.Client
}

// NewWebhookService 创建 Webhook 服务
func NewWebhookService(db *gorm.DB) WebhookService {
	return &WebhookServiceImpl{
		db: db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateWebhook 创建 Webhook
func (s *WebhookServiceImpl) CreateWebhook(ctx context.Context, webhook *Webhook) error {
	return s.db.WithContext(ctx).Create(webhook).Error
}

// UpdateWebhook 更新 Webhook
func (s *WebhookServiceImpl) UpdateWebhook(ctx context.Context, id uint, webhook *Webhook) error {
	webhook.ID = id
	return s.db.WithContext(ctx).Save(webhook).Error
}

// DeleteWebhook 删除 Webhook
func (s *WebhookServiceImpl) DeleteWebhook(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&Webhook{}, id).Error
}

// GetWebhook 获取 Webhook
func (s *WebhookServiceImpl) GetWebhook(ctx context.Context, id uint) (*Webhook, error) {
	var webhook Webhook
	if err := s.db.WithContext(ctx).First(&webhook, id).Error; err != nil {
		return nil, err
	}
	return &webhook, nil
}

// ListWebhooks 列出 Webhooks
func (s *WebhookServiceImpl) ListWebhooks(ctx context.Context, userID, tenantID uint) ([]*Webhook, error) {
	var webhooks []*Webhook
	query := s.db.WithContext(ctx)
	
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	
	if err := query.Find(&webhooks).Error; err != nil {
		return nil, err
	}
	
	return webhooks, nil
}

// TriggerEvent 触发事件
func (s *WebhookServiceImpl) TriggerEvent(ctx context.Context, event *Event) error {
	// 查找订阅此事件的 Webhooks
	var webhooks []*Webhook
	s.db.WithContext(ctx).
		Where("enabled = ? AND tenant_id = ?", true, event.TenantID).
		Find(&webhooks)
	
	// 异步触发所有匹配的 Webhooks
	for _, webhook := range webhooks {
		if s.isEventSubscribed(webhook, event.Type) {
			go s.deliverWebhook(context.Background(), webhook, event)
		}
	}
	
	return nil
}

// isEventSubscribed 检查 Webhook 是否订阅了该事件
func (s *WebhookServiceImpl) isEventSubscribed(webhook *Webhook, eventType EventType) bool {
	for _, subscribedEvent := range webhook.Events {
		if subscribedEvent == eventType {
			return true
		}
	}
	return false
}

// deliverWebhook 投递 Webhook
func (s *WebhookServiceImpl) deliverWebhook(ctx context.Context, webhook *Webhook, event *Event) {
	// 准备 payload
	payload, err := json.Marshal(event)
	if err != nil {
		s.logEvent(ctx, webhook.ID, event.Type, string(payload), 0, "", false, 0, err.Error())
		return
	}
	
	// 尝试投递（带重试）
	var lastErr error
	var statusCode int
	var response string
	
	for attempt := 0; attempt <= webhook.RetryCount; attempt++ {
		statusCode, response, lastErr = s.sendWebhook(webhook, payload)
		
		if lastErr == nil && statusCode >= 200 && statusCode < 300 {
			// 成功
			s.logEvent(ctx, webhook.ID, event.Type, string(payload), statusCode, response, true, attempt, "")
			return
		}
		
		// 失败，等待后重试
		if attempt < webhook.RetryCount {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}
	
	// 所有重试都失败
	errorMsg := ""
	if lastErr != nil {
		errorMsg = lastErr.Error()
	}
	s.logEvent(ctx, webhook.ID, event.Type, string(payload), statusCode, response, false, webhook.RetryCount, errorMsg)
}

// sendWebhook 发送 Webhook 请求
func (s *WebhookServiceImpl) sendWebhook(webhook *Webhook, payload []byte) (int, string, error) {
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(payload))
	if err != nil {
		return 0, "", err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "QWQ-Webhook/1.0")
	
	// 添加签名
	if webhook.Secret != "" {
		signature := s.generateSignature(payload, webhook.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}
	
	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(webhook.Timeout)*time.Second)
	defer cancel()
	req = req.WithContext(ctx)
	
	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	
	// 读取响应
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	responseBody := buf.String()
	
	return resp.StatusCode, responseBody, nil
}

// generateSignature 生成签名
func (s *WebhookServiceImpl) generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// logEvent 记录事件日志
func (s *WebhookServiceImpl) logEvent(ctx context.Context, webhookID uint, eventType EventType, payload string, statusCode int, response string, success bool, retryCount int, errorMsg string) {
	log := &EventLog{
		WebhookID:    webhookID,
		EventType:    eventType,
		Payload:      payload,
		StatusCode:   statusCode,
		Response:     response,
		Success:      success,
		RetryCount:   retryCount,
		ErrorMessage: errorMsg,
	}
	
	s.db.WithContext(ctx).Create(log)
}

// ListEvents 列出事件日志
func (s *WebhookServiceImpl) ListEvents(ctx context.Context, webhookID uint) ([]*EventLog, error) {
	var logs []*EventLog
	if err := s.db.WithContext(ctx).
		Where("webhook_id = ?", webhookID).
		Order("created_at DESC").
		Limit(100).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	
	return logs, nil
}

// VerifySignature 验证签名
func VerifySignature(payload []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// TableName 指定表名
func (Webhook) TableName() string {
	return "webhooks"
}

func (EventLog) TableName() string {
	return "webhook_event_logs"
}
