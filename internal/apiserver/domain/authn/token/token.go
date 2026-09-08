package token

import (
	"fmt"
	"strings"
	"time"

	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
)

// TokenType 表示 IAM 令牌的领域用途。
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// TokenMetadata 是两类令牌共享的身份与生命周期信息。
type TokenMetadata struct {
	// —— 身份信息 —— //
	ID string // 令牌ID

	// —— 生命周期信息 —— //
	IssuedAt  time.Time // 令牌颁发时间
	ExpiresAt time.Time // 令牌过期时间
}

// IsExpiredAt 返回令牌在指定时刻是否已过期。
func (m TokenMetadata) IsExpiredAt(now time.Time) bool {
	return now.After(m.ExpiresAt)
}

// IsExpired 返回令牌当前是否已过期。
func (m TokenMetadata) IsExpired() bool {
	return m.IsExpiredAt(time.Now())
}

// RemainingAt 返回令牌在指定时刻的剩余有效期。
func (m TokenMetadata) RemainingAt(now time.Time) time.Duration {
	if m.IsExpiredAt(now) {
		return 0
	}
	return m.ExpiresAt.Sub(now)
}

// RemainingDuration 返回令牌当前剩余有效期。
func (m TokenMetadata) RemainingDuration() time.Duration {
	return m.RemainingAt(time.Now())
}

// AccessToken 表示绑定用户认证上下文与 Session 的短期访问凭证。
type AccessToken struct {
	TokenMetadata
	Value string // 已颁发的凭证值，不属于元数据。

	// —— 主体信息 —— //
	Subject string // 令牌主题

	// —— 会话信息 —— //
	SessionID       string  // 会话ID
	UserID          meta.ID // 用户ID
	LoginIdentityID meta.ID // 登录身份ID
	TenantID        meta.ID // 租户ID

}

func (*AccessToken) Kind() TokenType { return TokenTypeAccess }

// NewAccessToken 创建访问令牌。
func NewAccessToken(id, value, sessionID string, userID, loginIdentityID, tenantID meta.ID, expiresIn time.Duration) *AccessToken {
	now := time.Now()
	return &AccessToken{
		TokenMetadata:   TokenMetadata{ID: id, IssuedAt: now, ExpiresAt: now.Add(expiresIn)},
		Value:           value,
		Subject:         userID.String(),
		SessionID:       sessionID,
		UserID:          userID,
		LoginIdentityID: loginIdentityID,
		TenantID:        tenantID,
	}
}

// RefreshToken 表示与认证 Session 绑定、可单次轮换的续期凭证。
type RefreshToken struct {
	TokenMetadata
	Value string // 已颁发的凭证值，不属于元数据。

	// —— 会话信息 —— //
	SessionID       string  // 会话ID
	UserID          meta.ID // 用户ID
	LoginIdentityID meta.ID // 登录身份ID
	TenantID        meta.ID // 租户ID

	// —— 认证信息 —— //
	// Deprecated: 以下字段只用于读取迁移前 Redis refresh JSON；新签发不再写入。
	AuthMethod    string            // 认证方法
	Realm         string            // 认证域
	AMR           []string          // 认证方法引用
	SessionClaims map[string]string // 认证声明
}

func (*RefreshToken) Kind() TokenType { return TokenTypeRefresh }

// NewRefreshToken 创建相对当前时间过期的刷新令牌。
func NewRefreshToken(id, value, sessionID string, userID, loginIdentityID, tenantID meta.ID, amr []string, sessionClaims map[string]string, expiresIn time.Duration) *RefreshToken {
	now := time.Now()
	return newRefreshToken(id, value, sessionID, userID, loginIdentityID, tenantID, amr, sessionClaims, now, now.Add(expiresIn))
}

// NewRefreshTokenWithExpiry 创建指定过期时间的刷新令牌。
func NewRefreshTokenWithExpiry(id, value, sessionID string, userID, loginIdentityID, tenantID meta.ID, amr []string, sessionClaims map[string]string, expiresAt time.Time) *RefreshToken {
	return newRefreshToken(id, value, sessionID, userID, loginIdentityID, tenantID, amr, sessionClaims, time.Now(), expiresAt)
}

func newRefreshToken(id, value, sessionID string, userID, loginIdentityID, tenantID meta.ID, amr []string, sessionClaims map[string]string, issuedAt, expiresAt time.Time) *RefreshToken {
	return &RefreshToken{
		TokenMetadata:   TokenMetadata{ID: id, IssuedAt: issuedAt, ExpiresAt: expiresAt},
		Value:           value,
		SessionID:       sessionID,
		UserID:          userID,
		LoginIdentityID: loginIdentityID,
		TenantID:        tenantID,
		AMR:             cloneStrings(amr),
		SessionClaims:   cloneStringMap(sessionClaims),
	}
}

// UserTokenSet 表示一次用户认证状态建立或续期产生的访问/刷新令牌集合。
type UserTokenSet struct {
	AccessToken  *AccessToken
	RefreshToken *RefreshToken
}

