package service

import (
"context"
"fmt"

domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
)

// OAuthWeChatComAuthStrategy 企业微信认证策略
type OAuthWeChatComAuthStrategy struct {
	scenario    domain.Scenario
	credRepo    port.CredentialRepository
	accountRepo port.AccountRepository
	idp         port.IdentityProvider
}

// 实现认证策略接口
var _ domain.AuthStrategy = (*OAuthWeChatComAuthStrategy)(nil)

// NewOAuthWeChatComAuthStrategy 构造函数（注入依赖）
func NewOAuthWeChatComAuthStrategy(
credRepo port.CredentialRepository,
accountRepo port.AccountRepository,
idp port.IdentityProvider,
) *OAuthWeChatComAuthStrategy {
	return &OAuthWeChatComAuthStrategy{
		scenario:    domain.AuthWecom,
		credRepo:    credRepo,
		accountRepo: accountRepo,
		idp:         idp,
	}
}

// Kind 返回认证策略类型
func (o *OAuthWeChatComAuthStrategy) Kind() domain.Scenario {
	return o.scenario
}

// Authenticate 执行企业微信认证
// 认证流程：
// 1. 调用企业微信API用code换取用户信息
// 2. 根据UserID查找凭据绑定
// 3. 检查账户状态
// 4. 返回认证判决
func (o *OAuthWeChatComAuthStrategy) Authenticate(ctx context.Context, in domain.AuthInput) (domain.AuthDecision, error) {
	// Step 1: 与企业微信IdP交互，用code换取用户信息
	openUserID, userID, err := o.idp.ExchangeWecomCode(ctx, in.WecomCorpID, in.WecomCode)
	if err != nil {
		// 系统异常或业务失败（code无效）
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrIDPExchangeFailed,
		}, fmt.Errorf("failed to exchange wecom code: %w", err)
	}

	// Step 2: 根据UserID查找凭据绑定（优先使用UserID，回退到OpenUserID）
	idpIdentifier := userID
	if idpIdentifier == "" {
		idpIdentifier = openUserID
	}

	accountID, uid, credentialID, err := o.credRepo.FindOAuthCredential(ctx, "wecom", in.WecomCorpID, idpIdentifier)
	if err != nil {
		return domain.AuthDecision{}, fmt.Errorf("failed to find wecom credential: %w", err)
	}
	if credentialID == 0 {
		// 业务失败：企业微信账号未绑定
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
		UserID:    uid,
		TenantID:  in.TenantID,
		AMR:       []string{string(domain.AMRWecom)},
		Claims: map[string]any{
			"wecom_corp_id":      in.WecomCorpID,
			"wecom_user_id":      userID,
			"wecom_open_user_id": openUserID,
			"auth_time":          ctx.Value("request_time"),
		},
	}

	return domain.AuthDecision{
		OK:           true,
		Principal:    principal,
		CredentialID: credentialID,
	}, nil
}
