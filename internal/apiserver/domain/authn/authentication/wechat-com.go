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

// WecomProofSpec 企业微信身份核验证明规格，用于构造 WecomProof 实例
type WecomProofSpec struct {
	CorpID         string // 企业微信企业 ID，作为身份命名空间
	ProviderUserID string // 企业内的用户标识
	OpenUserID     string // 企业微信返回的开放用户标识
}

// WecomProof 企业微信身份核验证明
// 此输入仅由已完成 IDP 核验的外部身份映射构造；构造函数不执行外部验真。
type WecomProof struct {
	CorpID         string // 企业微信企业 ID，作为身份命名空间
	ProviderUserID string // 企业内的用户标识
	OpenUserID     string // 企业微信返回的开放用户标识
}

// 确保 WecomProof 实现了 IdentityProof 接口
var _ IdentityProof = (*WecomProof)(nil)

// CredentialKind 返回身份核验证明类型
func (c *WecomProof) CredentialKind() CredentialKind {
	return CredentialKindWecom
}

// NewWecomProof 创建 WecomProof 实例
func NewWecomProof(spec WecomProofSpec) (IdentityProof, error) {
	if spec.CorpID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "wecom corpid is required for wecom authentication")
	}
	if spec.ProviderUserID == "" && spec.OpenUserID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "wecom userid or open_userid is required for wecom authentication")
	}
	return &WecomProof{
		CorpID:         spec.CorpID,
		ProviderUserID: spec.ProviderUserID,
		OpenUserID:     spec.OpenUserID,
	}, nil
}

// ================= 身份核验策略 ========================

// OAuthWeChatComAuthStrategy 企业微信身份核验策略
type OAuthWeChatComAuthStrategy struct {
	credentialKind CredentialKind
	identityRepo   LoginIdentityRepository
}

// 实现身份核验策略接口
var _ AuthStrategy = (*OAuthWeChatComAuthStrategy)(nil)

func NewOAuthWeChatComAuthStrategyWithLoginIdentity(
	identityRepo LoginIdentityRepository,
) *OAuthWeChatComAuthStrategy {
	return &OAuthWeChatComAuthStrategy{
		credentialKind: CredentialKindWecom,
		identityRepo:   identityRepo,
	}
}

// Kind 返回身份核验策略类型
func (o *OAuthWeChatComAuthStrategy) Kind() CredentialKind {
	return o.credentialKind
}

// Authenticate 执行企业微信认证
// 身份核验流程：
// 1. 根据已验证的 UserID/OpenUserID 查找凭据绑定
// 2. 检查 LoginIdentity 状态
// 3. 返回身份核验决策
func (o *OAuthWeChatComAuthStrategy) Authenticate(ctx context.Context, credential IdentityProof) (AuthDecision, error) {
	// 断言身份核验证明类型
	wecomCred, ok := credential.(*WecomProof)
	if !ok {
		return AuthDecision{}, fmt.Errorf("wecom strategy expects *WecomProof, got %T", credential)
	}

	// 根据已验证的 UserID/OpenUserID 查找登录身份
	identity := wecomIdentity{openUserID: wecomCred.OpenUserID, userID: wecomCred.ProviderUserID}
	lookup, err := o.findWecomIdentity(ctx, wecomCred, identity)
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
	return o.buildWecomSuccessDecision(wecomCred, lookup.LoginIdentityID, lookup.UserID), nil
}

// wecomIdentity 企业微信身份
type wecomIdentity struct {
	openUserID string
	userID     string
}

// findWecomIdentity 根据 UserID/OpenUserID 查找登录身份
func (o *OAuthWeChatComAuthStrategy) findWecomIdentity(
	ctx context.Context,
	credential *WecomProof,
	identity wecomIdentity,
) (*LoginIdentityLookup, error) {
	// 根据 UserID/OpenUserID 查找登录身份
	identifier := identity.userID
	if identifier == "" {
		identifier = identity.openUserID
	}
	lookup, err := o.identityRepo.FindLoginIdentityByProviderKey(ctx, loginidentity.ProviderWecom, credential.CorpID, identifier)
	if err != nil || lookup != nil || identity.openUserID == "" || identifier == identity.openUserID {
		return lookup, err
	}
	return o.identityRepo.FindLoginIdentityByProviderKey(ctx, loginidentity.ProviderWecom, credential.CorpID, identity.openUserID)
}

// buildWecomSuccessDecision 身份核验成功，构造Principal
func (o *OAuthWeChatComAuthStrategy) buildWecomSuccessDecision(
	credential *WecomProof,
	loginIdentityID meta.ID,
	userID meta.ID,
) AuthDecision {
	principal := &Principal{
		LoginIdentityID: loginIdentityID,
		UserID:          userID,
	}
	principal.ApplyAuthContext(NewAuthenticationContext(MethodWecom, credential.CorpID, []AMR{AMRWecom}, time.Now().UTC()))

	return AuthDecision{
		OK:        true,
		Principal: principal,
	}
}