// NewUserTokenSet 创建用户令牌集合。
func NewUserTokenSet(accessToken *AccessToken, refreshToken *RefreshToken) *UserTokenSet {
	return &UserTokenSet{AccessToken: accessToken, RefreshToken: refreshToken}
}

// ConsumedRefreshToken 是旧刷新令牌成功轮换后留下的最小重放检测事实。
type ConsumedRefreshToken struct {
	SessionID string
	UserID    meta.ID
}

// VerifiedTokenClaims 是验签、标准时间校验和 canonical issuer 校验后得到的领域事实。
// 它不是 JWT wire model，也不包含 JWT Header/Signature。
type VerifiedTokenClaims struct {
	// —— 令牌元数据 —— //
	TokenID   string    // 令牌ID
	TokenType TokenType // 令牌类型
	SessionID string    // 会话ID
	Subject   string    // 令牌主题

	// —— 令牌主体 —— //
	UserID          meta.ID // 用户ID
	LoginIdentityID meta.ID // 登录身份ID
	TenantDomain    string  // 租户域
	OrgID           meta.ID // 组织ID

	// —— 令牌认证 —— //
	Issuer          string    // 令牌颁发者
	AuthenticatedAt time.Time // 令牌认证时间

	// —— 令牌属性 —— //
	Audience   []string          // 受众，令牌预期给谁使用
	Attributes map[string]string // 属性，令牌携带的额外信息
	AMR        []string          // 认证方法引用

	// —— 令牌时间 —— //
	IssuedAt  time.Time // 令牌颁发时间
	NotBefore time.Time // 令牌生效时间
	ExpiresAt time.Time // 令牌过期时间
}

// NewVerifiedUserTokenClaims 构造并校验用户访问令牌事实。
func NewVerifiedUserTokenClaims(claims VerifiedTokenClaims) (*VerifiedTokenClaims, error) {
	claims.TokenType = TokenTypeAccess
	claims.normalize()
	if err := claims.Validate(); err != nil {
		return nil, err
	}
	return &claims, nil
}

// Validate 校验验签后仍需满足的类型相关领域不变量。
func (c *VerifiedTokenClaims) Validate() error {
	if c == nil {
		return fmt.Errorf("verified token claims are required")
	}
	if strings.TrimSpace(c.TokenID) == "" || strings.TrimSpace(c.Subject) == "" || strings.TrimSpace(c.Issuer) == "" {
		return fmt.Errorf("jti, sub and iss are required")
	}
	if len(c.Audience) == 0 {
		return fmt.Errorf("aud is required")
	}
	if c.IssuedAt.IsZero() || c.NotBefore.IsZero() || c.ExpiresAt.IsZero() {
		return fmt.Errorf("iat, nbf and exp are required")
	}
	if c.ExpiresAt.Before(c.NotBefore) || c.ExpiresAt.Equal(c.NotBefore) {
		return fmt.Errorf("exp must be after nbf")
	}
	switch c.TokenType {
	case TokenTypeAccess:
		if strings.TrimSpace(c.SessionID) == "" || c.UserID.IsZero() || c.LoginIdentityID.IsZero() {
			return fmt.Errorf("access token requires sid, user_id and login_identity_id")
		}
		if c.Subject != c.UserID.String() {
			return fmt.Errorf("access token sub must equal user_id")
		}
	default:
		return fmt.Errorf("unsupported token type %q", c.TokenType)
	}
	return nil
}

func (c *VerifiedTokenClaims) normalize() {
	c.TokenID = strings.TrimSpace(c.TokenID)
	c.Subject = strings.TrimSpace(c.Subject)
	c.SessionID = strings.TrimSpace(c.SessionID)
	c.Issuer = strings.TrimSpace(c.Issuer)
	c.TenantDomain = strings.TrimSpace(c.TenantDomain)
	c.Audience = cloneStrings(c.Audience)
	c.Attributes = cloneStringMap(c.Attributes)
	c.AMR = cloneStrings(c.AMR)
	c.IssuedAt = c.IssuedAt.UTC()
	c.NotBefore = c.NotBefore.UTC()
	c.ExpiresAt = c.ExpiresAt.UTC()
	if !c.AuthenticatedAt.IsZero() {
		c.AuthenticatedAt = c.AuthenticatedAt.UTC()
	}
}

// IsExpiredAt 返回声明在指定时刻是否已过期。
func (c *VerifiedTokenClaims) IsExpiredAt(now time.Time) bool {
	return c == nil || now.After(c.ExpiresAt)
}

// IsExpired 返回声明当前是否已过期。
func (c *VerifiedTokenClaims) IsExpired() bool { return c.IsExpiredAt(time.Now()) }

func (c *VerifiedTokenClaims) IsNotYetValidAt(now time.Time) bool {
	return c == nil || (!c.NotBefore.IsZero() && now.Before(c.NotBefore))
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
