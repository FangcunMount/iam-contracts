// Package mtls 提供 mTLS 双向认证配置和管理功能
package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Config mTLS 配置
type Config struct {
	// 服务端证书配置
	CertFile string `json:"cert_file" mapstructure:"cert-file"` // 服务端证书文件
	KeyFile  string `json:"key_file" mapstructure:"key-file"`   // 服务端私钥文件

	// CA 证书配置
	CAFile string `json:"ca_file" mapstructure:"ca-file"` // 根 CA 证书文件
	CADir  string `json:"ca_dir" mapstructure:"ca-dir"`   // CA 证书目录（支持多级 CA）

	// 客户端证书验证配置
	RequireClientCert bool     `json:"require_client_cert" mapstructure:"require-client-cert"` // 是否要求客户端证书
	AllowedCNs        []string `json:"allowed_cns" mapstructure:"allowed-cns"`                 // 允许的客户端证书 CN 列表
	AllowedOUs        []string `json:"allowed_ous" mapstructure:"allowed-ous"`                 // 允许的客户端证书 OU 列表
	AllowedDNSSANs    []string `json:"allowed_dns_sans" mapstructure:"allowed-dns-sans"`       // 允许的 DNS SAN 列表

	// TLS 版本控制
	MinVersion uint16 `json:"min_version" mapstructure:"min-version"` // 最低 TLS 版本 (默认 TLS 1.2)
	MaxVersion uint16 `json:"max_version" mapstructure:"max-version"` // 最高 TLS 版本 (默认 TLS 1.3)

	// 证书轮换配置
	EnableAutoReload    bool          `json:"enable_auto_reload" mapstructure:"enable-auto-reload"`       // 启用证书自动重载
	ReloadInterval      time.Duration `json:"reload_interval" mapstructure:"reload-interval"`             // 证书重载检查间隔
	CertExpiryThreshold time.Duration `json:"cert_expiry_threshold" mapstructure:"cert-expiry-threshold"` // 证书过期预警阈值
}

// DefaultConfig 返回默认的 mTLS 配置
func DefaultConfig() *Config {
	return &Config{
		RequireClientCert:   true,
		MinVersion:          tls.VersionTLS12,
		MaxVersion:          tls.VersionTLS13,
		EnableAutoReload:    true,
		ReloadInterval:      5 * time.Minute,
		CertExpiryThreshold: 7 * 24 * time.Hour, // 7天预警
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.CertFile == "" {
		return fmt.Errorf("cert_file is required")
	}
	if c.KeyFile == "" {
		return fmt.Errorf("key_file is required")
	}
	if c.RequireClientCert && c.CAFile == "" && c.CADir == "" {
		return fmt.Errorf("ca_file or ca_dir is required when require_client_cert is true")
	}
	return nil
}

// TLSConfigBuilder mTLS 配置构建器
type TLSConfigBuilder struct {
	config    *Config
	mu        sync.RWMutex
	tlsConfig *tls.Config
	stopCh    chan struct{}
}

// NewTLSConfigBuilder 创建 TLS 配置构建器
func NewTLSConfigBuilder(cfg *Config) (*TLSConfigBuilder, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid mtls config: %w", err)
	}

	builder := &TLSConfigBuilder{
		config: cfg,
		stopCh: make(chan struct{}),
	}

	// 初始化 TLS 配置
	if err := builder.buildTLSConfig(); err != nil {
		return nil, err
	}

	return builder, nil
}

// GetTLSConfig 获取当前 TLS 配置
func (b *TLSConfigBuilder) GetTLSConfig() *tls.Config {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.tlsConfig
}

// buildTLSConfig 构建 TLS 配置
func (b *TLSConfigBuilder) buildTLSConfig() error {
	// 加载服务端证书
	cert, err := tls.LoadX509KeyPair(b.config.CertFile, b.config.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load server certificate: %w", err)
	}

	// 加载 CA 证书池
	caPool, err := b.loadCACertPool()
	if err != nil {
		return fmt.Errorf("failed to load CA certificates: %w", err)
	}

	// 设置客户端认证模式
	clientAuth := tls.NoClientCert
	if b.config.RequireClientCert {
		clientAuth = tls.RequireAndVerifyClientCert
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		ClientAuth:   clientAuth,
		MinVersion:   b.config.MinVersion,
		MaxVersion:   b.config.MaxVersion,

		// 安全选项
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},

		// 自定义证书验证
		VerifyPeerCertificate: b.verifyPeerCertificate,
	}

	b.mu.Lock()
	b.tlsConfig = tlsConfig
	b.mu.Unlock()

	return nil
}

