package options

import (
	"github.com/spf13/pflag"
)

// SingleRedisOptions defines options for a single redis instance.
type SingleRedisOptions struct {
	Host                  string   `json:"host,omitempty"                         mapstructure:"host"`
	Port                  int      `json:"port,omitempty"                         mapstructure:"port"`
	Addrs                 []string `json:"addrs,omitempty"                        mapstructure:"addrs"`
	Password              string   `json:"-"                                      mapstructure:"password"`
	Database              int      `json:"database"                               mapstructure:"database"`
	MaxIdle               int      `json:"max-idle,omitempty"                     mapstructure:"max-idle"`
	MaxActive             int      `json:"max-active,omitempty"                   mapstructure:"max-active"`
	Timeout               int      `json:"timeout,omitempty"                      mapstructure:"timeout"`
	EnableCluster         bool     `json:"enable-cluster,omitempty"               mapstructure:"enable-cluster"`
	UseSSL                bool     `json:"use-ssl,omitempty"                      mapstructure:"use-ssl"`
	SSLInsecureSkipVerify bool     `json:"ssl-insecure-skip-verify,omitempty"     mapstructure:"ssl-insecure-skip-verify"`
}

// RedisOptions defines options for dual redis instances (cache + store).
type RedisOptions struct {
	// Cache Redis - 用于缓存、会话、限流等临时数据
	Cache *SingleRedisOptions `json:"cache" mapstructure:"cache"`
	// Store Redis - 用于持久化存储、队列、发布订阅等
	Store *SingleRedisOptions `json:"store" mapstructure:"store"`
}

// NewRedisOptions create a `zero` value instance.
func NewRedisOptions() *RedisOptions {
	return &RedisOptions{
		Cache: &SingleRedisOptions{
			Host:                  "127.0.0.1",
			Port:                  6379,
			Addrs:                 []string{},
			Password:              "",
			Database:              0, // 缓存使用 DB 0
			MaxIdle:               50,
			MaxActive:             100,
			Timeout:               5,
			EnableCluster:         false,
			UseSSL:                false,
			SSLInsecureSkipVerify: false,
		},
		Store: &SingleRedisOptions{
			Host:                  "127.0.0.1",
			Port:                  6379,
			Addrs:                 []string{},
			Password:              "",
			Database:              1, // 存储使用 DB 1
			MaxIdle:               50,
			MaxActive:             100,
			Timeout:               5,
			EnableCluster:         false,
			UseSSL:                false,
			SSLInsecureSkipVerify: false,
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

	// Store Redis flags
	fs.StringVar(&o.Store.Host, "redis.store.host", o.Store.Host, ""+
		"Redis store service host address.")

	fs.IntVar(&o.Store.Port, "redis.store.port", o.Store.Port, ""+
		"Redis store service port.")

	fs.StringSliceVar(&o.Store.Addrs, "redis.store.addrs", o.Store.Addrs, ""+
		"Redis store cluster addresses. If set, host and port will be ignored.")

	fs.StringVar(&o.Store.Password, "redis.store.password", o.Store.Password, ""+
		"Password for access to redis store service.")

	fs.IntVar(&o.Store.Database, "redis.store.database", o.Store.Database, ""+
		"Redis store database number (default: 1).")

	fs.IntVar(&o.Store.MaxIdle, "redis.store.max-idle", o.Store.MaxIdle, ""+
		"Maximum idle connections allowed to connect to redis store.")

	fs.IntVar(&o.Store.MaxActive, "redis.store.max-active", o.Store.MaxActive, ""+
		"Maximum active connections allowed to connect to redis store.")

	fs.IntVar(&o.Store.Timeout, "redis.store.timeout", o.Store.Timeout, ""+
		"Redis store connection timeout in seconds.")

	fs.BoolVar(&o.Store.EnableCluster, "redis.store.enable-cluster", o.Store.EnableCluster, ""+
		"Enable redis store cluster mode.")

	fs.BoolVar(&o.Store.UseSSL, "redis.store.use-ssl", o.Store.UseSSL, ""+
		"Enable SSL for redis store connection.")

	fs.BoolVar(&o.Store.SSLInsecureSkipVerify, "redis.store.ssl-insecure-skip-verify", o.Store.SSLInsecureSkipVerify, ""+
		"Skip SSL certificate verification for store.")
}
