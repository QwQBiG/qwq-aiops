package dbmanager

import (
	"context"
	"fmt"
	"strings"
)

// AIQueryOptimizer AI查询优化器
type AIQueryOptimizer struct {
	service DatabaseService
}

// NewAIQueryOptimizer 创建AI查询优化器
func NewAIQueryOptimizer(service DatabaseService) *AIQueryOptimizer {
	return &AIQueryOptimizer{
		service: service,
	}
}

// OptimizeQuery 优化SQL查询
func (opt *AIQueryOptimizer) OptimizeQuery(ctx context.Context, connectionID uint, sql string) (*QueryOptimization, error) {
	// 获取执行计划
	plan, err := opt.service.GetExecutionPlan(ctx, connectionID, sql)
	if err != nil {
		return nil, fmt.Errorf("获取执行计划失败: %w", err)
	}
	
	// 分析执行计划
	suggestions := opt.analyzeExecutionPlan(plan)
	
	// 生成索引推荐
	indexRecommendations := opt.generateIndexRecommendations(plan, sql)
	
	// 优化SQL语句
	optimizedSQL := opt.rewriteSQL(sql, suggestions)
	
	// 估算性能提升
	improvement := opt.estimateImprovement(plan, suggestions)
	
	return &QueryOptimization{
		OriginalSQL:          sql,
		OptimizedSQL:         optimizedSQL,
		Suggestions:          suggestions,
		EstimatedImprovement: improvement,
		IndexRecommendations: indexRecommendations,
	}, nil
}

// analyzeExecutionPlan 分析执行计划
func (opt *AIQueryOptimizer) analyzeExecutionPlan(plan *ExecutionPlan) []string {
	suggestions := make([]string, 0)
	
	for _, step := range plan.Steps {
		// 检查全表扫描
		if step.Type == "ALL" {
			suggestions = append(suggestions, 
				fmt.Sprintf("表 %s 正在进行全表扫描，建议添加索引", step.Table))
		}
		
		// 检查是否使用了索引
		if step.Key == "" && step.Type != "const" {
			suggestions = append(suggestions,
				fmt.Sprintf("表 %s 未使用索引，查询性能可能较差", step.Table))
		}
		
		// 检查扫描行数
		if step.Rows > 10000 {
			suggestions = append(suggestions,
				fmt.Sprintf("表 %s 扫描了 %d 行数据，建议优化查询条件或添加索引", step.Table, step.Rows))
		}
		
		// 检查临时表
		if strings.Contains(step.Extra, "Using temporary") {
			suggestions = append(suggestions,
				"查询使用了临时表，可能影响性能，建议优化GROUP BY或ORDER BY")
		}
		
		// 检查文件排序
		if strings.Contains(step.Extra, "Using filesort") {
			suggestions = append(suggestions,
				"查询使用了文件排序，建议在ORDER BY的列上添加索引")
		}
	}
	
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "查询执行计划良好，暂无优化建议")
	}
	
	return suggestions
}

// generateIndexRecommendations 生成索引推荐
func (opt *AIQueryOptimizer) generateIndexRecommendations(plan *ExecutionPlan, sql string) []IndexRecommendation {
	recommendations := make([]IndexRecommendation, 0)
	
	// 解析SQL获取WHERE条件中的列
	whereColumns := opt.extractWhereColumns(sql)
	
	for _, step := range plan.Steps {
		// 如果是全表扫描且有WHERE条件
		if step.Type == "ALL" && len(whereColumns) > 0 {
			for table, columns := range whereColumns {
				if table == step.Table {
					recommendations = append(recommendations, IndexRecommendation{
						Table:    table,
						Columns:  columns,
						Reason:   "WHERE条件中的列未使用索引，导致全表扫描",
						Priority: "high",
						CreateSQL: fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s (%s)",
							table, strings.Join(columns, "_"), table, strings.Join(columns, ", ")),
					})
				}
			}
		}
		
		// 如果使用了文件排序
		if strings.Contains(step.Extra, "Using filesort") {
			orderColumns := opt.extractOrderByColumns(sql)
			if len(orderColumns) > 0 {
				recommendations = append(recommendations, IndexRecommendation{
					Table:    step.Table,
					Columns:  orderColumns,
					Reason:   "ORDER BY使用了文件排序，建议添加索引",
					Priority: "medium",
					CreateSQL: fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s (%s)",
						step.Table, strings.Join(orderColumns, "_"), step.Table, strings.Join(orderColumns, ", ")),
				})
			}
		}
	}
	
	return recommendations
}

