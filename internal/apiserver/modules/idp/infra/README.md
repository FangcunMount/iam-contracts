# 基础设施层（Infrastructure Layer）

## 概述

基础设施层是 IDP 模块的技术实现层，提供与外部系统的集成和技术能力支撑。本层遵循**六边形架构**（Hexagonal Architecture / Ports & Adapters）的原则，确保与领域层的隔离。

## 架构原则

### 1. **依赖方向**

```
领域层 (Domain) ← 基础设施层 (Infrastructure)
```

- ✅ **基础设施层依赖领域层的端口（Port）接口**
- ❌ **基础设施层不能依赖领域对象（Domain Objects）**
- ✅ **基础设施层使用原始类型（primitives）作为输入输出**

### 2. **层次隔离**

基础设施层实现由领域层定义的端口接口，而领域服务层负责在领域对象和原始类型之间进行适配。

```go
// ✅ 正确：基础设施层使用原始类型
func (s *AuthService) Code2Session(ctx context.Context, appID, appSecret, jsCode string) (*Code2SessionResult, error)

// ❌ 错误：基础设施层不应接受领域对象
func (s *AuthService) Code2Session(ctx context.Context, app *domain.WechatApp, jsCode string) (*Code2SessionResult, error)
```

### 3. **端口实现**

所有基础设施服务都实现领域层定义的端口接口：

- MySQL 仓储 → 实现 `port.WechatAppRepository`
- Redis 缓存 → 实现 `port.AccessTokenCache`
- 加密服务 → 实现 `port.SecretVault`

## 目录结构

```
infra/
├── infrastructure.go           # 基础设施服务聚合器
├── mysql/                      # MySQL 持久化
│   ├── wechatapp_po.go        # 持久化对象（PO）
│   ├── wechatapp_repository.go # 仓储实现
│   └── schema.sql             # 数据库表结构
├── redis/                      # Redis 缓存
│   ├── accesstoken_cache.go   # 访问令牌缓存
│   └── wechatsession_repository.go # 会话仓储
├── crypto/                     # 加密服务
│   └── secret_vault.go        # 密钥加密（AES-GCM）
└── wechatapi/                  # 微信 API 集成
    ├── code2session_client.go # 微信认证服务
    └── token_provider.go      # 访问令牌提供器
```

## 技术栈

| 组件 | 技术选型 | 用途 |
|------|---------|------|
| MySQL | GORM v2 | 数据持久化 |
| Redis | go-redis/v9 | 缓存 & 会话存储 |
| 加密 | AES-256-GCM | 敏感信息加密 |
| 微信 SDK | silenceper/wechat/v2 | 微信 API 集成 |

## 服务说明

### 1. MySQL 持久化（mysql/）

提供数据持久化能力，将领域对象映射到数据库表。

**仓储实现：**

- `WechatAppRepository`: 微信应用仓储
  - `Create()`: 创建应用
  - `GetByID()`: 根据 ID 查询
  - `GetByAppID()`: 根据 AppID 查询
  - `Update()`: 更新应用信息

**持久化对象（PO）：**

```go
type WechatAppPO struct {
    ID                string  // 内部 ID
    AppID             string  // 微信 AppID
    AppType           string  // 应用类型
    EncryptedSecret   []byte  // 加密后的 AppSecret
    Nonce             []byte  // 加密 nonce
}
```

**数据库表：**

```sql
CREATE TABLE wechat_apps (
    id VARCHAR(36) PRIMARY KEY,
    app_id VARCHAR(128) UNIQUE NOT NULL,
    app_type VARCHAR(32) NOT NULL,
    encrypted_secret VARBINARY(512) NOT NULL,
    nonce VARBINARY(32) NOT NULL,
    ...
);
```

### 2. Redis 缓存（redis/）

提供高性能缓存和会话存储能力。

**访问令牌缓存（AccessTokenCache）：**

