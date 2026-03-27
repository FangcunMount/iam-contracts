package token

import (
	"time"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// TokenType 令牌类型
type TokenType string

const (
	// TokenTypeAccess 访问令牌
	TokenTypeAccess TokenType = "access"
	// TokenTypeRefresh 刷新令牌
	TokenTypeRefresh TokenType = "refresh"
	// TokenTypeService 服务间访问令牌
	TokenTypeService TokenType = "service"
)

// Token 令牌值对象
type Token struct {
	ID         string    // 令牌唯一标识（用于撤销）
	Type       TokenType // 令牌类型
	Value      string    // 令牌值（JWT 字符串或 UUID）
	Subject    string    // JWT sub，服务令牌或访问令牌的主体
	UserID     meta.ID   // 关联的用户 ID
	AccountID  meta.ID
	Audience   []string          // JWT audience
	Attributes map[string]string // 附加属性（主要用于服务令牌）
	IssuedAt   time.Time         // 颁发时间
	ExpiresAt  time.Time         // 过期时间
}

// NewAccessToken 创建访问令牌
func NewAccessToken(id, value string, userID meta.ID, accountID meta.ID, expiresIn time.Duration) *Token {
	now := time.Now()
	return &Token{
		ID:        id,
		Type:      TokenTypeAccess,
		Value:     value,
		Subject:   userID.String(),
		UserID:    userID,
		AccountID: accountID,
		IssuedAt:  now,
		ExpiresAt: now.Add(expiresIn),
	}
}

// NewServiceToken 创建服务间访问令牌。
func NewServiceToken(id, value, subject string, audience []string, attributes map[string]string, expiresIn time.Duration) *Token {
	now := time.Now()
	return &Token{
		ID:         id,
		Type:       TokenTypeService,
		Value:      value,
		Subject:    subject,
		Audience:   cloneStrings(audience),
		Attributes: cloneStringMap(attributes),
		IssuedAt:   now,
		ExpiresAt:  now.Add(expiresIn),
	}
}

// NewRefreshToken 创建刷新令牌
func NewRefreshToken(id, value string, userID meta.ID, accountID meta.ID, expiresIn time.Duration) *Token {
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
	TokenID    string // 令牌 ID
	TokenType  TokenType
	Subject    string
	UserID     meta.ID // 用户 ID
	AccountID  meta.ID
	Issuer     string
	Audience   []string
	Attributes map[string]string
	IssuedAt   time.Time // 颁发时间
	ExpiresAt  time.Time // 过期时间
}

// NewTokenClaims 创建令牌声明
func NewTokenClaims(tokenType TokenType, tokenID, subject string, userID meta.ID, accountID meta.ID, issuer string, audience []string, attributes map[string]string, issuedAt, expiresAt time.Time) *TokenClaims {
	return &TokenClaims{
		TokenID:    tokenID,
		TokenType:  tokenType,
		Subject:    subject,
		UserID:     userID,
		AccountID:  accountID,
		Issuer:     issuer,
		Audience:   cloneStrings(audience),
		Attributes: cloneStringMap(attributes),
		IssuedAt:   issuedAt,
		ExpiresAt:  expiresAt,
	}
}

// IsExpired 检查令牌声明是否过期
func (c *TokenClaims) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
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
	for k, v := range in {
		out[k] = v
	}
	return out
}
