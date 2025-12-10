// Package config 修复过程记录和验证模块
// 提供修复操作的记录、跟踪和验证功能
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RepairType 修复类型
type RepairType string

const (
	RepairFrontend     RepairType = "frontend"
	RepairConfig       RepairType = "config"
	RepairNotification RepairType = "notification"
	RepairPlatform     RepairType = "platform"
)

// DeploymentComponent 部署组件
type DeploymentComponent struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Status      ComponentStatus   `json:"status"`
	Config      map[string]string `json:"config"`
	Dependencies []string         `json:"dependencies"`
}

// ComponentStatus 组件状态
type ComponentStatus string

const (
	ComponentStatusHealthy   ComponentStatus = "healthy"
	ComponentStatusUnhealthy ComponentStatus = "unhealthy"
	ComponentStatusMissing   ComponentStatus = "missing"
	ComponentStatusError     ComponentStatus = "error"
)

// DeploymentValidationResult 部署验证结果
type DeploymentValidationResult struct {
	Valid              bool                           `json:"valid"`
	ComponentsChecked  []string                      `json:"components_checked"`
	HealthyComponents  []string                      `json:"healthy_components"`
	UnhealthyComponents []string                     `json:"unhealthy_components"`
	MissingComponents  []string                      `json:"missing_components"`
	ValidationErrors   []string                      `json:"validation_errors"`
	Suggestions        []string                      `json:"suggestions"`
	ComponentDetails   map[string]DeploymentComponent `json:"component_details"`
}

// RepairTracker 修复跟踪器
type RepairTracker struct {
	logPath string
	session *RepairSession
}

