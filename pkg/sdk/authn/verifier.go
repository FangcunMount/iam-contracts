package authnsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
	authnv1 "github.com/FangcunMount/iam-contracts/internal/apiserver/interface/authn/grpc/pb/iam/authn/v1"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Verifier JWT 验证器
// 负责验证 JWT token 的签名和声明
// 支持本地验证（使用 JWKS）和可选的远程验证（通过 gRPC）
type Verifier struct {
	cfg    Config       // 配置
	jwks   *JWKSManager // JWKS 管理器
	client *Client      // gRPC 客户端（可选）
}

// VerifyOptions 验证选项
// 控制单次验证调用的行为
type VerifyOptions struct {
	ForceRemote bool // 强制使用远程验证，即使本地验证成功
}

// NewVerifier builds a verifier using config & client.
// When client is nil the verifier works purely locally.
func NewVerifier(cfg Config, client *Client) (*Verifier, error) {
	if cfg.JWKSURL == "" {
		return nil, fmt.Errorf("jwks url is required")
	}
	cfg.setDefaults()
	log.Infof("[AuthN SDK] Initializing verifier with JWKS URL: %s", cfg.JWKSURL)
	if client != nil {
		log.Info("[AuthN SDK] Remote verification enabled")
	} else {
		log.Info("[AuthN SDK] Local verification only (no gRPC client)")
	}
	manager := newJWKSManager(cfg)
	return &Verifier{
		cfg:    cfg,
		jwks:   manager,
		client: client,
	}, nil
}

// Verify 验证 JWT token
// 首先进行本地验证（签名、过期时间、audience、issuer），
// 然后根据配置决定是否调用 IAM 进行远程验证
//
// 参数：
//   - ctx: 上下文
//   - token: JWT token 字符串
//   - opts: 验证选项（可选）
//
// 返回：
//   - *VerifyTokenResponse: 验证结果，包含 token 声明和状态
//   - error: 验证失败时返回错误
func (v *Verifier) Verify(ctx context.Context, token string, opts *VerifyOptions) (*authnv1.VerifyTokenResponse, error) {
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}
	log.Debug("[AuthN SDK] Starting token verification")
	localClaims, err := v.parseLocal(ctx, token)
	if err != nil {
		log.Warnf("[AuthN SDK] Local token verification failed: %v", err)
		return nil, err
	}
	log.Debugf("[AuthN SDK] Local verification successful, subject: %s, user_id: %s", localClaims.Subject, localClaims.UserId)
	resp := &authnv1.VerifyTokenResponse{
		Valid:  true,
		Status: authnv1.TokenStatus_TOKEN_STATUS_VALID,
		Claims: localClaims,
	}
	if v.shouldCallRemote(opts) {
		log.Debug("[AuthN SDK] Calling remote verification")
		if v.client == nil {
			return nil, fmt.Errorf("auth client not configured for remote verification")
		}
		remote, err := v.client.Auth().VerifyToken(ctx, &authnv1.VerifyTokenRequest{
			AccessToken:     token,
			ForceRemote:     true,
			IncludeMetadata: true,
		})
		if err != nil {
			log.Errorf("[AuthN SDK] Remote verification failed: %v", err)
			return nil, err
		}
		log.Debug("[AuthN SDK] Remote verification successful")
		resp = remote
	}
	log.Info("[AuthN SDK] Token verification completed successfully")
	return resp, nil
}

// shouldCallRemote 判断是否应该调用远程验证
// 根据调用选项和全局配置决定
//
// 参数：
//   - opts: 验证选项
//
// 返回：
//   - bool: true 表示应该调用远程验证
func (v *Verifier) shouldCallRemote(opts *VerifyOptions) bool {
	if opts != nil && opts.ForceRemote {
		return true
	}
	return v.cfg.ForceRemoteVerification
}

// parseLocal 本地解析和验证 JWT
// 验证签名、过期时间、audience 和 issuer
//
// 参数：
//   - ctx: 上下文
//   - token: JWT token 字符串
//
// 返回：
//   - *TokenClaims: 解析后的 token 声明
//   - error: 解析或验证失败时返回错误
func (v *Verifier) parseLocal(ctx context.Context, token string) (*authnv1.TokenClaims, error) {
	claims := jwt.MapClaims{}
	// jwt v4: 直接使用 ParseWithClaims，时钟偏差通过 MapClaims 自动处理
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512"}))
	log.Debug("[AuthN SDK] Parsing JWT token")
	if _, err := parser.ParseWithClaims(token, claims, v.jwks.Keyfunc(ctx)); err != nil {
		log.Debugf("[AuthN SDK] JWT parse failed: %v", err)
		return nil, err
	}
	if err := v.verifyAudience(claims); err != nil {
		log.Warnf("[AuthN SDK] Audience verification failed: %v", err)
		return nil, err
	}
	if err := v.verifyIssuer(claims); err != nil {
		log.Warnf("[AuthN SDK] Issuer verification failed: %v", err)
		return nil, err
	}
	return mapClaimsToProto(claims), nil
}

