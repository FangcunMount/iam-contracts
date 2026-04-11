package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/FangcunMount/component-base/pkg/logger"
	authnv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/authn/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/config"
	iamerrors "github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// ==============================================q===============================
// 策略模式：VerifyStrategy 接口
// =============================================================================

// VerifyStrategy 定义 Token 验证策略接口
type VerifyStrategy interface {
	// Verify 验证 Token
	Verify(ctx context.Context, token string, opts *VerifyOptions) (*VerifyResult, error)
	// Name 返回策略名称
	Name() string
}

// =============================================================================
// 1. LocalVerifyStrategy - 本地 JWKS 验证策略
// =============================================================================

// LocalVerifyStrategy 本地验证策略（使用 JWKS）
type LocalVerifyStrategy struct {
	config      *config.TokenVerifyConfig
	jwksManager *JWKSManager
}

// LocalStrategyOption 本地策略配置选项
type LocalStrategyOption func(*LocalVerifyStrategy)

// WithLocalConfig 设置验证配置
func WithLocalConfig(cfg *config.TokenVerifyConfig) LocalStrategyOption {
	return func(s *LocalVerifyStrategy) {
		s.config = cfg
	}
}

// NewLocalVerifyStrategy 创建本地验证策略
func NewLocalVerifyStrategy(jwksManager *JWKSManager, opts ...LocalStrategyOption) *LocalVerifyStrategy {
	s := &LocalVerifyStrategy{
		config:      &config.TokenVerifyConfig{},
		jwksManager: jwksManager,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *LocalVerifyStrategy) Name() string {
	return "local"
}

func (s *LocalVerifyStrategy) Verify(ctx context.Context, tokenString string, opts *VerifyOptions) (*VerifyResult, error) {
	log.Println("LocalVerifyStrategy Verify", "tokenString", tokenString, "opts", opts)
	log.Println("LocalVerifyStrategy jwksManager", "jwksManager", s.jwksManager)
	if s.jwksManager == nil {
		log.Fatal("LocalVerifyStrategy jwks manager not configured")
		return nil, fmt.Errorf("local-strategy: jwks manager not configured")
	}

	keySet, err := s.jwksManager.GetKeySet(ctx)
	if err != nil {
		log.Fatal("LocalVerifyStrategy get keys", "err", err)
		return nil, fmt.Errorf("local-strategy: get keys: %w", err)
	}

	// 构建验证选项
	var verifyOpts []jwt.ParseOption

	// 配置允许的签名算法
	algorithms := s.getAllowedAlgorithms()
	if len(algorithms) == 1 {
		// 单个算法
		verifyOpts = append(verifyOpts, jwt.WithKeySet(keySet, algorithms[0]))
	} else {
		// 多个算法，使用验证回调
		verifyOpts = append(verifyOpts, jwt.WithKeySet(keySet))
	}

	// Audience 验证
	audience := opts.ExpectedAudience
	if len(audience) == 0 && s.config != nil {
		audience = s.config.AllowedAudience
	}
	if len(audience) > 0 {
		for _, aud := range audience {
			verifyOpts = append(verifyOpts, jwt.WithAudience(aud))
		}
	}

	// Issuer 验证
	issuer := opts.ExpectedIssuer
	if issuer == "" && s.config != nil {
		issuer = s.config.AllowedIssuer
	}
	if issuer != "" {
		verifyOpts = append(verifyOpts, jwt.WithIssuer(issuer))
	}

	// 时钟偏差
	if s.config != nil && s.config.ClockSkew > 0 {
		verifyOpts = append(verifyOpts, jwt.WithAcceptableSkew(s.config.ClockSkew))
	}

	// 配置必需的声明
	if s.config != nil && len(s.config.RequiredClaims) > 0 {
		for _, claim := range s.config.RequiredClaims {
			verifyOpts = append(verifyOpts, jwt.WithRequiredClaim(claim))
		}
	}

	// 解析并验证
	token, err := jwt.Parse([]byte(tokenString), verifyOpts...)
	if err != nil {
		if jwt.IsValidationError(err) {
			return nil, iamerrors.ErrTokenExpired
		}
		return nil, fmt.Errorf("local-strategy: parse token: %w", err)
	}

	claims := extractClaims(token)

	return &VerifyResult{
		Valid:    true,
		Claims:   claims,
		RawToken: token,
	}, nil
}

// getAllowedAlgorithms 返回允许的签名算法列表
// 如果未配置，默认返回 [RS256]（最常用的 RSA 算法）
func (s *LocalVerifyStrategy) getAllowedAlgorithms() []jwa.SignatureAlgorithm {
	if s.config == nil || len(s.config.Algorithms) == 0 {
		// 默认只允许 RS256
		return []jwa.SignatureAlgorithm{jwa.RS256}
	}

	algorithms := make([]jwa.SignatureAlgorithm, 0, len(s.config.Algorithms))
	for _, alg := range s.config.Algorithms {
		switch alg {
		case "RS256":
			algorithms = append(algorithms, jwa.RS256)
		case "RS384":
			algorithms = append(algorithms, jwa.RS384)
		case "RS512":
			algorithms = append(algorithms, jwa.RS512)
		case "ES256":
			algorithms = append(algorithms, jwa.ES256)
		case "ES384":
			algorithms = append(algorithms, jwa.ES384)
		case "ES512":
			algorithms = append(algorithms, jwa.ES512)
		case "PS256":
			algorithms = append(algorithms, jwa.PS256)
		case "PS384":
			algorithms = append(algorithms, jwa.PS384)
		case "PS512":
			algorithms = append(algorithms, jwa.PS512)
		case "EdDSA":
			algorithms = append(algorithms, jwa.EdDSA)
		}
	}

	if len(algorithms) == 0 {
		// 如果配置的算法都无效，回退到默认
		return []jwa.SignatureAlgorithm{jwa.RS256}
	}

	return algorithms
}

// =============================================================================
// 2. RemoteVerifyStrategy - 远程 gRPC 验证策略
// =============================================================================

// RemoteVerifyStrategy 远程验证策略（调用 IAM 服务）
type RemoteVerifyStrategy struct {
	authClient *Client
	config     *config.TokenVerifyConfig
}

// NewRemoteVerifyStrategy 创建远程验证策略
func NewRemoteVerifyStrategy(authClient *Client, cfg *config.TokenVerifyConfig) *RemoteVerifyStrategy {
	return &RemoteVerifyStrategy{
		authClient: authClient,
		config:     cfg,
	}
}

func (s *RemoteVerifyStrategy) Name() string {
	return "remote"
}

func (s *RemoteVerifyStrategy) Verify(ctx context.Context, tokenString string, opts *VerifyOptions) (*VerifyResult, error) {
	log.Println("RemoteVerifyStrategy Verify", "tokenString", tokenString, "opts", opts)
	log.Println("RemoteVerifyStrategy authClient", "authClient", s.authClient)
	if s.authClient == nil {
		log.Fatal("RemoteVerifyStrategy auth client not configured")
		return nil, fmt.Errorf("remote-strategy: auth client not configured")
	}

	resp, err := s.authClient.VerifyToken(ctx, &authnv1.VerifyTokenRequest{
		AccessToken:      tokenString,
		ExpectedIssuer:   s.expectedIssuer(opts),
		ExpectedAudience: s.expectedAudience(opts),
	})
	log.Println("RemoteVerifyStrategy verify token", "resp", resp)
	if err != nil {
		log.Fatal("RemoteVerifyStrategy verify token", "err", err)
		return nil, err
	}

	if !resp.Valid {
		log.Fatal("RemoteVerifyStrategy verify token invalid")
		return nil, fmt.Errorf("remote-strategy: verify token invalid")
	}

	log.Println("RemoteVerifyStrategy verify token valid", "resp", resp)

	claims := &TokenClaims{
		Subject:   resp.Claims.Subject,
		UserID:    resp.Claims.UserId,
		AccountID: resp.Claims.AccountId,
		TenantID:  resp.Claims.TenantId,
		Issuer:    resp.Claims.Issuer,
		Audience:  resp.Claims.Audience,
		Extra:     make(map[string]interface{}),
	}
	log.Println("RemoteVerifyStrategy claims", "claims", claims)

	if resp.Claims.ExpiresAt != nil {
		claims.ExpiresAt = resp.Claims.ExpiresAt.AsTime()
		log.Println("RemoteVerifyStrategy expires_at", "expires_at", claims.ExpiresAt)
	}
	if resp.Claims.IssuedAt != nil {
		claims.IssuedAt = resp.Claims.IssuedAt.AsTime()
		log.Println("RemoteVerifyStrategy issued_at", "issued_at", claims.IssuedAt)
	}

	if resp.Claims.Attributes != nil {
		for k, v := range resp.Claims.Attributes {
			claims.Extra[k] = v
		}
		log.Println("RemoteVerifyStrategy extra", "extra", claims.Extra)
	}

	log.Println("RemoteVerifyStrategy verify token success", "claims", claims)
	return &VerifyResult{
		Valid:  true,
		Claims: claims,
	}, nil
}

func (s *RemoteVerifyStrategy) expectedAudience(opts *VerifyOptions) []string {
	if opts != nil && len(opts.ExpectedAudience) > 0 {
		return append([]string(nil), opts.ExpectedAudience...)
	}
	if s.config != nil && len(s.config.AllowedAudience) > 0 {
		return append([]string(nil), s.config.AllowedAudience...)
	}
	return nil
}

func (s *RemoteVerifyStrategy) expectedIssuer(opts *VerifyOptions) string {
	if opts != nil && opts.ExpectedIssuer != "" {
		return opts.ExpectedIssuer
	}
	if s.config != nil {
		return s.config.AllowedIssuer
	}
	return ""
}

// =============================================================================
// 3. FallbackVerifyStrategy - 降级策略（本地失败后尝试远程）
// =============================================================================

// FallbackVerifyStrategy 降级策略
type FallbackVerifyStrategy struct {
	primary  VerifyStrategy
	fallback VerifyStrategy
}

// NewFallbackVerifyStrategy 创建降级策略
func NewFallbackVerifyStrategy(primary, fallback VerifyStrategy) *FallbackVerifyStrategy {
	return &FallbackVerifyStrategy{
		primary:  primary,
		fallback: fallback,
	}
}

func (s *FallbackVerifyStrategy) Name() string {
	return fmt.Sprintf("fallback(%s->%s)", s.primary.Name(), s.fallback.Name())
}

func (s *FallbackVerifyStrategy) Verify(ctx context.Context, token string, opts *VerifyOptions) (*VerifyResult, error) {
	log.Println("FallbackVerifyStrategy Verify", "token", token, "opts", opts)
	log.Println("FallbackVerifyStrategy primary", "primary", s.primary)
	log.Println("FallbackVerifyStrategy fallback", "fallback", s.fallback)
	result, err := s.primary.Verify(ctx, token, opts)
	if err == nil {
		log.Println("FallbackVerifyStrategy primary verify success", "result", result)
		return result, nil
	}

	// 主策略失败，尝试降级策略
	if s.fallback != nil {
		log.Println("FallbackVerifyStrategy fallback verify", "token", token, "opts", opts)
		result, err := s.fallback.Verify(ctx, token, opts)
		if err == nil {
			log.Println("FallbackVerifyStrategy fallback verify success", "result", result)
			return result, nil
		}
		log.Fatal("FallbackVerifyStrategy fallback verify failed", "err", err)
		return nil, err
	}

	log.Fatal("FallbackVerifyStrategy verify failed", "err", err)
	return nil, err
}

// =============================================================================
// 4. CachingVerifyStrategy - 缓存策略（缓存验证结果）
// =============================================================================

// CachingVerifyStrategy 缓存验证结果的策略
type CachingVerifyStrategy struct {
	delegate VerifyStrategy
	cache    VerifyResultCache
	ttl      time.Duration
}

// VerifyResultCache 验证结果缓存接口
type VerifyResultCache interface {
	Get(token string) (*VerifyResult, bool)
	Set(token string, result *VerifyResult, ttl time.Duration)
}

// NewCachingVerifyStrategy 创建缓存策略
func NewCachingVerifyStrategy(delegate VerifyStrategy, cache VerifyResultCache, ttl time.Duration) *CachingVerifyStrategy {
	return &CachingVerifyStrategy{
		delegate: delegate,
		cache:    cache,
		ttl:      ttl,
	}
}

func (s *CachingVerifyStrategy) Name() string {
	return fmt.Sprintf("caching(%s)", s.delegate.Name())
}

func (s *CachingVerifyStrategy) Verify(ctx context.Context, token string, opts *VerifyOptions) (*VerifyResult, error) {
	log.Println("CachingVerifyStrategy Verify", "token", token, "opts", opts)
	log.Println("CachingVerifyStrategy delegate", "delegate", s.delegate)
	log.Println("CachingVerifyStrategy cache", "cache", s.cache)
	log.Println("CachingVerifyStrategy ttl", "ttl", s.ttl)
	// 检查缓存
	if cached, ok := s.cache.Get(token); ok {
		log.Println("CachingVerifyStrategy get cached", "cached", cached)
		return cached, nil
	}

	// 调用委托策略
	result, err := s.delegate.Verify(ctx, token, opts)
	if err != nil {
		log.Fatal("CachingVerifyStrategy delegate verify failed", "err", err)
		return nil, fmt.Errorf("caching-strategy: delegate verify failed: %w", err)
	}

	// 缓存结果
	s.cache.Set(token, result, s.ttl)

	return result, nil
}

// =============================================================================
// TokenVerifier - 使用策略模式的验证器
// =============================================================================

// TokenVerifier Token 验证器（使用策略模式）
type TokenVerifier struct {
	config   *config.TokenVerifyConfig
	strategy VerifyStrategy
}

// VerifyResult 验证结果
type VerifyResult struct {
	// Valid 是否有效
	Valid bool

	// Claims Token 声明
	Claims *TokenClaims

	// RawToken 原始 JWT Token
	RawToken jwt.Token
}

// TokenClaims Token 声明
type TokenClaims struct {
	// Subject 主题（通常是用户 ID）
	Subject string

	// Issuer 签发者
	Issuer string

	// Audience 受众
	Audience []string

	// ExpiresAt 过期时间
	ExpiresAt time.Time

	// IssuedAt 签发时间
	IssuedAt time.Time

	// NotBefore 生效时间
	NotBefore time.Time

	// UserID 用户 ID
	UserID string

	// AccountID 账户 ID（JWT claim account_id）
	AccountID string

	// TenantID 租户 ID
	TenantID string

	// Roles 角色列表
	Roles []string

	// Scopes 权限范围
	Scopes []string

	// TokenType Token 类型
	TokenType string

	// AMR 认证方法引用（JWT `amr` claim）
	AMR []string

	// Extra 额外声明
	Extra map[string]interface{}
}

// VerifyOptions 验证选项
type VerifyOptions struct {
	// ForceRemote 强制使用远程验证
	ForceRemote bool

	// IncludeMetadata 包含元数据
	IncludeMetadata bool

	// ExpectedAudience 期望的 audience（覆盖默认配置）
	ExpectedAudience []string

	// ExpectedIssuer 期望的 issuer（覆盖默认配置）
	ExpectedIssuer string
}

// TokenVerifierOption 验证器配置选项
type TokenVerifierOption func(*TokenVerifier)

// WithVerifyStrategy 设置验证策略
func WithVerifyStrategy(strategy VerifyStrategy) TokenVerifierOption {
	return func(v *TokenVerifier) {
		v.strategy = strategy
	}
}

// WithVerifyConfig 设置验证配置
func WithVerifyConfig(cfg *config.TokenVerifyConfig) TokenVerifierOption {
	return func(v *TokenVerifier) {
		v.config = cfg
	}
}

// =============================================================================
// StrategySelector - 策略选择器
// =============================================================================

// StrategySelector 策略选择器，根据条件选择合适的验证策略
type StrategySelector struct {
	cfg         *config.TokenVerifyConfig
	jwksManager *JWKSManager
	authClient  *Client
}

// NewStrategySelector 创建策略选择器
func NewStrategySelector(cfg *config.TokenVerifyConfig, jwksManager *JWKSManager, authClient *Client) *StrategySelector {
	if cfg == nil {
		cfg = &config.TokenVerifyConfig{}
	}
	return &StrategySelector{
		cfg:         cfg,
		jwksManager: jwksManager,
		authClient:  authClient,
	}
}

// Select 根据配置选择最佳策略
func (s *StrategySelector) Select() (VerifyStrategy, error) {
	// 检查可用资源
	hasJWKS := s.jwksManager != nil
	hasRemote := s.authClient != nil

	if !hasJWKS && !hasRemote {
		return nil, fmt.Errorf("strategy-selector: at least one of jwksManager or authClient is required")
	}

	// 根据配置选择策略
	switch {
	case s.cfg.ForceRemoteVerification:
		// 强制远程验证
		return s.selectRemoteOnly()

	case hasJWKS && hasRemote:
		// 两者都有，使用降级策略
		return s.selectWithFallback()

	case hasJWKS:
		// 仅本地
		return s.selectLocalOnly()

	case hasRemote:
		// 仅远程
		return s.selectRemoteOnly()

	default:
		return nil, fmt.Errorf("strategy-selector: no valid strategy available")
	}
}

func (s *StrategySelector) selectLocalOnly() (VerifyStrategy, error) {
	if s.jwksManager == nil {
		return nil, fmt.Errorf("strategy-selector: jwks manager required for local strategy")
	}
	return NewLocalVerifyStrategy(s.jwksManager, WithLocalConfig(s.cfg)), nil
}

func (s *StrategySelector) selectRemoteOnly() (VerifyStrategy, error) {
	if s.authClient == nil {
		return nil, fmt.Errorf("strategy-selector: auth client required for remote strategy")
	}
	return NewRemoteVerifyStrategy(s.authClient, s.cfg), nil
}

func (s *StrategySelector) selectWithFallback() (VerifyStrategy, error) {
	localStrategy := NewLocalVerifyStrategy(s.jwksManager, WithLocalConfig(s.cfg))
	remoteStrategy := NewRemoteVerifyStrategy(s.authClient, s.cfg)
	return NewFallbackVerifyStrategy(localStrategy, remoteStrategy), nil
}

// LocalStrategy 显式获取本地策略
func (s *StrategySelector) LocalStrategy() (*LocalVerifyStrategy, error) {
	if s.jwksManager == nil {
		return nil, fmt.Errorf("strategy-selector: jwks manager not available")
	}
	return NewLocalVerifyStrategy(s.jwksManager, WithLocalConfig(s.cfg)), nil
}

// RemoteStrategy 显式获取远程策略
func (s *StrategySelector) RemoteStrategy() (*RemoteVerifyStrategy, error) {
	if s.authClient == nil {
		return nil, fmt.Errorf("strategy-selector: auth client not available")
	}
	return NewRemoteVerifyStrategy(s.authClient, s.cfg), nil
}

// FallbackStrategy 显式获取降级策略
func (s *StrategySelector) FallbackStrategy() (*FallbackVerifyStrategy, error) {
	local, err := s.LocalStrategy()
	if err != nil {
		return nil, err
	}
	remote, err := s.RemoteStrategy()
	if err != nil {
		return nil, err
	}
	return NewFallbackVerifyStrategy(local, remote), nil
}

// =============================================================================
// NewTokenVerifier - 使用策略选择器
// =============================================================================

// NewTokenVerifier 创建 Token 验证器
//
// 使用策略选择器自动选择最佳验证策略
func NewTokenVerifier(cfg *config.TokenVerifyConfig, jwksManager *JWKSManager, authClient *Client) (*TokenVerifier, error) {
	logger.L(context.Background()).Debugw("NewTokenVerifier cfg", "cfg", cfg)
	logger.L(context.Background()).Debugw("NewTokenVerifier jwksManager", "jwksManager", jwksManager)
	logger.L(context.Background()).Debugw("NewTokenVerifier authClient", "authClient", authClient)

	// 使用策略选择器选择策略
	selector := NewStrategySelector(cfg, jwksManager, authClient)
	strategy, err := selector.Select()
	logger.L(context.Background()).Debugw("NewTokenVerifier strategy", "strategy", strategy)
	if err != nil {
		return nil, err
	}

	return &TokenVerifier{
		config:   cfg,
		strategy: strategy,
	}, nil
}

// NewTokenVerifierWithStrategy 使用自定义策略创建验证器
func NewTokenVerifierWithStrategy(strategy VerifyStrategy, opts ...TokenVerifierOption) *TokenVerifier {
	v := &TokenVerifier{
		config:   &config.TokenVerifyConfig{},
		strategy: strategy,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Verify 验证 Token
func (v *TokenVerifier) Verify(ctx context.Context, token string, opts *VerifyOptions) (*VerifyResult, error) {
	if opts == nil {
		opts = &VerifyOptions{}
	}

	return v.strategy.Verify(ctx, token, opts)
}

// Strategy 返回当前使用的策略
func (v *TokenVerifier) Strategy() VerifyStrategy {
	return v.strategy
}

// =============================================================================
// 辅助函数
// =============================================================================

func extractClaims(token jwt.Token) *TokenClaims {
	logger.L(context.Background()).Debugw("extractClaims token", "token", token)
	claims := &TokenClaims{
		Subject:   token.Subject(),
		Issuer:    token.Issuer(),
		Audience:  token.Audience(),
		ExpiresAt: token.Expiration(),
		IssuedAt:  token.IssuedAt(),
		NotBefore: token.NotBefore(),
		Extra:     make(map[string]interface{}),
	}
	logger.L(context.Background()).Debugw("extractClaims token", "token", token)
	logger.L(context.Background()).Debugw(
		"extractClaims claims", "claims", claims,
		"extractClaims subject", "subject", claims.Subject,
		"extractClaims issuer", "issuer", claims.Issuer,
		"extractClaims audience", "audience", claims.Audience,
		"extractClaims expires_at", "expires_at", claims.ExpiresAt,
		"extractClaims issued_at", "issued_at", claims.IssuedAt,
		"extractClaims not_before", "not_before", claims.NotBefore,
		"extractClaims extra", "extra", claims.Extra,
		"extractClaims user_id", "user_id", claims.UserID,
		"extractClaims tenant_id", "tenant_id", claims.TenantID,
		"extractClaims account_id", "account_id", claims.AccountID,
		"extractClaims roles", "roles", claims.Roles,
		"extractClaims scopes", "scopes", claims.Scopes,
		"extractClaims token_type", "token_type", claims.TokenType,
	)

	// 提取自定义声明
	if v, ok := token.Get("user_id"); ok {
		claims.UserID = claimString(v)
		logger.L(context.Background()).Debugw("extractClaims user_id", "user_id", claims.UserID)
	}

	if v, ok := token.Get("tenant_id"); ok {
		claims.TenantID = claimString(v)
		logger.L(context.Background()).Debugw("extractClaims tenant_id", "tenant_id", claims.TenantID)
	}
	if v, ok := token.Get("account_id"); ok {
		claims.AccountID = claimString(v)
		logger.L(context.Background()).Debugw("extractClaims account_id", "account_id", claims.AccountID)
	}
	if v, ok := token.Get("roles"); ok {
		if arr, ok := v.([]interface{}); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					claims.Roles = append(claims.Roles, s)
				}
			}
		}
		logger.L(context.Background()).Debugw("extractClaims roles", "roles", claims.Roles)
	}
	if v, ok := token.Get("scopes"); ok {
		if arr, ok := v.([]interface{}); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					claims.Scopes = append(claims.Scopes, s)
				}
			}
		}
		logger.L(context.Background()).Debugw("extractClaims scopes", "scopes", claims.Scopes)
	}
	if v, ok := token.Get("token_type"); ok {
		if s, ok := v.(string); ok {
			claims.TokenType = s
		}
	}
	if v, ok := token.Get("amr"); ok {
		if arr, ok := v.([]interface{}); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					claims.AMR = append(claims.AMR, s)
				}
			}
		}
	}
	if v, ok := token.Get("attributes"); ok {
		if m, ok := v.(map[string]interface{}); ok {
			for k, val := range m {
				claims.Extra[k] = val
			}
		}
	}

	return claims
}

func claimString(v interface{}) string {
	switch value := v.(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	case float64:
		return fmt.Sprintf("%.0f", value)
	case int64:
		return fmt.Sprintf("%d", value)
	case int:
		return fmt.Sprintf("%d", value)
	case uint64:
		return fmt.Sprintf("%d", value)
	case uint:
		return fmt.Sprintf("%d", value)
	default:
		return ""
	}
}
