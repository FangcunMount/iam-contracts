package authentication

import (
	"time"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
)

// TokenType 令牌类型
type TokenType string

const (
	// TokenTypeAccess 访问令牌
	TokenTypeAccess TokenType = "access"
	// TokenTypeRefresh 刷新令牌
	TokenTypeRefresh TokenType = "refresh"
)

// Token 令牌值对象
type Token struct {
	ID        string         // 令牌唯一标识（用于撤销）
	Type      TokenType      // 令牌类型
	Value     string         // 令牌值（JWT 字符串或 UUID）
	UserID    account.UserID // 关联的用户 ID
	AccountID account.AccountID
	IssuedAt  time.Time // 颁发时间
	ExpiresAt time.Time // 过期时间
}

// NewAccessToken 创建访问令牌
func NewAccessToken(id, value string, userID account.UserID, accountID account.AccountID, expiresIn time.Duration) *Token {
	now := time.Now()
	return &Token{
		ID:        id,
		Type:      TokenTypeAccess,
		Value:     value,
		UserID:    userID,
		AccountID: accountID,
		IssuedAt:  now,
		ExpiresAt: now.Add(expiresIn),
	}
}

// NewRefreshToken 创建刷新令牌
func NewRefreshToken(id, value string, userID account.UserID, accountID account.AccountID, expiresIn time.Duration) *Token {
	now := time.Now()
	return &Token{
		ID:        id,
		Type:      TokenTypeRefresh,
		Value:     value,
		UserID:    userID,
		AccountID: accountID,
		IssuedAt:  now,
		ExpiresAt: now.Add(expiresIn),
	}
}

// IsExpired 检查令牌是否过期
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// RemainingDuration 返回令牌剩余有效时长
func (t *Token) RemainingDuration() time.Duration {
	if t.IsExpired() {
		return 0
	}
	return time.Until(t.ExpiresAt)
}

// TokenPair 令牌对（访问令牌 + 刷新令牌）
type TokenPair struct {
	AccessToken  *Token
	RefreshToken *Token
}

// NewTokenPair 创建令牌对
func NewTokenPair(accessToken, refreshToken *Token) *TokenPair {
	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

// TokenClaims 令牌声明（从 JWT 解析出来的信息）
type TokenClaims struct {
	TokenID   string         // 令牌 ID
	UserID    account.UserID // 用户 ID
	AccountID account.AccountID
	IssuedAt  time.Time // 颁发时间
	ExpiresAt time.Time // 过期时间
}

// NewTokenClaims 创建令牌声明
func NewTokenClaims(tokenID string, userID account.UserID, accountID account.AccountID, issuedAt, expiresAt time.Time) *TokenClaims {
	return &TokenClaims{
		TokenID:   tokenID,
		UserID:    userID,
		AccountID: accountID,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}
}

// IsExpired 检查令牌声明是否过期
func (c *TokenClaims) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}