// rewriteSQL 重写SQL语句
func (opt *AIQueryOptimizer) rewriteSQL(sql string, suggestions []string) string {
	// 简单的SQL重写逻辑
	optimized := sql
	
	// 移除不必要的SELECT *
	if strings.Contains(strings.ToUpper(sql), "SELECT *") {
		// 实际应该分析需要的列，这里只是示例
		// optimized = strings.Replace(optimized, "SELECT *", "SELECT specific_columns", 1)
	}
	
	// 添加LIMIT限制（如果没有）
	upperSQL := strings.ToUpper(sql)
	if strings.HasPrefix(upperSQL, "SELECT") && !strings.Contains(upperSQL, "LIMIT") {
		// 可以考虑添加LIMIT，但需要根据实际情况判断
	}
	
	return optimized
}

// estimateImprovement 估算性能提升
func (opt *AIQueryOptimizer) estimateImprovement(plan *ExecutionPlan, suggestions []string) float64 {
	// 简单的性能提升估算
	improvement := 0.0
	
	for _, step := range plan.Steps {
		// 全表扫描的改进潜力最大
		if step.Type == "ALL" {
			improvement += 50.0
		}
		
		// 使用临时表
		if strings.Contains(step.Extra, "Using temporary") {
			improvement += 20.0
		}
		
		// 文件排序
		if strings.Contains(step.Extra, "Using filesort") {
			improvement += 15.0
		}
	}
	
	// 限制最大值为95%
	if improvement > 95.0 {
		improvement = 95.0
	}
	
	return improvement
}

// extractWhereColumns 提取WHERE条件中的列
func (opt *AIQueryOptimizer) extractWhereColumns(sql string) map[string][]string {
	columns := make(map[string][]string)
	
	upperSQL := strings.ToUpper(sql)
	whereIdx := strings.Index(upperSQL, "WHERE")
	if whereIdx == -1 {
		return columns
	}
	
	// 简化实现：提取WHERE后面的内容
	wherePart := sql[whereIdx+5:]
	
	// 查找可能的结束位置
	endKeywords := []string{"GROUP BY", "ORDER BY", "LIMIT", "HAVING"}
	endIdx := len(wherePart)
	for _, keyword := range endKeywords {
		if idx := strings.Index(strings.ToUpper(wherePart), keyword); idx != -1 && idx < endIdx {
			endIdx = idx
		}
	}
	
	wherePart = wherePart[:endIdx]
	
	// 简单提取列名（实际应该使用SQL解析器）
	// 这里只是示例实现
	words := strings.Fields(wherePart)
	for i, word := range words {
		if i > 0 && (words[i-1] == "=" || words[i-1] == ">" || words[i-1] == "<" || 
			words[i-1] == ">=" || words[i-1] == "<=" || words[i-1] == "!=") {
			// 前一个词是比较运算符，当前词可能是列名
			continue
		}
		
		// 检查是否包含表名.列名格式
		if strings.Contains(word, ".") {
			parts := strings.Split(word, ".")
			if len(parts) == 2 {
				table := parts[0]
				column := parts[1]
				columns[table] = append(columns[table], column)
			}
		}
	}
	
	return columns
}

// extractOrderByColumns 提取ORDER BY中的列
func (opt *AIQueryOptimizer) extractOrderByColumns(sql string) []string {
	columns := make([]string, 0)
	
	upperSQL := strings.ToUpper(sql)
	orderIdx := strings.Index(upperSQL, "ORDER BY")
	if orderIdx == -1 {
		return columns
	}
	
	// 提取ORDER BY后面的内容
	orderPart := sql[orderIdx+8:]
	
	// 查找可能的结束位置
	endKeywords := []string{"LIMIT", "OFFSET"}
	endIdx := len(orderPart)
	for _, keyword := range endKeywords {
		if idx := strings.Index(strings.ToUpper(orderPart), keyword); idx != -1 && idx < endIdx {
			endIdx = idx
		}
	}
	
	orderPart = strings.TrimSpace(orderPart[:endIdx])
	
	// 分割列名
	parts := strings.Split(orderPart, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// 移除ASC/DESC
		part = strings.TrimSuffix(strings.TrimSuffix(part, " DESC"), " ASC")
		part = strings.TrimSpace(part)
		
		// 移除表名前缀
		if strings.Contains(part, ".") {
			parts := strings.Split(part, ".")
			if len(parts) == 2 {
				part = parts[1]
			}
		}
		
		columns = append(columns, part)
	}
	
	return columns
}

