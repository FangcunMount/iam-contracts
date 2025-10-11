package databases

import (
	"context"
)

// DatabaseType 数据库类型
type DatabaseType string

const (
	MySQL DatabaseType = "mysql"
	Redis DatabaseType = "redis"
)

// DatabaseConnection 数据库连接接口
type DatabaseConnection interface {
	Type() DatabaseType
	Connect() error
	Close() error
	HealthCheck(ctx context.Context) error
	GetClient() interface{}
}
