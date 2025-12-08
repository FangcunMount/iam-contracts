package config

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ========== 从环境变量加载配置 ==========

// FromEnv 从环境变量加载配置
//
// 支持的环境变量：
//
//	IAM_ENDPOINT          - gRPC 服务地址（必填）
//	IAM_TIMEOUT           - 请求超时时间（默认：30s）
//	IAM_DIAL_TIMEOUT      - 连接超时时间（默认：10s）
//
//	IAM_TLS_ENABLED       - 是否启用 TLS（默认：true）
//	IAM_TLS_CA_CERT       - CA 证书文件路径
//	IAM_TLS_CLIENT_CERT   - 客户端证书文件路径
//	IAM_TLS_CLIENT_KEY    - 客户端私钥文件路径
//	IAM_TLS_SERVER_NAME   - 服务端名称（SNI）
//	IAM_TLS_SKIP_VERIFY   - 跳过证书验证（默认：false）
//
//	IAM_RETRY_ENABLED     - 是否启用重试（默认：true）
//	IAM_RETRY_MAX_ATTEMPTS - 最大重试次数（默认：3）
//
//	IAM_JWKS_URL          - JWKS 端点 URL
//	IAM_JWKS_REFRESH_INTERVAL - JWKS 刷新间隔（默认：5m）
func FromEnv() (*Config, error) {
	return FromEnvWithPrefix("IAM")
}

// FromEnvWithPrefix 从带前缀的环境变量加载配置
func FromEnvWithPrefix(prefix string) (*Config, error) {
	cfg := &Config{}

	// 基础配置
	cfg.Endpoint = getEnv(prefix, "ENDPOINT", "")
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("%s_ENDPOINT is required", prefix)
	}

	cfg.Timeout = getEnvDuration(prefix, "TIMEOUT", 30*time.Second)
	cfg.DialTimeout = getEnvDuration(prefix, "DIAL_TIMEOUT", 10*time.Second)
	cfg.LoadBalancer = getEnv(prefix, "LOAD_BALANCER", "round_robin")

	// TLS 配置
	tlsEnabled := getEnvBool(prefix, "TLS_ENABLED", true)
	if tlsEnabled {
		cfg.TLS = &TLSConfig{
			Enabled:            true,
			CACert:             getEnv(prefix, "TLS_CA_CERT", ""),
			ClientCert:         getEnv(prefix, "TLS_CLIENT_CERT", ""),
			ClientKey:          getEnv(prefix, "TLS_CLIENT_KEY", ""),
			ServerName:         getEnv(prefix, "TLS_SERVER_NAME", ""),
			InsecureSkipVerify: getEnvBool(prefix, "TLS_SKIP_VERIFY", false),
			MinVersion:         tls.VersionTLS12,
		}
	}

	// 重试配置
	retryEnabled := getEnvBool(prefix, "RETRY_ENABLED", true)
	if retryEnabled {
		cfg.Retry = &RetryConfig{
			Enabled:           true,
			MaxAttempts:       getEnvInt(prefix, "RETRY_MAX_ATTEMPTS", 3),
			InitialBackoff:    getEnvDuration(prefix, "RETRY_INITIAL_BACKOFF", 100*time.Millisecond),
			MaxBackoff:        getEnvDuration(prefix, "RETRY_MAX_BACKOFF", 10*time.Second),
			BackoffMultiplier: 2.0,
		}
	}

	// JWKS 配置
	jwksURL := getEnv(prefix, "JWKS_URL", "")
	if jwksURL != "" {
		cfg.JWKS = &JWKSConfig{
			URL:             jwksURL,
			RefreshInterval: getEnvDuration(prefix, "JWKS_REFRESH_INTERVAL", 5*time.Minute),
			RequestTimeout:  getEnvDuration(prefix, "JWKS_REQUEST_TIMEOUT", 5*time.Second),
		}
	}

	return cfg, nil
}

// ========== Viper 集成 ==========

// ViperLoader Viper 配置加载器
type ViperLoader struct {
	getter func(key string) interface{}
}

// NewViperLoader 创建 Viper 配置加载器
//
// 示例：
//
//	v := viper.New()
//	v.SetConfigFile("config.yaml")
//	v.ReadInConfig()
//
//	loader := config.NewViperLoader(v.Get)
//	cfg, err := loader.Load("iam")
func NewViperLoader(getter func(key string) interface{}) *ViperLoader {
	return &ViperLoader{getter: getter}
}

