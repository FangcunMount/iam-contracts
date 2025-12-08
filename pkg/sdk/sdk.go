// Package sdk 提供 IAM 系统的 Go SDK
//
// SDK 采用模块化设计，包含以下子包：
//
//   - config: 配置定义和加载
//   - transport: gRPC 传输层（连接、TLS、拦截器）
//   - observability: 可观测性（Metrics、Tracing）
//   - errors: 统一错误处理
//   - auth: 认证服务（Token 验证、JWKS、服务间认证）
//   - identity: 身份服务（用户管理、监护关系）
//
// 快速开始：
//
//	import "github.com/FangcunMount/iam-contracts/pkg/sdk"
//
//	client, err := sdk.NewClient(ctx, &sdk.Config{
//	    Endpoint: "iam.example.com:8081",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// 使用认证服务
//	resp, err := client.Auth().VerifyToken(ctx, &authnv1.VerifyTokenRequest{...})
//
//	// 使用身份服务
//	user, err := client.Identity().GetUser(ctx, "user-123")
//
//	// 使用监护关系服务
//	result, err := client.Guardianship().IsGuardian(ctx, "user-1", "child-1")
package sdk

import (
	"context"
	"fmt"

	authnv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/authn/v1"
	identityv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/identity/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/auth"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/config"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/identity"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/transport"
	"google.golang.org/grpc"
)

// ========== 类型别名（便于外部使用） ==========

// Config 配置
type Config = config.Config

// TLSConfig TLS 配置
type TLSConfig = config.TLSConfig

// RetryConfig 重试配置
type RetryConfig = config.RetryConfig

// JWKSConfig JWKS 配置
type JWKSConfig = config.JWKSConfig

// KeepaliveConfig Keepalive 配置
type KeepaliveConfig = config.KeepaliveConfig

// TokenVerifyConfig Token 验证配置
type TokenVerifyConfig = config.TokenVerifyConfig

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig = config.CircuitBreakerConfig

// ServiceAuthConfig 服务间认证配置
type ServiceAuthConfig = config.ServiceAuthConfig

// ClientOption 客户端选项
type ClientOption = config.ClientOption

// ========== 选项函数 ==========

// WithUnaryInterceptors 添加 Unary 拦截器
var WithUnaryInterceptors = config.WithUnaryInterceptors

// WithStreamInterceptors 添加 Stream 拦截器
var WithStreamInterceptors = config.WithStreamInterceptors

// WithDialOptions 添加 gRPC DialOption
var WithDialOptions = config.WithDialOptions

// ========== 配置加载 ==========

// ConfigFromEnv 从环境变量加载配置
var ConfigFromEnv = config.FromEnv

// ConfigFromEnvWithPrefix 从带前缀的环境变量加载配置
var ConfigFromEnvWithPrefix = config.FromEnvWithPrefix

// NewViperLoader 创建 Viper 配置加载器
var NewViperLoader = config.NewViperLoader

// DefaultConfig 返回默认配置
var DefaultConfig = config.DefaultConfig

// ========== Context 工具 ==========

// WithRequestID 设置 request-id
var WithRequestID = transport.WithRequestID

// WithTraceID 设置 trace-id
var WithTraceID = transport.WithTraceID

// GetRequestID 获取 request-id
var GetRequestID = transport.GetRequestID

// GetTraceID 获取 trace-id
var GetTraceID = transport.GetTraceID

// ========== Client ==========

// Client IAM 统一客户端
type Client struct {
	conn *grpc.ClientConn
	cfg  *Config

	// 子客户端
	authClient         *auth.Client
	identityClient     *identity.Client
	guardianshipClient *identity.GuardianshipClient
}

// NewClient 创建 IAM 客户端
func NewClient(ctx context.Context, cfg *Config, opts ...ClientOption) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("sdk: config is required")
	}

	// 填充默认值
	cfg = cfg.WithDefaults()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 应用选项
	clientOpts := config.ApplyOptions(opts...)

	// 添加默认拦截器
	clientOpts.UnaryInterceptors = append(
		[]grpc.UnaryClientInterceptor{transport.RequestIDInterceptor()},
		clientOpts.UnaryInterceptors...,
	)

	if len(cfg.Metadata) > 0 {
		clientOpts.UnaryInterceptors = append(
			clientOpts.UnaryInterceptors,
			transport.MetadataInterceptor(cfg.Metadata),
		)
	}

	// 建立连接
	conn, err := transport.Dial(ctx, cfg, clientOpts)
	if err != nil {
		return nil, err
	}

	client := &Client{
		conn: conn,
		cfg:  cfg,
	}

	// 初始化子客户端
	client.initSubClients()

	return client, nil
}

