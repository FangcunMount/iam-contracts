package database

import (
	"context"
	"fmt"
	"sync"

	"github.com/fangcun-mount/iam-contracts/pkg/database/databases"
)

// Registry 数据库注册器
type Registry struct {
	connections map[databases.DatabaseType]databases.DatabaseConnection
	configs     map[databases.DatabaseType]interface{}
	mutex       sync.RWMutex
}

// NewRegistry 创建数据库注册器
func NewRegistry() *Registry {
	return &Registry{
		connections: make(map[databases.DatabaseType]databases.DatabaseConnection),
		configs:     make(map[databases.DatabaseType]interface{}),
	}
}

// Register 注册数据库连接
func (r *Registry) Register(dbType databases.DatabaseType, config interface{}, connection databases.DatabaseConnection) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.connections[dbType]; exists {
		return fmt.Errorf("database connection for type %s already registered", dbType)
	}

	r.connections[dbType] = connection
	r.configs[dbType] = config

	return nil
}

// GetConnection 获取数据库连接
func (r *Registry) GetConnection(dbType databases.DatabaseType) (databases.DatabaseConnection, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	connection, exists := r.connections[dbType]
	if !exists {
		return nil, fmt.Errorf("database connection for type %s not found", dbType)
	}

	return connection, nil
}

// GetClient 获取数据库客户端
func (r *Registry) GetClient(dbType databases.DatabaseType) (interface{}, error) {
	connection, err := r.GetConnection(dbType)
	if err != nil {
		return nil, err
	}

	return connection.GetClient(), nil
}

// Init 初始化所有数据库连接
func (r *Registry) Init() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for dbType, connection := range r.connections {
		if err := connection.Connect(); err != nil {
			return fmt.Errorf("failed to connect to %s: %w", dbType, err)
		}
	}

	return nil
}

// Close 关闭所有数据库连接
func (r *Registry) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var lastErr error
	for dbType, connection := range r.connections {
		if err := connection.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close %s connection: %w", dbType, err)
		}
	}

	return lastErr
}

// HealthCheck 健康检查
func (r *Registry) HealthCheck(ctx context.Context) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for dbType, connection := range r.connections {
		if err := connection.HealthCheck(ctx); err != nil {
			return fmt.Errorf("health check failed for %s: %w", dbType, err)
		}
	}

	return nil
}
