package grpc

import (
	"time"
)

// Config GRPC 服务器配置
type Config struct {
	BindAddress           string
	BindPort              int
	HealthzPort           int
	MaxMsgSize            int
	MaxConnectionAge      time.Duration
	MaxConnectionAgeGrace time.Duration
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration

	// TLS 配置（单向 TLS）
	TLSCertFile string
	TLSKeyFile  string

	// mTLS 配置（双向认证）
	MTLS MTLSConfig

	// 应用层认证配置
	Auth AuthConfig

	// ACL 权限控制配置
	ACL ACLConfig

	// 审计日志配置
	Audit AuditConfig

	EnableReflection  bool
	EnableHealthCheck bool
	Insecure          bool // 是否使用不安全连接
}

// MTLSConfig mTLS 双向认证配置
type MTLSConfig struct {
	Enabled           bool          // 是否启用 mTLS
	CAFile            string        // CA 证书文件（用于验证客户端证书）
	CADir             string        // CA 证书目录（支持多级 CA）
	RequireClientCert bool          // 是否强制要求客户端证书
	AllowedCNs        []string      // 允许的客户端证书 CN 列表
	AllowedOUs        []string      // 允许的客户端证书 OU 列表
	AllowedSANs       []string      // 允许的客户端证书 DNS SAN 列表
	MinTLSVersion     string        // 最低 TLS 版本 (1.0, 1.1, 1.2, 1.3)，默认 1.2
	EnableAutoReload  bool          // 启用证书自动重载
	ReloadInterval    time.Duration // 证书重载检查间隔
}

// AuthConfig 应用层认证配置
type AuthConfig struct {
	Enabled               bool          // 是否启用应用层认证
	EnableBearer          bool          // 启用 Bearer Token 认证
	EnableHMAC            bool          // 启用 HMAC 签名认证
	EnableAPIKey          bool          // 启用 API Key 认证
	HMACTimestampValidity time.Duration // HMAC 时间戳有效期
	RequireIdentityMatch  bool          // 是否要求 mTLS 身份与凭证身份一致
}

// ACLConfig ACL 权限控制配置
type ACLConfig struct {
	Enabled       bool   // 是否启用 ACL
	ConfigFile    string // ACL 配置文件路径
	DefaultPolicy string // 默认策略：deny 或 allow
}

// AuditConfig 审计日志配置
type AuditConfig struct {
	Enabled bool // 是否启用审计日志
}

// NewConfig 创建默认的 GRPC 服务器配置
func NewConfig() *Config {
	return &Config{
		BindAddress:           "0.0.0.0",
		BindPort:              9090,
		HealthzPort:           9091,
		MaxMsgSize:            4 * 1024 * 1024,  // 4MB
		MaxConnectionAge:      2 * time.Hour,    // 连接最大存活时间
		MaxConnectionAgeGrace: 10 * time.Second, // 连接优雅终止等待时间
		ReadTimeout:           5 * time.Second,  // 读取超时时间
		WriteTimeout:          5 * time.Second,  // 写入超时时间
		EnableReflection:      true,             // 启用反射
		EnableHealthCheck:     true,             // 启用健康检查
		Insecure:              true,             // 默认使用不安全连接
		MTLS: MTLSConfig{
			Enabled:           false,
			RequireClientCert: true,
			MinTLSVersion:     "1.2",
			EnableAutoReload:  true,
			ReloadInterval:    5 * time.Minute,
		},
		Auth: AuthConfig{
			Enabled:               false,
			EnableBearer:          true,
			EnableHMAC:            true,
			EnableAPIKey:          true,
			HMACTimestampValidity: 5 * time.Minute,
			RequireIdentityMatch:  true,
		},
		ACL: ACLConfig{
			Enabled:       false,
			DefaultPolicy: "deny",
		},
		Audit: AuditConfig{
			Enabled: true,
		},
	}
}

// CompletedConfig GRPC 服务器的完成配置
type CompletedConfig struct {
	*Config
}

// Complete 填充任何未设置的字段，这些字段是必需的，并且可以从其他字段派生出来
func (c *Config) Complete() CompletedConfig {
	// 设置默认值
	if c.BindAddress == "" {
		c.BindAddress = "0.0.0.0"
	}
	if c.BindPort == 0 {
		c.BindPort = 8090
	}
	if c.HealthzPort == 0 {
		c.HealthzPort = 9091
	}
	if c.MaxMsgSize == 0 {
		c.MaxMsgSize = 4 * 1024 * 1024
	}
	if c.MaxConnectionAge == 0 {
		c.MaxConnectionAge = 2 * time.Hour
	}
	if c.MaxConnectionAgeGrace == 0 {
		c.MaxConnectionAgeGrace = 10 * time.Second
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = 5 * time.Second
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = 5 * time.Second
	}

	return CompletedConfig{c}
}

// New 从给定的配置创建一个新的 GRPC 服务器实例
func (c CompletedConfig) New() (*Server, error) {
	return NewServer(c.Config)
}
