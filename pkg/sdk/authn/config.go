package authnsdk

import "time"

// Config 认证 SDK 配置
// 控制 SDK 如何连接到 IAM 服务以及如何验证 JWT
type Config struct {
	// GRPCEndpoint gRPC 服务地址
	// 格式："host:port"，例如 "localhost:8081"
	GRPCEndpoint string

	// JWKSURL JWKS 端点 URL
	// 指向 IAM 的 /.well-known/jwks.json 端点
	// 用于获取 JWT 签名验证的公钥
	JWKSURL string

	// JWKSRefreshInterval JWKS 缓存主动刷新间隔
	// 控制多久主动刷新一次 JWKS 缓存
	// 默认值：5 分钟
	JWKSRefreshInterval time.Duration

	// JWKSRequestTimeout JWKS HTTP 请求超时时间
	// 控制通过 HTTP 获取 JWKS 时的超时时间
	// 默认值：3 秒
	JWKSRequestTimeout time.Duration

	// JWKSCacheTTL JWKS 缓存 TTL
	// 当服务器未提供 Cache-Control 头时的回退最大 TTL
	// 默认值：10 分钟
	JWKSCacheTTL time.Duration

	// AllowedAudience 允许的 JWT 受众列表（可选）
	// 如果设置，JWT 的 aud 声明必须匹配其中之一
	AllowedAudience []string

	// AllowedIssuer 允许的 JWT 签发者（可选）
	// 如果设置，JWT 的 iss 声明必须匹配此值
	AllowedIssuer string

	// ClockSkew 时钟偏差容忍度
	// 检查 exp/nbf 时允许的时间差
	// 默认值：60 秒
	ClockSkew time.Duration

	// ForceRemoteVerification 强制远程验证
	// 如果为 true，即使本地验证成功也会调用 IAM 的 VerifyToken RPC
	ForceRemoteVerification bool
}

// setDefaults 设置配置的默认值
// 对于未设置或设置为 0 的配置项，使用合理的默认值
func (c *Config) setDefaults() {
	// JWKS 刷新间隔默认 5 分钟
	if c.JWKSRefreshInterval <= 0 {
		c.JWKSRefreshInterval = 5 * time.Minute
	}
	// JWKS 请求超时默认 3 秒
	if c.JWKSRequestTimeout <= 0 {
		c.JWKSRequestTimeout = 3 * time.Second
	}
	// JWKS 缓存 TTL 默认 10 分钟
	if c.JWKSCacheTTL <= 0 {
		c.JWKSCacheTTL = 10 * time.Minute
	}
	// 时钟偏差默认 1 分钟
	if c.ClockSkew <= 0 {
		c.ClockSkew = time.Minute
	}
}