- `Get()`: 获取缓存的令牌
- `Set()`: 设置令牌缓存
- `Delete()`: 删除令牌缓存
- `TryLockRefresh()`: 分布式锁（防止并发刷新）
- `UnlockRefresh()`: 释放分布式锁

**会话仓储（WechatSessionRepository）：**

- `Save()`: 保存会话
- `Get()`: 获取会话
- `Delete()`: 删除会话
- `Refresh()`: 刷新会话 TTL

### 3. 加密服务（crypto/）

提供敏感信息加密能力，使用 **AES-256-GCM** 算法。

**密钥加密（SecretVault）：**

```go
// 加密明文
ciphertext, err := vault.Encrypt(ctx, plaintext)

// 解密密文
plaintext, err := vault.Decrypt(ctx, ciphertext)
```

**特性：**

- AES-256-GCM：认证加密，防止篡改
- 随机 nonce：每次加密生成唯一 nonce
- 主密钥：32 字节（256 位）

### 4. 微信 API 集成（wechatapi/）

提供与微信平台的交互能力，使用 **silenceper/wechat/v2** SDK。

**认证服务（AuthService）：**

```go
// 小程序登录认证
result, err := authService.Code2Session(ctx, appID, appSecret, jsCode)
// 返回：OpenID, SessionKey, UnionID

// 解密手机号
result, err := authService.DecryptPhone(ctx, sessionKey, encryptedData, iv)
// 返回：PhoneNumber, CountryCode, PurePhoneNumber
```

**令牌提供器（TokenProvider）：**

```go
// 获取小程序访问令牌
result, err := tokenProvider.FetchMiniProgramToken(ctx, appID, appSecret)
// 返回：Token, ExpiresAt

// 获取公众号访问令牌
result, err := tokenProvider.FetchOfficialAccountToken(ctx, appID, appSecret)
// 返回：Token, ExpiresAt
```

**架构亮点：**

✅ **不依赖领域对象**：方法参数使用 `string`、`[]byte` 等原始类型

✅ **返回值对象（Result）**：简单的数据结构，不包含业务逻辑

✅ **错误处理**：返回标准 Go error，不传播 SDK 特定错误类型

## 使用示例

### 初始化基础设施服务

```go
import (
    "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/infra"
)

// 准备依赖
deps := &infra.InfrastructureDependencies{
    DB:             gormDB,           // *gorm.DB
    RedisClient:    redisClient,      // *redis.Client
    EncryptionKey:  encryptionKey,    // []byte (32 bytes)
    WechatSDKCache: wechatCache,      // cache.Cache (可选)
}

// 创建基础设施服务
infraServices, err := infra.NewInfrastructureServices(deps)
if err != nil {
    log.Fatal(err)
}
```

### 使用仓储

```go
// 创建微信应用
app := &domain.WechatApp{
    ID:     "app-123",
    AppID:  "wx1234567890abcdef",
    Type:   domain.WechatAppTypeMiniProgram,
    Secret: "app-secret-encrypted",
}
err := infraServices.WechatAppRepository.Create(ctx, app)

// 查询微信应用
app, err := infraServices.WechatAppRepository.GetByAppID(ctx, "wx1234567890abcdef")
```

### 使用缓存

```go
// 保存访问令牌
token := &domain.AppAccessToken{
    Token:     "ACCESS_TOKEN",
    ExpiresAt: time.Now().Add(2 * time.Hour),
}
err := infraServices.AccessTokenCache.Set(ctx, "wx1234567890abcdef", token)

// 获取访问令牌
token, err := infraServices.AccessTokenCache.Get(ctx, "wx1234567890abcdef")
```

### 使用加密服务

```go
// 加密 AppSecret
plaintext := []byte("my-app-secret")
ciphertext, err := infraServices.SecretVault.Encrypt(ctx, plaintext)

// 解密 AppSecret
plaintext, err := infraServices.SecretVault.Decrypt(ctx, ciphertext)
```

### 使用微信 API