// Load 从 Viper 加载配置
func (l *ViperLoader) Load(prefix string) (*Config, error) {
	cfg := &Config{}

	// 基础配置
	cfg.Endpoint = l.getString(prefix + ".endpoint")
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("%s.endpoint is required", prefix)
	}

	cfg.Timeout = l.getDuration(prefix+".timeout", 30*time.Second)
	cfg.DialTimeout = l.getDuration(prefix+".dial_timeout", 10*time.Second)
	cfg.LoadBalancer = l.getStringDefault(prefix+".load_balancer", "round_robin")

	// TLS 配置
	if l.getBool(prefix+".tls.enabled", true) {
		cfg.TLS = &TLSConfig{
			Enabled:            true,
			CACert:             l.getString(prefix + ".tls.ca_cert"),
			ClientCert:         l.getString(prefix + ".tls.client_cert"),
			ClientKey:          l.getString(prefix + ".tls.client_key"),
			ServerName:         l.getString(prefix + ".tls.server_name"),
			InsecureSkipVerify: l.getBool(prefix+".tls.insecure_skip_verify", false),
			MinVersion:         tls.VersionTLS12,
		}
	}

	// 重试配置
	if l.getBool(prefix+".retry.enabled", true) {
		cfg.Retry = &RetryConfig{
			Enabled:           true,
			MaxAttempts:       l.getInt(prefix+".retry.max_attempts", 3),
			InitialBackoff:    l.getDuration(prefix+".retry.initial_backoff", 100*time.Millisecond),
			MaxBackoff:        l.getDuration(prefix+".retry.max_backoff", 10*time.Second),
			BackoffMultiplier: 2.0,
		}
	}

	// JWKS 配置
	jwksURL := l.getString(prefix + ".jwks.url")
	if jwksURL != "" {
		cfg.JWKS = &JWKSConfig{
			URL:             jwksURL,
			RefreshInterval: l.getDuration(prefix+".jwks.refresh_interval", 5*time.Minute),
			RequestTimeout:  l.getDuration(prefix+".jwks.request_timeout", 5*time.Second),
		}
	}

	return cfg, nil
}

func (l *ViperLoader) getString(key string) string {
	v := l.getter(key)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func (l *ViperLoader) getStringDefault(key, def string) string {
	s := l.getString(key)
	if s == "" {
		return def
	}
	return s
}

func (l *ViperLoader) getBool(key string, def bool) bool {
	v := l.getter(key)
	if v == nil {
		return def
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}

func (l *ViperLoader) getInt(key string, def int) int {
	v := l.getter(key)
	if v == nil {
		return def
	}
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	}
	return def
}

func (l *ViperLoader) getDuration(key string, def time.Duration) time.Duration {
	v := l.getter(key)
	if v == nil {
		return def
	}
	switch t := v.(type) {
	case time.Duration:
		return t
	case string:
		if d, err := time.ParseDuration(t); err == nil {
			return d
		}
	case int64:
		return time.Duration(t)
	}
	return def
}

// ========== 环境变量工具函数 ==========

func getEnv(prefix, key, def string) string {
	fullKey := prefix + "_" + key
	if v := os.Getenv(fullKey); v != "" {
		return v
	}
	return def
}

func getEnvBool(prefix, key string, def bool) bool {
	fullKey := prefix + "_" + key
	v := os.Getenv(fullKey)
	if v == "" {
		return def
	}
	v = strings.ToLower(v)
	return v == "true" || v == "1" || v == "yes"
}

func getEnvInt(prefix, key string, def int) int {
	fullKey := prefix + "_" + key
	v := os.Getenv(fullKey)
	if v == "" {
		return def
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}
	return def
}

func getEnvDuration(prefix, key string, def time.Duration) time.Duration {
	fullKey := prefix + "_" + key
	v := os.Getenv(fullKey)
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	return def
}

// ========== 配置示例生成 ==========

// GenerateEnvExample 生成环境变量配置示例
func GenerateEnvExample(prefix string) string {
	return fmt.Sprintf(`# IAM SDK 环境变量配置示例
# 基础配置
%s_ENDPOINT=iam.example.com:8081
%s_TIMEOUT=30s
%s_DIAL_TIMEOUT=10s

# TLS 配置
%s_TLS_ENABLED=true
%s_TLS_CA_CERT=/etc/iam/certs/ca.crt
%s_TLS_CLIENT_CERT=/etc/iam/certs/client.crt
%s_TLS_CLIENT_KEY=/etc/iam/certs/client.key
%s_TLS_SERVER_NAME=iam.example.com
%s_TLS_SKIP_VERIFY=false

# 重试配置
%s_RETRY_ENABLED=true
%s_RETRY_MAX_ATTEMPTS=3
%s_RETRY_INITIAL_BACKOFF=100ms
%s_RETRY_MAX_BACKOFF=10s

# JWKS 配置
%s_JWKS_URL=https://iam.example.com/.well-known/jwks.json
%s_JWKS_REFRESH_INTERVAL=5m
`, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix)
}

// GenerateYAMLExample 生成 YAML 配置示例
func GenerateYAMLExample() string {
	return `# IAM SDK YAML 配置示例
iam:
  endpoint: "iam.example.com:8081"
  timeout: 30s
  dial_timeout: 10s
  load_balancer: round_robin

  tls:
    enabled: true
    ca_cert: "/etc/iam/certs/ca.crt"
    client_cert: "/etc/iam/certs/client.crt"
    client_key: "/etc/iam/certs/client.key"
    server_name: "iam.example.com"
    insecure_skip_verify: false

  retry:
    enabled: true
    max_attempts: 3
    initial_backoff: 100ms
    max_backoff: 10s

  jwks:
    url: "https://iam.example.com/.well-known/jwks.json"
    refresh_interval: 5m
    request_timeout: 5s
`
}
