package monitoring

import (
	"sync"
	"time"
)

// MetricsStore 指标存储接口
type MetricsStore interface {
	Store(metric *Metric) error
	Query(query *MetricQuery) ([]*MetricData, error)
}

// InMemoryMetricsStore 内存指标存储
type InMemoryMetricsStore struct {
	mu      sync.RWMutex
	metrics map[string][]*MetricData // key: metric_name
	maxSize int
}

// NewInMemoryMetricsStore 创建内存指标存储
func NewInMemoryMetricsStore() *InMemoryMetricsStore {
	return &InMemoryMetricsStore{
		metrics: make(map[string][]*MetricData),
		maxSize: 10000, // 每个指标最多保存10000个数据点
	}
}

// Store 存储指标
func (s *InMemoryMetricsStore) Store(metric *Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	data := &MetricData{
		Timestamp: metric.Timestamp,
		Value:     metric.Value,
		Labels:    metric.Labels,
	}
	
	// 添加到对应的指标列表
	s.metrics[metric.Name] = append(s.metrics[metric.Name], data)
	
	// 限制大小
	if len(s.metrics[metric.Name]) > s.maxSize {
		s.metrics[metric.Name] = s.metrics[metric.Name][1:]
	}
	
	return nil
}

// Query 查询指标
func (s *InMemoryMetricsStore) Query(query *MetricQuery) ([]*MetricData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	allData, exists := s.metrics[query.MetricName]
	if !exists {
		return []*MetricData{}, nil
	}
	
	// 过滤时间范围和标签
	var filtered []*MetricData
	for _, data := range allData {
		// 检查时间范围
		if !query.StartTime.IsZero() && data.Timestamp.Before(query.StartTime) {
			continue
		}
		if !query.EndTime.IsZero() && data.Timestamp.After(query.EndTime) {
			continue
		}
		
		// 检查标签匹配
		if !s.matchLabels(data.Labels, query.Labels) {
			continue
		}
		
		filtered = append(filtered, data)
	}
	
	// 聚合（如果需要）
	if query.Step > 0 {
		filtered = s.aggregate(filtered, query.Step, query.Aggregation)
	}
	
	return filtered, nil
}

// matchLabels 检查标签是否匹配
func (s *InMemoryMetricsStore) matchLabels(dataLabels, queryLabels map[string]string) bool {
	if len(queryLabels) == 0 {
		return true
	}
	
	for key, value := range queryLabels {
		if dataLabels[key] != value {
			return false
		}
	}
	
	return true
}

// aggregate 聚合数据
func (s *InMemoryMetricsStore) aggregate(data []*MetricData, step time.Duration, aggregation string) []*MetricData {
	if len(data) == 0 {
		return data
	}
	
	// 按时间窗口分组
	buckets := make(map[int64][]*MetricData)
	for _, point := range data {
		bucket := point.Timestamp.Unix() / int64(step.Seconds())
		buckets[bucket] = append(buckets[bucket], point)
	}
	
	// 聚合每个桶
	var result []*MetricData
	for bucket, points := range buckets {
		timestamp := time.Unix(bucket*int64(step.Seconds()), 0)
		value := s.aggregateValues(points, aggregation)
		
		result = append(result, &MetricData{
			Timestamp: timestamp,
			Value:     value,
			Labels:    points[0].Labels, // 使用第一个点的标签
		})
	}
	
	return result
}

// aggregateValues 聚合值
func (s *InMemoryMetricsStore) aggregateValues(points []*MetricData, aggregation string) float64 {
	if len(points) == 0 {
		return 0
	}
	
	switch aggregation {
	case "sum":
		var sum float64
		for _, point := range points {
			sum += point.Value
		}
		return sum
		
	case "min":
		min := points[0].Value
		for _, point := range points {
			if point.Value < min {
				min = point.Value
			}
		}
		return min
		
	case "max":
		max := points[0].Value
		for _, point := range points {
			if point.Value > max {
				max = point.Value
			}
		}
		return max
		
	case "avg":
		fallthrough
	default:
		var sum float64
		for _, point := range points {
			sum += point.Value
		}
		return sum / float64(len(points))
	}
}
