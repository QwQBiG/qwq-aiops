package monitoring

import (
	"context"
	"encoding/json"
	"time"
)

// AlertEvaluator 告警评估器
type AlertEvaluator struct {
	service *MonitoringServiceImpl
	ticker  *time.Ticker
}

// NewAlertEvaluator 创建告警评估器
func NewAlertEvaluator(service *MonitoringServiceImpl) *AlertEvaluator {
	return &AlertEvaluator{
		service: service,
		ticker:  time.NewTicker(30 * time.Second), // 每30秒评估一次
	}
}

// Start 启动评估器
func (e *AlertEvaluator) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			e.ticker.Stop()
			return
		case <-e.ticker.C:
			e.evaluate(ctx)
		}
	}
}

// evaluate 评估所有告警规则
func (e *AlertEvaluator) evaluate(ctx context.Context) {
	// 获取所有启用的告警规则
	var rules []*AlertRule
	e.service.db.WithContext(ctx).Where("enabled = ?", true).Find(&rules)
	
	for _, rule := range rules {
		e.evaluateRule(ctx, rule)
	}
}

// evaluateRule 评估单个告警规则
func (e *AlertEvaluator) evaluateRule(ctx context.Context, rule *AlertRule) {
	// 解析标签
	var labels map[string]string
	if rule.Labels != "" {
		json.Unmarshal([]byte(rule.Labels), &labels)
	}
	
	// 查询指标数据
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(rule.Duration) * time.Second)
	if rule.Duration == 0 {
		startTime = endTime.Add(-5 * time.Minute) // 默认5分钟
	}
	
	query := &MetricQuery{
		MetricName:  rule.MetricName,
		Labels:      labels,
		StartTime:   startTime,
		EndTime:     endTime,
		Aggregation: rule.Aggregation,
	}
	
	data, err := e.service.QueryMetrics(ctx, query)
	if err != nil || len(data) == 0 {
		return
	}
	
	// 计算聚合值
	value := e.calculateAggregatedValue(data, rule.Aggregation)
	
	// 评估条件
	if e.evaluateCondition(value, rule.Operator, rule.Threshold) {
		// 触发告警
		e.service.FireAlert(ctx, rule, value, labels)
	}
}

// calculateAggregatedValue 计算聚合值
func (e *AlertEvaluator) calculateAggregatedValue(data []*MetricData, aggregation string) float64 {
	if len(data) == 0 {
		return 0
	}
	
	switch aggregation {
	case "sum":
		var sum float64
		for _, point := range data {
			sum += point.Value
		}
		return sum
		
	case "min":
		min := data[0].Value
		for _, point := range data {
			if point.Value < min {
				min = point.Value
			}
		}
		return min
		
	case "max":
		max := data[0].Value
		for _, point := range data {
			if point.Value > max {
				max = point.Value
			}
		}
		return max
		
	case "avg":
		fallthrough
	default:
		var sum float64
		for _, point := range data {
			sum += point.Value
		}
		return sum / float64(len(data))
	}
}

// evaluateCondition 评估条件
func (e *AlertEvaluator) evaluateCondition(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}
