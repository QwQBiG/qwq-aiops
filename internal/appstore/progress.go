package appstore

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProgressStore 进度存储
type ProgressStore struct {
	mu        sync.RWMutex
	progresses map[string]*InstallationProgress
}

// NewProgressStore 创建进度存储实例
func NewProgressStore() *ProgressStore {
	return &ProgressStore{
		progresses: make(map[string]*InstallationProgress),
	}
}

// Create 创建新的安装进度
func (s *ProgressStore) Create(instanceID uint) *InstallationProgress {
	s.mu.Lock()
	defer s.mu.Unlock()

	progress := &InstallationProgress{
		ID:             uuid.New().String(),
		InstanceID:     instanceID,
		Status:         StatusPending,
		CurrentStep:    "Initializing",
		TotalSteps:     5,
		CompletedSteps: 0,
		Message:        "Installation queued",
		StartTime:      time.Now(),
	}

	s.progresses[progress.ID] = progress
	return progress
}

// Get 获取安装进度
func (s *ProgressStore) Get(progressID string) *InstallationProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.progresses[progressID]
}

// Update 更新安装进度
func (s *ProgressStore) Update(progressID string, status InstallationStatus, message string, completedSteps, totalSteps int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	progress, exists := s.progresses[progressID]
	if !exists {
		return
	}

	progress.Status = status
	progress.Message = message
	progress.CompletedSteps = completedSteps
	progress.TotalSteps = totalSteps
	progress.CurrentStep = message

	// 如果是完成或失败状态，设置结束时间
	if status == StatusCompleted || status == StatusFailed || status == StatusRolledBack {
		now := time.Now()
		progress.EndTime = &now
	}
}

// SetError 设置错误信息
func (s *ProgressStore) SetError(progressID string, errorMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	progress, exists := s.progresses[progressID]
	if !exists {
		return
	}

	progress.Error = errorMsg
	progress.Status = StatusFailed
	now := time.Now()
	progress.EndTime = &now
}

// SetRollback 设置回滚信息
func (s *ProgressStore) SetRollback(progressID string, reason, failedStep string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	progress, exists := s.progresses[progressID]
	if !exists {
		return
	}

	progress.Status = StatusRollingBack
	progress.RollbackInfo = &RollbackInfo{
		Reason:       reason,
		FailedStep:   failedStep,
		RollbackTime: time.Now(),
	}
}

// Delete 删除进度记录
func (s *ProgressStore) Delete(progressID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.progresses, progressID)
}

// List 列出所有进度
func (s *ProgressStore) List() []*InstallationProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*InstallationProgress, 0, len(s.progresses))
	for _, progress := range s.progresses {
		result = append(result, progress)
	}

	return result
}

// Cleanup 清理过期的进度记录（超过24小时）
func (s *ProgressStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, progress := range s.progresses {
		if progress.EndTime != nil && now.Sub(*progress.EndTime) > 24*time.Hour {
			delete(s.progresses, id)
		}
	}
}

// GetByInstanceID 根据实例ID获取进度
func (s *ProgressStore) GetByInstanceID(instanceID uint) []*InstallationProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*InstallationProgress, 0)
	for _, progress := range s.progresses {
		if progress.InstanceID == instanceID {
			result = append(result, progress)
		}
	}

	return result
}

// GetProgress 获取进度百分比
func (p *InstallationProgress) GetProgress() float64 {
	if p.TotalSteps == 0 {
		return 0
	}
	return float64(p.CompletedSteps) / float64(p.TotalSteps) * 100
}

// GetDuration 获取执行时长
func (p *InstallationProgress) GetDuration() time.Duration {
	if p.EndTime != nil {
		return p.EndTime.Sub(p.StartTime)
	}
	return time.Since(p.StartTime)
}

// IsCompleted 是否已完成
func (p *InstallationProgress) IsCompleted() bool {
	return p.Status == StatusCompleted
}

// IsFailed 是否失败
func (p *InstallationProgress) IsFailed() bool {
	return p.Status == StatusFailed
}

// IsInProgress 是否进行中
func (p *InstallationProgress) IsInProgress() bool {
	return p.Status == StatusPending || p.Status == StatusValidating || p.Status == StatusInstalling
}

// String 返回进度的字符串表示
func (p *InstallationProgress) String() string {
	return fmt.Sprintf("Progress[%s]: %s - %s (%d/%d steps, %.1f%%)",
		p.ID, p.Status, p.Message, p.CompletedSteps, p.TotalSteps, p.GetProgress())
}
