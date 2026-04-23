package transport

import (
	"time"

	"google.golang.org/grpc/codes"
)

// MethodConfig 方法级别配置。
type MethodConfig struct {
	Timeout time.Duration
	Retry   *MethodRetryConfig
}

// MethodRetryConfig 方法级别重试配置。
type MethodRetryConfig struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
	RetryableCodes    []codes.Code
}

// MethodConfigs 方法配置表。
type MethodConfigs struct {
	Default  *MethodConfig
	Services map[string]*MethodConfig
	Methods  map[string]*MethodConfig
}

// NewMethodConfigs 创建方法配置表。
func NewMethodConfigs() *MethodConfigs {
	return &MethodConfigs{
		Default: &MethodConfig{
			Timeout: 30 * time.Second,
			Retry:   DefaultMethodRetryConfig(),
		},
		Services: make(map[string]*MethodConfig),
		Methods:  make(map[string]*MethodConfig),
	}
}

// GetConfig 获取方法配置。
func (mc *MethodConfigs) GetConfig(fullMethod string) *MethodConfig {
	if cfg, ok := mc.Methods[fullMethod]; ok {
		return cfg
	}

	service := extractServiceName(fullMethod)
	if cfg, ok := mc.Services[service]; ok {
		return cfg
	}

	return mc.Default
}

// SetMethodConfig 设置方法配置。
func (mc *MethodConfigs) SetMethodConfig(method string, cfg *MethodConfig) {
	mc.Methods[method] = cfg
}

// SetServiceConfig 设置服务配置。
func (mc *MethodConfigs) SetServiceConfig(service string, cfg *MethodConfig) {
	mc.Services[service] = cfg
}