// RepairSession 修复会话
type RepairSession struct {
	ID        string                 `json:"id"`
	StartTime time.Time              `json:"start_time"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Status    RepairSessionStatus    `json:"status"`
	Operations []RepairOperationLog  `json:"operations"`
	Summary   *RepairSummary         `json:"summary,omitempty"`
}

// RepairSessionStatus 修复会话状态
type RepairSessionStatus string

const (
	SessionStatusRunning   RepairSessionStatus = "running"
	SessionStatusCompleted RepairSessionStatus = "completed"
	SessionStatusFailed    RepairSessionStatus = "failed"
)

// RepairOperationLog 修复操作日志
type RepairOperationLog struct {
	ID          string                 `json:"id"`
	Type        RepairType             `json:"type"`
	Description string                 `json:"description"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Status      RepairOperationStatus  `json:"status"`
	Commands    []string               `json:"commands"`
	Output      string                 `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RepairOperationStatus 修复操作状态
type RepairOperationStatus string

const (
	OperationStatusPending   RepairOperationStatus = "pending"
	OperationStatusRunning   RepairOperationStatus = "running"
	OperationStatusCompleted RepairOperationStatus = "completed"
	OperationStatusFailed    RepairOperationStatus = "failed"
	OperationStatusSkipped   RepairOperationStatus = "skipped"
)

// RepairSummary 修复摘要
type RepairSummary struct {
	TotalOperations     int                        `json:"total_operations"`
	CompletedOperations int                        `json:"completed_operations"`
	FailedOperations    int                        `json:"failed_operations"`
	SkippedOperations   int                        `json:"skipped_operations"`
	Duration            time.Duration              `json:"duration"`
	ValidationResult    *DeploymentValidationResult `json:"validation_result,omitempty"`
	Recommendations     []string                   `json:"recommendations"`
}

// NewRepairTracker 创建新的修复跟踪器
func NewRepairTracker(logPath string) *RepairTracker {
	return &RepairTracker{
		logPath: logPath,
	}
}

// StartSession 开始修复会话
func (rt *RepairTracker) StartSession() error {
	sessionID := fmt.Sprintf("repair_%d", time.Now().Unix())
	
	rt.session = &RepairSession{
		ID:         sessionID,
		StartTime:  time.Now(),
		Status:     SessionStatusRunning,
		Operations: []RepairOperationLog{},
	}
	
	fmt.Printf("🚀 开始修复会话: %s\n", sessionID)
	return rt.saveSession()
}

// AddOperation 添加修复操作
func (rt *RepairTracker) AddOperation(opType RepairType, description string, commands []string) string {
	if rt.session == nil {
		return ""
	}
	
	operationID := fmt.Sprintf("op_%d", len(rt.session.Operations)+1)
	
	operation := RepairOperationLog{
		ID:          operationID,
		Type:        opType,
		Description: description,
		StartTime:   time.Now(),
		Status:      OperationStatusPending,
		Commands:    commands,
		Metadata:    make(map[string]interface{}),
	}
	
	rt.session.Operations = append(rt.session.Operations, operation)
	
	fmt.Printf("📝 添加修复操作: %s - %s\n", operationID, description)
	return operationID
}

// StartOperation 开始执行修复操作
func (rt *RepairTracker) StartOperation(operationID string) error {
	if rt.session == nil {
		return fmt.Errorf("没有活动的修复会话")
	}
	
	for i, op := range rt.session.Operations {
		if op.ID == operationID {
			rt.session.Operations[i].Status = OperationStatusRunning
			rt.session.Operations[i].StartTime = time.Now()
			
			fmt.Printf("▶️  开始执行操作: %s\n", op.Description)
			return rt.saveSession()
		}
	}
	
	return fmt.Errorf("操作 %s 不存在", operationID)
}

// CompleteOperation 完成修复操作
func (rt *RepairTracker) CompleteOperation(operationID string, output string, err error) error {
	if rt.session == nil {
		return fmt.Errorf("没有活动的修复会话")
	}
	
	for i, op := range rt.session.Operations {
		if op.ID == operationID {
			endTime := time.Now()
			rt.session.Operations[i].EndTime = &endTime
			rt.session.Operations[i].Output = output
			
			if err != nil {
				rt.session.Operations[i].Status = OperationStatusFailed
				rt.session.Operations[i].Error = err.Error()
				fmt.Printf("❌ 操作失败: %s - %v\n", op.Description, err)
			} else {
				rt.session.Operations[i].Status = OperationStatusCompleted
				fmt.Printf("✅ 操作完成: %s\n", op.Description)
			}
			
			return rt.saveSession()
		}
	}
	
	return fmt.Errorf("操作 %s 不存在", operationID)
}

// SkipOperation 跳过修复操作
func (rt *RepairTracker) SkipOperation(operationID string, reason string) error {
	if rt.session == nil {
		return fmt.Errorf("没有活动的修复会话")
	}
	
	for i, op := range rt.session.Operations {
		if op.ID == operationID {
			endTime := time.Now()
			rt.session.Operations[i].EndTime = &endTime
			rt.session.Operations[i].Status = OperationStatusSkipped
			rt.session.Operations[i].Output = fmt.Sprintf("跳过原因: %s", reason)
			
			fmt.Printf("⏭️  跳过操作: %s - %s\n", op.Description, reason)
			return rt.saveSession()
		}
	}
	
	return fmt.Errorf("操作 %s 不存在", operationID)
}

// EndSession 结束修复会话
func (rt *RepairTracker) EndSession(validationResult *DeploymentValidationResult) error {
	if rt.session == nil {
		return fmt.Errorf("没有活动的修复会话")
	}
	
	endTime := time.Now()
	rt.session.EndTime = &endTime
	
	// 生成摘要
	summary := rt.generateSummary(validationResult)
	rt.session.Summary = summary
	
	// 确定会话状态
	if summary.FailedOperations > 0 {
		rt.session.Status = SessionStatusFailed
	} else {
		rt.session.Status = SessionStatusCompleted
	}
	
	fmt.Printf("🏁 修复会话结束: %s\n", rt.session.Status)
	fmt.Printf("📊 操作统计: 总计 %d, 完成 %d, 失败 %d, 跳过 %d\n",
		summary.TotalOperations, summary.CompletedOperations,
		summary.FailedOperations, summary.SkippedOperations)
	
	return rt.saveSession()
}

// generateSummary 生成修复摘要
func (rt *RepairTracker) generateSummary(validationResult *DeploymentValidationResult) *RepairSummary {
	summary := &RepairSummary{
		TotalOperations:     len(rt.session.Operations),
		CompletedOperations: 0,
		FailedOperations:    0,
		SkippedOperations:   0,
		Duration:            time.Since(rt.session.StartTime),
		ValidationResult:    validationResult,
		Recommendations:     []string{},
	}
	
	// 统计操作状态
	for _, op := range rt.session.Operations {
		switch op.Status {
		case OperationStatusCompleted:
			summary.CompletedOperations++
		case OperationStatusFailed:
			summary.FailedOperations++
		case OperationStatusSkipped:
			summary.SkippedOperations++
		}
	}
	
	// 生成建议
	if summary.FailedOperations > 0 {
		summary.Recommendations = append(summary.Recommendations,
			"有修复操作失败，请检查错误日志并手动处理")
	}
	
	if validationResult != nil && !validationResult.Valid {
		summary.Recommendations = append(summary.Recommendations,
			"部署验证未通过，请检查系统状态")
		summary.Recommendations = append(summary.Recommendations, validationResult.Suggestions...)
	}
	
	if summary.CompletedOperations == summary.TotalOperations {
		summary.Recommendations = append(summary.Recommendations,
			"所有修复操作已完成，建议重启服务以确保更改生效")
	}
	
	return summary
}

// saveSession 保存修复会话
func (rt *RepairTracker) saveSession() error {
	if rt.session == nil {
		return fmt.Errorf("没有活动的修复会话")
	}
	
	// 确保日志目录存在
	logDir := filepath.Dir(rt.logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}
	
	// 序列化会话数据
	data, err := json.MarshalIndent(rt.session, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化会话数据失败: %v", err)
	}
	
	// 写入文件
	sessionFile := filepath.Join(logDir, fmt.Sprintf("%s.json", rt.session.ID))
	if err := os.WriteFile(sessionFile, data, 0644); err != nil {
		return fmt.Errorf("保存会话文件失败: %v", err)
	}
	
	return nil
}

// LoadSession 加载修复会话
func (rt *RepairTracker) LoadSession(sessionID string) error {
	sessionFile := filepath.Join(filepath.Dir(rt.logPath), fmt.Sprintf("%s.json", sessionID))
	
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return fmt.Errorf("读取会话文件失败: %v", err)
	}
	
	var session RepairSession
	if err := json.Unmarshal(data, &session); err != nil {
		return fmt.Errorf("解析会话数据失败: %v", err)
	}
	
	rt.session = &session
	return nil
}

// GetCurrentSession 获取当前会话
func (rt *RepairTracker) GetCurrentSession() *RepairSession {
	return rt.session
}

// ListSessions 列出所有修复会话
func (rt *RepairTracker) ListSessions() ([]string, error) {
	logDir := filepath.Dir(rt.logPath)
	
	files, err := os.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("读取日志目录失败: %v", err)
	}
	
	var sessions []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			sessionID := strings.TrimSuffix(file.Name(), ".json")
			if strings.HasPrefix(sessionID, "repair_") {
				sessions = append(sessions, sessionID)
			}
		}
	}
	
	return sessions, nil
}

// PrintSessionSummary 打印会话摘要
func (rt *RepairTracker) PrintSessionSummary() {
	if rt.session == nil || rt.session.Summary == nil {
		fmt.Println("没有可用的会话摘要")
		return
	}
	
	summary := rt.session.Summary
	
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("修复会话摘要: %s\n", rt.session.ID)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("开始时间: %s\n", rt.session.StartTime.Format("2006-01-02 15:04:05"))
	if rt.session.EndTime != nil {
		fmt.Printf("结束时间: %s\n", rt.session.EndTime.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("持续时间: %v\n", summary.Duration)
	fmt.Printf("会话状态: %s\n", rt.session.Status)
	
	fmt.Println("\n操作统计:")
	fmt.Printf("  总计: %d\n", summary.TotalOperations)
	fmt.Printf("  完成: %d\n", summary.CompletedOperations)
	fmt.Printf("  失败: %d\n", summary.FailedOperations)
	fmt.Printf("  跳过: %d\n", summary.SkippedOperations)
	
	if len(summary.Recommendations) > 0 {
		fmt.Println("\n建议:")
		for _, rec := range summary.Recommendations {
			fmt.Printf("  - %s\n", rec)
		}
	}
	
	if summary.ValidationResult != nil {
		fmt.Println("\n部署验证结果:")
		if summary.ValidationResult.Valid {
			fmt.Println("  ✅ 验证通过")
		} else {
			fmt.Println("  ❌ 验证失败")
			for _, err := range summary.ValidationResult.ValidationErrors {
				fmt.Printf("    - %s\n", err)
			}
		}
	}
	
	fmt.Println(strings.Repeat("=", 60))
}