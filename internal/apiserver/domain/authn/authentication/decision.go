package authentication

import (
	"context"
	"fmt"

	credDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/credential"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// CredentialEffect 描述认证后对长期凭据的副作用。
type CredentialEffect int

const (
	CredentialEffectNone CredentialEffect = iota
	CredentialEffectRecordFailure
	CredentialEffectRecordSuccess
)

// AuthDecision 身份核验决策。
type AuthDecision struct {
	// ---- 核验结果 ----
	OK bool // 身份核验是否成功，不代表登录流程已全部完成

	// Code 表示身份核验未通过的业务原因；只有 OK=false 时有效。
	Code int

	// ---- 成功主体 ----
	// Principal 表示身份核验成功后的主体；只有 OK=true 时有效。
	Principal *Principal

	// ---- 失败身份定位 ----
	// RejectedLoginIdentityID 仅定位核验失败时的登录身份；成功时从 Principal 读取。
	RejectedLoginIdentityID meta.ID

	// ---- 凭据更新意图 ----
	CredentialUpdate *CredentialUpdate // 待执行的凭据记录意图；决策本身不写入凭据
}

// loginIdentityStatusFailureDecision 检查登录身份状态是否为失败
func loginIdentityStatusFailureDecision(ctx context.Context, identityRepo LoginIdentityRepository, loginIdentityID meta.ID) (*AuthDecision, error) {
	active, err := identityRepo.IsLoginIdentityActive(ctx, loginIdentityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get login identity status: %w", err)
	}
	if !active {
		return &AuthDecision{
			OK:   false,
			Code: code.ErrLoginIdentityDisabled,

			RejectedLoginIdentityID: loginIdentityID,
		}, nil
	}
	return nil, nil
}

// CredentialUpdate 聚合长期凭据记录意图；Rotation 仅在核验成功时存在。
type CredentialUpdate struct {
	CredentialID meta.ID                      // 需要记录核验结果的凭据 ID
	Effect       CredentialEffect             // 凭据记录动作，与核验成功或失败结果一致
	Rotation     *credDomain.MaterialRotation // 成功核验时可附带的新材料，未设置则不轮换
}

// Validate 检查凭据更新意图是否与身份核验结果一致。
func (u *CredentialUpdate) Validate(succeeded bool) error {
	if u == nil {
		return nil
	}
	if u.CredentialID.IsZero() {
		return fmt.Errorf("credential update requires credential ID")
	}
	switch u.Effect {
	case CredentialEffectNone:
		if u.Rotation != nil {
			return fmt.Errorf("rotation requires success effect")
		}
	case CredentialEffectRecordFailure:
		if succeeded || u.Rotation != nil {
			return fmt.Errorf("failure effect conflicts with authentication result")
		}
	case CredentialEffectRecordSuccess:
		if !succeeded {
			return fmt.Errorf("success effect conflicts with authentication result")
		}
		if u.Rotation != nil && len(u.Rotation.Material) == 0 {
			return fmt.Errorf("rotation material is required")
		}
	default:
		return fmt.Errorf("unknown credential effect")
	}
	return nil
}

// LoginIdentityID 返回本次决策涉及的登录身份，成功时以 Principal 为唯一来源。
func (d AuthDecision) LoginIdentityID() meta.ID {
	if d.OK && d.Principal != nil {
		return d.Principal.LoginIdentityID
	}
	return d.RejectedLoginIdentityID
}

// Validate 检查领域决策的关联字段，避免矛盾结果进入后续记录与登录流程。
func (d AuthDecision) Validate() error {
	if d.OK {
		if d.Principal == nil || d.Code != 0 || !d.RejectedLoginIdentityID.IsZero() {
			return fmt.Errorf("invalid successful authentication decision")
		}
	} else if d.Principal != nil {
		return fmt.Errorf("rejected authentication cannot contain principal")
	}
	return d.CredentialUpdate.Validate(d.OK)
}
