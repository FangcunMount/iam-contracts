package config

import "time"

// TLSConfig TLS/mTLS 配置。
type TLSConfig struct {
	Enabled bool

	CACert    string
	CACertPEM []byte

	ClientCert    string
	ClientCertPEM []byte
	ClientKey     string
	ClientKeyPEM  []byte

	ServerName         string
	InsecureSkipVerify bool
	MinVersion         uint16
}

// KeepaliveConfig gRPC 连接保活配置。
type KeepaliveConfig struct {
	Time                time.Duration
	Timeout             time.Duration
	PermitWithoutStream bool
}

// RetryConfig 重试配置。
type RetryConfig struct {
	Enabled           bool
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
	RetryableCodes    []string
}