// loadCACertPool 加载 CA 证书池
func (b *TLSConfigBuilder) loadCACertPool() (*x509.CertPool, error) {
	caPool := x509.NewCertPool()

	// 从文件加载
	if b.config.CAFile != "" {
		caCert, err := os.ReadFile(b.config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
	}

	// 从目录加载
	if b.config.CADir != "" {
		entries, err := os.ReadDir(b.config.CADir)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || (filepath.Ext(entry.Name()) != ".pem" && filepath.Ext(entry.Name()) != ".crt") {
				continue
			}

			caPath := filepath.Join(b.config.CADir, entry.Name())
			caCert, err := os.ReadFile(caPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA file %s: %w", caPath, err)
			}
			caPool.AppendCertsFromPEM(caCert)
		}
	}

	return caPool, nil
}

// verifyPeerCertificate 自定义客户端证书验证
func (b *TLSConfigBuilder) verifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(verifiedChains) == 0 || len(verifiedChains[0]) == 0 {
		return fmt.Errorf("no verified certificate chain")
	}

	// 获取客户端证书
	clientCert := verifiedChains[0][0]

	// 验证 CN
	if len(b.config.AllowedCNs) > 0 {
		cnAllowed := false
		for _, cn := range b.config.AllowedCNs {
			if clientCert.Subject.CommonName == cn {
				cnAllowed = true
				break
			}
		}
		if !cnAllowed {
			return fmt.Errorf("client certificate CN %q not in allowed list", clientCert.Subject.CommonName)
		}
	}

	// 验证 OU
	if len(b.config.AllowedOUs) > 0 {
		ouAllowed := false
		for _, clientOU := range clientCert.Subject.OrganizationalUnit {
			for _, allowedOU := range b.config.AllowedOUs {
				if clientOU == allowedOU {
					ouAllowed = true
					break
				}
			}
		}
		if !ouAllowed {
			return fmt.Errorf("client certificate OU not in allowed list")
		}
	}

	// 验证 DNS SAN
	if len(b.config.AllowedDNSSANs) > 0 {
		sanAllowed := false
		for _, clientSAN := range clientCert.DNSNames {
			for _, allowedSAN := range b.config.AllowedDNSSANs {
				if MatchDNSSAN(clientSAN, allowedSAN) {
					sanAllowed = true
					break
				}
			}
		}
		if !sanAllowed {
			return fmt.Errorf("client certificate DNS SAN not in allowed list")
		}
	}

	return nil
}

// MatchDNSSAN 匹配 DNS SAN（支持通配符）
func MatchDNSSAN(actual, pattern string) bool {
	if pattern == actual {
		return true
	}
	// 支持简单通配符匹配 *.example.com
	if len(pattern) > 2 && pattern[:2] == "*." {
		suffix := pattern[1:] // .example.com
		if len(actual) > len(suffix) {
			return actual[len(actual)-len(suffix):] == suffix
		}
	}
	return false
}

// StartAutoReload 启动证书自动重载
func (b *TLSConfigBuilder) StartAutoReload() {
	if !b.config.EnableAutoReload {
		return
	}

	go func() {
		ticker := time.NewTicker(b.config.ReloadInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := b.buildTLSConfig(); err != nil {
					// 记录错误，但不影响现有配置
					fmt.Printf("failed to reload TLS config: %v\n", err)
				}
			case <-b.stopCh:
				return
			}
		}
	}()
}

// Stop 停止自动重载
func (b *TLSConfigBuilder) Stop() {
	close(b.stopCh)
}

// CheckCertExpiry 检查证书过期时间
func (b *TLSConfigBuilder) CheckCertExpiry() (time.Duration, error) {
	certPEM, err := os.ReadFile(b.config.CertFile)
	if err != nil {
		return 0, err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return 0, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return 0, err
	}

	remaining := time.Until(cert.NotAfter)
	return remaining, nil
}
