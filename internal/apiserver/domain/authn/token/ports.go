package token

import (
	"context"
	"time"

	admissiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/admission"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
)

// Store 持久化 RefreshToken、消费事实与 Bearer Token 撤销事实。
type Store interface {
	// SaveRefreshToken 保存刷新令牌
	SaveRefreshToken(ctx context.Context, token *RefreshToken) error
	// GetRefreshToken 获取刷新令牌
	GetRefreshToken(ctx context.Context, tokenValue string) (*RefreshToken, error)
	// GetConsumedRefreshToken 获取已消费的刷新令牌
	GetConsumedRefreshToken(ctx context.Context, tokenValue string) (*ConsumedRefreshToken, error)
	// RotateRefreshToken 轮换刷新令牌
	RotateRefreshToken(ctx context.Context, oldValue, expectedOldID string, newToken *RefreshToken) (bool, error)
	// DeleteRefreshToken 删除刷新令牌
	DeleteRefreshToken(ctx context.Context, tokenValue string) error
	// MarkBearerTokenRevoked 标记 access bearer token 已撤销
	MarkBearerTokenRevoked(ctx context.Context, tokenID string, expiry time.Duration) error
	// IsBearerTokenRevoked 检查 access bearer token 是否已撤销
	IsBearerTokenRevoked(ctx context.Context, tokenID string) (bool, error)
}

// AccessTokenEncoder encodes an already assembled claims set; it does not create issuance facts.
type AccessTokenEncoder interface {
	EncodeAccessToken(context.Context, *AccessTokenClaims) (string, error)
}

// AccessTokenSignatureVerifier verifies signature, canonical issuer, time and claim invariants.
// Recipient audience and online authentication state are checked by Verifier.
type AccessTokenSignatureVerifier interface {
	VerifySignatureAndClaims(context.Context, string) (*AccessTokenClaims, error)
}

// AccessTokenIssueContext 是访问令牌编码的签发输入，不是 JWT sub 或完整 Claims Set。
// 领域层完成投影后，JWT adapter 只负责序列化，不再从任意 Claims 推断授权域。
type AccessTokenIssueContext struct {
	UserID          meta.ID
	LoginIdentityID meta.ID
	TenantID        meta.ID
	SessionID       string
	TenantDomain    string
	OrgID           string
	AMR             []string
	AuthenticatedAt time.Time
	// Attributes 是已经过准入的对外附加字段，不再代表任意 Principal.Claims。
	Attributes map[string]string
}

// LegacyAuthenticationContextSnapshotDecoder 只负责读取迁移前 RefreshToken 中的认证上下文快照。
// 新写入以 Session.AuthContext/TokenContext 为权威来源，不再生成该快照。
type LegacyAuthenticationContextSnapshotDecoder interface {
	Decode(map[string]string) map[string]any
}

// SessionLoader 是会话加载器
type SessionLoader = sessiondomain.Loader

// SessionRevoker 是会话撤销器
type SessionRevoker = sessiondomain.Revoker

// SessionExtender 是会话延期器
type SessionExtender = sessiondomain.Extender

// SessionRefreshExpirer 是会话刷新过期时间计算器
type SessionRefreshExpirer = sessiondomain.RefreshExpirer

// AdmissionPolicy 是登录准入策略
type AdmissionPolicy = admissiondomain.Policy

// TokenSetMinter 在既有 Session 上签发尚未持久化的用户令牌集合
type TokenSetMinter interface {
	// MintTokenSet 颁发用户令牌。
	MintTokenSet(ctx context.Context, sess *sessiondomain.Session) (*UserTokenSet, error)
}

// Refresher 轮换 RefreshToken 并延续认证状态。
type Refresher interface {
	// RefreshToken 轮换刷新令牌。
	RefreshToken(ctx context.Context, refreshTokenValue string) (*UserTokenSet, error)
	// RevokeRefreshToken 撤销刷新令牌。
	RevokeRefreshToken(ctx context.Context, refreshTokenValue string) error
}

// Verifier 在线验证 access token 及用户认证状态。
type Verifier interface {
	// VerifyToken 验证令牌。
	VerifyToken(ctx context.Context, tokenValue string, expectedAudience []string) (*AccessTokenClaims, error)
}

// Revoker 撤销 bearer token 及其关联 Session。
type Revoker interface {
	// RevokeBearerToken 撤销令牌。
	RevokeBearerToken(ctx context.Context, tokenValue string) error
}
