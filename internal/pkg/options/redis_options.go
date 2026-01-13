package options

import (
	"github.com/spf13/pflag"
)

// SingleRedisOptions defines options for a single redis instance.
type SingleRedisOptions struct {
	Host                  string   `json:"host,omitempty"                         mapstructure:"host"`
	Port                  int      `json:"port,omitempty"                         mapstructure:"port"`
	Addrs                 []string `json:"addrs,omitempty"                        mapstructure:"addrs"`
	Username              string   `json:"username,omitempty"                     mapstructure:"username"`
	Password              string   `json:"-"                                      mapstructure:"password"`
	Database              int      `json:"database"                               mapstructure:"database"`
	MaxIdle               int      `json:"max-idle,omitempty"                     mapstructure:"max-idle"`
	MaxActive             int      `json:"max-active,omitempty"                   mapstructure:"max-active"`
	Timeout               int      `json:"timeout,omitempty"                      mapstructure:"timeout"`
	EnableCluster         bool     `json:"enable-cluster,omitempty"               mapstructure:"enable-cluster"`
	UseSSL                bool     `json:"use-ssl,omitempty"                      mapstructure:"use-ssl"`
	SSLInsecureSkipVerify bool     `json:"ssl-insecure-skip-verify,omitempty"     mapstructure:"ssl-insecure-skip-verify"`
	EnableLogging         bool     `json:"enable-logging,omitempty"               mapstructure:"enable-logging"`
}

// RedisOptions defines options for a single redis instance (cache).
type RedisOptions struct {
	// Cache Redis - 用于缓存、会话、限流等临时数据
	Cache *SingleRedisOptions `json:"cache" mapstructure:"cache"`
}

// NewRedisOptions create a `zero` value instance.
func NewRedisOptions() *RedisOptions {
	return &RedisOptions{
		Cache: &SingleRedisOptions{
			Host:                  "127.0.0.1",
			Port:                  6379,
			Addrs:                 []string{},
			Username:              "",
			Password:              "",
			Database:              0, // 缓存使用 DB 0
			MaxIdle:               50,
			MaxActive:             100,
			Timeout:               5,
			EnableCluster:         false,
			UseSSL:                false,
			SSLInsecureSkipVerify: false,
			EnableLogging:         false, // 默认不开启 Redis 命令日志
		},
	}
}

// Validate verifies flags passed to RedisOptions.
func (o *RedisOptions) Validate() []error {
	errs := []error{}
	// 可以添加验证逻辑
	return errs
}

// AddFlags adds flags related to redis storage for a specific APIServer to the specified FlagSet.
func (o *RedisOptions) AddFlags(fs *pflag.FlagSet) {
	// Cache Redis flags
	fs.StringVar(&o.Cache.Host, "redis.cache.host", o.Cache.Host, ""+
		"Redis cache service host address.")

	fs.IntVar(&o.Cache.Port, "redis.cache.port", o.Cache.Port, ""+
		"Redis cache service port.")

	fs.StringSliceVar(&o.Cache.Addrs, "redis.cache.addrs", o.Cache.Addrs, ""+
		"Redis cache cluster addresses. If set, host and port will be ignored.")

	fs.StringVar(&o.Cache.Username, "redis.cache.username", o.Cache.Username, ""+
		"Username for Redis 6.0+ ACL authentication (cache).")

	fs.StringVar(&o.Cache.Password, "redis.cache.password", o.Cache.Password, ""+
		"Password for access to redis cache service.")

	fs.IntVar(&o.Cache.Database, "redis.cache.database", o.Cache.Database, ""+
		"Redis cache database number (default: 0).")

	fs.IntVar(&o.Cache.MaxIdle, "redis.cache.max-idle", o.Cache.MaxIdle, ""+
		"Maximum idle connections allowed to connect to redis cache.")

	fs.IntVar(&o.Cache.MaxActive, "redis.cache.max-active", o.Cache.MaxActive, ""+
		"Maximum active connections allowed to connect to redis cache.")

	fs.IntVar(&o.Cache.Timeout, "redis.cache.timeout", o.Cache.Timeout, ""+
		"Redis cache connection timeout in seconds.")

	fs.BoolVar(&o.Cache.EnableCluster, "redis.cache.enable-cluster", o.Cache.EnableCluster, ""+
		"Enable redis cache cluster mode.")

	fs.BoolVar(&o.Cache.UseSSL, "redis.cache.use-ssl", o.Cache.UseSSL, ""+
		"Enable SSL for redis cache connection.")

	fs.BoolVar(&o.Cache.SSLInsecureSkipVerify, "redis.cache.ssl-insecure-skip-verify", o.Cache.SSLInsecureSkipVerify, ""+
		"Skip SSL certificate verification for cache.")
}
