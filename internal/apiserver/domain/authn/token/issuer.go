package token

import (
	"context"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
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
func (s *tokenSetMinter) MintTokenSet(ctx context.Context, principal *authentication.Principal, sess *sessiondomain.Session) (*UserTokenSet, error) {
	// 参数校验
	if principal == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "principal is required")
	}
	// 会话校验
	if sess == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "session is required")
	}
	// 主体与会话校验
	if err := validatePrincipalSessionAlignment(principal, sess); err != nil {
		return nil, err
	}

	// 构建访问令牌主体
	subject := accessTokenSubjectFromAuth(principal, sess)
	now := time.Now().UTC()
	subject.Attributes = cloneStringMap(subject.Attributes)
	if subject.Attributes == nil {
		subject.Attributes = map[string]string{}
	}
	if !subject.AuthenticatedAt.IsZero() {
		authTime := subject.AuthenticatedAt.UTC().Format(time.RFC3339)
		subject.Attributes["auth_time"] = authTime
	}

	// 颁发访问令牌
	accessToken, err := s.tokenCodec.IssueAccessToken(ctx, &AccessTokenSubject{
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
func (s *tokenSetMinter) issueRefreshToken(subject *AccessTokenSubject, sess *sessiondomain.Session, now time.Time) (*RefreshToken, error) {
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
