package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
)

// MonitoringServiceImpl 监控服务实现
type MonitoringServiceImpl struct {
	db            *gorm.DB
	metricsStore  MetricsStore
	alertEvaluator *AlertEvaluator
}

// NewMonitoringService 创建监控服务
func NewMonitoringService(db *gorm.DB) MonitoringService {
	service := &MonitoringServiceImpl{
		db:           db,
		metricsStore: NewInMemoryMetricsStore(),
	}
	
	service.alertEvaluator = NewAlertEvaluator(service)
	
	// 启动告警评估器
	go service.alertEvaluator.Start(context.Background())
	
	return service
}

// RecordMetric 记录指标
func (s *MonitoringServiceImpl) RecordMetric(ctx context.Context, metric *Metric) error {
	// 存储到时序数据库（这里使用内存存储）
	return s.metricsStore.Store(metric)
}

// QueryMetrics 查询指标
func (s *MonitoringServiceImpl) QueryMetrics(ctx context.Context, query *MetricQuery) ([]*MetricData, error) {
	return s.metricsStore.Query(query)
}

// ListMetrics 列出指标定义
func (s *MonitoringServiceImpl) ListMetrics(ctx context.Context, userID, tenantID uint) ([]*MetricDefinition, error) {
	var metrics []*MetricDefinition
	q := s.db.WithContext(ctx)
	
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	
	if err := q.Find(&metrics).Error; err != nil {
		return nil, err
	}
	
	return metrics, nil
}

// CreateAlertRule 创建告警规则
func (s *MonitoringServiceImpl) CreateAlertRule(ctx context.Context, rule *AlertRule) error {
	return s.db.WithContext(ctx).Create(rule).Error
}

// UpdateAlertRule 更新告警规则
func (s *MonitoringServiceImpl) UpdateAlertRule(ctx context.Context, id uint, rule *AlertRule) error {
	rule.ID = id
	return s.db.WithContext(ctx).Save(rule).Error
}

// DeleteAlertRule 删除告警规则
func (s *MonitoringServiceImpl) DeleteAlertRule(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&AlertRule{}, id).Error
}

// GetAlertRule 获取告警规则
func (s *MonitoringServiceImpl) GetAlertRule(ctx context.Context, id uint) (*AlertRule, error) {
	var rule AlertRule
	if err := s.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// ListAlertRules 列出告警规则
func (s *MonitoringServiceImpl) ListAlertRules(ctx context.Context, userID, tenantID uint) ([]*AlertRule, error) {
	var rules []*AlertRule
	q := s.db.WithContext(ctx)
	
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	
	if err := q.Find(&rules).Error; err != nil {
		return nil, err
	}
	
	return rules, nil
}

// ListAlerts 列出告警
func (s *MonitoringServiceImpl) ListAlerts(ctx context.Context, filters *AlertFilters) ([]*Alert, error) {
	var alerts []*Alert
	q := s.db.WithContext(ctx).Preload("Rule")
	
	if filters != nil {
		if filters.Status != "" {
			q = q.Where("status = ?", filters.Status)
		}
		if filters.Severity != "" {
			q = q.Where("severity = ?", filters.Severity)
		}
		if filters.RuleID > 0 {
			q = q.Where("rule_id = ?", filters.RuleID)
		}
		if !filters.StartTime.IsZero() {
			q = q.Where("fired_at >= ?", filters.StartTime)
		}
		if !filters.EndTime.IsZero() {
			q = q.Where("fired_at <= ?", filters.EndTime)
		}
		if filters.UserID > 0 {
			q = q.Where("user_id = ?", filters.UserID)
		}
		if filters.TenantID > 0 {
			q = q.Where("tenant_id = ?", filters.TenantID)
		}
	}
	
	if err := q.Order("fired_at DESC").Find(&alerts).Error; err != nil {
		return nil, err
	}
	
	return alerts, nil
}

// AcknowledgeAlert 确认告警
func (s *MonitoringServiceImpl) AcknowledgeAlert(ctx context.Context, alertID uint, userID uint) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&Alert{}).
		Where("id = ?", alertID).
		Updates(map[string]interface{}{
			"status":           AlertStatusAcknowledged,
			"acknowledged_at":  now,
			"acknowledged_by":  userID,
		}).Error
}

// ResolveAlert 解决告警
func (s *MonitoringServiceImpl) ResolveAlert(ctx context.Context, alertID uint, userID uint) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&Alert{}).
		Where("id = ?", alertID).
		Updates(map[string]interface{}{
			"status":      AlertStatusResolved,
			"resolved_at": now,
			"resolved_by": userID,
		}).Error
}

// PredictIssues AI 预测问题
func (s *MonitoringServiceImpl) PredictIssues(ctx context.Context, resourceType, resourceID string) (*PredictionResult, error) {
	// 查询历史指标数据
	query := &MetricQuery{
		MetricName: fmt.Sprintf("%s_health", resourceType),
		Labels:     map[string]string{"resource_id": resourceID},
		StartTime:  time.Now().Add(-24 * time.Hour),
		EndTime:    time.Now(),
		Step:       5 * time.Minute,
	}
	
	data, err := s.QueryMetrics(ctx, query)
	if err != nil {
		return nil, err
	}
	
	// AI 分析（简化版）
	predictions := s.analyzeTimeSeries(data)
	
	return &PredictionResult{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Predictions:  predictions,
		Confidence:   0.85,
		PredictedAt:  time.Now(),
	}, nil
}

