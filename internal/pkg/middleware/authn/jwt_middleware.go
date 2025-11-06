package authn

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/token"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/pkg/core"
)

// JWTAuthMiddleware JWT 认证中间件
// 使用新的认证模块来验证令牌
type JWTAuthMiddleware struct {
	tokenService token.TokenApplicationService
}

// NewJWTAuthMiddleware 创建 JWT 认证中间件
func NewJWTAuthMiddleware(tokenService token.TokenApplicationService) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{
		tokenService: tokenService,
	}
}

// AuthRequired 认证必需中间件
// 验证请求中的 JWT 令牌,如果无效则返回 401
func (m *JWTAuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenValue := m.extractToken(c)
		if tokenValue == "" {
			core.WriteResponse(c, errors.WithCode(code.ErrTokenInvalid, "Missing authorization token"), nil)
			c.Abort()
			return
		}

		// 验证令牌
		resp, err := m.tokenService.VerifyToken(c.Request.Context(), tokenValue)
		if err != nil {
			log.Warnf("Token verification failed: %v", err)
			core.WriteResponse(c, errors.WithCode(code.ErrTokenInvalid, "Token verification failed"), nil)
			c.Abort()
			return
		}

		if !resp.Valid {
			core.WriteResponse(c, errors.WithCode(code.ErrTokenInvalid, "Invalid or expired token"), nil)
			c.Abort()
			return
		}

		// 将用户信息存入上下文（从 Claims 中读取）
		if resp.Claims != nil {
			c.Set(ContextKeyUserID, resp.Claims.UserID.String())
			c.Set(ContextKeyAccountID, resp.Claims.AccountID.String())
			c.Set(ContextKeyTokenID, resp.Claims.TokenID)
		}

		c.Next()
	}
}

// AuthOptional 可选认证中间件
// 如果有令牌则验证,没有令牌也允许通过(但不设置用户信息)
func (m *JWTAuthMiddleware) AuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenValue := m.extractToken(c)
		if tokenValue == "" {
			// 没有令牌,允许通过
			c.Next()
			return
		}

		// 验证令牌
		resp, err := m.tokenService.VerifyToken(c.Request.Context(), tokenValue)
		if err != nil {
			log.Warnf("Token verification failed (optional): %v", err)
			// 令牌无效,但允许通过(不设置用户信息)
			c.Next()
			return
		}

		if !resp.Valid {
			// 令牌无效,但允许通过(不设置用户信息)
			c.Next()
			return
		}

		// 将用户信息存入上下文（从 Claims 中读取）
		if resp.Claims != nil {
			c.Set(ContextKeyUserID, resp.Claims.UserID.String())
			c.Set(ContextKeyAccountID, resp.Claims.AccountID.String())
			c.Set(ContextKeyTokenID, resp.Claims.TokenID)
		}

		c.Next()
	}
}

// RequireRole 要求特定角色的中间件
// 必须在 AuthRequired 之后使用
func (m *JWTAuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户信息
		accountID, exists := c.Get(ContextKeyAccountID)
		if !exists {
			core.WriteResponse(c, errors.WithCode(code.ErrUnauthorized, "Not authenticated"), nil)
			c.Abort()
			return
		}

		// TODO: 从数据库或缓存查询用户角色
		// 这里需要注入 AccountRepository 或 RoleService
		log.Infof("Checking roles for account: %v, required roles: %v", accountID, roles)

		// 暂时放行,待实现角色系统
		c.Next()
	}
}

// RequirePermission 要求特定权限的中间件
// 必须在 AuthRequired 之后使用
func (m *JWTAuthMiddleware) RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户信息
		accountID, exists := c.Get(ContextKeyAccountID)
		if !exists {
			core.WriteResponse(c, errors.WithCode(code.ErrUnauthorized, "Not authenticated"), nil)
			c.Abort()
			return
		}

		// TODO: 从数据库或缓存查询用户权限
		log.Infof("Checking permissions for account: %v, required permissions: %v", accountID, permissions)

		// 暂时放行,待实现权限系统
		c.Next()
	}
}

// extractToken 从请求中提取令牌
// 支持多种方式：Authorization Header, Query Parameter, Cookie
func (m *JWTAuthMiddleware) extractToken(c *gin.Context) string {
	// 1. 从 Authorization Header 提取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// 支持 "Bearer <token>" 格式
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
		// 也支持直接传递令牌（无 Bearer 前缀）
		return authHeader
	}

	// 2. 从查询参数提取
	if token := c.Query("token"); token != "" {
		return token
	}

	// 3. 从 Cookie 提取
	if token, err := c.Cookie("access_token"); err == nil && token != "" {
		return token
	}

	return ""
}

// GetCurrentUserID 从上下文获取当前用户 ID
func GetCurrentUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	if id, ok := userID.(string); ok {
		return id, true
	}
	return "", false
}

// GetCurrentAccountID 从上下文获取当前账户 ID
func GetCurrentAccountID(c *gin.Context) (string, bool) {
	accountID, exists := c.Get("account_id")
	if !exists {
		return "", false
	}
	if id, ok := accountID.(string); ok {
		return id, true
	}
	return "", false
}

// GetCurrentSessionID 从上下文获取当前会话 ID
func GetCurrentSessionID(c *gin.Context) (string, bool) {
	sessionID, exists := c.Get("session_id")
	if !exists {
		return "", false
	}
	if id, ok := sessionID.(string); ok {
		return id, true
	}
	return "", false
}

// RequireAuth 便捷函数：创建认证必需中间件
func RequireAuth(tokenService token.TokenApplicationService) gin.HandlerFunc {
	return NewJWTAuthMiddleware(tokenService).AuthRequired()
}

// OptionalAuth 便捷函数：创建可选认证中间件
func OptionalAuth(tokenService token.TokenApplicationService) gin.HandlerFunc {
	return NewJWTAuthMiddleware(tokenService).AuthOptional()
}