func (c *Client) initSubClients() {
	// Auth 客户端
	authService := authnv1.NewAuthServiceClient(c.conn)
	jwksService := authnv1.NewJWKSServiceClient(c.conn)
	c.authClient = auth.NewClient(authService, jwksService)

	// Identity 客户端
	readService := identityv1.NewIdentityReadClient(c.conn)
	lifecycleService := identityv1.NewIdentityLifecycleClient(c.conn)
	c.identityClient = identity.NewClient(readService, lifecycleService)

	// Guardianship 客户端
	queryService := identityv1.NewGuardianshipQueryClient(c.conn)
	commandService := identityv1.NewGuardianshipCommandClient(c.conn)
	c.guardianshipClient = identity.NewGuardianshipClient(queryService, commandService)
}

// Auth 返回认证服务客户端
func (c *Client) Auth() *auth.Client {
	return c.authClient
}

// Identity 返回身份服务客户端
func (c *Client) Identity() *identity.Client {
	return c.identityClient
}

// Guardianship 返回监护关系服务客户端
func (c *Client) Guardianship() *identity.GuardianshipClient {
	return c.guardianshipClient
}

// Conn 返回底层 gRPC 连接
func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ========== 便捷构造函数 ==========

// NewTokenVerifier 创建 Token 验证器
func NewTokenVerifier(verifyCfg *TokenVerifyConfig, jwksCfg *JWKSConfig, client *Client) (*auth.TokenVerifier, error) {
	var authClient *auth.Client
	if client != nil {
		authClient = client.Auth()
	}

	var jwksManager *auth.JWKSManager
	if jwksCfg != nil {
		var err error
		jwksOpts := []auth.JWKSManagerOption{
			auth.WithCacheEnabled(true),
		}
		if authClient != nil {
			jwksOpts = append(jwksOpts, auth.WithAuthClient(authClient))
		}

		jwksManager, err = auth.NewJWKSManager(
			&config.JWKSConfig{
				URL:             jwksCfg.URL,
				GRPCEndpoint:    jwksCfg.GRPCEndpoint,
				RefreshInterval: jwksCfg.RefreshInterval,
				RequestTimeout:  jwksCfg.RequestTimeout,
				CacheTTL:        jwksCfg.CacheTTL,
				HTTPClient:      jwksCfg.HTTPClient,
				CustomHeaders:   jwksCfg.CustomHeaders,
				FallbackOnError: jwksCfg.FallbackOnError,
			},
			jwksOpts...,
		)
		if err != nil {
			return nil, err
		}
	}

	return auth.NewTokenVerifier(verifyCfg, jwksManager, authClient)
}

// NewServiceAuthHelper 创建服务间认证助手
func NewServiceAuthHelper(cfg *ServiceAuthConfig, client *Client, opts ...auth.ServiceAuthOption) (*auth.ServiceAuthHelper, error) {
	if client == nil {
		return nil, fmt.Errorf("sdk: client is required for service auth")
	}
	return auth.NewServiceAuthHelper(cfg, client.Auth(), opts...)
}

// NewJWKSManager 创建 JWKS 管理器
func NewJWKSManager(cfg *JWKSConfig, opts ...auth.JWKSManagerOption) (*auth.JWKSManager, error) {
	return auth.NewJWKSManager(cfg, opts...)
}

// NewJWKSManagerWithClient 创建带 gRPC 客户端降级的 JWKS 管理器
func NewJWKSManagerWithClient(cfg *JWKSConfig, client *Client, cbConfig *CircuitBreakerConfig) (*auth.JWKSManager, error) {
	opts := []auth.JWKSManagerOption{
		auth.WithCacheEnabled(true),
	}
	if client != nil {
		opts = append(opts, auth.WithAuthClient(client.Auth()))
	}
	if cbConfig != nil {
		opts = append(opts, auth.WithCircuitBreakerConfig(cbConfig))
	}
	return auth.NewJWKSManager(cfg, opts...)
}
