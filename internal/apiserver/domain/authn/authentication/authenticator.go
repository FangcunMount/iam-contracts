package authentication

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/logger"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// Authenticator 身份核验器
type Authenticator struct {
	strategies map[CredentialKind]AuthStrategy
}

// NewAuthenticator 创建身份核验器
func NewAuthenticator(strategies ...AuthStrategy) *Authenticator {
	authenticator := &Authenticator{
		strategies: make(map[CredentialKind]AuthStrategy, len(strategies)),
	}
	for _, strategy := range strategies {
		authenticator.Register(strategy)
	}
	return authenticator
}

// Register 注册身份核验策略
func (a *Authenticator) Register(strategy AuthStrategy) {
	if a == nil || strategy == nil {
		return
	}
	if a.strategies == nil {
		a.strategies = make(map[CredentialKind]AuthStrategy)
	}
	a.strategies[strategy.Kind()] = strategy
}

// Authenticate 执行身份核验
// 统一流程：
// 1. 获取领域凭据类型对应的身份核验策略
// 2. 执行身份核验
func (a *Authenticator) Authenticate(ctx context.Context, proof IdentityProof) (AuthDecision, error) {
	l := logger.L(ctx)
	if proof == nil {
		return AuthDecision{}, perrors.WithCode(code.ErrInvalidArgument, "authentication credential is required")
	}

	// 获取身份核验证明类型
	credentialKind := proof.CredentialKind()
	if credentialKind == "" {
		return AuthDecision{}, perrors.WithCode(code.ErrInvalidArgument, "unsupported authentication credential kind: %s", credentialKind)
	}

	// 获取身份核验策略
	strategy := a.strategyFor(credentialKind)
	if strategy == nil {
		l.Errorw("不支持的身份核验场景", "action", logger.ActionLogin, "credential_kind", string(credentialKind))
		return AuthDecision{}, perrors.WithCode(code.ErrInvalidArgument, "unsupported authentication credential kind: %s", credentialKind)
	}

	// 执行身份核验
	decision, err := strategy.Authenticate(ctx, proof)
	if err != nil {
		l.Errorw("身份核验策略执行出错",
			"action", logger.ActionLogin,
			"credential_kind", string(credentialKind),
			"result", "failed",
			"error_category", "authentication",
			"retryable", false,
		)
		return AuthDecision{}, err
	}

	if err := decision.Validate(); err != nil {
		return AuthDecision{}, perrors.WrapC(err, code.ErrInternalServerError, "invalid authentication decision")
	}

	// 身份核验未通过
	if !decision.OK {
		l.Warnw("身份核验未通过", "action", logger.ActionLogin, "credential_kind", string(credentialKind), "code", decision.Code)
		return decision, nil
	}

	// 身份核验通过
	l.Debugw("身份核验成功", "action", logger.ActionLogin, "credential_kind", string(credentialKind), "user_id", decision.Principal.UserID.String(), "login_identity_id", decision.Principal.LoginIdentityID.String())

	return decision, nil
}

// strategyFor 获取身份核验策略
func (a *Authenticator) strategyFor(credentialKind CredentialKind) AuthStrategy {
	return a.strategies[credentialKind]
}
