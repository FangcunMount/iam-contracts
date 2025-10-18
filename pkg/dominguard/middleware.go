// Package dominguard 中间件
package dominguard

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件配置
type AuthMiddleware struct {
	guard        *DomainGuard
	getUserID    func(*gin.Context) string // 从请求中提取用户ID
	getTenantID  func(*gin.Context) string // 从请求中提取租户ID
	errorHandler func(*gin.Context, error) // 错误处理
	skipPaths    map[string]bool           // 跳过认证的路径
}

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	Guard        *DomainGuard
	GetUserID    func(*gin.Context) string
	GetTenantID  func(*gin.Context) string
	ErrorHandler func(*gin.Context, error)
	SkipPaths    []string // 跳过认证的路径列表
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(config MiddlewareConfig) *AuthMiddleware {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}

	return &AuthMiddleware{
		guard:        config.Guard,
		getUserID:    config.GetUserID,
		getTenantID:  config.GetTenantID,
		errorHandler: config.ErrorHandler,
		skipPaths:    skipPaths,
	}
}

// RequirePermission 要求特定权限的中间件
//
// 用法:
//
//	router.GET("/users", authMiddleware.RequirePermission("user", "read"), handler)
func (m *AuthMiddleware) RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否跳过
		if m.skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// 提取用户ID和租户ID
		userID := m.getUserID(c)
		if userID == "" {
			m.errorHandler(c, &PermissionError{
				Code:    "UNAUTHORIZED",
				Message: "用户未登录",
			})
			c.Abort()
			return
		}

		tenantID := m.getTenantID(c)
		if tenantID == "" {
			m.errorHandler(c, &PermissionError{
				Code:    "INVALID_TENANT",
				Message: "租户ID缺失",
			})
			c.Abort()
			return
		}

		// 检查权限
		allowed, err := m.guard.CheckPermission(c.Request.Context(), userID, tenantID, resource, action)
		if err != nil {
			m.errorHandler(c, err)
			c.Abort()
			return
		}

		if !allowed {
			resourceName := m.guard.GetResourceDisplayName(resource)
			m.errorHandler(c, &PermissionError{
				Code:    "PERMISSION_DENIED",
				Message: "没有权限执行此操作",
				Details: map[string]string{
					"resource": resourceName,
					"action":   action,
				},
			})
			c.Abort()
			return
		}

		// 权限检查通过，继续执行
		c.Next()
	}
}

// RequireAnyPermission 要求任意一个权限的中间件
func (m *AuthMiddleware) RequireAnyPermission(permissions []Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		userID := m.getUserID(c)
		if userID == "" {
			m.errorHandler(c, &PermissionError{Code: "UNAUTHORIZED", Message: "用户未登录"})
			c.Abort()
			return
		}

		tenantID := m.getTenantID(c)
		if tenantID == "" {
			m.errorHandler(c, &PermissionError{Code: "INVALID_TENANT", Message: "租户ID缺失"})
			c.Abort()
			return
		}

		// 检查是否有任意一个权限
		for _, perm := range permissions {
			allowed, err := m.guard.CheckPermission(c.Request.Context(), userID, tenantID, perm.Resource, perm.Action)
			if err != nil {
				continue
			}
			if allowed {
				c.Next()
				return
			}
		}

		// 没有任何权限
		m.errorHandler(c, &PermissionError{
			Code:    "PERMISSION_DENIED",
			Message: "没有权限执行此操作",
		})
		c.Abort()
	}
}

// RequireAllPermissions 要求所有权限的中间件
func (m *AuthMiddleware) RequireAllPermissions(permissions []Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		userID := m.getUserID(c)
		if userID == "" {
			m.errorHandler(c, &PermissionError{Code: "UNAUTHORIZED", Message: "用户未登录"})
			c.Abort()
			return
		}

		tenantID := m.getTenantID(c)
		if tenantID == "" {
			m.errorHandler(c, &PermissionError{Code: "INVALID_TENANT", Message: "租户ID缺失"})
			c.Abort()
			return
		}

		// 批量检查所有权限
		results, err := m.guard.BatchCheckPermissions(c.Request.Context(), userID, tenantID, permissions)
		if err != nil {
			m.errorHandler(c, err)
			c.Abort()
			return
		}

		// 检查是否全部通过
		for _, allowed := range results {
			if !allowed {
				m.errorHandler(c, &PermissionError{
					Code:    "PERMISSION_DENIED",
					Message: "没有足够的权限执行此操作",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// PermissionError 权限错误
type PermissionError struct {
	Code    string
	Message string
	Details map[string]string
}

func (e *PermissionError) Error() string {
	return e.Message
}

// defaultErrorHandler 默认错误处理器
func defaultErrorHandler(c *gin.Context, err error) {
	if permErr, ok := err.(*PermissionError); ok {
		statusCode := http.StatusForbidden
		if permErr.Code == "UNAUTHORIZED" {
			statusCode = http.StatusUnauthorized
		}

		c.JSON(statusCode, gin.H{
			"error": gin.H{
				"code":    permErr.Code,
				"message": permErr.Message,
				"details": permErr.Details,
			},
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": err.Error(),
		},
	})
}

// ExtractBearerToken 从请求头中提取 Bearer Token
func ExtractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}
