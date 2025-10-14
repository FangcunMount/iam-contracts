# 六边形架构容器设计

## 🎯 容器设计理念

依赖注入容器是六边形架构的核心组件，负责管理所有模块的依赖关系、生命周期和模块组装。容器实现了松耦合的模块设计，使得系统更加灵活、可测试和可维护。

## 🏗️ 容器架构

```text
┌─────────────────────────────────────────────────────────────┐
│                        Container                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                    Modules                              │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │ UserModule  │  │ AuthModule  │  │ OtherModule │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                infra                           │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │   MySQL     │  │   Redis     │  │   HTTP      │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 📦 容器实现

### 容器结构

```go
// Container 容器
type Container struct {
    // 数据库连接
    mysqlDB *gorm.DB

    // 业务模块
    AuthModule *assembler.AuthModule
    UserModule *assembler.UserModule

    // 容器状态
    initialized bool
}
```

### 容器创建

```go
// NewContainer 创建容器
func NewContainer(mysqlDB *gorm.DB) *Container {
    return &Container{
        mysqlDB: mysqlDB,
    }
}
```

### 容器初始化

```go
// Initialize 初始化容器
func (c *Container) Initialize() error {
    if c.initialized {
        return fmt.Errorf("container already initialized")
    }

    // 初始化认证模块
    if err := c.initAuthModule(); err != nil {
        return fmt.Errorf("failed to initialize auth module: %w", err)
    }

    // 初始化用户模块
    if err := c.initUserModule(); err != nil {
        return fmt.Errorf("failed to initialize user module: %w", err)
    }

    c.initialized = true
    fmt.Printf("🏗️  Container initialized with modules: user, auth\n")

    return nil
}
```

## 🔧 模块组装器

### 模块接口

```go
// Module 模块接口
type Module interface {
    // Initialize 初始化模块
    Initialize(db *gorm.DB) error
    
    // CheckHealth 健康检查
    CheckHealth() error
    
    // Cleanup 清理资源
    Cleanup() error
    
    // ModuleInfo 模块信息
    ModuleInfo() ModuleInfo
}
```

### 用户模块组装器

```go
// UserModule 用户模块
type UserModule struct {
    // handler 层
    UserHandler *handler.UserHandler
}

// NewUserModule 创建用户模块
func NewUserModule() *UserModule {
    return &UserModule{}
}

// Initialize 初始化用户模块
func (m *UserModule) Initialize(db *gorm.DB) error {
    if db == nil {
        return errors.WithCode(code.ErrModuleInitializationFailed, "database connection is nil")
    }

    // 初始化 handler 层
    m.UserHandler = handler.NewUserHandler()
    
    return nil
}
```

### 认证模块组装器

```go
// AuthModule 认证模块
type AuthModule struct {
    // 这里可以添加认证相关的组件
}

// NewAuthModule 创建认证模块
func NewAuthModule() *AuthModule {
    return &AuthModule{}
}

// Initialize 初始化认证模块
func (m *AuthModule) Initialize(db *gorm.DB) error {
    if db == nil {
        return errors.WithCode(code.ErrModuleInitializationFailed, "database connection is nil")
    }

    // 这里可以初始化认证相关的组件
    // 目前简化处理，不依赖具体的业务逻辑
    
    return nil
}
```

## 🔄 生命周期管理

### 初始化流程

```go
// 1. 创建容器
container := container.NewContainer(mysqlDB)

// 2. 初始化容器
if err := container.Initialize(); err != nil {
    log.Fatalf("Failed to initialize container: %v", err)
}

