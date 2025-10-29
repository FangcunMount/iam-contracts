// Package restful IDP 模块 REST API 路由注册
package restful

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/interface/restful/handler"
)

// Dependencies IDP 模块的依赖
type Dependencies struct {
	WechatAppHandler *handler.WechatAppHandler
	// WechatAuthHandler 已移除 - 认证功能由 authn 模块统一提供
}

var deps Dependencies

// Provide 存储依赖供 Register 使用
func Provide(d Dependencies) {
	deps = d
}

// Register 注册 IDP 模块的所有路由
//
// IDP 模块职责：
// - 微信应用管理（创建、查询、凭据轮换、令牌管理）
// - 提供基础设施服务供其他模块使用（通过容器依赖注入）
//
// 认证功能由 authn 模块统一提供：
// - POST /api/v1/auth/login (method: "wx:minip") - 微信小程序登录
func Register(engine *gin.Engine) {
	if engine == nil {
		return
	}

	idpGroup := engine.Group("/api/v1/idp")
	{
		// 健康检查
		idpGroup.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
				"module": "idp",
			})
		})

		// 如果依赖未初始化，只注册健康检查
		if deps.WechatAppHandler == nil {
			return
		}

		// ============ 微信应用管理 ============
		wechatApps := idpGroup.Group("/wechat-apps")
		{
			// 创建微信应用
			wechatApps.POST("", deps.WechatAppHandler.CreateWechatApp)

			// 查询微信应用
			wechatApps.GET("/:app_id", deps.WechatAppHandler.GetWechatApp)

			// 获取访问令牌
			wechatApps.GET("/:app_id/access-token", deps.WechatAppHandler.GetAccessToken)

			// 轮换认证密钥
			wechatApps.POST("/rotate-auth-secret", deps.WechatAppHandler.RotateAuthSecret)

			// 轮换消息密钥
			wechatApps.POST("/rotate-msg-secret", deps.WechatAppHandler.RotateMsgSecret)

			// 刷新访问令牌
			wechatApps.POST("/refresh-access-token", deps.WechatAppHandler.RefreshAccessToken)
		}

		// ============ 微信认证 ============
		// 已移除 - 认证功能由 authn 模块统一提供
		// 使用方式：
		//   POST /api/v1/auth/login
		//   {
		//     "method": "wx:minip",
		//     "credentials": {
		//       "app_id": "wx1234567890",
		//       "js_code": "code_from_wx_login"
		//     }
		//   }
	}
}
