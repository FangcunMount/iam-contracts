package authentication

import (
	"context"
	"fmt"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	credDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/credential"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// ====================== 身份核验证明（请求级输入） ========================

// PasswordProofSpec 密码身份核验证明规格，用于构造 PasswordProof 实例
type PasswordProofSpec struct {
	Username string
	Password string
}

// PasswordProof 用户名+密码身份核验证明
type PasswordProof struct {
	Username string
	Password string
}

// 确保 PasswordProof 实现了 IdentityProof 接口
var _ IdentityProof = (*PasswordProof)(nil)

// CredentialKind 返回身份核验证明类型
func (c *PasswordProof) CredentialKind() CredentialKind {
	return CredentialKindPassword
}

// NewPasswordProof 构造密码身份核验证明
func NewPasswordProof(spec PasswordProofSpec) (IdentityProof, error) {
	if spec.Username == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "username is required for password authentication")
	}
	if spec.Password == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "password is required for password authentication")
	}

	return &PasswordProof{

		Username: spec.Username,
		Password: spec.Password,
	}, nil
}

// ================= 身份核验策略 ========================

// PasswordAuthStrategy 用户名+密码身份核验策略
type PasswordAuthStrategy struct {
	credentialKind CredentialKind
	credRepo       LoginIdentityCredentialRepository
	identityRepo   LoginIdentityRepository
	hasher         PasswordHasher
}

// 实现身份核验策略接口
var _ AuthStrategy = (*PasswordAuthStrategy)(nil)

func NewPasswordAuthStrategyWithLoginIdentity(
	credRepo LoginIdentityCredentialRepository,
	identityRepo LoginIdentityRepository,
	hasher PasswordHasher,
) *PasswordAuthStrategy {
	return &PasswordAuthStrategy{
		credentialKind: CredentialKindPassword,
		credRepo:       credRepo,
		identityRepo:   identityRepo,
		hasher:         hasher,
	}
}

// Kind 返回身份核验策略类型
func (p *PasswordAuthStrategy) Kind() CredentialKind {
	return p.credentialKind
}