```go
// 小程序登录
result, err := infraServices.WechatAuthService.Code2Session(
    ctx,
    "wx1234567890abcdef", // appID
    "app-secret",         // appSecret
    "071xYZ123456",       // jsCode
)
// result: *wechatapi.Code2SessionResult { OpenID, SessionKey, UnionID }

// 获取访问令牌
tokenResult, err := infraServices.WechatTokenProvider.FetchMiniProgramToken(
    ctx,
    "wx1234567890abcdef", // appID
    "app-secret",         // appSecret
)
// tokenResult: *wechatapi.AccessTokenResult { Token, ExpiresAt }
```

## 与领域层的协作

基础设施层通过**端口（Port）接口**与领域层协作：

```go
// 领域层定义端口接口（domain/wechatapp/port/driven.go）
type WechatAppRepository interface {
    Create(ctx context.Context, app *WechatApp) error
    GetByAppID(ctx context.Context, appID string) (*WechatApp, error)
    // ...
}

// 基础设施层实现端口接口（infra/mysql/wechatapp_repository.go）
type wechatAppRepository struct {
    db *gorm.DB
}

func (r *wechatAppRepository) Create(ctx context.Context, app *domain.WechatApp) error {
    // 领域对象 -> PO
    po := toPO(app)
    // 持久化到数据库
    return r.db.WithContext(ctx).Create(po).Error
}
```

## 测试建议

### 单元测试

对每个基础设施组件编写单元测试：

```go
func TestWechatAppRepository_Create(t *testing.T) {
    // 准备测试数据库
    db := setupTestDB()
    repo := mysql.NewWechatAppRepository(db)
    
    // 创建测试数据
    app := &domain.WechatApp{...}
    err := repo.Create(context.Background(), app)
    assert.NoError(t, err)
    
    // 验证数据已持久化
    ...
}
```

### 集成测试

在真实环境中测试基础设施服务：

```go
func TestInfrastructureServices_Integration(t *testing.T) {
    // 连接真实 MySQL、Redis
    deps := setupRealDependencies()
    infraServices, err := infra.NewInfrastructureServices(deps)
    assert.NoError(t, err)
    
    // 测试端到端流程
    ...
}
```

## 注意事项

### 1. 依赖注入

所有外部依赖通过构造函数注入，便于测试和替换：

```go
func NewWechatAppRepository(db *gorm.DB) port.WechatAppRepository {
    return &wechatAppRepository{db: db}
}
```

### 2. 错误处理

基础设施层应该返回清晰的错误信息：

```go
if err != nil {
    return fmt.Errorf("failed to create wechat app: %w", err)
}
```

### 3. 上下文传递

所有方法都应接受 `context.Context` 参数，支持超时和取消：

```go
func (r *wechatAppRepository) Create(ctx context.Context, app *domain.WechatApp) error {
    return r.db.WithContext(ctx).Create(po).Error
}
```

### 4. 事务处理

数据库操作应支持事务（通过依赖注入传入的 `*gorm.DB` 可能已经在事务中）：

```go
// 领域服务层控制事务
tx := db.Begin()
repo := mysql.NewWechatAppRepository(tx)
// ...操作...
tx.Commit()
```

## 扩展指南

### 添加新的仓储

1. 在 `infra/mysql/` 创建 PO 和仓储实现
2. 在领域层的 `port/driven.go` 定义端口接口
3. 在 `infrastructure.go` 中聚合新服务

### 添加新的外部 API 集成

1. 在 `infra/` 创建新的子包（如 `infra/alipayapi/`）
2. 使用原始类型作为输入输出，不依赖领域对象
3. 创建 Result 结构体封装返回数据
4. 在 `infrastructure.go` 中聚合新服务

## 相关文档

- [IDP 模块总览](../README.md)
- [领域层文档](../domain/README.md)
- [应用层文档](../application/README.md)
- [架构设计](../docs/ARCHITECTURE.md)
