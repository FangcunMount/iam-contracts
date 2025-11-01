package service

import (
	"context"
	"fmt"

	appPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp/port"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession"
	sessionPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession/port"
)

type Authenticator struct {
	// 依赖的仓储或外部服务接口
	authProvider     sessionPort.AuthProvider
	wechatAppQuerier appPort.WechatAppQuerier
	secretVault      appPort.SecretVault
}

// 确保 Authenticator 实现了相应的接口
var _ sessionPort.Authenticator = (*Authenticator)(nil)

// NewAuthenticator 创建认证器实例
func NewAuthenticator(authProvider sessionPort.AuthProvider, wechatAppQuerier appPort.WechatAppQuerier) *Authenticator {
	return &Authenticator{
		authProvider:     authProvider,
		wechatAppQuerier: wechatAppQuerier,
	}
}

// LoginWithCode 使用登录码进行登录，返回统一声明和微信会话
func (a *Authenticator) LoginWithCode(ctx context.Context, appID string, jsCode string) (*domain.ExternalClaim, *domain.WechatSession, error) {
	// 参数校验
	if appID == "" || jsCode == "" {
		return nil, nil, nil
	}

	// 获取 wechatApp
	wechatApp, err := a.wechatAppQuerier.QueryByAppID(ctx, appID)
	if err != nil || wechatApp == nil {
		return nil, nil, err
	}

	// 检查凭据是否存在
	if wechatApp.Cred == nil || wechatApp.Cred.Auth == nil || len(wechatApp.Cred.Auth.AppSecretCipher) == 0 {
		return nil, nil, fmt.Errorf("wechat app %s has no auth credentials configured", appID)
	}

	// 调用外部服务进行登录
	plainSecret, err := a.secretVault.Decrypt(ctx, wechatApp.Cred.Auth.AppSecretCipher)
	if err != nil {
		return nil, nil, err
	}
	result, err := a.authProvider.Code2Session(
		ctx,
		wechatApp.AppID,
		string(plainSecret),
		jsCode,
	)
	if err != nil {
		return nil, nil, err
	}

	// 构造 ExternalClaim 和 WechatSession
	claim := &domain.ExternalClaim{
		Provider: "wechat",
		AppID:    wechatApp.AppID,
		Subject:  result.OpenID,
		UnionID:  &result.UnionID,
		// 微信小程序没有 DisplayName 和 AvatarURL 信息
		DisplayName: nil,
		AvatarURL:   nil,
		Phone:       nil,
		Email:       nil,
	}

	session := domain.NewWechatSession(
		wechatApp.AppID,
		result.OpenID,
		domain.WithWechatSessionUnionID(result.UnionID),
		domain.WithWechatSessionSKCipher([]byte(result.SessionKey)),
	)

	return claim, session, nil
}

// DecryptPhone 解密用户手机号
func (a *Authenticator) DecryptPhone(ctx context.Context, appID, openID, encryptedData, iv string) (string, error) {
	return "", nil
}
