package signin

import (
	"context"
	"errors"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/logger"
	admissionapp "github.com/FangcunMount/iam/v4/internal/apiserver/application/authn/admission"
	admissiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/admission"
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v4/internal/pkg/code"
)

// completeLogin 编排登录准入、会话建立和令牌颁发，并负责失败补偿。
func (s *SignIn) completeLogin(ctx context.Context, principal *authentication.Principal, creationContext sessiondomain.CreationContext) (*Result, error) {
	// 参数校验
	if principal == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "principal is required")
	}

	// 登录准入：业务拒绝由 Decision 表达；error 表示策略无法完成评估。
	if s.deps.AdmissionPolicy == nil {
		return nil, admissionapp.MapError(&admissiondomain.EvaluationError{Err: errors.New("admission policy is not configured")})
	}
	decision, err := s.deps.AdmissionPolicy.Evaluate(ctx, admissiondomain.Subject{
		UserID:          principal.UserID,
		LoginIdentityID: principal.LoginIdentityID,
	})
	if err != nil {
		return nil, admissionapp.MapError(&admissiondomain.EvaluationError{Err: err})
	}
	if !decision.IsAdmitted() {
		return nil, admissionapp.MapError(&admissiondomain.DeniedError{Decision: decision})
	}

	// 依赖校验
	if s.deps.SessionCreator == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "session creator is not configured")
	}
	if s.deps.SessionRevoker == nil || s.deps.TokenIssuer == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "authentication grant dependencies are not configured")
	}

	// 会话建立
	sess, err := s.deps.SessionCreator.Create(ctx, principal, creationContext)
	if err != nil {
		if perrors.IsCode(err, code.ErrInvalidArgument) {
			return nil, err
		}
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to create session")
	}
	if sess == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "session creator returned no session")
	}

	// 请求租户与核验主体分开校验，保持历史非零租户一致性约束。
	if !creationContext.RequestedTenantID.IsZero() && !sess.TenantID.IsZero() && creationContext.RequestedTenantID != sess.TenantID {
		cause := perrors.WithCode(code.ErrInvalidArgument, "requested tenant does not match session")
		return nil, s.revokeFailedEstablishment(ctx, sess.SessionID, principal.UserID.String(), cause)
	}

	if err := validatePrincipalSessionAlignment(principal, sess); err != nil {
		return nil, s.revokeFailedEstablishment(ctx, sess.SessionID, principal.UserID.String(), err)
	}

	// 令牌颁发：初始刷新令牌保存成功后才交付登录结果。
	tokenPair, err := s.deps.TokenIssuer.IssueInitialTokens(ctx, sess)
	if err != nil {
		return nil, s.revokeFailedEstablishment(ctx, sess.SessionID, principal.UserID.String(), err)
	}
	if tokenPair == nil || tokenPair.AccessToken == nil || tokenPair.RefreshToken == nil {
		cause := perrors.WithCode(code.ErrInternalServerError, "token set minter returned incomplete token set")
		return nil, s.revokeFailedEstablishment(ctx, sess.SessionID, principal.UserID.String(), cause)
	}
	logger.L(ctx).Debugw("登录成功", "action", logger.ActionLogin, "user_id", principal.UserID.String(), "session_id", sess.SessionID, "result", "success")
	return resultFromSession(principal, sess, tokenPair), nil
}

// 客户端取消请求不能取消补偿；补偿有独立的短超时，并保留两阶段错误供排障。
func (s *SignIn) revokeFailedEstablishment(ctx context.Context, sessionID, userID string, cause error) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.deps.SessionRevoker.Revoke(cleanupCtx, sessionID, "authentication_grant_failed", userID); err != nil {
		logger.L(ctx).Errorw("authentication grant session compensation failed", "session_id", sessionID, "error_category", "session_store", "result", "failed")
		return perrors.WrapC(errors.Join(cause, err), code.ErrInternalServerError, "authentication grant failed and session compensation failed")
	}
	return cause
}
