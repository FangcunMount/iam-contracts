package token

import (
	"context"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v4/internal/pkg/code"
	"github.com/google/uuid"
)

// tokenSetMinter 是用户令牌颁发器的实现。
type tokenSetMinter struct {
	tokenCodec     BearerTokenCodec
	refreshExpirer SessionRefreshExpirer
	accessTTL      time.Duration
}

// 确保 tokenSetMinter 实现 TokenSetMinter 接口。
var _ TokenSetMinter = &tokenSetMinter{}

// newTokenSetMinter 创建用户令牌颁发器。
func newTokenSetMinter(tokenCodec BearerTokenCodec, refreshExpirer SessionRefreshExpirer, accessTTL time.Duration) TokenSetMinter {
	return &tokenSetMinter{
		tokenCodec: tokenCodec, refreshExpirer: refreshExpirer, accessTTL: accessTTL,
	}
}

// MintTokenSet 颁发用户令牌。
func (s *tokenSetMinter) MintTokenSet(ctx context.Context, sess *sessiondomain.Session) (*UserTokenSet, error) {
	// 会话校验
	if sess == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "session is required")
	}
	// 构建访问令牌主体
	subject := accessTokenSubjectFromSession(sess)
	now := time.Now().UTC()
	// 颁发访问令牌
	accessToken, err := s.tokenCodec.IssueAccessToken(ctx, &AccessTokenIssueContext{
		UserID: subject.UserID, LoginIdentityID: subject.LoginIdentityID, SessionID: subject.SessionID,
		TenantID: subject.TenantID, TenantDomain: subject.TenantDomain, OrgID: subject.OrgID,
		AMR: append([]string(nil), subject.AMR...), AuthenticatedAt: subject.AuthenticatedAt,
		Attributes: cloneStringMap(subject.Attributes),
	}, s.accessTTL)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to generate access token")
	}

	// 颁发刷新令牌
	refreshToken, err := s.issueRefreshToken(subject, sess, now)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to generate refresh token")
	}

	// 返回令牌集
	return NewUserTokenSet(accessToken, refreshToken), nil
}

// issueRefreshToken 颁发刷新令牌。
func (s *tokenSetMinter) issueRefreshToken(subject *AccessTokenIssueContext, sess *sessiondomain.Session, now time.Time) (*RefreshToken, error) {
	// 计算刷新令牌过期时间
	refreshExpiresAt, err := s.refreshExpirer.NextRefreshExpiresAt(now, sess)
	if err != nil {
		return nil, err
	}
	// 颁发刷新令牌
	token := NewRefreshTokenWithExpiry(
		uuid.NewString(), uuid.NewString(), sess.SessionID, subject.UserID, subject.LoginIdentityID,
		subject.TenantID, nil, nil, refreshExpiresAt,
	)
	// 设置颁发时间
	token.IssuedAt = now
	// 返回刷新令牌
	return token, nil
}
