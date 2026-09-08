package signin

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v4/internal/apiserver/application/authn/authfailure"
	authnexternal "github.com/FangcunMount/iam/v4/internal/apiserver/application/authn/externalidentity"
	"github.com/FangcunMount/iam/v4/internal/apiserver/application/authn/signin/method"
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v4/internal/pkg/code"
	"github.com/FangcunMount/iam/v4/pkg/tenant"
)

// SignIn 编排完整登录流程：身份核验 → 登录准入 → 会话建立 → 令牌颁发。
type SignIn struct {
	deps Dependencies // 依赖
}

// New 创建 SignIn 用例。
// 参数：deps 依赖
// 返回：SignIn 用例
// 职责：创建 SignIn 用例，返回 SignIn 用例
func New(deps Dependencies) *SignIn {
	return &SignIn{deps: deps}
}

// Execute 执行登录。
func (s *SignIn) Execute(ctx context.Context, cmd method.LoginRequest) (*Result, error) {
	// 确保依赖已准备好
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	// 构建身份核验所需的凭据
	credential, err := s.buildCredential(ctx, cmd)
	if err != nil {
		return nil, err
	}

	// 身份核验：验证凭据并确认主体，形成领域身份核验决策。
	decision, err := s.authenticate(ctx, credential)
	if err != nil {
		return nil, err
	}

	if err := decision.Validate(); err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "invalid authentication decision")
	}

	// 身份核验失败也必须先记录，避免绕过失败计数与锁定策略。
	if err := s.recordCredential(ctx, decision); err != nil {
		return nil, err
	}

	// 将领域身份核验决策映射为稳定的应用错误契约。
	if !decision.OK {
		return nil, authfailure.Error(decision.Code)
	}
	if decision.Principal == nil {
		return nil, perrors.WithCode(code.ErrAuthenticationFailed, "authentication principal is missing")
	}

	// 身份核验成功后继续登录准入、会话建立与令牌颁发；全部完成才算登录成功。
	result, err := s.completeLogin(ctx, decision.Principal, sessiondomain.TokenContext{TenantDomain: tenant.DefaultID})
	if err != nil {
		return nil, wrapStageError(err, code.ErrAuthenticationFailed, "failed to issue authentication grant")
	}
	return result, nil
}

// ensureReady 确保依赖已准备好
// 参数：s SignIn 用例
// 返回：错误
// 职责：确保依赖已准备好，返回错误
func (s *SignIn) ensureReady() error {
	if s == nil {
		return perrors.WithCode(code.ErrInvalidArgument, "login service is not initialized")
	}
	d := s.deps
	if d.TokenIssuer == nil || d.AdmissionPolicy == nil || d.SessionCreator == nil || d.SessionRevoker == nil || d.MethodRegistry == nil || d.ProofFactory == nil || d.Authenticator == nil {
		return perrors.WithCode(code.ErrInvalidArgument, "login service is not initialized")
	}
	return nil
}

// buildCredential 构建领域认证凭据
// 参数：ctx 上下文, cmd 登录命令
// 返回：领域认证凭据, 错误
// 职责：构建领域认证凭据，返回领域认证凭据
func (s *SignIn) buildCredential(ctx context.Context, cmd method.LoginRequest) (authentication.IdentityProof, error) {
	// 选择登录方式
	selection, err := s.deps.MethodRegistry.Select(ctx, cmd)
	if err != nil {
		return nil, wrapStageError(err, code.ErrUnsupportedAuthMethod, "failed to select login method")
	}

	// 构建领域认证凭据
	credential, err := s.deps.ProofFactory.Build(ctx, selection)
	if err != nil {
		if cause, ok := authnexternal.AuthenticationCause(err); ok {
			return nil, perrors.WrapC(cause, code.ErrInternalServerError, "failed to authenticate")
		}
		return nil, wrapStageError(err, code.ErrProofBuildFailed, "failed to build credential")
	}
	return credential, nil
}

// authenticate 根据凭据核验登录身份并确认主体。
// 参数：ctx 上下文, credential 领域认证凭据
// 返回：身份核验决策, 错误
// 职责：完成身份核验并返回决策，尚未建立登录态
func (s *SignIn) authenticate(ctx context.Context, credential authentication.IdentityProof) (authentication.AuthDecision, error) {
	// 执行身份核验
	decision, err := s.deps.Authenticator.Authenticate(ctx, credential)
	if err != nil {
		return authentication.AuthDecision{}, perrors.WrapC(err, code.ErrInternalServerError, "failed to authenticate")
	}

	// 返回身份核验决策
	return decision, nil
}

// recordCredential 记录认证结果
// 参数：ctx 上下文, decision 身份核验决策
// 返回：错误
// 职责：记录认证结果，返回错误
func (s *SignIn) recordCredential(ctx context.Context, decision authentication.AuthDecision) error {
	if s.deps.CredentialRecorder == nil {
		return nil
	}
	if err := s.deps.CredentialRecorder.Record(ctx, decision); err != nil {
		return perrors.WrapC(err, code.ErrInternalServerError, "failed to record credential")
	}
	return nil
}

// wrapStageError 包装阶段错误
// 参数：err 错误, fallbackCode 默认错误码, message 错误消息
// 返回：错误
// 职责：包装阶段错误，返回错误
func wrapStageError(err error, fallbackCode int, message string) error {
	codeValue := fallbackCode
	if coder := perrors.ParseCoder(err); coder != nil && coder.Code() != 0 {
		codeValue = coder.Code()
	}
	return perrors.WrapC(err, codeValue, "%s", message)
}