// 3. 使用模块
userHandler := container.UserModule.UserHandler
authModule := container.AuthModule
```

### 健康检查

```go
// HealthCheck 健康检查
func (c *Container) HealthCheck(ctx context.Context) error {
    // 检查MySQL连接
    if c.mysqlDB != nil {
        if err := c.mysqlDB.WithContext(ctx).Raw("SELECT 1").Error; err != nil {
            return fmt.Errorf("mysql health check failed: %w", err)
        }
    }

    return nil
}
```

### 状态监控

```go
// PrintStatus 打印容器状态
func (c *Container) PrintStatus() {
    fmt.Printf("📊 Container Status:\n")
    fmt.Printf("   • Initialized: %t\n", c.initialized)
    
    // 数据库连接状态
    fmt.Printf("   • MySQL: ")
    if c.mysqlDB != nil {
        fmt.Printf("✅\n")
    } else {
        fmt.Printf("❌\n")
    }

    // 模块状态
    fmt.Printf("   • Auth Module: ")
    if c.AuthModule != nil {
        fmt.Printf("✅\n")
    } else {
        fmt.Printf("❌\n")
    }

    fmt.Printf("   • User Module: ")
    if c.UserModule != nil {
        fmt.Printf("✅\n")
    } else {
        fmt.Printf("❌\n")
    }
}
```

## 🎨 设计模式

### 1. 依赖注入模式

容器通过依赖注入管理模块间的依赖关系，实现松耦合。

```go
// 模块依赖注入
func (c *Container) initUserModule() error {
    userModule := assembler.NewUserModule()
    if err := userModule.Initialize(c.mysqlDB); err != nil {
        return fmt.Errorf("failed to initialize user module: %w", err)
    }
    c.UserModule = userModule
    return nil
}
```

### 2. 工厂模式

使用工厂模式创建模块实例。

```go
// 模块工厂
func NewUserModule() *UserModule {
    return &UserModule{}
}

func NewAuthModule() *AuthModule {
    return &AuthModule{}
}
```

### 3. 模板方法模式

定义模块的标准生命周期。

```go
// 模块生命周期模板
type Module interface {
    Initialize(db *gorm.DB) error
    CheckHealth() error
    Cleanup() error
}
```

## 📈 扩展指南

### 添加新模块

1.**定义模块接口**

```go
type NewModule struct {
    // 模块组件
    Handler *handler.NewHandler
    Service *service.NewService
}
```

2.**实现模块方法**

```go
func (m *NewModule) Initialize(db *gorm.DB) error {
    // 初始化逻辑
    return nil
}

func (m *NewModule) CheckHealth() error {
    // 健康检查逻辑
    return nil
}

func (m *NewModule) Cleanup() error {
    // 清理逻辑
    return nil
}
```

3.**在容器中注册**

```go
// 在Container结构体中添加
type Container struct {
    // ... 其他字段
    NewModule *assembler.NewModule
}

// 在Initialize方法中初始化
func (c *Container) Initialize() error {
    // ... 其他初始化
    
    // 初始化新模块
    if err := c.initNewModule(); err != nil {
        return fmt.Errorf("failed to initialize new module: %w", err)
    }
    
    return nil
}

func (c *Container) initNewModule() error {
    newModule := assembler.NewNewModule()
    if err := newModule.Initialize(c.mysqlDB); err != nil {
        return fmt.Errorf("failed to initialize new module: %w", err)
    }
    c.NewModule = newModule
    return nil
}
```

## 🧪 测试策略

### 单元测试

```go
func TestContainer_Initialize(t *testing.T) {
    // 创建测试数据库
    db := setupTestDB(t)
    
    // 创建容器
    container := container.NewContainer(db)
    
    // 测试初始化
    err := container.Initialize()
    assert.NoError(t, err)
    assert.True(t, container.IsInitialized())
    assert.NotNil(t, container.UserModule)
    assert.NotNil(t, container.AuthModule)
}
```

### 集成测试

```go
func TestContainer_HealthCheck(t *testing.T) {
    // 创建容器
    container := setupContainer(t)
    
    // 测试健康检查
    ctx := context.Background()
    err := container.HealthCheck(ctx)
    assert.NoError(t, err)
}
```

## 🎯 最佳实践

1. **单一职责**: 每个模块只负责一个业务领域
2. **依赖注入**: 通过容器管理依赖关系
3. **生命周期管理**: 统一的初始化和清理流程
4. **健康检查**: 提供完整的健康检查机制
5. **状态监控**: 实时监控容器和模块状态
6. **错误处理**: 统一的错误处理和日志记录
