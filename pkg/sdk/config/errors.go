package config

import "errors"

// 配置相关错误
var (
	ErrEndpointRequired = errors.New("config: endpoint is required")
)
