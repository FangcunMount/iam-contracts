// Package jwt implements IAM access token encoding with JWS compact JWT.
package jwt

import (
	"context"
	"crypto/rsa"
	"fmt"
	"strconv"
	"strings"
	"time"

	tokendomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/token"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	pkgauth "github.com/FangcunMount/iam/v5/pkg/auth"
	jwtv4 "github.com/golang-jwt/jwt/v4"
)

// SigningKey 签名密钥
type SigningKey struct {
	Kid        string          // 密钥ID
	Algorithm  string          // 算法
	PrivateKey *rsa.PrivateKey // 私钥
}

// VerificationKey 验签密钥
type VerificationKey struct {
	Kid       string         // 密钥ID
	Algorithm string         // 算法
	PublicKey *rsa.PublicKey // 公钥
}

// JWSKeySource 签名密钥源
type JWSKeySource interface {
	// ActiveSigningKey 获取活动签名密钥
	ActiveSigningKey(ctx context.Context) (*SigningKey, error)
	// VerificationKey 获取验签密钥
	VerificationKey(ctx context.Context, kid string) (*VerificationKey, error)
}

// SignedJWTCodec 将 IAM access claims 编码为 JWS Compact Signed JWT，
// 并把验签通过的 JWT Claims Set 投影为领域事实。
type SignedJWTCodec struct {
	issuer    string       // 签发者域名
	keySource JWSKeySource // JWS 签名与验签密钥源，由 JWSKeySource 接口实现
}

// 同一个 wire adapter 实现两个独立的密码学能力端口。
var _ tokendomain.AccessTokenEncoder = (*SignedJWTCodec)(nil)
var _ tokendomain.AccessTokenSignatureVerifier = (*SignedJWTCodec)(nil)

// NewSignedJWTCodec 创建 Signed JWT 编解码器。
func NewSignedJWTCodec(issuer string, keySource JWSKeySource) *SignedJWTCodec {
	return &SignedJWTCodec{issuer: issuer, keySource: keySource}
}

