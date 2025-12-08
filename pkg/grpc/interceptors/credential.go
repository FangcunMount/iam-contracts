// Package interceptors 提供凭证验证拦截器
package interceptors

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ===== 凭证提取 =====

// MetadataCredentialExtractor 从 gRPC metadata 提取凭证
type MetadataCredentialExtractor struct {
	// 支持的 header 名称
	AuthorizationHeader string // 默认 "authorization"
	APIKeyHeader        string // 默认 "x-api-key"
	SignatureHeader     string // 默认 "x-api-signature"
	TimestampHeader     string // 默认 "x-api-timestamp"
	NonceHeader         string // 默认 "x-api-nonce"
}

// NewMetadataCredentialExtractor 创建 metadata 凭证提取器
func NewMetadataCredentialExtractor() *MetadataCredentialExtractor {
	return &MetadataCredentialExtractor{
		AuthorizationHeader: "authorization",
		APIKeyHeader:        "x-api-key",
		SignatureHeader:     "x-api-signature",
		TimestampHeader:     "x-api-timestamp",
		NonceHeader:         "x-api-nonce",
	}
}

// Extract 从上下文提取凭证
func (e *MetadataCredentialExtractor) Extract(ctx context.Context) (*ServiceCredential, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	// 优先检查 Bearer Token
	if tokens := md.Get(e.AuthorizationHeader); len(tokens) > 0 {
		token := tokens[0]
		if strings.HasPrefix(strings.ToLower(token), "bearer ") {
			return &ServiceCredential{
				Type:  CredentialTypeBearer,
				Token: strings.TrimSpace(token[7:]), // 去掉 "Bearer " 前缀
			}, nil
		}
	}

	// 检查 HMAC 签名
	if signatures := md.Get(e.SignatureHeader); len(signatures) > 0 {
		cred := &ServiceCredential{
			Type:      CredentialTypeHMAC,
			Signature: signatures[0],
		}

		if keys := md.Get(e.APIKeyHeader); len(keys) > 0 {
			cred.AccessKey = keys[0]
		}
		if timestamps := md.Get(e.TimestampHeader); len(timestamps) > 0 {
			ts, _ := strconv.ParseInt(timestamps[0], 10, 64)
			cred.Timestamp = ts
		}
		if nonces := md.Get(e.NonceHeader); len(nonces) > 0 {
			cred.Nonce = nonces[0]
		}

		return cred, nil
	}

	// 检查 API Key
	if keys := md.Get(e.APIKeyHeader); len(keys) > 0 {
		return &ServiceCredential{
			Type:      CredentialTypeAPIKey,
			AccessKey: keys[0],
		}, nil
	}

	return nil, fmt.Errorf("no credential found in metadata")
}

// ===== HMAC 验证器 =====

// HMACValidator HMAC 签名验证器
type HMACValidator struct {
	// 获取服务密钥的回调
	GetSecretKey func(accessKey string) (string, error)
	// 时间戳有效期
	TimestampValidity time.Duration
	// Nonce 存储（防重放）
	NonceStore NonceStore
}

// NonceStore Nonce 存储接口（防重放攻击）
type NonceStore interface {
	// Exists 检查 nonce 是否已存在
	Exists(ctx context.Context, nonce string) (bool, error)
	// Store 存储 nonce
	Store(ctx context.Context, nonce string, ttl time.Duration) error
}

// NewHMACValidator 创建 HMAC 验证器
func NewHMACValidator(getSecretKey func(accessKey string) (string, error)) *HMACValidator {
	return &HMACValidator{
		GetSecretKey:      getSecretKey,
		TimestampValidity: 5 * time.Minute,
	}
}

