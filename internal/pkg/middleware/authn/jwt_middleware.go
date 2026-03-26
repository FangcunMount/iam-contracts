package authn

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/token"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/security/sanitize"
	"github.com/FangcunMount/iam-contracts/pkg/core"
)

// CasbinEnforcer 可选注入的授权判定端口（由 authz Casbin 适配器实现）。
// 为 nil 时 RequireRole / RequirePermission 返回服务不可用。
type CasbinEnforcer interface {
	Enforce(ctx context.Context, sub, dom, obj, act string) (bool, error)
	GetRolesForUser(ctx context.Context, user, domain string) ([]string, error)
}

// JWTAuthMiddleware JWT 认证中间件
// 使用新的认证模块来验证令牌
type JWTAuthMiddleware struct {
	tokenService token.TokenApplicationService
	casbin       CasbinEnforcer
}

// NewJWTAuthMiddleware 创建 JWT 认证中间件。
// casbin 可为 nil（仅 JWT 校验，不做角色/权限判定）。
func NewJWTAuthMiddleware(tokenService token.TokenApplicationService, casbin CasbinEnforcer) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{
		tokenService: tokenService,
		casbin:       casbin,
	}
}

// AuthRequired 认证必需中间件
// 验证请求中的 JWT 令牌,如果无效则返回 401
func (m *JWTAuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenValue, source := m.extractToken(c)
		if tokenValue == "" {
			log.Warnw("authorization token missing",
				"path", c.FullPath(),
				"method", c.Request.Method,
				"token_source", source,
				"request_id", requestIDFromContext(c),
			)
			core.WriteResponse(c, errors.WithCode(code.ErrTokenInvalid, "Missing authorization token"), nil)
			c.Abort()
			return
		}

		// 不记录完整 token，仅在中央验证处输出必要信息

		// 验证令牌
		resp, err := m.tokenService.VerifyToken(c.Request.Context(), tokenValue)
		if err != nil {
			log.Errorw("token verification request failed",
				"error", err,
				"path", c.FullPath(),
				"method", c.Request.Method,
				"token_source", source,
				"request_id", requestIDFromContext(c),
			)
			core.WriteResponse(c, errors.WithCode(code.ErrTokenInvalid, "Token verification failed"), nil)
			c.Abort()
			return
		}
		if resp == nil || !resp.Valid {
			log.Warnw("token rejected by verification",
				"path", c.FullPath(),
				"method", c.Request.Method,
				"token_source", source,
				"request_id", requestIDFromContext(c),
				"token_hint", sanitize.MaskToken(tokenValue),
			)
			core.WriteResponse(c, errors.WithCode(code.ErrTokenInvalid, "Invalid or expired token"), nil)
			c.Abort()
			return
		}

		// 将用户信息存入上下文（从 Claims 中读取）
		if resp.Claims != nil {
			ctx := context.WithValue(c.Request.Context(), ContextKeyUserID, resp.Claims.UserID)
			c.Request = c.Request.WithContext(ctx)
			c.Set(ContextKeyClaims, resp.Claims)
			uid := resp.Claims.UserID.String()
			aid := resp.Claims.AccountID.String()
			tid := resp.Claims.TokenID
			_ = tid
			c.Set(ContextKeyUserID, uid)
			c.Set(ContextKeyAccountID, aid)
			c.Set(ContextKeyTokenID, tid)
		}

		c.Next()
	}
}

// AuthOptional 可选认证中间件
// 如果有令牌则验证,没有令牌也允许通过(但不设置用户信息)
func (m *JWTAuthMiddleware) AuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenValue, source := m.extractToken(c)
		if tokenValue == "" {
			// 没有令牌,允许通过
			c.Next()
			return
		}

		// 验证令牌
		resp, err := m.tokenService.VerifyToken(c.Request.Context(), tokenValue)
		if err != nil {
			// 验证失败，允许通过（不设置用户信息）
			log.Debugw("token verification failed in optional auth",
				"error", err,
				"path", c.FullPath(),
				"method", c.Request.Method,
				"token_source", source,
				"request_id", requestIDFromContext(c),
			)
			c.Next()
			return
		}

		if resp == nil || !resp.Valid {
			// 令牌无效,但允许通过(不设置用户信息)
			log.Debugw("token invalid in optional auth",
				"path", c.FullPath(),
				"method", c.Request.Method,
				"token_source", source,
				"request_id", requestIDFromContext(c),
				"token_hint", sanitize.MaskToken(tokenValue),
			)
			c.Next()
			return
		}

		// 将用户信息存入上下文（从 Claims 中读取）
		if resp.Claims != nil {
			ctx := context.WithValue(c.Request.Context(), ContextKeyUserID, resp.Claims.UserID)
			c.Request = c.Request.WithContext(ctx)
			c.Set(ContextKeyClaims, resp.Claims)
			uid := resp.Claims.UserID.String()
			aid := resp.Claims.AccountID.String()
			tid := resp.Claims.TokenID
			_ = tid
			c.Set(ContextKeyUserID, uid)
			c.Set(ContextKeyAccountID, aid)
			c.Set(ContextKeyTokenID, tid)
		}

		c.Next()
	}
}

