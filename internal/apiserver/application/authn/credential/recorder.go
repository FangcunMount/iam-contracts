package credential

import (
	"context"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/logger"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/authentication"
	credDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/credential"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// Recorder 记录长期 Credential 的认证生命周期状态。
type Recorder interface {
	Record(ctx context.Context, decision authentication.AuthDecision) error
}

// Dependencies 记录长期 Credential 的认证生命周期状态的依赖。
type Dependencies struct {
	Credentials   credDomain.Repository
	LockoutPolicy credDomain.LockoutPolicy
	Now           func() time.Time
}

// recorder 记录长期 Credential 的认证生命周期状态。
type recorder struct {
	deps Dependencies
}

// 确保 recorder 实现了 Recorder 接口。
var _ Recorder = (*recorder)(nil)

// NewRecorder 创建记录长期 Credential 的认证生命周期状态的 recorder。
func NewRecorder(deps Dependencies) Recorder {
	return &recorder{deps: deps}
}

// Record 记录长期 Credential 的认证生命周期状态。
func (r *recorder) Record(ctx context.Context, decision authentication.AuthDecision) error {
	if err := decision.CredentialUpdate.Validate(decision.OK); err != nil {
		return perrors.WrapC(err, code.ErrInternalServerError, "invalid credential update")
	}
	if r == nil || r.deps.Credentials == nil || decision.CredentialUpdate == nil {
		return nil
	}
	switch decision.CredentialUpdate.Effect {
	case authentication.CredentialEffectRecordFailure:
		return r.recordFailure(ctx, decision.CredentialUpdate.CredentialID, r.now())
	case authentication.CredentialEffectRecordSuccess:
		return r.recordSuccess(ctx, decision)
	default:
		return nil
	}
}

// recordSuccess 记录身份核验成功状态。
func (r *recorder) recordSuccess(ctx context.Context, decision authentication.AuthDecision) error {
	rotation := decision.CredentialUpdate.Rotation

	// 创建成功迁移
	transition := credDomain.NewSuccessTransition(decision.CredentialUpdate.CredentialID, r.now(), rotation)

	// 应用成功迁移
	_, err := r.deps.Credentials.ApplyAuthenticationTransition(ctx, transition)
	// 如果凭据不存在，则返回
	if perrors.IsCode(err, code.ErrCredentialNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	// 返回成功
	return nil
}

// recordFailure 记录身份核验失败状态。
func (r *recorder) recordFailure(ctx context.Context, credentialID meta.ID, now time.Time) error {
	transition := credDomain.NewFailureTransition(credentialID, now, r.deps.LockoutPolicy)
	state, err := r.deps.Credentials.ApplyAuthenticationTransition(ctx, transition)
	if perrors.IsCode(err, code.ErrCredentialNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if state.NewlyLocked {
		logger.L(ctx).Warnw("credential locked after consecutive authentication failures",
			"credential_id", credentialID.String(),
			"failed_attempts", state.FailedAttempts,
			"locked_until", state.LockedUntil,
		)
	}
	return nil
}

// now 获取当前时间。
func (r *recorder) now() time.Time {
	if r.deps.Now != nil {
		return r.deps.Now()
	}
	return time.Now()
}
