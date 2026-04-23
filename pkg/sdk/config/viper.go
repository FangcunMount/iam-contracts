package config

import "fmt"

type viperGetter interface {
	Get(key string) interface{}
}

// FromViper 从 Viper 风格的 getter 加载配置，默认读取 `iam.*` 前缀。
func FromViper(getter viperGetter) (*Config, error) {
	return FromViperWithPrefix(getter, "iam")
}

// FromViperWithPrefix 从 Viper 风格的 getter 加载带前缀的配置。
func FromViperWithPrefix(getter viperGetter, prefix string) (*Config, error) {
	return NewViperLoader(getter.Get).Load(prefix)
}

// ViperLoader Viper 配置加载器。
type ViperLoader struct {
	getter func(key string) interface{}
}

// NewViperLoader 创建 Viper 配置加载器。
func NewViperLoader(getter func(key string) interface{}) *ViperLoader {
	return &ViperLoader{getter: getter}
}

// Load 从 Viper 加载配置。
func (l *ViperLoader) Load(prefix string) (*Config, error) {
	cfg := &Config{}

	cfg.Endpoint = l.getString(prefix + ".endpoint")
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("%s.endpoint is required", prefix)
	}

	cfg.Timeout = l.getDuration(prefix+".timeout", defaultTimeout)
	cfg.DialTimeout = l.getDuration(prefix+".dial_timeout", defaultDialTimeout)
	cfg.LoadBalancer = l.getStringDefault(prefix+".load_balancer", defaultLoadBalancer)
	cfg.TLS = l.loadTLS(prefix)
	cfg.Retry = l.loadRetry(prefix)
	cfg.Keepalive = l.loadKeepalive(prefix)
	cfg.JWKS = l.loadJWKS(prefix)
	cfg.CircuitBreaker = l.loadCircuitBreaker(prefix)
	cfg.Observability = l.loadObservability(prefix)

	return cfg, nil
}
