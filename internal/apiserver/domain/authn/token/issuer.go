package token

import (
	"context"
	"fmt"
	"strings"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v4/internal/pkg/code"
	"github.com/google/uuid"
)

// IssuanceConfig is the authoritative access-token issuance policy.
type IssuanceConfig struct {
	Issuer    string
	Audience  []string
	AccessTTL time.Duration
}

func (c IssuanceConfig) Validate() error {
	if strings.TrimSpace(c.Issuer) == "" || c.AccessTTL < time.Second {
		return fmt.Errorf("issuer and access TTL of at least one second are required")
	}
	_, err := normalizeAudience(c.Audience)
	return err
}

type tokenSetMinter struct {
	encoder        AccessTokenEncoder
	refreshExpirer SessionRefreshExpirer
	config         IssuanceConfig
	now            func() time.Time
}

func newTokenSetMinter(encoder AccessTokenEncoder, refreshExpirer SessionRefreshExpirer, config IssuanceConfig, now func() time.Time) TokenSetMinter {
	if now == nil {
		now = time.Now
	}
	if normalized, err := normalizeAudience(config.Audience); err == nil {
		config.Audience = normalized
	} else {
		config.Audience = cloneStrings(config.Audience)
	}
	return &tokenSetMinter{encoder: encoder, refreshExpirer: refreshExpirer, config: config, now: now}
}

// MintTokenSet 颁发用户令牌。
func (s *tokenSetMinter) MintTokenSet(ctx context.Context, sess *sessiondomain.Session) (*UserTokenSet, error) {
	// 会话校验
	if sess == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "session is required")
	}
	// 构建访问令牌主体
	if err := s.config.Validate(); err != nil {
		audienceFailures.WithLabelValues("configuration_error").Inc()
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "invalid issuance config")
	}
	subject := accessTokenSubjectFromSession(sess)
	now := s.now().UTC()
	issuedAt := now.Truncate(time.Second)
	claims, err := NewAccessTokenClaims(AccessTokenClaims{
		TokenID: uuid.NewString(), Subject: subject.UserID.String(), SessionID: subject.SessionID,
		UserID: subject.UserID, LoginIdentityID: subject.LoginIdentityID, OrgID: sess.TokenContext.OrgID,
		TenantDomain: subject.TenantDomain, AMR: subject.AMR, Attributes: subject.Attributes,
		AuthenticatedAt: subject.AuthenticatedAt, Issuer: s.config.Issuer, Audience: s.config.Audience,
		IssuedAt: issuedAt, NotBefore: issuedAt, ExpiresAt: now.Add(s.config.AccessTTL).Truncate(time.Second),
	})
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "invalid access claims")
	}
	value, err := s.encoder.EncodeAccessToken(ctx, claims)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to generate access token")
	}
	accessToken := NewAccessToken(claims.TokenID, value, sess.SessionID, sess.UserID, sess.LoginIdentityID, sess.TenantID, claims.IssuedAt, claims.ExpiresAt)

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
	token := NewRefreshToken(
		uuid.NewString(), uuid.NewString(), sess.SessionID, subject.UserID, subject.LoginIdentityID,
		subject.TenantID, now, refreshExpiresAt,
	)
	// 返回刷新令牌
	return token, nil
}