// verifyAudience 验证 JWT 的 audience 声明
// 如果配置了允许的 audience 列表，则 token 的 aud 必须匹配其中之一
//
// 参数：
//   - claims: JWT 声明
//
// 返回：
//   - error: 验证失败时返回错误
func (v *Verifier) verifyAudience(claims jwt.MapClaims) error {
	if len(v.cfg.AllowedAudience) == 0 {
		return nil // 未配置则跳过验证
	}
	// 检查是否匹配任一允许的 audience
	for _, aud := range v.cfg.AllowedAudience {
		if claims.VerifyAudience(aud, true) {
			return nil
		}
	}
	return fmt.Errorf("audience mismatch")
}

// verifyIssuer 验证 JWT 的 issuer 声明
// 如果配置了允许的 issuer，则 token 的 iss 必须匹配
//
// 参数：
//   - claims: JWT 声明
//
// 返回：
//   - error: 验证失败时返回错误
func (v *Verifier) verifyIssuer(claims jwt.MapClaims) error {
	if v.cfg.AllowedIssuer == "" {
		return nil // 未配置则跳过验证
	}
	if claims.VerifyIssuer(v.cfg.AllowedIssuer, true) {
		return nil
	}
	return fmt.Errorf("issuer mismatch")
}

// mapClaimsToProto 将 JWT 声明映射为 Proto 消息
// 提取标准声明和自定义属性
//
// 参数：
//   - claims: JWT 声明
//
// 返回：
//   - *TokenClaims: Proto 格式的 token 声明
func mapClaimsToProto(claims jwt.MapClaims) *authnv1.TokenClaims {
	accountID := claimString(claims, "account_id")
	if accountID == "" {
		accountID = claimString(claims, "acct")
	}
	aud := claimStrings(claims["aud"])
	p := &authnv1.TokenClaims{
		TokenId:    claimString(claims, "jti"),
		Subject:    claimString(claims, "sub"),
		UserId:     claimString(claims, "user_id"),
		AccountId:  accountID,
		Issuer:     claimString(claims, "iss"),
		TenantId:   claimString(claims, "tenant_id"),
		Audience:   aud,
		IssuedAt:   timestampFromClaim(claims["iat"]),
		ExpiresAt:  timestampFromClaim(claims["exp"]),
		Attributes: make(map[string]string),
	}
	for k, v := range claims {
		switch k {
		case "jti", "sub", "user_id", "account_id", "acct", "iss", "tenant_id", "aud", "iat", "exp", "nbf":
			continue
		default:
			if str := toString(v); str != "" {
				p.Attributes[k] = str
			}
		}
	}
	if len(p.Attributes) == 0 {
		p.Attributes = nil
	}
	if len(p.Audience) == 0 {
		p.Audience = nil
	}
	return p
}

// claimString 从声明中提取字符串值
// 将声明值转换为字符串类型
//
// 参数：
//   - claims: JWT 声明
//   - key: 声明键名
//
// 返回：
//   - string: 字符串值，不存在则返回空字符串
func claimString(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key]; ok {
		return toString(val)
	}
	return ""
}

// claimStrings 从声明中提取字符串数组
// 支持单个字符串或字符串数组
//
// 参数：
//   - v: 声明值
//
// 返回：
//   - []string: 字符串数组
func claimStrings(v interface{}) []string {
	switch vv := v.(type) {
	case string:
		return []string{vv}
	case []interface{}:
		out := make([]string, 0, len(vv))
		for _, item := range vv {
			if s := toString(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// timestampFromClaim 从声明值创建时间戳
// 将 JWT 的数字时间戳转换为 Proto 时间戳
//
// 参数：
//   - v: 声明值（数字或数字字符串）
//
// 返回：
//   - *timestamppb.Timestamp: Proto 时间戳，解析失败返回 nil
func timestampFromClaim(v interface{}) *timestamppb.Timestamp {
	seconds := parseNumeric(v)
	if seconds == 0 {
		return nil
	}
	return timestamppb.New(time.Unix(seconds, 0))
}

// parseNumeric 解析数字类型的声明值
// 支持多种数字类型：int、float、json.Number
//
// 参数：
//   - v: 声明值
//
// 返回：
//   - int64: 整数值，解析失败返回 0
func parseNumeric(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	case int64:
		return val
	case int32:
		return int64(val)
	case int:
		return int64(val)
	case json.Number:
		if i, err := strconv.ParseInt(string(val), 10, 64); err == nil {
			return i
		}
	}
	return 0
}

// toString 将任意类型转换为字符串
// 支持多种类型的转换：string、Stringer、数字类型等
//
// 参数：
//   - v: 任意类型的值
//
// 返回：
//   - string: 字符串表示，无法转换则返回空字符串
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	case float64:
		if math.Trunc(val) == val {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case float32:
		if math.Trunc(float64(val)) == float64(val) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case int, int32, int64:
		return fmt.Sprintf("%d", val)
	case uint, uint32, uint64:
		return fmt.Sprintf("%d", val)
	default:
		return ""
	}
}
