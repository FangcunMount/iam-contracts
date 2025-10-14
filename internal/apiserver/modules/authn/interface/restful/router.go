package restful

import (
	"github.com/gin-gonic/gin"

	authhandler "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/interface/restful/handler"
)

// Dependencies describes the external collaborators needed to expose authn endpoints.
type Dependencies struct {
	AuthHandler    *authhandler.AuthHandler // 新的认证处理器
	AccountHandler *authhandler.AccountHandler
}

var deps Dependencies

// Provide wires the dependencies for subsequent Register calls.
func Provide(d Dependencies) {
	deps = d
}

// Register exposes the authentication endpoints that issue and refresh tokens.
func Register(engine *gin.Engine) {
	if engine == nil {
		return
	}

	api := engine.Group("/api")

	// 注册符合 API 文档的认证端点
	registerAuthEndpointsV2(api.Group("/v1/auth"), deps.AuthHandler)

	// 注册账户管理端点
	registerAccountEndpoints(api.Group("/v1"), deps.AccountHandler)

	// 注册 JWKS 端点
	registerJWKSEndpoints(engine, deps.AuthHandler)
}

// registerAuthEndpointsV2 注册符合 API 文档的认证端点
func registerAuthEndpointsV2(group *gin.RouterGroup, handler *authhandler.AuthHandler) {
	if group == nil || handler == nil {
		return
	}

	// 认证端点(符合 API 文档)
	group.POST("/login", handler.Login)                // POST /v1/auth/login - 统一登录
	group.POST("/refresh_token", handler.RefreshToken) // POST /v1/auth/refresh_token - 刷新令牌
	group.POST("/logout", handler.Logout)              // POST /v1/auth/logout - 登出
	group.POST("/verify", handler.VerifyToken)         // POST /v1/auth/verify - 验证令牌
}

// registerJWKSEndpoints 注册 JWKS 端点
func registerJWKSEndpoints(engine *gin.Engine, handler *authhandler.AuthHandler) {
	if engine == nil || handler == nil {
		return
	}

	// JWKS 端点（符合 API 文档）
	engine.GET("/.well-known/jwks.json", handler.GetJWKS)
}

func registerAccountEndpoints(v1 *gin.RouterGroup, h *authhandler.AccountHandler) {
	if v1 == nil || h == nil {
		return
	}

	accounts := v1.Group("/accounts")
	accounts.POST("/operation", h.CreateOperationAccount)
	accounts.PATCH("/operation/:username", h.UpdateOperationCredential)
	accounts.POST("/operation/:username:change", h.ChangeOperationUsername)
	accounts.POST("/wechat:bind", h.BindWeChatAccount)
	accounts.PATCH("/:accountId/wechat:profile", h.UpsertWeChatProfile)
	accounts.PATCH("/:accountId/wechat:unionid", h.SetWeChatUnionID)
	accounts.GET("/:accountId", h.GetAccount)
	accounts.POST("/:accountId:enable", h.EnableAccount)
	accounts.POST("/:accountId:disable", h.DisableAccount)
	accounts.GET("/operation/:username", h.GetOperationAccountByUsername)

	v1.GET("/accounts:by-ref", h.FindAccountByRef)

	users := v1.Group("/users")
	users.GET("/:userId/accounts", h.ListAccountsByUser)
}
