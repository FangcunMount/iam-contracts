// Package mtls 提供 gRPC 服务端和客户端的 mTLS 凭证配置
package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// ServerCredentials 服务端 mTLS 凭证
type ServerCredentials struct {
	config  *Config
	builder *TLSConfigBuilder
}

// NewServerCredentials 创建服务端 mTLS 凭证
func NewServerCredentials(cfg *Config) (*ServerCredentials, error) {
	builder, err := NewTLSConfigBuilder(cfg)
	if err != nil {
		return nil, err
	}

	return &ServerCredentials{
		config:  cfg,
		builder: builder,
	}, nil
}

// TransportCredentials 返回 gRPC 传输凭证
func (s *ServerCredentials) TransportCredentials() credentials.TransportCredentials {
	return credentials.NewTLS(s.builder.GetTLSConfig())
}

// GRPCServerOption 返回 gRPC 服务器选项
func (s *ServerCredentials) GRPCServerOption() grpc.ServerOption {
	return grpc.Creds(s.TransportCredentials())
}

// StartAutoReload 启动证书自动重载
func (s *ServerCredentials) StartAutoReload() {
	s.builder.StartAutoReload()
}

// Stop 停止自动重载
func (s *ServerCredentials) Stop() {
	s.builder.Stop()
}

// ClientCredentialsConfig 客户端 mTLS 凭证配置
type ClientCredentialsConfig struct {
	// 客户端证书
	CertFile string `json:"cert_file" mapstructure:"cert-file"` // 客户端证书文件
	KeyFile  string `json:"key_file" mapstructure:"key-file"`   // 客户端私钥文件

	// CA 证书（用于验证服务端）
	CAFile string `json:"ca_file" mapstructure:"ca-file"` // Root CA 证书文件

	// 服务端名称（用于 SNI）
	ServerName string `json:"server_name" mapstructure:"server-name"` // 服务端主机名

	// TLS 配置
	InsecureSkipVerify bool `json:"insecure_skip_verify" mapstructure:"insecure-skip-verify"` // 跳过服务端证书验证（仅用于测试）
}

// Validate 验证客户端凭证配置
func (c *ClientCredentialsConfig) Validate() error {
	if c.CertFile == "" {
		return fmt.Errorf("cert_file is required for mTLS client")
	}
	if c.KeyFile == "" {
		return fmt.Errorf("key_file is required for mTLS client")
	}
	if c.CAFile == "" && !c.InsecureSkipVerify {
		return fmt.Errorf("ca_file is required when insecure_skip_verify is false")
	}
	return nil
}

// ClientCredentials 客户端 mTLS 凭证
type ClientCredentials struct {
	config    *ClientCredentialsConfig
	tlsConfig *tls.Config
}

// NewClientCredentials 创建客户端 mTLS 凭证
func NewClientCredentials(cfg *ClientCredentialsConfig) (*ClientCredentials, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid client credentials config: %w", err)
	}

	// 加载客户端证书
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// 加载 CA 证书
	var caPool *x509.CertPool
	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		caPool = x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caPool,
		ServerName:         cfg.ServerName,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	return &ClientCredentials{
		config:    cfg,
		tlsConfig: tlsConfig,
	}, nil
}

// TransportCredentials 返回 gRPC 传输凭证
func (c *ClientCredentials) TransportCredentials() credentials.TransportCredentials {
	return credentials.NewTLS(c.tlsConfig)
}

// GRPCDialOption 返回 gRPC Dial 选项
func (c *ClientCredentials) GRPCDialOption() grpc.DialOption {
	return grpc.WithTransportCredentials(c.TransportCredentials())
}

// NewMutualTLSConfig 快速创建双向 TLS 配置（用于服务端）
func NewMutualTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	// 加载服务端证书
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// 加载 CA 证书
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
	}, nil
}

// NewClientTLSConfig 快速创建客户端 TLS 配置
func NewClientTLSConfig(certFile, keyFile, caFile, serverName string) (*tls.Config, error) {
	// 加载客户端证书
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// 加载 CA 证书
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
