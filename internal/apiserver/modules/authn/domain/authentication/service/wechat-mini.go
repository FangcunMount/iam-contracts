package service

import (
"context"
"fmt"

domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
)

// OAuthWechatMinipAuthStrategy 微信小程序认证策略
type OAuthWechatMinipAuthStrategy struct {
	scenario domain.Scenario
	credRepo port.CredentialRepository
	accountRepo port.AccountRepository
	idp      port.IdentityProvider
}

// 实现认证策略接口
var _ domain.AuthStrategy = (*OAuthWechatMinipAuthStrategy)(nil)

// NewOAuthWechatMinipAuthStrategy 构造函数（注入依赖）
func NewOAuthWechatMinipAuthStrategy(
credRepo port.CredentialRepository,
accountRepo port.AccountRepository,
idp port.IdentityProvider,
) *OAuthWechatMinipAuthStrategy {
	return &OAuthWechatMinipAuthStrategy{
		scenario:    domain.AuthWxMinip,
		credRepo:    credRepo,
		accountRepo: accountRepo,
		idp:         idp,
	}
}

// Kind 返回认证策略类型
func (o *OAuthWechatMinipAuthStrategy) Kind() domain.Scenario {
	return o.scenario
}

// Authenticate 执行微信小程序认证
// 认证流程：
// 1. 调用微信API用jsCode换取openID/unionID
// 2. 根据openID查找凭据绑定
// 3. 检查账户状态
// 4. 返回认证判决
func (o *OAuthWechatMinipAuthStrategy) Authenticate(ctx context.Context, in domain.AuthInput) (domain.AuthDecision, error) {
	// Step 1: 与微信IdP交互，用jsCode换取openID
	openID, unionID, err := o.idp.ExchangeWxMinipCode(ctx, in.WxAppID, in.WxJsCode)
	if err != nil {
		// 系统异常或业务失败（code无效）
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrIDPExchangeFailed,
		}, fmt.Errorf("failed to exchange wx minip code: %w", err)
	}

	// Step 2: 根据openID查找凭据绑定（优先使用unionID，回退到openID）
	idpIdentifier := openID
	if unionID != "" {
		idpIdentifier = unionID
	}

	accountID, userID, credentialID, err := o.credRepo.FindOAuthCredential(ctx, "wx_minip", in.WxAppID, idpIdentifier)
	if err != nil {
		return domain.AuthDecision{}, fmt.Errorf("failed to find wx minip credential: %w", err)
	}
	if credentialID == 0 {
		// 业务失败：微信账号未绑定
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrNoBinding,
		}, nil
	}

	// Step 3: 检查账户状态
	enabled, locked, err := o.accountRepo.GetAccountStatus(ctx, accountID)
	if err != nil {
		return domain.AuthDecision{}, fmt.Errorf("failed to get account status: %w", err)
	}
	if !enabled {
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrDisabled,
		}, nil
	}
	if locked {
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrLocked,
		}, nil
	}

	// Step 4: 认证成功，构造Principal
	principal := &domain.Principal{
		AccountID: accountID,
		UserID:    userID,
		TenantID:  in.TenantID,
		AMR:       []string{string(domain.AMRWx)},
		Claims: map[string]any{
			"wx_openid":  openID,
			"wx_unionid": unionID,
			"auth_time":  ctx.Value("request_time"),
		},
	}

	return domain.AuthDecision{
		OK:           true,
		Principal:    principal,
		CredentialID: credentialID,
	}, nil
}