// jwtPayloadClaims 是 JWT Payload 的 wire model，不向领域层泄漏。
type jwtPayloadClaims struct {
	TokenType       string `json:"token_type,omitempty"`
	SessionID       string `json:"sid,omitempty"`
	UserID          string `json:"user_id,omitempty"`
	LoginIdentityID string `json:"login_identity_id,omitempty"`
	OrgID           string `json:"org_id,omitempty"`

	AuthTime   int64             `json:"auth_time,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	AMR        []string          `json:"amr,omitempty"`
	jwtv4.RegisteredClaims
}

// EncodeAccessToken maps domain claims to the JWT wire format and signs them.
func (g *SignedJWTCodec) EncodeAccessToken(ctx context.Context, claims *tokendomain.AccessTokenClaims) (string, error) {
	if err := claims.Validate(); err != nil {
		return "", err
	}
	if strings.TrimSpace(g.issuer) == "" || claims.Issuer != g.issuer {
		return "", fmt.Errorf("unexpected token issuer: %q", claims.Issuer)
	}
	attributes := cloneStringMap(claims.Attributes)
	authTime := int64(0)
	if !claims.AuthenticatedAt.IsZero() {
		authTime = claims.AuthenticatedAt.UTC().Unix()
		if attributes == nil {
			attributes = map[string]string{}
		}
		// Legacy consumers still read the string form during the compatibility window.
		attributes["auth_time"] = claims.AuthenticatedAt.UTC().Format(time.RFC3339)
	}
	orgID := ""
	if !claims.OrgID.IsZero() {
		orgID = claims.OrgID.String()
	}
	return g.signClaims(ctx, jwtPayloadClaims{
		TokenType: string(claims.TokenType), SessionID: claims.SessionID,
		UserID: claims.UserID.String(), LoginIdentityID: claims.LoginIdentityID.String(),
		OrgID: orgID, AuthTime: authTime,
		Attributes: attributes, AMR: cloneStrings(claims.AMR),
		RegisteredClaims: jwtv4.RegisteredClaims{
			ID: claims.TokenID, Subject: claims.Subject, Issuer: claims.Issuer,
			Audience:  jwtv4.ClaimStrings(cloneStrings(claims.Audience)),
			IssuedAt:  jwtv4.NewNumericDate(claims.IssuedAt),
			NotBefore: jwtv4.NewNumericDate(claims.NotBefore), ExpiresAt: jwtv4.NewNumericDate(claims.ExpiresAt),
		},
	})
}

// VerifySignatureAndClaims 验证 access bearer token。

func (g *SignedJWTCodec) VerifySignatureAndClaims(ctx context.Context, tokenValue string) (*tokendomain.AccessTokenClaims, error) {
	// 解析 JWT
	parsed, err := jwtv4.ParseWithClaims(tokenValue, &jwtPayloadClaims{},
		func(token *jwtv4.Token) (interface{}, error) {
			headerAlgorithm, ok := token.Header["alg"].(string)
			if !ok || headerAlgorithm != pkgauth.TokenProfileAlgorithm || token.Method.Alg() != pkgauth.TokenProfileAlgorithm {
				return nil, fmt.Errorf("unexpected signing algorithm: %v", token.Header["alg"])
			}
			// 获取签名密钥 ID
			kidInterface, ok := token.Header["kid"]
			if !ok {
				return nil, fmt.Errorf("missing kid in token header")
			}
			kid, ok := kidInterface.(string)
			if !ok {
				return nil, fmt.Errorf("invalid kid type in token header")
			}
			if kid == "" {
				return nil, fmt.Errorf("empty kid in token header")
			}

			// 获取签名密钥
			key, err := g.keySource.VerificationKey(ctx, kid)
			if err != nil {
				return nil, fmt.Errorf("failed to get key %s: %w", kid, err)
			}
			if key == nil {
				return nil, fmt.Errorf("key not found for kid %s", kid)
			}
			if key.Kid != kid {
				return nil, fmt.Errorf("verification key kid mismatch: header=%s key=%s", kid, key.Kid)
			}
			if key.Algorithm != headerAlgorithm || key.Algorithm != pkgauth.TokenProfileAlgorithm {
				return nil, fmt.Errorf("verification algorithm mismatch: header=%s key=%s", headerAlgorithm, key.Algorithm)
			}
			if key.PublicKey == nil {
				return nil, fmt.Errorf("verification key is nil for kid %s", kid)
			}

			// 返回签名密钥
			return key.PublicKey, nil
		},
		jwtv4.WithValidMethods([]string{pkgauth.TokenProfileAlgorithm}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// 解析 JWT 声明
	claims, ok := parsed.Claims.(*jwtPayloadClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	if claims.Issuer != g.issuer {
		return nil, fmt.Errorf("unexpected token issuer: %q", claims.Issuer)
	}

	// 获取令牌类型。兼容窗口内缺失 token_type 仍按 access 处理，并记录有界指标。
	tokenType := tokendomain.TokenType(claims.TokenType)
	switch tokenType {
	case "":
		missingTokenTypeTotal.Inc()
		tokenType = tokendomain.TokenTypeAccess
	case tokendomain.TokenTypeAccess:
	default:
		return nil, fmt.Errorf("unsupported token_type: %q", claims.TokenType)
	}

	// 解析登录身份 ID
	loginIdentityID := parseStringID(claims.LoginIdentityID)
	// 解析组织 ID
	orgID := parseStringID(claims.OrgID)
	attributes := cloneStringMap(claims.Attributes)
	authTime := time.Time{}
	if claims.AuthTime > 0 {
		authTime = time.Unix(claims.AuthTime, 0).UTC()
		if attributes == nil {
			attributes = map[string]string{}
		}
		if attributes["auth_time"] == "" {
			attributes["auth_time"] = authTime.Format(time.RFC3339)
		}
	} else if raw := strings.TrimSpace(attributes["auth_time"]); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			legacyAttributeAuthTimeFallbackTotal.Inc()
			authTime = parsed.UTC()
		}
	}

	// 构造并校验令牌事实
	verified := tokendomain.AccessTokenClaims{
		TokenID: claims.ID, TokenType: tokenType, Subject: claims.Subject, SessionID: claims.SessionID,
		UserID: parseStringID(claims.UserID), LoginIdentityID: loginIdentityID,
		OrgID: orgID, Issuer: claims.Issuer,
		Audience: []string(claims.Audience), Attributes: attributes, AMR: claims.AMR,
		AuthenticatedAt: authTime, IssuedAt: numericDateTime(claims.IssuedAt),
		NotBefore: numericDateTime(claims.NotBefore), ExpiresAt: numericDateTime(claims.ExpiresAt),
	}
	return tokendomain.NewAccessTokenClaims(verified)
}

// signClaims 签名 JWT 声明

func (g *SignedJWTCodec) signClaims(ctx context.Context, claims jwtPayloadClaims) (string, error) {
	// 获取活动签名密钥
	key, err := g.keySource.ActiveSigningKey(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to resolve active signing key: %w", err)
	}
	if key == nil || key.PrivateKey == nil {
		return "", fmt.Errorf("active signing key is nil")
	}
	if key.Kid == "" {
		return "", fmt.Errorf("active signing key kid is empty")
	}
	if key.Algorithm != pkgauth.TokenProfileAlgorithm {
		return "", fmt.Errorf("active signing key algorithm %q is not allowed", key.Algorithm)
	}
	// 创建 JWT 令牌
	token := jwtv4.NewWithClaims(jwtv4.SigningMethodRS256, claims)
	// 设置令牌类型
	token.Header["typ"] = headerTypeJWT
	// 设置签名密钥 ID
	token.Header["kid"] = key.Kid
	// 签名 JWT 令牌
	return token.SignedString(key.PrivateKey)
}

// cloneStrings 克隆字符串切片
func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

// cloneStringMap 克隆字符串映射
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

// parseStringID 解析字符串 ID
func parseStringID(raw string) meta.ID {
	if raw == "" {
		return meta.FromUint64(0)
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return meta.FromUint64(0)
	}
	return meta.FromUint64(value)
}

// numericDateTime 解析数字日期时间
func numericDateTime(v *jwtv4.NumericDate) time.Time {
	if v == nil {
		return time.Time{}
	}
	return v.Time
}
