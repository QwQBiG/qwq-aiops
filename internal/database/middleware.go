package database

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ContextKey 上下文键类型
type ContextKey string

const (
	// UserIDKey 用户ID的上下文键
	UserIDKey ContextKey = "user_id"
	// TenantIDKey 租户ID的上下文键
	TenantIDKey ContextKey = "tenant_id"
	// UserRoleKey 用户角色的上下文键
	UserRoleKey ContextKey = "user_role"
)

// PermissionMiddleware 权限检查中间件
type PermissionMiddleware struct {
	rbacService  *RBACService
	auditService *AuditService
}

// NewPermissionMiddleware 创建权限中间件实例
func NewPermissionMiddleware(rbacService *RBACService, auditService *AuditService) *PermissionMiddleware {
	return &PermissionMiddleware{
		rbacService:  rbacService,
		auditService: auditService,
	}
}

// RequirePermission 返回一个需要特定权限的中间件函数
func (m *PermissionMiddleware) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 从上下文获取用户ID
			userID, ok := ctx.Value(UserIDKey).(uint)
			if !ok || userID == 0 {
				m.respondError(w, http.StatusUnauthorized, "未认证")
				return
			}

			// 检查权限
			hasPermission, err := m.rbacService.CheckPermission(ctx, userID, resource, action)
			if err != nil {
				m.respondError(w, http.StatusInternalServerError, fmt.Sprintf("权限检查失败: %v", err))
				return
			}

			if !hasPermission {
				// 记录权限拒绝日志
				tenantID, _ := ctx.Value(TenantIDKey).(uint)
				_ = m.auditService.LogFailure(
					ctx,
					userID,
					tenantID,
					action,
					resource,
					"",
					nil,
					"权限不足",
					m.getClientIP(r),
					r.UserAgent(),
				)

				m.respondError(w, http.StatusForbidden, "权限不足")
				return
			}

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

// RequireTenantAccess 返回一个需要租户访问权限的中间件函数
func (m *PermissionMiddleware) RequireTenantAccess() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 从上下文获取用户ID
			userID, ok := ctx.Value(UserIDKey).(uint)
			if !ok || userID == 0 {
				m.respondError(w, http.StatusUnauthorized, "未认证")
				return
			}

			// 从请求中获取租户ID（可以从URL参数、请求头或请求体中获取）
			tenantIDStr := r.URL.Query().Get("tenant_id")
			if tenantIDStr == "" {
				tenantIDStr = r.Header.Get("X-Tenant-ID")
			}

			if tenantIDStr == "" {
				m.respondError(w, http.StatusBadRequest, "缺少租户ID")
				return
			}

			tenantID, err := strconv.ParseUint(tenantIDStr, 10, 32)
			if err != nil {
				m.respondError(w, http.StatusBadRequest, "无效的租户ID")
				return
			}

			// 检查租户访问权限
			hasAccess, err := m.rbacService.CheckTenantAccess(ctx, userID, uint(tenantID))
			if err != nil {
				m.respondError(w, http.StatusInternalServerError, fmt.Sprintf("租户访问检查失败: %v", err))
				return
			}

			if !hasAccess {
				// 记录访问拒绝日志
				_ = m.auditService.LogFailure(
					ctx,
					userID,
					uint(tenantID),
					"access",
					"tenant",
					tenantIDStr,
					nil,
					"无权访问该租户资源",
					m.getClientIP(r),
					r.UserAgent(),
				)

				m.respondError(w, http.StatusForbidden, "无权访问该租户资源")
				return
			}

			// 将租户ID添加到上下文
			ctx = context.WithValue(ctx, TenantIDKey, uint(tenantID))
			r = r.WithContext(ctx)

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

// AuditLog 返回一个记录审计日志的中间件函数
func (m *PermissionMiddleware) AuditLog(resource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 从上下文获取用户ID和租户ID
			userID, _ := ctx.Value(UserIDKey).(uint)
			tenantID, _ := ctx.Value(TenantIDKey).(uint)

			// 确定操作类型
			action := m.getActionFromMethod(r.Method)

			// 创建响应记录器以捕获状态码
			recorder := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// 处理请求
			next.ServeHTTP(recorder, r)

			// 记录审计日志
			if recorder.statusCode >= 200 && recorder.statusCode < 400 {
				_ = m.auditService.LogSuccess(
					ctx,
					userID,
					tenantID,
					action,
					resource,
					"",
					map[string]string{
						"method": r.Method,
						"path":   r.URL.Path,
					},
					m.getClientIP(r),
					r.UserAgent(),
				)
			} else {
				_ = m.auditService.LogFailure(
					ctx,
					userID,
					tenantID,
					action,
					resource,
					"",
					map[string]string{
						"method": r.Method,
						"path":   r.URL.Path,
					},
					fmt.Sprintf("HTTP %d", recorder.statusCode),
					m.getClientIP(r),
					r.UserAgent(),
				)
			}
		})
	}
}

// getActionFromMethod 根据HTTP方法确定操作类型
func (m *PermissionMiddleware) getActionFromMethod(method string) string {
	switch method {
	case http.MethodGet:
		return "read"
	case http.MethodPost:
		return "write"
	case http.MethodPut, http.MethodPatch:
		return "write"
	case http.MethodDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// getClientIP 获取客户端IP地址
func (m *PermissionMiddleware) getClientIP(r *http.Request) string {
	// 尝试从X-Forwarded-For头获取
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 尝试从X-Real-IP头获取
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 使用RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// respondError 返回错误响应
func (m *PermissionMiddleware) respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, `{"error": "%s"}`, message)
}

// responseRecorder 响应记录器，用于捕获状态码
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 记录状态码
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
