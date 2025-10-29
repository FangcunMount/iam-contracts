// Package wechat 微信认证适配器
package wechat

import (
	"context"
	"fmt"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatsession"
)

// AuthAdapter 微信认证适配器
//
// 实现 authentication.port.WeChatAuthPort 接口
// 通过调用 IDP 模块的应用服务来换取用户 openID
//
// 架构说明：
// - authn 模块提供统一的认证入口
// - 通过端口接口（WeChatAuthPort）依赖 IDP 模块的应用服务
// - 遵循正确的层次调用：authn.infra -> idp.application
type AuthAdapter struct {
	// IDP 模块的微信认证应用服务
	wechatAuthService wechatsession.WechatAuthApplicationService
}

// NewAuthAdapter 创建微信认证适配器
//
// 参数：
//   - wechatAuthService: IDP 模块的微信认证应用服务
//
// 返回：
//   - *AuthAdapter: 微信认证适配器实例
func NewAuthAdapter(
	wechatAuthService wechatsession.WechatAuthApplicationService,
) *AuthAdapter {
	return &AuthAdapter{
		wechatAuthService: wechatAuthService,
	}
}

// ExchangeOpenID 通过微信授权码换取 openID
//
// 实现 WeChatAuthPort 接口，调用 IDP 模块的应用服务
func (a *AuthAdapter) ExchangeOpenID(ctx context.Context, code, appID string) (string, error) {
	// 调用 IDP 模块的应用服务
	result, err := a.wechatAuthService.LoginWithCode(ctx, wechatsession.LoginWithCodeDTO{
		AppID:  appID,
		JSCode: code,
	})
	if err != nil {
		return "", fmt.Errorf("failed to login with wechat: %w", err)
	}

	if result.OpenID == "" {
		return "", fmt.Errorf("openid is empty in login result")
	}

	return result.OpenID, nil
}
