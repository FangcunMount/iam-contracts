package application

import (
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatsession"
	wechatappport "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp/port"
	wechatsessionport "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession/port"
)

// ApplicationServices IDP 模块所有应用服务的集合
type ApplicationServices struct {
	// 微信应用管理
	WechatApp           wechatapp.WechatAppApplicationService
	WechatAppCredential wechatapp.WechatAppCredentialApplicationService
	WechatAppToken      wechatapp.WechatAppTokenApplicationService

	// 微信认证
	WechatAuth wechatsession.WechatAuthApplicationService
}

// ApplicationServicesDependencies 应用服务依赖的端口集合
type ApplicationServicesDependencies struct {
	// WechatApp 依赖
	WechatAppRepo     wechatappport.WechatAppRepository
	WechatAppCreator  wechatappport.WechatAppCreator
	WechatAppQuerier  wechatappport.WechatAppQuerier
	CredentialRotater wechatappport.CredentialRotater
	AccessTokenCacher wechatappport.AccessTokenCacher
	AppTokenProvider  wechatappport.AppTokenProvider
	AccessTokenCache  wechatappport.AccessTokenCache

	// WechatSession 依赖
	WechatAuthenticator wechatsessionport.Authenticator
}

// NewApplicationServices 创建所有应用服务实例
func NewApplicationServices(deps ApplicationServicesDependencies) *ApplicationServices {
	return &ApplicationServices{
		// 微信应用管理服务
		WechatApp: wechatapp.NewWechatAppApplicationService(
			deps.WechatAppRepo,
			deps.WechatAppCreator,
			deps.WechatAppQuerier,
			deps.CredentialRotater,
		),
		WechatAppCredential: wechatapp.NewWechatAppCredentialApplicationService(
			deps.WechatAppRepo,
			deps.WechatAppQuerier,
			deps.CredentialRotater,
		),
		WechatAppToken: wechatapp.NewWechatAppTokenApplicationService(
			deps.WechatAppQuerier,
			deps.AccessTokenCacher,
			deps.AppTokenProvider,
			deps.AccessTokenCache,
		),

		// 微信认证服务
		WechatAuth: wechatsession.NewWechatAuthApplicationService(
			deps.WechatAuthenticator,
		),
	}
}
