package authentication

import (
	"context"
	"fmt"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/loginidentity"
	"github.com/FangcunMount/iam/v4/internal/pkg/code"
	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
)

// WechatOpenProofSpec 微信开放平台身份核验证明规格
type WechatOpenProofSpec struct {
	AppID   string
	OpenID  string
	UnionID string
}

// WechatOpenProof 微信开放平台身份核验证明
// 此输入仅由已完成 IDP 核验的外部身份映射构造；构造函数不执行外部验真。
type WechatOpenProof struct {
	AppID   string
	OpenID  string
	UnionID string
}

// 确保 WechatOpenProof 实现了 IdentityProof 接口
var _ IdentityProof = (*WechatOpenProof)(nil)

// CredentialKind 返回身份核验证明类型
func (c *WechatOpenProof) CredentialKind() CredentialKind {
	return CredentialKindWechatOpen
}

// NewWechatOpenProof 构造微信开放平台身份核验证明
func NewWechatOpenProof(spec WechatOpenProofSpec) (IdentityProof, error) {
	if spec.AppID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "wechat appid is required for wechat authentication")
	}
	if spec.OpenID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "wechat openid is required for wechat authentication")
	}
	return &WechatOpenProof{
		AppID:   spec.AppID,
		OpenID:  spec.OpenID,
		UnionID: spec.UnionID,
	}, nil
}

// ================= 身份核验策略 ========================

// OAuthWechatOpenAuthStrategy 微信开放平台身份核验策略
type OAuthWechatOpenAuthStrategy struct {
	credentialKind CredentialKind
	identityRepo   LoginIdentityRepository
}

// 实现身份核验策略接口
var _ AuthStrategy = (*OAuthWechatOpenAuthStrategy)(nil)

// NewOAuthWechatOpenAuthStrategyWithLoginIdentity 创建微信开放平台身份核验策略
func NewOAuthWechatOpenAuthStrategyWithLoginIdentity(
	identityRepo LoginIdentityRepository,
) *OAuthWechatOpenAuthStrategy {
	return &OAuthWechatOpenAuthStrategy{
		credentialKind: CredentialKindWechatOpen,
		identityRepo:   identityRepo,
	}
}

// Kind 返回身份核验策略类型
func (o *OAuthWechatOpenAuthStrategy) Kind() CredentialKind {
	return o.credentialKind
}

// Authenticate 执行微信开放平台认证
// 身份核验流程：
// 1. 按已验证的 openID 查找 LoginIdentity，必要时用 unionID 回退
// 2. 检查 LoginIdentity 状态
// 3. 返回身份核验决策
func (o *OAuthWechatOpenAuthStrategy) Authenticate(ctx context.Context, credential IdentityProof) (AuthDecision, error) {
	// 断言身份核验证明类型
	wechatCred, ok := credential.(*WechatOpenProof)
	if !ok {
		return AuthDecision{}, fmt.Errorf("wechat open strategy expects *WechatOpenProof, got %T", credential)
	}

	identity := wechatOpenIdentity{openID: wechatCred.OpenID, unionID: wechatCred.UnionID}

	// 根据openID查找登录身份
	lookup, err := o.findWechatOpenIdentity(ctx, wechatCred, identity)
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
	if statusFailure != nil {
		return *statusFailure, nil
	}

	// 构造身份核验成功决策
	return o.buildWechatOpenSuccessDecision(wechatCred, lookup.LoginIdentityID, lookup.UserID), nil
}

// wechatOpenIdentity 微信开放平台身份
type wechatOpenIdentity struct {
	openID  string
	unionID string
}

// findWechatOpenIdentity 根据 openID 查找 LoginIdentity，必要时用 unionID 回退。
func (o *OAuthWechatOpenAuthStrategy) findWechatOpenIdentity(ctx context.Context, credential *WechatOpenProof, identity wechatOpenIdentity) (*LoginIdentityLookup, error) {
	return findWechatIdentityByOpenIDThenUnionID(
		ctx,
		o.identityRepo,
		loginidentity.ProviderWechatOpen,
		credential.AppID,
		identity.openID,
		identity.unionID,
	)
}

// buildWechatOpenSuccessDecision 身份核验成功，构造Principal
func (o *OAuthWechatOpenAuthStrategy) buildWechatOpenSuccessDecision(credential *WechatOpenProof, loginIdentityID meta.ID, userID meta.ID) AuthDecision {
	// 构造Principal
	principal := &Principal{
		LoginIdentityID: loginIdentityID,
		UserID:          userID,
	}
	principal.ApplyAuthContext(NewAuthenticationContext(MethodWechatOpen, credential.AppID, []AMR{AMRWxOpen}, time.Now().UTC()))

	// 返回身份核验决策
	return AuthDecision{
		OK:        true,
		Principal: principal,
	}
}
