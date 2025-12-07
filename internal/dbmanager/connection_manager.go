package dbmanager

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"
)

// ConnectionManager 数据库连接管理器
type ConnectionManager struct {
	mu            sync.RWMutex
	connections   map[uint]DatabaseAdapter // 连接ID -> 适配器实例
	factory       *AdapterFactory
	encryptionKey []byte // 用于加密密码的密钥
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager(encryptionKey string) *ConnectionManager {
	// 如果没有提供加密密钥，使用默认密钥（生产环境应该从配置读取）
	if encryptionKey == "" {
		encryptionKey = "qwq-aiops-default-encryption-key-32" // 32字节
	}
	
	return &ConnectionManager{
		connections:   make(map[uint]DatabaseAdapter),
		factory:       NewAdapterFactory(),
		encryptionKey: []byte(encryptionKey)[:32], // AES-256需要32字节密钥
	}
}

// GetConnection 获取数据库连接
func (cm *ConnectionManager) GetConnection(ctx context.Context, connID uint) (DatabaseAdapter, error) {
	cm.mu.RLock()
	adapter, exists := cm.connections[connID]
	cm.mu.RUnlock()
	
	if exists {
		// 测试连接是否仍然有效
		if err := adapter.Ping(ctx); err == nil {
			return adapter, nil
		}
		// 连接失效，移除并重新创建
		cm.mu.Lock()
		delete(cm.connections, connID)
		cm.mu.Unlock()
	}
	
	return nil, ErrConnectionNotFound
}

// CreateConnection 创建新的数据库连接
func (cm *ConnectionManager) CreateConnection(ctx context.Context, config *DatabaseConnection) (DatabaseAdapter, error) {
	// 创建适配器
	adapter, err := cm.factory.Create(config.Type)
	if err != nil {
		return nil, err
	}
	
	// 建立连接
	if err := adapter.Connect(ctx, config); err != nil {
		return nil, fmt.Errorf("建立数据库连接失败: %w", err)
	}
	
	// 保存连接
	cm.mu.Lock()
	cm.connections[config.ID] = adapter
	cm.mu.Unlock()
	
	return adapter, nil
}

// CloseConnection 关闭数据库连接
func (cm *ConnectionManager) CloseConnection(ctx context.Context, connID uint) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	adapter, exists := cm.connections[connID]
	if !exists {
		return nil // 连接不存在，视为已关闭
	}
	
	if err := adapter.Disconnect(ctx); err != nil {
		return fmt.Errorf("断开数据库连接失败: %w", err)
	}
	
	delete(cm.connections, connID)
	return nil
}

// CloseAllConnections 关闭所有数据库连接
func (cm *ConnectionManager) CloseAllConnections(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	var lastErr error
	for connID, adapter := range cm.connections {
		if err := adapter.Disconnect(ctx); err != nil {
			lastErr = err
		}
		delete(cm.connections, connID)
	}
	
	return lastErr
}

// TestConnection 测试数据库连接
func (cm *ConnectionManager) TestConnection(ctx context.Context, config *DatabaseConnection) error {
	// 创建临时适配器
	adapter, err := cm.factory.Create(config.Type)
	if err != nil {
		return err
	}
	
	// 尝试连接
	if err := adapter.Connect(ctx, config); err != nil {
		return fmt.Errorf("连接测试失败: %w", err)
	}
	
	// 测试ping
	if err := adapter.Ping(ctx); err != nil {
		adapter.Disconnect(ctx)
		return fmt.Errorf("连接测试失败: %w", err)
	}
	
	// 断开临时连接
	return adapter.Disconnect(ctx)
}

// MonitorConnections 监控连接状态
func (cm *ConnectionManager) MonitorConnections(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.checkConnections(ctx)
		}
	}
}

// checkConnections 检查所有连接的健康状态
func (cm *ConnectionManager) checkConnections(ctx context.Context) {
	cm.mu.RLock()
	connIDs := make([]uint, 0, len(cm.connections))
	for connID := range cm.connections {
		connIDs = append(connIDs, connID)
	}
	cm.mu.RUnlock()
	
	for _, connID := range connIDs {
		adapter, err := cm.GetConnection(ctx, connID)
		if err != nil {
			continue
		}
		
		// 测试连接
		if err := adapter.Ping(ctx); err != nil {
			// 连接失效，移除
			cm.mu.Lock()
			delete(cm.connections, connID)
			cm.mu.Unlock()
		}
	}
}

// EncryptPassword 加密密码
func (cm *ConnectionManager) EncryptPassword(password string) (string, error) {
	if password == "" {
		return "", nil
	}
	
	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("创建加密器失败: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成随机数失败: %w", err)
	}
	
	ciphertext := gcm.Seal(nonce, nonce, []byte(password), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptPassword 解密密码
func (cm *ConnectionManager) DecryptPassword(encryptedPassword string) (string, error) {
	if encryptedPassword == "" {
		return "", nil
	}
	
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", fmt.Errorf("解码密码失败: %w", err)
	}
	
	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("创建解密器失败: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}
	
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("密文长度无效")
	}
	
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密密码失败: %w", err)
	}
	
	return string(plaintext), nil
}

// GetConnectionStats 获取连接统计信息
func (cm *ConnectionManager) GetConnectionStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_connections": len(cm.connections),
		"connection_ids":    make([]uint, 0, len(cm.connections)),
	}
	
	for connID := range cm.connections {
		stats["connection_ids"] = append(stats["connection_ids"].([]uint), connID)
	}
	
	return stats
}