// analyzeTimeSeries 分析时序数据
func (s *MonitoringServiceImpl) analyzeTimeSeries(data []*MetricData) []Prediction {
	if len(data) < 10 {
		return []Prediction{}
	}
	
	// 计算趋势
	var sum, sumX, sumY, sumXY, sumX2 float64
	n := float64(len(data))
	
	for i, point := range data {
		x := float64(i)
		y := point.Value
		sum += y
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	// 线性回归
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	
	predictions := []Prediction{}
	
	// 如果趋势向下且当前值较低，预测可能出现故障
	if slope < -0.1 && data[len(data)-1].Value < 50 {
		predictions = append(predictions, Prediction{
			Issue:       "服务健康度持续下降，可能即将出现故障",
			Probability: 0.75,
			TimeToIssue: 3600, // 1小时
			Severity:    SeverityWarning,
			Recommendations: []string{
				"检查服务日志",
				"增加资源配额",
				"准备故障转移方案",
			},
		})
	}
	
	// 如果值波动剧烈，预测可能不稳定
	variance := s.calculateVariance(data)
	if variance > 100 {
		predictions = append(predictions, Prediction{
			Issue:       "服务指标波动剧烈，系统可能不稳定",
			Probability: 0.65,
			TimeToIssue: 1800, // 30分钟
			Severity:    SeverityInfo,
			Recommendations: []string{
				"检查负载均衡配置",
				"分析流量模式",
				"优化资源分配",
			},
		})
	}
	
	return predictions
}

// calculateVariance 计算方差
func (s *MonitoringServiceImpl) calculateVariance(data []*MetricData) float64 {
	if len(data) == 0 {
		return 0
	}
	
	var sum float64
	for _, point := range data {
		sum += point.Value
	}
	mean := sum / float64(len(data))
	
	var variance float64
	for _, point := range data {
		diff := point.Value - mean
		variance += diff * diff
	}
	
	return variance / float64(len(data))
}

// AnalyzeCapacity 分析容量
func (s *MonitoringServiceImpl) AnalyzeCapacity(ctx context.Context, resourceType string) (*CapacityAnalysis, error) {
	// 查询资源使用情况
	query := &MetricQuery{
		MetricName: fmt.Sprintf("%s_usage", resourceType),
		StartTime:  time.Now().Add(-7 * 24 * time.Hour),
		EndTime:    time.Now(),
		Step:       1 * time.Hour,
	}
	
	data, err := s.QueryMetrics(ctx, query)
	if err != nil {
		return nil, err
	}
	
	if len(data) == 0 {
		return nil, fmt.Errorf("no data available")
	}
	
	// 计算当前使用率
	currentUsage := data[len(data)-1].Value
	capacity := 100.0 // 假设容量为100
	usagePercent := (currentUsage / capacity) * 100
	
	// 计算趋势
	trend := "stable"
	if len(data) > 1 {
		recentAvg := s.calculateAverage(data[len(data)-24:])
		oldAvg := s.calculateAverage(data[:24])
		
		if recentAvg > oldAvg*1.1 {
			trend = "increasing"
		} else if recentAvg < oldAvg*0.9 {
			trend = "decreasing"
		}
	}
	
	// 预测满载时间
	timeToFull := -1
	if trend == "increasing" && usagePercent < 100 {
		// 简单线性预测
		growthRate := (currentUsage - data[0].Value) / float64(len(data))
		if growthRate > 0 {
			remainingCapacity := capacity - currentUsage
			timeToFull = int(remainingCapacity / growthRate / 24) // 天数
		}
	}
	
	// 生成建议
	recommendations := []string{}
	if usagePercent > 80 {
		recommendations = append(recommendations, "资源使用率超过80%，建议扩容")
	}
	if trend == "increasing" {
		recommendations = append(recommendations, "资源使用呈上升趋势，建议监控并规划扩容")
	}
	if timeToFull > 0 && timeToFull < 30 {
		recommendations = append(recommendations, fmt.Sprintf("预计%d天后资源将满载，请尽快扩容", timeToFull))
	}
	
	return &CapacityAnalysis{
		ResourceType:    resourceType,
		CurrentUsage:    currentUsage,
		Capacity:        capacity,
		UsagePercent:    usagePercent,
		Trend:           trend,
		TimeToFull:      timeToFull,
		Recommendations: recommendations,
		AnalyzedAt:      time.Now(),
	}, nil
}

// calculateAverage 计算平均值
func (s *MonitoringServiceImpl) calculateAverage(data []*MetricData) float64 {
	if len(data) == 0 {
		return 0
	}
	
	var sum float64
	for _, point := range data {
		sum += point.Value
	}
	
	return sum / float64(len(data))
}

// FireAlert 触发告警
func (s *MonitoringServiceImpl) FireAlert(ctx context.Context, rule *AlertRule, value float64, labels map[string]string) error {
	// 检查是否在冷却期内
	var lastAlert Alert
	err := s.db.WithContext(ctx).
		Where("rule_id = ? AND status = ?", rule.ID, AlertStatusFiring).
		Order("fired_at DESC").
		First(&lastAlert).Error
	
	if err == nil {
		cooldownEnd := lastAlert.FiredAt.Add(time.Duration(rule.Cooldown) * time.Second)
		if time.Now().Before(cooldownEnd) {
			return nil // 在冷却期内，不触发新告警
		}
	}
	
	// 创建告警
	labelsJSON, _ := json.Marshal(labels)
	alert := &Alert{
		RuleID:   rule.ID,
		Status:   AlertStatusFiring,
		Severity: rule.Severity,
		Message:  fmt.Sprintf("%s: %s %s %.2f (当前值: %.2f)", rule.Name, rule.MetricName, rule.Operator, rule.Threshold, value),
		Value:    value,
		Labels:   string(labelsJSON),
		FiredAt:  time.Now(),
		UserID:   rule.UserID,
		TenantID: rule.TenantID,
	}
	
	if err := s.db.WithContext(ctx).Create(alert).Error; err != nil {
		return err
	}
	
	// TODO: 发送通知
	
	return nil
}
