# 数据库注册器设计

## 🎯 设计理念

数据库注册器（Registry）是框架的核心组件，负责统一管理多种数据库连接。它提供了抽象的数据库接口，支持MySQL、Redis等不同类型的数据库，实现了数据库连接的注册、初始化、健康检查和资源管理。

## 🏗️ 架构设计

```text
┌─────────────────────────────────────────────────────────────┐
│                    Database Registry                        │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Connections Map                          │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │   MySQL     │  │   Redis     │  │   Other     │    │ │
│  │  │ Connection  │  │ Connection  │  │ Connection  │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Configs Map                              │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │ MySQL Config│  │ Redis Config│  │ Other Config│    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 📦 核心接口

### 数据库类型定义

```go
// DatabaseType 数据库类型
type DatabaseType string

const (
    MySQL DatabaseType = "mysql"
    Redis DatabaseType = "redis"
)
```

### 数据库连接接口

```go
// DatabaseConnection 数据库连接接口
type DatabaseConnection interface {
    Type() DatabaseType
    Connect() error
    Close() error
    HealthCheck(ctx context.Context) error
    GetClient() interface{}
}
```

### 注册器结构

```go
// Registry 数据库注册器
type Registry struct {
    connections map[databases.DatabaseType]databases.DatabaseConnection
    configs     map[databases.DatabaseType]interface{}
    mutex       sync.RWMutex
}
```

## 🔧 实现细节

### 注册器创建

```go
// NewRegistry 创建数据库注册器
func NewRegistry() *Registry {
    return &Registry{
        connections: make(map[databases.DatabaseType]databases.DatabaseConnection),
        configs:     make(map[databases.DatabaseType]interface{}),
    }
}
```

### 连接注册

```go
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
```

### 连接获取

```go
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
```

## 🔄 生命周期管理

### 初始化

```go
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
```

### 健康检查

```go
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
```

### 资源清理

```go
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
```

## 🗄️ 数据库适配器

### MySQL适配器

```go
// MySQLConnection MySQL 连接实现
type MySQLConnection struct {
    config *MySQLConfig
    client *gorm.DB
}

// NewMySQLConnection 创建 MySQL 连接
func NewMySQLConnection(config *MySQLConfig) *MySQLConnection {
    return &MySQLConnection{
        config: config,
    }
}

// Connect 连接 MySQL 数据库
func (m *MySQLConnection) Connect() error {
    dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        m.config.Username, m.config.Password, m.config.Host, m.config.Database)

    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.LogLevel(m.config.LogLevel)),
    })
    if err != nil {
        return fmt.Errorf("failed to connect to MySQL: %w", err)
    }

    sqlDB, err := db.DB()
    if err != nil {
        return fmt.Errorf("failed to get sql.DB: %w", err)
    }

    sqlDB.SetMaxIdleConns(m.config.MaxIdleConnections)
    sqlDB.SetMaxOpenConns(m.config.MaxOpenConnections)
    sqlDB.SetConnMaxLifetime(m.config.MaxConnectionLifeTime)

    m.client = db
    return nil
}

// HealthCheck 检查 MySQL 连接是否健康
func (m *MySQLConnection) HealthCheck(ctx context.Context) error {
    if m.client == nil {
        return fmt.Errorf("MySQL client is nil")
    }

    return m.client.WithContext(ctx).Raw("SELECT 1").Error
}

// GetClient 获取 MySQL 客户端
func (m *MySQLConnection) GetClient() interface{} {
    return m.client
}
```

### Redis适配器

```go
// RedisConnection Redis 连接实现
type RedisConnection struct {
    config *RedisConfig
    client redis.UniversalClient
}

// NewRedisConnection 创建 Redis 连接
func NewRedisConnection(config *RedisConfig) *RedisConnection {
    return &RedisConnection{
        config: config,
    }
}

