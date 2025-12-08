package auth

import (
	"context"
	"fmt"
	"time"

	authnv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/authn/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/config"
	iamerrors "github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// =============================================================================
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
	if s.jwksManager == nil {
		return nil, fmt.Errorf("local-strategy: jwks manager not configured")
	}

	keySet, err := s.jwksManager.GetKeySet(ctx)
	if err != nil {
		return nil, fmt.Errorf("local-strategy: get keys: %w", err)
	}

	// 构建验证选项
	verifyOpts := []jwt.ParseOption{
		jwt.WithKeySet(keySet, jwa.RS256),
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
	if s.config != nil && s.config.AllowedIssuer != "" {
		verifyOpts = append(verifyOpts, jwt.WithIssuer(s.config.AllowedIssuer))
	}

	// 时钟偏差
	if s.config != nil && s.config.ClockSkew > 0 {
		verifyOpts = append(verifyOpts, jwt.WithAcceptableSkew(s.config.ClockSkew))
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

// =============================================================================
// 2. RemoteVerifyStrategy - 远程 gRPC 验证策略
// =============================================================================

// RemoteVerifyStrategy 远程验证策略（调用 IAM 服务）
type RemoteVerifyStrategy struct {
	authClient *Client
}

// NewRemoteVerifyStrategy 创建远程验证策略
func NewRemoteVerifyStrategy(authClient *Client) *RemoteVerifyStrategy {
	return &RemoteVerifyStrategy{
		authClient: authClient,
	}
}

func (s *RemoteVerifyStrategy) Name() string {
	return "remote"
}

func (s *RemoteVerifyStrategy) Verify(ctx context.Context, tokenString string, opts *VerifyOptions) (*VerifyResult, error) {
	if s.authClient == nil {
		return nil, fmt.Errorf("remote-strategy: auth client not configured")
	}

	resp, err := s.authClient.VerifyToken(ctx, &authnv1.VerifyTokenRequest{
		AccessToken: tokenString,
	})
	if err != nil {
		return nil, err
	}

	if !resp.Valid {
		return &VerifyResult{
			Valid: false,
		}, nil
	}

	claims := &TokenClaims{
		Subject:  resp.Claims.Subject,
		UserID:   resp.Claims.UserId,
		TenantID: resp.Claims.TenantId,
		Issuer:   resp.Claims.Issuer,
		Audience: resp.Claims.Audience,
		Extra:    make(map[string]interface{}),
	}

	if resp.Claims.ExpiresAt != nil {
		claims.ExpiresAt = resp.Claims.ExpiresAt.AsTime()
	}
	if resp.Claims.IssuedAt != nil {
		claims.IssuedAt = resp.Claims.IssuedAt.AsTime()
	}

	if resp.Claims.Attributes != nil {
		for k, v := range resp.Claims.Attributes {
			claims.Extra[k] = v
		}
	}

	return &VerifyResult{
		Valid:  true,
		Claims: claims,
	}, nil
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
	result, err := s.primary.Verify(ctx, token, opts)
	if err == nil {
		return result, nil
	}

	// 主策略失败，尝试降级策略
	if s.fallback != nil {
		return s.fallback.Verify(ctx, token, opts)
	}

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
	// 检查缓存
	if cached, ok := s.cache.Get(token); ok {
		return cached, nil
	}

	// 调用委托策略
	result, err := s.delegate.Verify(ctx, token, opts)
	if err != nil {
		return nil, err
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

	// TenantID 租户 ID
	TenantID string

	// Roles 角色列表
	Roles []string

	// Scopes 权限范围
	Scopes []string

	// TokenType Token 类型
	TokenType string

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
	return NewRemoteVerifyStrategy(s.authClient), nil
}

func (s *StrategySelector) selectWithFallback() (VerifyStrategy, error) {
	localStrategy := NewLocalVerifyStrategy(s.jwksManager, WithLocalConfig(s.cfg))
	remoteStrategy := NewRemoteVerifyStrategy(s.authClient)
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
	return NewRemoteVerifyStrategy(s.authClient), nil
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
	// 使用策略选择器选择策略
	selector := NewStrategySelector(cfg, jwksManager, authClient)
	strategy, err := selector.Select()
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
	claims := &TokenClaims{
		Subject:   token.Subject(),
		Issuer:    token.Issuer(),
		Audience:  token.Audience(),
		ExpiresAt: token.Expiration(),
		IssuedAt:  token.IssuedAt(),
		NotBefore: token.NotBefore(),
		Extra:     make(map[string]interface{}),
	}

	// 提取自定义声明
	if v, ok := token.Get("user_id"); ok {
		if s, ok := v.(string); ok {
			claims.UserID = s
		}
	}
	if v, ok := token.Get("tenant_id"); ok {
		if s, ok := v.(string); ok {
			claims.TenantID = s
		}
	}
	if v, ok := token.Get("roles"); ok {
		if arr, ok := v.([]interface{}); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					claims.Roles = append(claims.Roles, s)
				}
			}
		}
	}
	if v, ok := token.Get("scopes"); ok {
		if arr, ok := v.([]interface{}); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					claims.Scopes = append(claims.Scopes, s)
				}
			}
		}
	}
	if v, ok := token.Get("token_type"); ok {
		if s, ok := v.(string); ok {
			claims.TokenType = s
		}
	}

	return claims
}
