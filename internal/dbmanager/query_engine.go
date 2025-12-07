package dbmanager

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// QueryEngine SQL查询执行引擎
type QueryEngine struct {
	service DatabaseService
}

// NewQueryEngine 创建查询执行引擎
func NewQueryEngine(service DatabaseService) *QueryEngine {
	return &QueryEngine{
		service: service,
	}
}

// Execute 执行SQL语句
func (qe *QueryEngine) Execute(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	// 验证SQL语句
	if err := qe.validateSQL(req.SQL); err != nil {
		return nil, err
	}
	
	// 设置默认超时
	if req.Timeout == 0 {
		req.Timeout = 30 // 默认30秒
	}
	
	// 设置默认最大行数
	if req.MaxRows == 0 {
		req.MaxRows = 1000 // 默认最多返回1000行
	}
	
	// 执行查询
	return qe.service.ExecuteQuery(ctx, req)
}

// ExecuteBatch 批量执行SQL语句
func (qe *QueryEngine) ExecuteBatch(ctx context.Context, connectionID uint, sqlStatements []string) ([]*QueryResult, error) {
	results := make([]*QueryResult, 0, len(sqlStatements))
	
	for i, sql := range sqlStatements {
		req := &QueryRequest{
			ConnectionID: connectionID,
			SQL:          sql,
			Timeout:      30,
			MaxRows:      1000,
		}
		
		result, err := qe.Execute(ctx, req)
		if err != nil {
			return results, fmt.Errorf("执行第%d条SQL失败: %w", i+1, err)
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// ExecuteWithPagination 分页执行查询
func (qe *QueryEngine) ExecuteWithPagination(ctx context.Context, req *QueryRequest, page, pageSize int) (*PaginatedResult, error) {
	// 验证分页参数
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 100 // 默认每页100条
	}
	
	// 计算总数（如果是SELECT语句）
	var total int64
	if qe.isSelectStatement(req.SQL) {
		countSQL := qe.buildCountSQL(req.SQL)
		countReq := &QueryRequest{
			ConnectionID: req.ConnectionID,
			SQL:          countSQL,
			Timeout:      req.Timeout,
		}
		
		countResult, err := qe.service.ExecuteQuery(ctx, countReq)
		if err == nil && len(countResult.Rows) > 0 {
			if count, ok := countResult.Rows[0]["count"].(int64); ok {
				total = count
			}
		}
	}
	
	// 添加分页限制
	offset := (page - 1) * pageSize
	paginatedSQL := fmt.Sprintf("%s LIMIT %d OFFSET %d", req.SQL, pageSize, offset)
	
	req.SQL = paginatedSQL
	req.MaxRows = pageSize
	
	result, err := qe.Execute(ctx, req)
	if err != nil {
		return nil, err
	}
	
	return &PaginatedResult{
		QueryResult: *result,
		Page:        page,
		PageSize:    pageSize,
		Total:       total,
		TotalPages:  int((total + int64(pageSize) - 1) / int64(pageSize)),
	}, nil
}

// ExplainQuery 解释查询执行计划
func (qe *QueryEngine) ExplainQuery(ctx context.Context, connectionID uint, sql string) (*ExecutionPlan, error) {
	return qe.service.GetExecutionPlan(ctx, connectionID, sql)
}

// validateSQL 验证SQL语句
func (qe *QueryEngine) validateSQL(sql string) error {
	if sql == "" {
		return ErrInvalidSQL
	}
	
	// 移除前后空白
	sql = strings.TrimSpace(sql)
	
	// 检查危险操作（可以根据需要扩展）
	dangerousKeywords := []string{
		"DROP DATABASE",
		"DROP SCHEMA",
		"TRUNCATE DATABASE",
	}
	
	upperSQL := strings.ToUpper(sql)
	for _, keyword := range dangerousKeywords {
		if strings.Contains(upperSQL, keyword) {
			return fmt.Errorf("禁止执行危险操作: %s", keyword)
		}
	}
	
	return nil
}

// isSelectStatement 判断是否为SELECT语句
func (qe *QueryEngine) isSelectStatement(sql string) bool {
	sql = strings.TrimSpace(strings.ToUpper(sql))
	return strings.HasPrefix(sql, "SELECT")
}

// buildCountSQL 构建COUNT查询
func (qe *QueryEngine) buildCountSQL(sql string) string {
	// 简单实现：将SELECT语句包装为子查询
	return fmt.Sprintf("SELECT COUNT(*) as count FROM (%s) as subquery", sql)
}

// PaginatedResult 分页查询结果
type PaginatedResult struct {
	QueryResult
	Page       int   `json:"page"`        // 当前页码
	PageSize   int   `json:"page_size"`   // 每页大小
	Total      int64 `json:"total"`       // 总记录数
	TotalPages int   `json:"total_pages"` // 总页数
}

// QueryParser SQL查询解析器
type QueryParser struct{}

// NewQueryParser 创建查询解析器
func NewQueryParser() *QueryParser {
	return &QueryParser{}
}

// Parse 解析SQL语句
func (qp *QueryParser) Parse(sql string) (*ParsedQuery, error) {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return nil, ErrInvalidSQL
	}
	
	upperSQL := strings.ToUpper(sql)
	
	parsed := &ParsedQuery{
		OriginalSQL: sql,
		Type:        qp.detectQueryType(upperSQL),
		Tables:      qp.extractTables(sql),
		Columns:     qp.extractColumns(sql),
	}
	
	return parsed, nil
}

// detectQueryType 检测查询类型
func (qp *QueryParser) detectQueryType(upperSQL string) string {
	switch {
	case strings.HasPrefix(upperSQL, "SELECT"):
		return "SELECT"
	case strings.HasPrefix(upperSQL, "INSERT"):
		return "INSERT"
	case strings.HasPrefix(upperSQL, "UPDATE"):
		return "UPDATE"
	case strings.HasPrefix(upperSQL, "DELETE"):
		return "DELETE"
	case strings.HasPrefix(upperSQL, "CREATE"):
		return "CREATE"
	case strings.HasPrefix(upperSQL, "ALTER"):
		return "ALTER"
	case strings.HasPrefix(upperSQL, "DROP"):
		return "DROP"
	default:
		return "UNKNOWN"
	}
}

// extractTables 提取表名（简化实现）
func (qp *QueryParser) extractTables(sql string) []string {
	// 这是一个简化的实现，实际应该使用SQL解析器
	tables := make([]string, 0)
	
	upperSQL := strings.ToUpper(sql)
	
	// 查找FROM子句
	if idx := strings.Index(upperSQL, "FROM"); idx != -1 {
		afterFrom := sql[idx+4:]
		// 简单提取第一个单词作为表名
		words := strings.Fields(afterFrom)
		if len(words) > 0 {
			tableName := strings.Trim(words[0], ",;")
			tables = append(tables, tableName)
		}
	}
	
	return tables
}

// extractColumns 提取列名（简化实现）
func (qp *QueryParser) extractColumns(sql string) []string {
	// 这是一个简化的实现
	columns := make([]string, 0)
	
	upperSQL := strings.ToUpper(sql)
	
	// 只处理SELECT语句
	if !strings.HasPrefix(upperSQL, "SELECT") {
		return columns
	}
	
	// 查找SELECT和FROM之间的内容
	selectIdx := strings.Index(upperSQL, "SELECT")
	fromIdx := strings.Index(upperSQL, "FROM")
	
	if selectIdx != -1 && fromIdx != -1 && fromIdx > selectIdx {
		columnsPart := sql[selectIdx+6 : fromIdx]
		columnsPart = strings.TrimSpace(columnsPart)
		
		// 如果是SELECT *，返回空列表
		if columnsPart == "*" {
			return columns
		}
		
		// 分割列名
		parts := strings.Split(columnsPart, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			// 移除别名
			if idx := strings.Index(strings.ToUpper(part), " AS "); idx != -1 {
				part = part[:idx]
			}
			columns = append(columns, strings.TrimSpace(part))
		}
	}
	
	return columns
}

// ParsedQuery 解析后的查询
type ParsedQuery struct {
	OriginalSQL string   `json:"original_sql"` // 原始SQL
	Type        string   `json:"type"`         // 查询类型
	Tables      []string `json:"tables"`       // 涉及的表
	Columns     []string `json:"columns"`      // 涉及的列
}

// QueryFormatter SQL查询格式化器
type QueryFormatter struct{}

// NewQueryFormatter 创建查询格式化器
func NewQueryFormatter() *QueryFormatter {
	return &QueryFormatter{}
}

// Format 格式化SQL语句
func (qf *QueryFormatter) Format(sql string) string {
	// 简单的格式化实现
	sql = strings.TrimSpace(sql)
	
	// 替换多个空格为单个空格
	sql = strings.Join(strings.Fields(sql), " ")
	
	// 关键字大写
	keywords := []string{
		"SELECT", "FROM", "WHERE", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER",
		"ON", "AND", "OR", "ORDER BY", "GROUP BY", "HAVING", "LIMIT", "OFFSET",
		"INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE", "CREATE", "ALTER",
		"DROP", "TABLE", "INDEX", "VIEW", "DATABASE",
	}
	
	for _, keyword := range keywords {
		sql = strings.ReplaceAll(sql, " "+strings.ToLower(keyword)+" ", " "+keyword+" ")
	}
	
	return sql
}

// QueryValidator SQL查询验证器
type QueryValidator struct {
	maxQueryLength int
	allowedTypes   map[string]bool
}

// NewQueryValidator 创建查询验证器
func NewQueryValidator() *QueryValidator {
	return &QueryValidator{
		maxQueryLength: 10000, // 最大查询长度
		allowedTypes: map[string]bool{
			"SELECT": true,
			"INSERT": true,
			"UPDATE": true,
			"DELETE": true,
		},
	}
}

// Validate 验证SQL查询
func (qv *QueryValidator) Validate(sql string) error {
	// 检查长度
	if len(sql) > qv.maxQueryLength {
		return fmt.Errorf("SQL语句过长，最大允许%d字符", qv.maxQueryLength)
	}
	
	// 检查是否为空
	if strings.TrimSpace(sql) == "" {
		return ErrInvalidSQL
	}
	
	// 解析查询类型
	parser := NewQueryParser()
	parsed, err := parser.Parse(sql)
	if err != nil {
		return err
	}
	
	// 检查是否允许该类型的查询
	if !qv.allowedTypes[parsed.Type] {
		return fmt.Errorf("不允许执行%s类型的查询", parsed.Type)
	}
	
	return nil
}

// SetAllowedTypes 设置允许的查询类型
func (qv *QueryValidator) SetAllowedTypes(types []string) {
	qv.allowedTypes = make(map[string]bool)
	for _, t := range types {
		qv.allowedTypes[strings.ToUpper(t)] = true
	}
}

// QueryCache 查询缓存
type QueryCache struct {
	cache map[string]*CachedResult
	ttl   time.Duration
}

// CachedResult 缓存的查询结果
type CachedResult struct {
	Result    *QueryResult
	CachedAt  time.Time
	ExpiresAt time.Time
}

// NewQueryCache 创建查询缓存
func NewQueryCache(ttl time.Duration) *QueryCache {
	return &QueryCache{
		cache: make(map[string]*CachedResult),
		ttl:   ttl,
	}
}

// Get 获取缓存的查询结果
func (qc *QueryCache) Get(key string) (*QueryResult, bool) {
	cached, exists := qc.cache[key]
	if !exists {
		return nil, false
	}
	
	// 检查是否过期
	if time.Now().After(cached.ExpiresAt) {
		delete(qc.cache, key)
		return nil, false
	}
	
	return cached.Result, true
}

// Set 设置查询结果缓存
func (qc *QueryCache) Set(key string, result *QueryResult) {
	now := time.Now()
	qc.cache[key] = &CachedResult{
		Result:    result,
		CachedAt:  now,
		ExpiresAt: now.Add(qc.ttl),
	}
}

// Clear 清空缓存
func (qc *QueryCache) Clear() {
	qc.cache = make(map[string]*CachedResult)
}

// GenerateCacheKey 生成缓存键
func GenerateCacheKey(connectionID uint, sql string) string {
	return fmt.Sprintf("%d:%s", connectionID, sql)
}