// Authenticate 执行用户名+密码认证
// 身份核验流程：
// 1. 根据用户名查找 LoginIdentity
// 2. 检查 LoginIdentity 状态
// 3. 查找密码凭据
// 4. 验证密码（带pepper）
// 5. 检查是否需要密码rehash（算法升级）
// 6. 返回身份核验决策
func (p *PasswordAuthStrategy) Authenticate(ctx context.Context, credential IdentityProof) (AuthDecision, error) {
	// 断言身份核验证明类型
	passwordCredential, ok := credential.(*PasswordProof)
	if !ok {
		return AuthDecision{}, fmt.Errorf("password strategy expects *PasswordProof, got %T", credential)
	}

	// 根据用户名查找登录身份
	lookup, err := p.identityRepo.FindUsernameIdentity(ctx, passwordCredential.Username)
	if err != nil {
		return AuthDecision{}, fmt.Errorf("failed to find login identity: %w", err)
	}
	// 如果登录身份不存在，则返回身份核验失败
	if lookup == nil || lookup.LoginIdentityID.IsZero() {
		return AuthDecision{
			OK:   false,
			Code: code.ErrInvalidCredentials,
		}, nil
	}
	// 获取登录身份ID和用户ID
	loginIdentityID, userID := lookup.LoginIdentityID, lookup.UserID

	// 检查登录身份状态
	statusFailure, err := loginIdentityStatusFailureDecision(ctx, p.identityRepo, loginIdentityID)
	if err != nil {
		return AuthDecision{}, err
	}
	if statusFailure != nil {
		return *statusFailure, nil
	}

	// 查找密码凭据
	passwordRecord, found, err := p.findPasswordCredential(ctx, loginIdentityID)
	if err != nil {
		return AuthDecision{}, err
	}
	// 如果密码凭据不存在，则返回身份核验失败
	if !found {
		return AuthDecision{
			OK:   false,
			Code: code.ErrInvalidCredentials,

			RejectedLoginIdentityID: loginIdentityID,
		}, nil
	}
	// 获取密码凭据ID
	credentialID := passwordRecord.CredentialID
	// 检查密码凭据状态
	if passwordRecord.Status == credDomain.CredStatusDisabled {
		return AuthDecision{
			OK:   false,
			Code: code.ErrCredentialDisabled,

			RejectedLoginIdentityID: loginIdentityID,

			CredentialUpdate: &CredentialUpdate{CredentialID: credentialID},
		}, nil
	}
	// 检查密码凭据是否被锁定
	if passwordRecord.LockedUntil != nil && time.Now().Before(*passwordRecord.LockedUntil) {
		return AuthDecision{
			OK:   false,
			Code: code.ErrCredentialLocked,

			RejectedLoginIdentityID: loginIdentityID,

			CredentialUpdate: &CredentialUpdate{CredentialID: credentialID},
		}, nil
	}

	// 验证密码是否匹配
	plaintextWithPepper := passwordCredential.Password + p.hasher.Pepper()
	storedHash := passwordRecord.PasswordHash
	if !p.passwordMatches(storedHash, plaintextWithPepper) {
		return AuthDecision{
			OK:   false,
			Code: code.ErrInvalidCredentials,

			RejectedLoginIdentityID: loginIdentityID,

			CredentialUpdate: &CredentialUpdate{CredentialID: credentialID, Effect: CredentialEffectRecordFailure},
		}, nil
	}

	// 尝试生成升级后的密码 hash
	rotation := p.rotationMaterial(storedHash, plaintextWithPepper)
	// 构造身份核验成功决策
	return p.buildPasswordSuccessDecision(lookup, loginIdentityID, userID, credentialID, rotation), nil
}

// ================= 辅助方法 ========================

func (p *PasswordAuthStrategy) findPasswordCredential(ctx context.Context, loginIdentityID meta.ID) (*PasswordCredentialLookup, bool, error) {
	record, err := p.credRepo.FindPasswordCredentialByLoginIdentity(ctx, loginIdentityID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to find password credential by login identity: %w", err)
	}
	if record == nil || record.CredentialID.IsZero() {
		return nil, false, nil
	}
	return record, true, nil
}

// 验证密码是否匹配
func (p *PasswordAuthStrategy) passwordMatches(storedHash string, plaintextWithPepper string) bool {
	return p.hasher.Verify(storedHash, plaintextWithPepper)
}

// rotationMaterial 尝试生成升级后的密码 hash。
// rehash 失败不应该把一次已经成功的身份核验变成身份核验失败。
func (p *PasswordAuthStrategy) rotationMaterial(storedHash string, plaintextWithPepper string) *credDomain.MaterialRotation {
	if !p.hasher.NeedRehash(storedHash) {
		return nil
	}
	newHash, err := p.hasher.Hash(plaintextWithPepper)
	if err != nil {
		return nil
	}
	return &credDomain.MaterialRotation{Material: []byte(newHash)}
}

func (p *PasswordAuthStrategy) buildPasswordSuccessDecision(
	lookup *LoginIdentityLookup,
	loginIdentityID meta.ID,
	userID meta.ID,
	credentialID meta.ID,
	rotation *credDomain.MaterialRotation,
) AuthDecision {
	realm := lookup.Realm
	principal := &Principal{
		LoginIdentityID: loginIdentityID,
		UserID:          userID,
	}
	principal.ApplyAuthContext(NewAuthenticationContext(MethodPassword, realm, []AMR{AMRPassword}, time.Now().UTC()))

	return AuthDecision{
		OK:        true,
		Principal: principal,

		CredentialUpdate: &CredentialUpdate{CredentialID: credentialID, Effect: CredentialEffectRecordSuccess, Rotation: rotation},
	}
}