// Validate 验证 HMAC 签名
func (v *HMACValidator) Validate(ctx context.Context, cred *ServiceCredential) (*ServiceCredential, error) {
	if cred.Type != CredentialTypeHMAC {
		return nil, fmt.Errorf("invalid credential type: expected hmac, got %s", cred.Type)
	}

	// 验证时间戳
	timestamp := time.Unix(cred.Timestamp, 0)
	if time.Since(timestamp) > v.TimestampValidity {
		return nil, fmt.Errorf("timestamp expired")
	}
	if time.Since(timestamp) < -v.TimestampValidity {
		return nil, fmt.Errorf("timestamp too far in future")
	}

	// 验证 nonce（防重放）
	if v.NonceStore != nil && cred.Nonce != "" {
		exists, err := v.NonceStore.Exists(ctx, cred.Nonce)
		if err != nil {
			return nil, fmt.Errorf("failed to check nonce: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("nonce already used (replay attack)")
		}
	}

	// 获取密钥
	secretKey, err := v.GetSecretKey(cred.AccessKey)
	if err != nil {
		return nil, fmt.Errorf("invalid access key: %w", err)
	}

	// 计算并验证签名
	expectedSig := ComputeHMACSignature(cred.AccessKey, secretKey, cred.Timestamp, cred.Nonce)
	if !hmac.Equal([]byte(cred.Signature), []byte(expectedSig)) {
		return nil, fmt.Errorf("signature verification failed")
	}

	// 存储 nonce
	if v.NonceStore != nil && cred.Nonce != "" {
		_ = v.NonceStore.Store(ctx, cred.Nonce, v.TimestampValidity*2)
	}

	// 返回验证后的凭证
	return &ServiceCredential{
		Type:      CredentialTypeHMAC,
		AccessKey: cred.AccessKey,
		Subject:   cred.AccessKey,
		ExpiresAt: time.Now().Add(v.TimestampValidity),
	}, nil
}

// ComputeHMACSignature 计算 HMAC 签名
func ComputeHMACSignature(accessKey, secretKey string, timestamp int64, nonce string) string {
	message := fmt.Sprintf("%s%d%s", accessKey, timestamp, nonce)
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// ===== API Key 验证器 =====

// APIKeyInfo API Key 信息
type APIKeyInfo struct {
	ServiceName string
	Permissions []string
	ExpiresAt   time.Time
	Revoked     bool
}

// APIKeyStore API Key 存储接口
type APIKeyStore interface {
	// Get 根据 API Key 获取服务信息
	Get(ctx context.Context, apiKey string) (*APIKeyInfo, error)
}

// APIKeyValidator API Key 验证器
type APIKeyValidator struct {
	KeyStore APIKeyStore
}

// NewAPIKeyValidator 创建 API Key 验证器
func NewAPIKeyValidator(store APIKeyStore) *APIKeyValidator {
	return &APIKeyValidator{KeyStore: store}
}

// Validate 验证 API Key
func (v *APIKeyValidator) Validate(ctx context.Context, cred *ServiceCredential) (*ServiceCredential, error) {
	if cred.Type != CredentialTypeAPIKey {
		return nil, fmt.Errorf("invalid credential type: expected api_key, got %s", cred.Type)
	}

	info, err := v.KeyStore.Get(ctx, cred.AccessKey)
	if err != nil {
		return nil, fmt.Errorf("invalid API key: %w", err)
	}

	if info.Revoked {
		return nil, fmt.Errorf("API key has been revoked")
	}

	if !info.ExpiresAt.IsZero() && time.Now().After(info.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	return &ServiceCredential{
		Type:        CredentialTypeAPIKey,
		AccessKey:   cred.AccessKey,
		Subject:     info.ServiceName,
		Permissions: info.Permissions,
		ExpiresAt:   info.ExpiresAt,
	}, nil
}

// ===== 组合验证器 =====

// CompositeValidator 组合验证器（支持多种凭证类型）
type CompositeValidator struct {
	validators map[CredentialType]CredentialValidator
	mu         sync.RWMutex
}

// NewCompositeValidator 创建组合验证器
func NewCompositeValidator() *CompositeValidator {
	return &CompositeValidator{
		validators: make(map[CredentialType]CredentialValidator),
	}
}

// Register 注册验证器
func (v *CompositeValidator) Register(credType CredentialType, validator CredentialValidator) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.validators[credType] = validator
}

// Validate 验证凭证
func (v *CompositeValidator) Validate(ctx context.Context, cred *ServiceCredential) (*ServiceCredential, error) {
	v.mu.RLock()
	validator, ok := v.validators[cred.Type]
	v.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unsupported credential type: %s", cred.Type)
	}

	return validator.Validate(ctx, cred)
}

// ===== 凭证拦截器 =====

// CredentialInterceptor 应用层凭证验证拦截器
func CredentialInterceptor(extractor CredentialExtractor, validator CredentialValidator, opts ...CredentialOption) grpc.UnaryServerInterceptor {
	options := defaultCredentialOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 检查是否跳过
		if options.skipMatcher.Match(info.FullMethod) {
			return handler(ctx, req)
		}

		// 提取凭证
		cred, err := extractor.Extract(ctx)
		if err != nil {
			if options.optional {
				return handler(ctx, req)
			}
			if options.logger != nil {
				options.logger.LogError("credential extraction failed",
					map[string]interface{}{
						"method": info.FullMethod,
						"error":  err.Error(),
					})
			}
			return nil, status.Error(codes.Unauthenticated, "missing or invalid credentials")
		}

		// 验证凭证
		validatedCred, err := validator.Validate(ctx, cred)
		if err != nil {
			if options.logger != nil {
				options.logger.LogError("credential validation failed",
					map[string]interface{}{
						"method": info.FullMethod,
						"type":   string(cred.Type),
						"error":  err.Error(),
					})
			}
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}

		// 双重校验：证书服务名与 token subject 一致
		if options.requireIdentityMatch {
			if err := validateIdentityMatch(ctx, validatedCred); err != nil {
				if options.logger != nil {
					options.logger.LogError("identity mismatch",
						map[string]interface{}{
							"method": info.FullMethod,
							"error":  err.Error(),
						})
				}
				return nil, status.Error(codes.Unauthenticated, "identity mismatch")
			}
		}

		// 将凭证注入上下文
		ctx = ContextWithCredential(ctx, validatedCred)

		if options.logger != nil {
			options.logger.LogInfo("credential validation succeeded",
				map[string]interface{}{
					"method":  info.FullMethod,
					"type":    string(validatedCred.Type),
					"subject": validatedCred.Subject,
				})
		}

		return handler(ctx, req)
	}
}

