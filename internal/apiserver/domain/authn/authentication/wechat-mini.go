package authentication

import (
	"context"
	"fmt"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/loginidentity"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// ====================== 身份核验证明（请求级输入） ========================

// WechatMiniProofSpec 微信小程序身份核验证明规格
type WechatMiniProofSpec struct {
	AppID   string
	OpenID  string
	UnionID string
}

// WechatMiniProof 微信小程序身份核验证明
// 此输入仅由已完成 IDP 核验的外部身份映射构造；构造函数不执行外部验真。
type WechatMiniProof struct {
	AppID   string
	OpenID  string
	UnionID string
}

// 确保 WechatMiniProof 实现了 IdentityProof 接口
var _ IdentityProof = (*WechatMiniProof)(nil)

// CredentialKind 返回身份核验证明类型
func (c *WechatMiniProof) CredentialKind() CredentialKind {
	return CredentialKindWechatMinip
}

// NewWechatMiniProof 创建 WechatMiniProof 实例
func NewWechatMiniProof(spec WechatMiniProofSpec) (IdentityProof, error) {
	if spec.AppID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "wechat appid is required for wechat authentication")
	}
	if spec.OpenID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "wechat openid is required for wechat authentication")
	}
	return &WechatMiniProof{
		AppID:   spec.AppID,
		OpenID:  spec.OpenID,
		UnionID: spec.UnionID,
	}, nil
}

// ================= 身份核验策略 ========================

// OAuthWechatMinipAuthStrategy 微信小程序身份核验策略
type OAuthWechatMinipAuthStrategy struct {
	credentialKind CredentialKind
	identityRepo   LoginIdentityRepository
}

// 实现身份核验策略接口
var _ AuthStrategy = (*OAuthWechatMinipAuthStrategy)(nil)

func NewOAuthWechatMinipAuthStrategyWithLoginIdentity(
	identityRepo LoginIdentityRepository,
) *OAuthWechatMinipAuthStrategy {
	return &OAuthWechatMinipAuthStrategy{
		credentialKind: CredentialKindWechatMinip,
		identityRepo:   identityRepo,
	}
}

// Kind 返回身份核验策略类型
func (o *OAuthWechatMinipAuthStrategy) Kind() CredentialKind {
	return o.credentialKind
}

// Authenticate 执行微信小程序认证
// 身份核验流程：
// 1. 根据已验证的 openID/unionID 查找凭据绑定
// 2. 检查 LoginIdentity 状态
// 3. 返回身份核验决策
func (o *OAuthWechatMinipAuthStrategy) Authenticate(ctx context.Context, credential IdentityProof) (AuthDecision, error) {
	// 断言身份核验证明类型
	wechatCred, ok := credential.(*WechatMiniProof)
	if !ok {
		return AuthDecision{}, fmt.Errorf("wechat minip strategy expects *WechatMiniProof, got %T", credential)
	}
	identity := wechatMinipIdentity{openID: wechatCred.OpenID, unionID: wechatCred.UnionID}

	// 根据已验证的 openID/unionID 查找登录身份
	lookup, err := o.findWechatMinipIdentity(ctx, wechatCred, identity)
	if err != nil {
		return AuthDecision{}, err
	}
	// 如果登录身份不存在，则返回身份核验失败
	if lookup == nil || lookup.LoginIdentityID.IsZero() {
		return AuthDecision{
			OK:   false,
			Code: code.ErrNoBinding,
		}, nil
	}

	// 检查登录身份状态
	statusFailure, err := loginIdentityStatusFailureDecision(ctx, o.identityRepo, lookup.LoginIdentityID)
	if err != nil {
		return AuthDecision{}, err
	}
	// 如果登录身份状态为失败，则返回身份核验失败
	if statusFailure != nil {
		return *statusFailure, nil
	}

	// 构造身份核验成功决策
	return o.buildWechatMinipSuccessDecision(wechatCred, lookup.LoginIdentityID, lookup.UserID), nil
}

// wechatMinipIdentity 微信小程序身份
type wechatMinipIdentity struct {
	openID  string
	unionID string
}

func (o *OAuthWechatMinipAuthStrategy) findWechatMinipIdentity(
	ctx context.Context,
	credential *WechatMiniProof,
	identity wechatMinipIdentity,
) (*LoginIdentityLookup, error) {
	return findWechatIdentityByOpenIDThenUnionID(
		ctx,
		o.identityRepo,
		loginidentity.ProviderWechatMinip,
		credential.AppID,
		identity.openID,
		identity.unionID,
	)
}

// buildWechatMinipSuccessDecision 身份核验成功，构造Principal
func (o *OAuthWechatMinipAuthStrategy) buildWechatMinipSuccessDecision(
	credential *WechatMiniProof,
	loginIdentityID meta.ID,
	userID meta.ID,
) AuthDecision {
	principal := &Principal{
		LoginIdentityID: loginIdentityID,
		UserID:          userID,
	}
	principal.ApplyAuthContext(NewAuthenticationContext(MethodWechatMinip, credential.AppID, []AMR{AMRWx}, time.Now().UTC()))

	return AuthDecision{
		OK:        true,
		Principal: principal,
	}
}
