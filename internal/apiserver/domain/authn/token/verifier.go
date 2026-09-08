package token

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	admissiondomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/admission"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

type verifier struct {
	tokenCodec      AccessTokenSignatureVerifier
	tokenStore      Store
	sessionLoader   SessionLoader
	admissionPolicy AdmissionPolicy
}

// 实现 Verifier 接口
var _ Verifier = &verifier{}

func newVerifier(tokenCodec AccessTokenSignatureVerifier, tokenStore Store, sessionLoader SessionLoader, admissionPolicy AdmissionPolicy) Verifier {
	return &verifier{
		tokenCodec: tokenCodec, tokenStore: tokenStore,
		sessionLoader: sessionLoader, admissionPolicy: admissionPolicy,
	}
}

func (s *verifier) VerifyToken(ctx context.Context, tokenValue string, expectedAudience []string) (*AccessTokenClaims, error) {
	expectedAudience, err := NormalizeExpectedAudience(expectedAudience)
	if err != nil {
		return nil, err
	}
	// 解析令牌（codec 负责签名、alg/kid、canonical issuer、exp/nbf/iat）
	claims, err := s.tokenCodec.VerifySignatureAndClaims(ctx, tokenValue)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrTokenInvalid, "failed to parse bearer token")
	}
	if claims.TokenType != TokenTypeAccess {
		return nil, perrors.WithCode(code.ErrTokenInvalid, "unsupported token type for online verification: %s", claims.TokenType)
	}

	if !matchesAudience(claims.Audience, expectedAudience) {
		audienceFailures.WithLabelValues("mismatch").Inc()
		return nil, perrors.WithCode(code.ErrTokenInvalid, "token audience does not match recipient")
	}
	// 用户访问令牌依次检查撤销标记、Session 和准入状态。
	if err := s.checkTokenValid(ctx, claims); err != nil {
		return nil, err
	}
	// 检查会话是否活跃
	if err := s.checkSessionActive(ctx, claims.SessionID); err != nil {
		return nil, err
	}
	// 检查准入策略
	if err := s.requireAdmission(ctx, claims.UserID, claims.LoginIdentityID); err != nil {
		return nil, err
	}
	return claims, nil
}

func (s *verifier) checkTokenValid(ctx context.Context, claims *AccessTokenClaims) error {
	isRevoked, err := s.tokenStore.IsBearerTokenRevoked(ctx, claims.TokenID)
	if err != nil {
		return perrors.WrapC(err, code.ErrInternalServerError, "failed to check revoked bearer token")
	}
	if isRevoked {
		return perrors.WithCode(code.ErrTokenInvalid, "bearer token has been revoked")
	}
	return nil
}

func (s *verifier) checkSessionActive(ctx context.Context, sessionID string) error {
	_, err := s.sessionLoader.GetActive(ctx, sessionID)
	if err != nil {
		if perrors.IsCode(err, code.ErrSessionInactive) {
			return err
		}
		return perrors.WrapC(err, code.ErrInternalServerError, "failed to load session")
	}
	return nil
}

func (s *verifier) requireAdmission(ctx context.Context, userID, loginIdentityID meta.ID) error {
	return admissiondomain.Require(ctx, s.admissionPolicy, admissiondomain.Subject{
		UserID: userID, LoginIdentityID: loginIdentityID,
	})
}
