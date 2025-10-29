// Package restful IDP 模块 REST API 路由注册
package restful

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/interface/restful/handler"
)

// Dependencies IDP 模块的依赖
type Dependencies struct {
	WechatAppHandler  *handler.WechatAppHandler
	WechatAuthHandler *handler.WechatAuthHandler
}

var deps Dependencies

// Provide 存储依赖供 Register 使用
func Provide(d Dependencies) {
	deps = d
}

// Register 注册 IDP 模块的所有路由
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
		if deps.WechatAppHandler == nil || deps.WechatAuthHandler == nil {
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
		wechatAuth := idpGroup.Group("/wechat")
		{
			// 微信登录
			wechatAuth.POST("/login", deps.WechatAuthHandler.LoginWithCode)

			// 解密手机号
			wechatAuth.POST("/decrypt-phone", deps.WechatAuthHandler.DecryptUserPhone)
		}
	}
}