// CredentialStreamInterceptor 流式凭证验证拦截器
func CredentialStreamInterceptor(extractor CredentialExtractor, validator CredentialValidator, opts ...CredentialOption) grpc.StreamServerInterceptor {
	options := defaultCredentialOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if options.skipMatcher.Match(info.FullMethod) {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		cred, err := extractor.Extract(ctx)
		if err != nil {
			if options.optional {
				return handler(srv, ss)
			}
			return status.Error(codes.Unauthenticated, "missing or invalid credentials")
		}

		validatedCred, err := validator.Validate(ctx, cred)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid credentials")
		}

		if options.requireIdentityMatch {
			if err := validateIdentityMatch(ctx, validatedCred); err != nil {
				return status.Error(codes.Unauthenticated, "identity mismatch")
			}
		}

		ctx = ContextWithCredential(ctx, validatedCred)
		wrappedStream := &WrappedServerStream{
			ServerStream: ss,
			Ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// validateIdentityMatch 验证 mTLS 身份与 Token 身份一致
func validateIdentityMatch(ctx context.Context, cred *ServiceCredential) error {
	identity, ok := ServiceIdentityFromContext(ctx)
	if !ok {
		return nil // 没有 mTLS 身份，跳过检查
	}

	if cred.Subject == "" {
		return nil // 凭证没有 subject，跳过检查
	}

	if identity.ServiceName != cred.Subject {
		return fmt.Errorf("mTLS service %q does not match credential subject %q",
			identity.ServiceName, cred.Subject)
	}

	return nil
}

// ===== 凭证选项 =====

type credentialOptions struct {
	skipMatcher          *SkipMethodMatcher
	optional             bool
	requireIdentityMatch bool
	logger               InterceptorLogger
}

func defaultCredentialOptions() *credentialOptions {
	return &credentialOptions{
		skipMatcher:          NewSkipMethodMatcher(DefaultSkipMethods()...),
		optional:             false,
		requireIdentityMatch: true,
	}
}

// CredentialOption 凭证拦截器选项函数
type CredentialOption func(*credentialOptions)

// WithCredentialSkipMethods 设置跳过验证的方法
func WithCredentialSkipMethods(methods ...string) CredentialOption {
	return func(o *credentialOptions) {
		o.skipMatcher.Add(methods...)
	}
}

// WithOptionalCredential 设置凭证为可选
func WithOptionalCredential() CredentialOption {
	return func(o *credentialOptions) {
		o.optional = true
	}
}

// WithoutIdentityMatch 禁用身份匹配检查
func WithoutIdentityMatch() CredentialOption {
	return func(o *credentialOptions) {
		o.requireIdentityMatch = false
	}
}

// WithCredentialLogger 设置日志记录器
func WithCredentialLogger(logger InterceptorLogger) CredentialOption {
	return func(o *credentialOptions) {
		o.logger = logger
	}
}

// ===== 内存存储实现（用于开发/测试）=====

// InMemoryNonceStore 内存 Nonce 存储
type InMemoryNonceStore struct {
	nonces map[string]time.Time
	mu     sync.RWMutex
}

// NewInMemoryNonceStore 创建内存 Nonce 存储
func NewInMemoryNonceStore() *InMemoryNonceStore {
	store := &InMemoryNonceStore{
		nonces: make(map[string]time.Time),
	}
	go store.cleanup()
	return store
}

// Exists 检查 nonce 是否存在
func (s *InMemoryNonceStore) Exists(ctx context.Context, nonce string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.nonces[nonce]
	return exists, nil
}

// Store 存储 nonce
func (s *InMemoryNonceStore) Store(ctx context.Context, nonce string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonces[nonce] = time.Now().Add(ttl)
	return nil
}

func (s *InMemoryNonceStore) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for nonce, expiry := range s.nonces {
			if now.After(expiry) {
				delete(s.nonces, nonce)
			}
		}
		s.mu.Unlock()
	}
}