// PerformanceAnalyzer 性能分析器
type PerformanceAnalyzer struct {
	service DatabaseService
}

// NewPerformanceAnalyzer 创建性能分析器
func NewPerformanceAnalyzer(service DatabaseService) *PerformanceAnalyzer {
	return &PerformanceAnalyzer{
		service: service,
	}
}

// AnalyzeQuery 分析查询性能
func (pa *PerformanceAnalyzer) AnalyzeQuery(ctx context.Context, connectionID uint, sql string) (*PerformanceReport, error) {
	// 获取执行计划
	plan, err := pa.service.GetExecutionPlan(ctx, connectionID, sql)
	if err != nil {
		return nil, fmt.Errorf("获取执行计划失败: %w", err)
	}
	
	// 执行查询并测量时间
	req := &QueryRequest{
		ConnectionID: connectionID,
		SQL:          sql,
		Timeout:      30,
		MaxRows:      1,
	}
	
	result, err := pa.service.ExecuteQuery(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	
	// 分析性能指标
	report := &PerformanceReport{
		SQL:           sql,
		ExecutionTime: result.ExecutionTime,
		RowsScanned:   pa.calculateRowsScanned(plan),
		RowsReturned:  int64(len(result.Rows)),
		IndexUsed:     pa.checkIndexUsage(plan),
		Issues:        pa.identifyIssues(plan, result),
		Recommendations: pa.generateRecommendations(plan, result),
	}
	
	return report, nil
}

// calculateRowsScanned 计算扫描的行数
func (pa *PerformanceAnalyzer) calculateRowsScanned(plan *ExecutionPlan) int64 {
	var total int64
	for _, step := range plan.Steps {
		total += step.Rows
	}
	return total
}

// checkIndexUsage 检查索引使用情况
func (pa *PerformanceAnalyzer) checkIndexUsage(plan *ExecutionPlan) bool {
	for _, step := range plan.Steps {
		if step.Key != "" {
			return true
		}
	}
	return false
}

// identifyIssues 识别性能问题
func (pa *PerformanceAnalyzer) identifyIssues(plan *ExecutionPlan, result *QueryResult) []string {
	issues := make([]string, 0)
	
	// 检查执行时间
	if result.ExecutionTime > 1000 {
		issues = append(issues, fmt.Sprintf("查询执行时间过长: %.2f ms", result.ExecutionTime))
	}
	
	// 检查扫描行数
	rowsScanned := int64(0)
	for _, step := range plan.Steps {
		rowsScanned += step.Rows
		
		if step.Type == "ALL" {
			issues = append(issues, fmt.Sprintf("表 %s 进行了全表扫描", step.Table))
		}
	}
	
	if rowsScanned > 100000 {
		issues = append(issues, fmt.Sprintf("扫描了大量数据: %d 行", rowsScanned))
	}
	
	return issues
}

// generateRecommendations 生成优化建议
func (pa *PerformanceAnalyzer) generateRecommendations(plan *ExecutionPlan, result *QueryResult) []string {
	recommendations := make([]string, 0)
	
	for _, step := range plan.Steps {
		if step.Type == "ALL" {
			recommendations = append(recommendations,
				fmt.Sprintf("在表 %s 上添加适当的索引以避免全表扫描", step.Table))
		}
		
		if strings.Contains(step.Extra, "Using temporary") {
			recommendations = append(recommendations,
				"优化GROUP BY或DISTINCT操作以避免使用临时表")
		}
		
		if strings.Contains(step.Extra, "Using filesort") {
			recommendations = append(recommendations,
				"在ORDER BY的列上添加索引以避免文件排序")
		}
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "查询性能良好，暂无优化建议")
	}
	
	return recommendations
}

// PerformanceReport 性能分析报告
type PerformanceReport struct {
	SQL             string   `json:"sql"`              // SQL语句
	ExecutionTime   float64  `json:"execution_time"`   // 执行时间（毫秒）
	RowsScanned     int64    `json:"rows_scanned"`     // 扫描行数
	RowsReturned    int64    `json:"rows_returned"`    // 返回行数
	IndexUsed       bool     `json:"index_used"`       // 是否使用索引
	Issues          []string `json:"issues"`           // 性能问题
	Recommendations []string `json:"recommendations"`  // 优化建议
}
