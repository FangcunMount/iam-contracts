package grant

import (
	"context"
	"errors"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/logger"
	admissiondomain "github.com/FangcunMount/iam/v3/internal/apiserver/domain/authn/admission"
	"github.com/FangcunMount/iam/v3/internal/apiserver/domain/authn/authentication"
	"github.com/FangcunMount/iam/v3/internal/pkg/code"
)

// Dependencies 是认证结果颁发所需的领域协作者。
type Dependencies struct {
	AdmissionPolicy   admissiondomain.Policy
	SessionCreator    SessionCreator
	SessionRevoker    SessionRevoker
	TokenSetMinter    TokenSetMinter
	RefreshTokenSaver RefreshTokenSaver
}

// issuer 是认证结果颁发器的实现。
type issuer struct {
	admissionPolicy   admissiondomain.Policy
	sessionCreator    SessionCreator
	sessionRevoker    SessionRevoker
	tokenSetMinter    TokenSetMinter
	refreshTokenSaver RefreshTokenSaver
}

// 确保 issuer 实现 Issuer 接口。
var _ Issuer = &issuer{}

// NewIssuer 创建认证结果颁发器。
func NewIssuer(deps Dependencies) Issuer {
	return &issuer{
		admissionPolicy:   deps.AdmissionPolicy,
		sessionCreator:    deps.SessionCreator,
		sessionRevoker:    deps.SessionRevoker,
		tokenSetMinter:    deps.TokenSetMinter,
		refreshTokenSaver: deps.RefreshTokenSaver,
	}
}

// Issue 在准入通过后建立 Session、颁发 TokenSet，并保存初始 RefreshToken。
func (s *issuer) Issue(ctx context.Context, principal *authentication.Principal) (*AuthenticationGrant, error) {
	// 参数校验
	if principal == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "principal is required")
	}

	// 准入校验
	if err := admissiondomain.Require(ctx, s.admissionPolicy, admissiondomain.Subject{
		UserID:          principal.UserID,
		LoginIdentityID: principal.LoginIdentityID,
	}); err != nil {
		return nil, err
	}

	// 依赖校验
	if s.sessionCreator == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "session creator is not configured")
	}
	if s.sessionRevoker == nil || s.tokenSetMinter == nil || s.refreshTokenSaver == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "authentication grant dependencies are not configured")
	}

	// 创建会话
	sess, err := s.sessionCreator.Create(ctx, principal)
	if err != nil {
		if perrors.IsCode(err, code.ErrInvalidArgument) {
			return nil, err
		}
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to create session")
	}
	if sess == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "session creator returned no session")
	}

	// 颁发令牌集
	set, err := s.tokenSetMinter.MintTokenSet(ctx, principal, sess)
	if err != nil {
		return nil, s.revokeFailedGrant(ctx, sess.SessionID, principal.UserID.String(), err)
	}
	if set == nil || set.AccessToken == nil || set.RefreshToken == nil {
		err := perrors.WithCode(code.ErrInternalServerError, "token set minter returned incomplete token set")
		return nil, s.revokeFailedGrant(ctx, sess.SessionID, principal.UserID.String(), err)
	}

	// 保存刷新令牌
	if err := s.refreshTokenSaver.SaveRefreshToken(ctx, set.RefreshToken); err != nil {
		cause := perrors.WrapC(err, code.ErrInternalServerError, "failed to save refresh token")
		return nil, s.revokeFailedGrant(ctx, sess.SessionID, principal.UserID.String(), cause)
	}

	// 返回认证结果
	return NewAuthenticationGrant(sess, set), nil
}

// 客户端取消请求不能取消补偿；补偿有独立的短超时，并保留两阶段错误供排障。
func (s *issuer) revokeFailedGrant(ctx context.Context, sessionID, userID string, cause error) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.sessionRevoker.Revoke(cleanupCtx, sessionID, "authentication_grant_failed", userID); err != nil {
		logger.L(ctx).Errorw("authentication grant session compensation failed", "session_id", sessionID, "error_category", "session_store", "result", "failed")
		return perrors.WrapC(errors.Join(cause, err), code.ErrInternalServerError, "authentication grant failed and session compensation failed")
	}
	return cause
}