// InMemoryAPIKeyStore 内存 API Key 存储
type InMemoryAPIKeyStore struct {
	keys map[string]*APIKeyInfo
	mu   sync.RWMutex
}

// NewInMemoryAPIKeyStore 创建内存 API Key 存储
func NewInMemoryAPIKeyStore() *InMemoryAPIKeyStore {
	return &InMemoryAPIKeyStore{
		keys: make(map[string]*APIKeyInfo),
	}
}

// Register 注册 API Key
func (s *InMemoryAPIKeyStore) Register(apiKey string, info *APIKeyInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[apiKey] = info
}

// Get 获取 API Key 信息
func (s *InMemoryAPIKeyStore) Get(ctx context.Context, apiKey string) (*APIKeyInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, ok := s.keys[apiKey]
	if !ok {
		return nil, fmt.Errorf("API key not found")
	}
	return info, nil
}

// Revoke 撤销 API Key
func (s *InMemoryAPIKeyStore) Revoke(apiKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if info, ok := s.keys[apiKey]; ok {
		info.Revoked = true
		return nil
	}
	return fmt.Errorf("API key not found")
}

// InMemorySecretStore 内存密钥存储（用于 HMAC）
type InMemorySecretStore struct {
	secrets map[string]string
	mu      sync.RWMutex
}

// NewInMemorySecretStore 创建内存密钥存储
func NewInMemorySecretStore() *InMemorySecretStore {
	return &InMemorySecretStore{
		secrets: make(map[string]string),
	}
}

// Register 注册密钥
func (s *InMemorySecretStore) Register(accessKey, secretKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[accessKey] = secretKey
}

// GetSecretKey 获取密钥
func (s *InMemorySecretStore) GetSecretKey(accessKey string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	secret, ok := s.secrets[accessKey]
	if !ok {
		return "", fmt.Errorf("access key not found")
	}
	return secret, nil
}

// ===== 客户端辅助函数 =====

// GenerateHMACCredentials 生成 HMAC 认证凭证（供客户端使用）
func GenerateHMACCredentials(accessKey, secretKey, nonce string) map[string]string {
	timestamp := time.Now().Unix()
	signature := ComputeHMACSignature(accessKey, secretKey, timestamp, nonce)

	return map[string]string{
		"x-api-key":       accessKey,
		"x-api-signature": signature,
		"x-api-timestamp": strconv.FormatInt(timestamp, 10),
		"x-api-nonce":     nonce,
	}
}
