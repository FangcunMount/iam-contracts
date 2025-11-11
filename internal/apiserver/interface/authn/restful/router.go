package restful

import (
	"github.com/gin-gonic/gin"

	authhandler "github.com/FangcunMount/iam-contracts/internal/apiserver/interface/authn/restful/handler"
)

// Dependencies describes the external collaborators needed to expose authn endpoints.
type Dependencies struct {
	AuthHandler    *authhandler.AuthHandler    // 新的认证处理器
	AccountHandler *authhandler.AccountHandler // 账户管理处理器
	JWKSHandler    *authhandler.JWKSHandler    // JWKS 处理器
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

	// 注册 JWKS 端点（公开端点）
	registerJWKSPublicEndpoints(engine, deps.JWKSHandler)

	// 注册 JWKS 管理端点（管理员接口）
	registerJWKSAdminEndpoints(api.Group("/v1/admin"), deps.JWKSHandler)
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

// registerJWKSPublicEndpoints 注册 JWKS 公开端点
func registerJWKSPublicEndpoints(engine *gin.Engine, handler *authhandler.JWKSHandler) {
	if engine == nil || handler == nil {
		return
	}

	// JWKS 公开端点（无需认证）
	engine.GET("/.well-known/jwks.json", handler.GetJWKS)
}

// registerJWKSAdminEndpoints 注册 JWKS 管理端点
func registerJWKSAdminEndpoints(admin *gin.RouterGroup, handler *authhandler.JWKSHandler) {
	if admin == nil || handler == nil {
		return
	}

	// JWKS 管理端点（需要管理员权限）
	jwks := admin.Group("/jwks")
	{
		// 密钥管理
		jwks.POST("/keys", handler.CreateKey)                        // 创建密钥
		jwks.GET("/keys", handler.ListKeys)                          // 列出密钥
		jwks.GET("/keys/:kid", handler.GetKey)                       // 获取密钥详情
		jwks.POST("/keys/:kid/retire", handler.RetireKey)            // 退役密钥
		jwks.POST("/keys/:kid/force-retire", handler.ForceRetireKey) // 强制退役密钥
		jwks.POST("/keys/:kid/grace", handler.EnterGracePeriod)      // 进入宽限期
		jwks.POST("/keys/cleanup", handler.CleanupExpiredKeys)       // 清理过期密钥
		jwks.GET("/keys/publishable", handler.GetPublishableKeys)    // 获取可发布的密钥
	}
}

func registerAccountEndpoints(v1 *gin.RouterGroup, h *authhandler.AccountHandler) {
	if v1 == nil || h == nil {
		return
	}

	accounts := v1.Group("/accounts")

	// 微信注册（公开端点，无需认证）
	accounts.POST("/wechat/register", h.RegisterWithWeChat)

	// 账户查询和管理（需要认证）
	accounts.GET("/:accountId", h.GetAccountByID)
	accounts.PUT("/:accountId/profile", h.UpdateProfile)
	accounts.PUT("/:accountId/unionid", h.SetUnionID)
	accounts.POST("/:accountId/enable", h.EnableAccount)
	accounts.POST("/:accountId/disable", h.DisableAccount)

	// TODO: 以下端点待实现
	// accounts.GET("/:accountId/credentials", h.GetCredentials) // 待实现凭据查询服务
	// accounts.POST("/operation", h.CreateOperationAccount)
	// accounts.PATCH("/operation/:username", h.UpdateOperationCredential)
	// accounts.POST("/operation/:username/change", h.ChangeOperationUsername)
	// accounts.POST("/wechat/bind", h.BindWeChatAccount)
	// accounts.GET("/operation/:username", h.GetOperationAccountByUsername)
	// v1.GET("/accounts/by-ref", h.FindAccountByRef)
	// users := v1.Group("/users")
	// users.GET("/:userId/accounts", h.ListAccountsByUser)
}
