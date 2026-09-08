package token

import (
	"time"

	tokendomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/token"
	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
)

// IssuedTokenDTO 是向 application 调用方返回的令牌 DTO；领域模型位于 domain/authn/token。
type IssuedTokenDTO struct {
	ID string // 令牌ID

	// --- 令牌主体信息 ---
	Type    TokenType // 令牌类型
	Value   string    // 令牌值
	Subject string    // 令牌主体

	// --- 令牌主体会话信息 ---
	SessionID       string  // 会话ID
	UserID          meta.ID // 用户ID
	LoginIdentityID meta.ID // 登录身份ID
	TenantID        meta.ID // 租户ID

	// --- 令牌期限 ---
	IssuedAt  time.Time // 颁发时间
	ExpiresAt time.Time // 过期时间
}

// NewAccessToken constructs a DTO from explicit access-token facts.
func NewAccessToken(id, value, sessionID string, userID meta.ID, loginIdentityID meta.ID, tenantID meta.ID, issuedAt, expiresAt time.Time) *IssuedTokenDTO {
	return tokenFromAccess(tokendomain.NewAccessToken(id, value, sessionID, userID, loginIdentityID, tenantID, issuedAt, expiresAt))
}

// NewRefreshToken constructs a DTO from explicit refresh-token facts.
func NewRefreshToken(id, value, sessionID string, userID, loginIdentityID, tenantID meta.ID, issuedAt, expiresAt time.Time) *IssuedTokenDTO {
	return tokenFromRefresh(tokendomain.NewRefreshToken(id, value, sessionID, userID, loginIdentityID, tenantID, issuedAt, expiresAt))
}

// IsExpired 检查令牌是否已过期
func (t *IssuedTokenDTO) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// RemainingDuration 返回令牌剩余时间
func (t *IssuedTokenDTO) RemainingDuration() time.Duration {
	if t.IsExpired() {
		return 0
	}
	return time.Until(t.ExpiresAt)
}

// TokenPair 令牌对
type TokenPair struct {
	AccessToken  *IssuedTokenDTO // 访问令牌
	RefreshToken *IssuedTokenDTO // 刷新令牌
}

// NewTokenPair 创建令牌对
func NewTokenPair(accessToken, refreshToken *IssuedTokenDTO) *TokenPair {
	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

func tokenPairFromDomain(set *tokendomain.UserTokenSet) *TokenPair {
	if set == nil {
		return nil
	}
	return NewTokenPair(tokenFromAccess(set.AccessToken), tokenFromRefresh(set.RefreshToken))
}

func tokenFromAccess(token *tokendomain.AccessToken) *IssuedTokenDTO {
	if token == nil {
		return nil
	}
	return &IssuedTokenDTO{
		ID: token.ID, Type: TokenTypeAccess, Value: token.Value, Subject: token.Subject,
		SessionID: token.SessionID, UserID: token.UserID, LoginIdentityID: token.LoginIdentityID,
		TenantID: token.TenantID,
		IssuedAt: token.IssuedAt, ExpiresAt: token.ExpiresAt,
	}
}

func tokenFromRefresh(token *tokendomain.RefreshToken) *IssuedTokenDTO {
	if token == nil {
		return nil
	}
	return &IssuedTokenDTO{
		ID: token.ID, Type: TokenTypeRefresh, Value: token.Value, SessionID: token.SessionID,
		UserID: token.UserID, LoginIdentityID: token.LoginIdentityID, TenantID: token.TenantID,
		IssuedAt: token.IssuedAt, ExpiresAt: token.ExpiresAt,
	}
}