// Connect 连接 Redis 数据库
func (r *RedisConnection) Connect() error {
    options := &redis.Options{
        Addr:     fmt.Sprintf("%s:%d", r.config.Host, r.config.Port),
        Password: r.config.Password,
        DB:       r.config.Database,
        PoolSize: r.config.MaxActive,
    }

    r.client = redis.NewClient(options)

    // 测试连接
    ctx, cancel := context.WithTimeout(context.Background(), r.config.Timeout)
    defer cancel()

    if err := r.client.Ping(ctx).Err(); err != nil {
        return fmt.Errorf("failed to ping Redis: %w", err)
    }

    return nil
}

// HealthCheck 检查 Redis 连接是否健康
func (r *RedisConnection) HealthCheck(ctx context.Context) error {
    if r.client == nil {
        return fmt.Errorf("Redis client is nil")
    }

    return r.client.Ping(ctx).Err()
}

// GetClient 获取 Redis 客户端
func (r *RedisConnection) GetClient() interface{} {
    return r.client
}
```

## 🎨 设计模式

### 1. 注册模式

通过注册器统一管理不同类型的数据库连接。

```go
// 注册MySQL连接
mysqlConn := databases.NewMySQLConnection(mysqlConfig)
registry.Register(databases.MySQL, mysqlConfig, mysqlConn)

// 注册Redis连接
redisConn := databases.NewRedisConnection(redisConfig)
registry.Register(databases.Redis, redisConfig, redisConn)
```

### 2. 工厂模式

使用工厂方法创建不同类型的数据库连接。

```go
// 工厂方法
func NewMySQLConnection(config *MySQLConfig) *MySQLConnection
func NewRedisConnection(config *RedisConfig) *RedisConnection
```

### 3. 策略模式

通过接口实现不同数据库的连接策略。

```go
// 策略接口
type DatabaseConnection interface {
    Connect() error
    HealthCheck(ctx context.Context) error
    Close() error
}
```

## 📈 扩展指南

### 添加新数据库类型

1.**定义数据库类型**

```go
const (
    MySQL DatabaseType = "mysql"
    Redis DatabaseType = "redis"
    PostgreSQL DatabaseType = "postgresql"  // 新增
)
```

2.**实现连接接口**

```go
type PostgreSQLConnection struct {
    config *PostgreSQLConfig
    client *gorm.DB
}

func (p *PostgreSQLConnection) Type() DatabaseType {
    return PostgreSQL
}

func (p *PostgreSQLConnection) Connect() error {
    // 实现连接逻辑
    return nil
}

func (p *PostgreSQLConnection) HealthCheck(ctx context.Context) error {
    // 实现健康检查
    return nil
}

func (p *PostgreSQLConnection) Close() error {
    // 实现关闭逻辑
    return nil
}

func (p *PostgreSQLConnection) GetClient() interface{} {
    return p.client
}
```

3.**创建工厂方法**

```go
func NewPostgreSQLConnection(config *PostgreSQLConfig) *PostgreSQLConnection {
    return &PostgreSQLConnection{
        config: config,
    }
}
```

4.**注册到注册器**

```go
postgresConn := databases.NewPostgreSQLConnection(postgresConfig)
registry.Register(databases.PostgreSQL, postgresConfig, postgresConn)
```

## 🧪 测试策略

### 单元测试

```go
func TestRegistry_Register(t *testing.T) {
    registry := database.NewRegistry()
    
    // 创建模拟连接
    mockConn := &MockDatabaseConnection{}
    config := &MockConfig{}
    
    // 测试注册
    err := registry.Register(databases.MySQL, config, mockConn)
    assert.NoError(t, err)
    
    // 测试重复注册
    err = registry.Register(databases.MySQL, config, mockConn)
    assert.Error(t, err)
}
```

### 集成测试

```go
func TestRegistry_HealthCheck(t *testing.T) {
    registry := setupTestRegistry(t)
    
    ctx := context.Background()
    err := registry.HealthCheck(ctx)
    assert.NoError(t, err)
}
```

## 🎯 最佳实践

1. **线程安全**: 使用互斥锁保护共享资源
2. **错误处理**: 提供详细的错误信息
3. **资源管理**: 确保连接正确关闭
4. **健康检查**: 定期检查连接状态
5. **配置验证**: 验证配置参数的有效性
6. **日志记录**: 记录重要的操作事件
