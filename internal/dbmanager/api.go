package dbmanager

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// API 数据库管理API
type API struct {
	service DatabaseService
}

// NewAPI 创建数据库管理API
func NewAPI(service DatabaseService) *API {
	return &API{
		service: service,
	}
}

// RegisterRoutes 注册路由
func (api *API) RegisterRoutes(router *gin.RouterGroup) {
	db := router.Group("/database")
	{
		// 连接管理
		db.POST("/connections", api.CreateConnection)
		db.GET("/connections", api.ListConnections)
		db.GET("/connections/:id", api.GetConnection)
		db.PUT("/connections/:id", api.UpdateConnection)
		db.DELETE("/connections/:id", api.DeleteConnection)
		db.POST("/connections/test", api.TestConnection)
		
		// SQL执行
		db.POST("/query", api.ExecuteQuery)
		
		// 数据库操作
		db.GET("/connections/:id/databases", api.ListDatabases)
		db.GET("/connections/:id/databases/:database/tables", api.ListTables)
		db.GET("/connections/:id/databases/:database/tables/:table/schema", api.GetTableSchema)
		db.GET("/connections/:id/databases/:database/tables/:table/indexes", api.GetTableIndexes)
		
		// AI优化
		db.POST("/connections/:id/optimize", api.OptimizeQuery)
		db.POST("/connections/:id/explain", api.GetExecutionPlan)
		
		// 备份管理
		db.POST("/backup/configs", api.CreateBackupConfig)
		db.GET("/backup/configs", api.ListBackupConfigs)
		db.PUT("/backup/configs/:id", api.UpdateBackupConfig)
		db.DELETE("/backup/configs/:id", api.DeleteBackupConfig)
		db.POST("/backup/configs/:id/execute", api.ExecuteBackup)
		db.GET("/backup/configs/:id/records", api.ListBackupRecords)
		db.POST("/backup/records/:id/restore", api.RestoreBackup)
	}
}

// CreateConnection 创建数据库连接
func (api *API) CreateConnection(c *gin.Context) {
	var conn DatabaseConnection
	if err := c.ShouldBindJSON(&conn); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 从上下文获取用户信息（实际应该从认证中间件获取）
	// conn.UserID = c.GetUint("user_id")
	// conn.TenantID = c.GetUint("tenant_id")
	
	if err := api.service.CreateConnection(c.Request.Context(), &conn); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, conn)
}

// ListConnections 列出数据库连接
func (api *API) ListConnections(c *gin.Context) {
	// 从上下文获取用户信息
	userID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")
	
	connections, err := api.service.ListConnections(c.Request.Context(), userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, connections)
}

// GetConnection 获取数据库连接
func (api *API) GetConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的连接ID"})
		return
	}
	
	conn, err := api.service.GetConnection(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, conn)
}

// UpdateConnection 更新数据库连接
func (api *API) UpdateConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的连接ID"})
		return
	}
	
	var conn DatabaseConnection
	if err := c.ShouldBindJSON(&conn); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := api.service.UpdateConnection(c.Request.Context(), uint(id), &conn); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DeleteConnection 删除数据库连接
func (api *API) DeleteConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的连接ID"})
		return
	}
	
	if err := api.service.DeleteConnection(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// TestConnection 测试数据库连接
func (api *API) TestConnection(c *gin.Context) {
	var conn DatabaseConnection
	if err := c.ShouldBindJSON(&conn); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := api.service.TestConnection(c.Request.Context(), &conn); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "连接测试成功"})
}

// ExecuteQuery 执行SQL查询
func (api *API) ExecuteQuery(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	result, err := api.service.ExecuteQuery(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, result)
}

// ListDatabases 列出所有数据库
func (api *API) ListDatabases(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的连接ID"})
		return
	}
	
	databases, err := api.service.ListDatabases(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, databases)
}

// ListTables 列出数据库的所有表
func (api *API) ListTables(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的连接ID"})
		return
	}
	
	database := c.Param("database")
	
	tables, err := api.service.ListTables(c.Request.Context(), uint(id), database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, tables)
}

// GetTableSchema 获取表结构
func (api *API) GetTableSchema(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的连接ID"})
		return
	}
	
	database := c.Param("database")
	table := c.Param("table")
	
	schema, err := api.service.GetTableSchema(c.Request.Context(), uint(id), database, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, schema)
}

// GetTableIndexes 获取表索引
func (api *API) GetTableIndexes(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的连接ID"})
		return
	}
	
	database := c.Param("database")
	table := c.Param("table")
	
	indexes, err := api.service.GetTableIndexes(c.Request.Context(), uint(id), database, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, indexes)
}

// OptimizeQuery AI查询优化
func (api *API) OptimizeQuery(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的连接ID"})
		return
	}
	
	var req struct {
		SQL string `json:"sql" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	optimization, err := api.service.OptimizeQuery(c.Request.Context(), uint(id), req.SQL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, optimization)
}

// GetExecutionPlan 获取执行计划
func (api *API) GetExecutionPlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的连接ID"})
		return
	}
	
	var req struct {
		SQL string `json:"sql" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	plan, err := api.service.GetExecutionPlan(c.Request.Context(), uint(id), req.SQL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, plan)
}

// CreateBackupConfig 创建备份配置
func (api *API) CreateBackupConfig(c *gin.Context) {
	var config BackupConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := api.service.CreateBackupConfig(c.Request.Context(), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, config)
}

// ListBackupConfigs 列出备份配置
func (api *API) ListBackupConfigs(c *gin.Context) {
	userID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")
	
	configs, err := api.service.ListBackupConfigs(c.Request.Context(), userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, configs)
}

// UpdateBackupConfig 更新备份配置
func (api *API) UpdateBackupConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置ID"})
		return
	}
	
	var config BackupConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := api.service.UpdateBackupConfig(c.Request.Context(), uint(id), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DeleteBackupConfig 删除备份配置
func (api *API) DeleteBackupConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置ID"})
		return
	}
	
	if err := api.service.DeleteBackupConfig(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ExecuteBackup 执行备份
func (api *API) ExecuteBackup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置ID"})
		return
	}
	
	if err := api.service.ExecuteBackup(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "备份任务已启动"})
}

// ListBackupRecords 列出备份记录
func (api *API) ListBackupRecords(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置ID"})
		return
	}
	
	records, err := api.service.ListBackupRecords(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, records)
}

// RestoreBackup 恢复备份
func (api *API) RestoreBackup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的记录ID"})
		return
	}
	
	var req struct {
		TargetDatabase string `json:"target_database" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := api.service.RestoreBackup(c.Request.Context(), uint(id), req.TargetDatabase); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "恢复任务已启动"})
}