// RequireRole 要求用户拥有任一角色名（与 Casbin `role:<name>` 对齐，入参传 name 即可）。
// 必须在 AuthRequired 之后使用。
func (m *JWTAuthMiddleware) RequireRole(roleNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.casbin == nil {
			core.WriteResponse(c, errors.WithCode(code.ErrInternalServerError, "Authorization engine not configured"), nil)
			c.Abort()
			return
		}
		userID, ok := c.Get(ContextKeyUserID)
		if !ok {
			core.WriteResponse(c, errors.WithCode(code.ErrUnauthorized, "Not authenticated"), nil)
			c.Abort()
			return
		}
		uid, ok := userID.(string)
		if !ok || uid == "" {
			core.WriteResponse(c, errors.WithCode(code.ErrUnauthorized, "Not authenticated"), nil)
			c.Abort()
			return
		}
		sub := "user:" + uid
		dom := tenantIDFromGin(c)
		roles, err := m.casbin.GetRolesForUser(c.Request.Context(), sub, dom)
		if err != nil {
			log.Errorw("casbin GetRolesForUser failed", "error", err, "sub", sub, "dom", dom)
			core.WriteResponse(c, errors.WithCode(code.ErrInternalServerError, "Authorization check failed"), nil)
			c.Abort()
			return
		}
		want := make(map[string]struct{}, len(roleNames))
		for _, n := range roleNames {
			if n == "" {
				continue
			}
			want["role:"+n] = struct{}{}
		}
		for _, got := range roles {
			if _, ok := want[got]; ok {
				c.Next()
				return
			}
		}
		core.WriteResponse(c, errors.WithCode(code.ErrPermissionDenied, "Forbidden"), nil)
		c.Abort()
	}
}

// RequirePermission 对资源键与动作执行 Casbin Enforce（与 PDP 一致）。
// 必须在 AuthRequired 之后使用。
func (m *JWTAuthMiddleware) RequirePermission(resourceObj, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.casbin == nil {
			core.WriteResponse(c, errors.WithCode(code.ErrInternalServerError, "Authorization engine not configured"), nil)
			c.Abort()
			return
		}
		userID, ok := c.Get(ContextKeyUserID)
		if !ok {
			core.WriteResponse(c, errors.WithCode(code.ErrUnauthorized, "Not authenticated"), nil)
			c.Abort()
			return
		}
		uid, ok := userID.(string)
		if !ok || uid == "" {
			core.WriteResponse(c, errors.WithCode(code.ErrUnauthorized, "Not authenticated"), nil)
			c.Abort()
			return
		}
		sub := "user:" + uid
		dom := tenantIDFromGin(c)
		allowed, err := m.casbin.Enforce(c.Request.Context(), sub, dom, resourceObj, action)
		if err != nil {
			log.Errorw("casbin Enforce failed", "error", err, "sub", sub, "dom", dom)
			core.WriteResponse(c, errors.WithCode(code.ErrInternalServerError, "Authorization check failed"), nil)
			c.Abort()
			return
		}
		if !allowed {
			core.WriteResponse(c, errors.WithCode(code.ErrPermissionDenied, "Forbidden"), nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

func tenantIDFromGin(c *gin.Context) string {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return "default"
	}
	if id, ok := tenantID.(string); ok && id != "" {
		return id
	}
	return "default"
}

// extractToken 从请求中提取令牌
// 支持多种方式：Authorization Header, Query Parameter, Cookie
func (m *JWTAuthMiddleware) extractToken(c *gin.Context) (string, string) {
	// 1. 从 Authorization Header 提取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// 支持 "Bearer <token>" 格式
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1], "header"
		}
		// 也支持直接传递令牌（无 Bearer 前缀）
		return authHeader, "header"
	}

	// 2. 从查询参数提取
	if token := c.Query("token"); token != "" {
		return token, "query"
	}

	// 3. 从 Cookie 提取
	if token, err := c.Cookie("access_token"); err == nil && token != "" {
		return token, "cookie"
	}

	return "", "none"
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

func requestIDFromContext(c *gin.Context) string {
	if rid, ok := c.Get("request_id"); ok {
		if v, ok := rid.(string); ok && v != "" {
			return v
		}
	}
	if rid := c.GetHeader("X-Request-Id"); rid != "" {
		return rid
	}
	return ""
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
	return NewJWTAuthMiddleware(tokenService, nil).AuthRequired()
}

// OptionalAuth 便捷函数：创建可选认证中间件
func OptionalAuth(tokenService token.TokenApplicationService) gin.HandlerFunc {
	return NewJWTAuthMiddleware(tokenService, nil).AuthOptional()
}
