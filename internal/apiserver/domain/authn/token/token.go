package token

import (
	"fmt"
	"strings"
	"time"

	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
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
	// ---- 令牌标识与有效期 ----
	TokenMetadata

	// ---- 凭证值 ----
	Value string // 已颁发的凭证值，不属于元数据。

	// —— 会话信息 —— //
	SessionID       string  // 会话ID
	UserID          meta.ID // 用户ID
	LoginIdentityID meta.ID // 登录身份ID
}

// Subject 从用户身份派生，避免保存第二套可变主体事实。
func (t *AccessToken) Subject() string { return t.UserID.String() }

func (*AccessToken) Kind() TokenType { return TokenTypeAccess }

// NewAccessToken 创建访问令牌。
func NewAccessToken(id, value, sessionID string, userID, loginIdentityID meta.ID, issuedAt, expiresAt time.Time) *AccessToken {
	return &AccessToken{
		TokenMetadata: TokenMetadata{ID: id, IssuedAt: issuedAt, ExpiresAt: expiresAt},
		Value:         value, SessionID: sessionID,
		UserID: userID, LoginIdentityID: loginIdentityID,
	}
}

// RefreshToken 表示与认证 Session 绑定、可单次轮换的续期凭证。
type RefreshToken struct {
	// ---- 令牌标识与有效期 ----
	TokenMetadata

	// ---- 凭证值 ----
	Value string // 已颁发的凭证值，不属于元数据。

	// —— 会话信息 —— //
	SessionID       string  // 会话ID
	UserID          meta.ID // 用户ID
	LoginIdentityID meta.ID // 登录身份ID

	// ---- 历史兼容输入 ----
	// Deprecated: 以下字段只用于读取迁移前 Redis refresh JSON；新签发不再写入。
	AuthMethod    string            // 认证方法
	Realm         string            // 认证域
	AMR           []string          // 认证方法引用
	SessionClaims map[string]string // 认证声明
}

func (*RefreshToken) Kind() TokenType { return TokenTypeRefresh }

// NewRefreshToken creates a new refresh credential with explicit lifetime facts.
func NewRefreshToken(id, value, sessionID string, userID, loginIdentityID meta.ID, issuedAt, expiresAt time.Time) *RefreshToken {
	return &RefreshToken{
		TokenMetadata: TokenMetadata{ID: id, IssuedAt: issuedAt, ExpiresAt: expiresAt}, Value: value,
		SessionID: sessionID, UserID: userID, LoginIdentityID: loginIdentityID,
	}
}

// LegacyRefreshContext is read-only migration input from historical Redis records.
type LegacyRefreshContext struct {
	AuthMethod    string            // 历史记录中的认证方法
	Realm         string            // 历史登录身份命名空间
	AMR           []string          // 历史认证手段
	SessionClaims map[string]string // 历史会话声明，仅供兼容恢复
}

// RestoreRefreshToken reads old storage without inventing a persisted issued_at.
// Redis did not store IssuedAt; the previous read-time timestamp is retained for compatibility.
func RestoreRefreshToken(id, value, sessionID string, userID, loginIdentityID meta.ID, expiresAt time.Time, legacy LegacyRefreshContext) *RefreshToken {
	token := NewRefreshToken(id, value, sessionID, userID, loginIdentityID, time.Now(), expiresAt)
	token.AuthMethod = legacy.AuthMethod
	token.Realm = legacy.Realm
	token.AMR = cloneStrings(legacy.AMR)
	token.SessionClaims = cloneStringMap(legacy.SessionClaims)
	return token
}

// UserTokenSet 表示一次用户认证状态建立或续期产生的访问/刷新令牌集合。
type UserTokenSet struct {
	AccessToken  *AccessToken  // 本次交付的访问令牌
	RefreshToken *RefreshToken // 本次交付的刷新令牌
}

// NewUserTokenSet 创建用户令牌集合。
func NewUserTokenSet(accessToken *AccessToken, refreshToken *RefreshToken) *UserTokenSet {
	return &UserTokenSet{AccessToken: accessToken, RefreshToken: refreshToken}
}

// ConsumedRefreshToken 是旧刷新令牌成功轮换后留下的最小重放检测事实。
type ConsumedRefreshToken struct {
	SessionID string  // 已消费刷新令牌所属会话，用于重放时定位撤销目标
	UserID    meta.ID // 该会话所属用户 ID
}

// AccessTokenClaims 表达访问令牌声明；类型本身不保证已验签或已通过在线验证。
// 它不是 JWT wire model，也不包含 JWT Header/Signature。
type AccessTokenClaims struct {
	// ---- 令牌标识与用途 ----
	TokenID   string    // 令牌ID
	TokenType TokenType // 令牌类型

	// ---- 会话与主体声明 ----
	SessionID string // 会话ID
	Subject   string // 用户主体声明，必须等于 UserID 的字符串表示

	// ---- 身份与业务归属 ----
	UserID          meta.ID // 用户ID
	LoginIdentityID meta.ID // 登录身份ID
	OrgID           meta.ID // 会话业务快照中的组织 ID

	// ---- 签发来源 ----
	Issuer string // 令牌颁发者

	// ---- 原始认证时间 ----
	AuthenticatedAt time.Time // 原始身份核验时间，刷新不应提升为新的认证时间

	// ---- 预期受众 ----
	Audience []string // 受众，令牌预期给谁使用

	// ---- 附加声明 ----
	Attributes map[string]string // 属性，令牌携带的额外信息

	// ---- 认证手段 ----
	AMR []string // 本次认证使用的手段声明

	// ---- 令牌有效期 ----
	IssuedAt  time.Time // 令牌颁发时间
	NotBefore time.Time // 令牌生效时间
	ExpiresAt time.Time // 令牌过期时间
}

// NewAccessTokenClaims 仅规范化并校验声明不变量，不执行验签。
func NewAccessTokenClaims(claims AccessTokenClaims) (*AccessTokenClaims, error) {
	if claims.TokenType == "" {
		claims.TokenType = TokenTypeAccess
	}
	claims.normalize()
	if err := claims.Validate(); err != nil {
		return nil, err
	}
	return &claims, nil
}

// Validate 仅校验声明的领域不变量，不执行密码学或在线验证。
func (c *AccessTokenClaims) Validate() error {
	if c == nil {
		return fmt.Errorf("access token claims are required")
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

func (c *AccessTokenClaims) normalize() {
	c.TokenID = strings.TrimSpace(c.TokenID)
	c.Subject = strings.TrimSpace(c.Subject)
	c.SessionID = strings.TrimSpace(c.SessionID)
	c.Issuer = strings.TrimSpace(c.Issuer)
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
func (c *AccessTokenClaims) IsExpiredAt(now time.Time) bool {
	return c == nil || now.After(c.ExpiresAt)
}

// IsExpired 返回声明当前是否已过期。
func (c *AccessTokenClaims) IsExpired() bool { return c.IsExpiredAt(time.Now()) }

func (c *AccessTokenClaims) IsNotYetValidAt(now time.Time) bool {
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
