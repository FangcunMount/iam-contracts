# IDP 模块 - 应用服务层

## 概述

应用服务层（Application Layer）是 IDP 模块的用例协调层，负责编排领域服务、管理事务边界，为接口层（Interface Layer）提供粗粒度的业务操作接口。

## 架构原则

### 六边形架构（Hexagonal Architecture）

```
┌─────────────────────────────────────────────────────────────┐
│                      Interface Layer                         │
│              (HTTP API / gRPC / Event Handlers)              │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   Application Layer                          │
│                  (Use Case Orchestration)                    │
│                                                               │
│  ┌──────────────────────┐    ┌──────────────────────────┐  │
│  │ WechatApp Services   │    │ WechatSession Services   │  │
│  │  - App Management    │    │  - Authentication        │  │
│  │  - Credential Mgmt   │    │  - Phone Decryption      │  │
│  │  - Token Management  │    │                          │  │
│  └──────────────────────┘    └──────────────────────────┘  │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      Domain Layer                            │
│                 (Business Logic & Rules)                     │
│                                                               │
│  ┌──────────────────────┐    ┌──────────────────────────┐  │
│  │ WechatApp Domain     │    │ WechatSession Domain     │  │
│  │  - Entities          │    │  - Entities              │  │
│  │  - Value Objects     │    │  - Value Objects         │  │
│  │  - Domain Services   │    │  - Domain Services       │  │
│  └──────────────────────┘    └──────────────────────────┘  │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  Infrastructure Layer                        │
│            (Database / Cache / External APIs)                │
└─────────────────────────────────────────────────────────────┘
```

## 模块结构

```
application/
├── wechatapp/              # 微信应用应用服务
│   ├── services.go         # 应用服务接口定义 + DTOs
│   └── services_impl.go    # 应用服务实现
├── wechatsession/          # 微信会话应用服务
│   ├── services.go         # 应用服务接口定义 + DTOs
│   └── services_impl.go    # 应用服务实现
└── services.go             # 应用服务集合工厂
```

## 应用服务

### 1. 微信应用管理服务（WechatApp）

#### 1.1 WechatAppApplicationService - 应用基本管理

**职责**：微信应用的创建和查询

**接口**：
- `CreateApp(ctx, dto) (*WechatAppResult, error)` - 创建微信应用
- `GetApp(ctx, appID) (*WechatAppResult, error)` - 查询微信应用

**用例流程**：
```go
// 创建微信应用
1. 调用领域服务验证 AppID 唯一性
2. 调用领域服务创建应用实体
3. 分配内部 ID
4. 初始化凭据结构
5. 如果提供了 AppSecret，设置认证密钥
6. 持久化到仓储
7. 返回结果 DTO
```

#### 1.2 WechatAppCredentialApplicationService - 凭据管理

**职责**：微信应用凭据的轮换和管理

**接口**：
- `RotateAuthSecret(ctx, appID, newSecret) error` - 轮换认证密钥（AppSecret）
- `RotateMsgSecret(ctx, appID, callbackToken, encodingAESKey) error` - 轮换消息加解密密钥

**用例流程**：
```go
// 轮换认证密钥
1. 查询应用实体
2. 验证应用状态（不能是归档状态）
3. 调用领域服务轮换密钥（加密存储、更新指纹、版本递增）
4. 持久化到仓储
```

#### 1.3 WechatAppTokenApplicationService - 访问令牌管理

**职责**：微信应用访问令牌的获取和刷新（带缓存）

**接口**：
- `GetAccessToken(ctx, appID) (string, error)` - 获取访问令牌（自动缓存和刷新）
- `RefreshAccessToken(ctx, appID) (string, error)` - 强制刷新访问令牌

**用例流程**：
```go
// 获取访问令牌（带缓存）
1. 查询应用实体
2. 调用领域服务的访问令牌缓存器
3. 缓存器自动处理：
   - 读取缓存
   - 检查是否过期（提前 120s 刷新）
   - 单飞刷新（避免并发重复刷新）
   - 更新缓存
4. 返回令牌

// 强制刷新访问令牌
1. 查询应用实体
2. 直接调用令牌提供器获取新令牌
3. 更新缓存
4. 返回新令牌
```

### 2. 微信认证服务（WechatSession）

#### 2.1 WechatAuthApplicationService - 微信认证

**职责**：微信小程序/公众号登录和用户信息解密

**接口**：
- `LoginWithCode(ctx, dto) (*LoginResult, error)` - 使用微信登录码进行登录
- `DecryptUserPhone(ctx, dto) (string, error)` - 解密用户手机号

**用例流程**：
```go
// 微信登录
1. 参数校验（AppID、JSCode）
2. 调用领域服务的认证器
3. 认证器执行：
   - 调用微信 API 进行 code2Session
   - 创建/更新 WechatSession 实体
   - 加密存储 session_key
   - 生成外部身份声明（ExternalClaim）
4. 转换为登录结果 DTO
5. 返回结果（包含 OpenID、UnionID、SessionKey 等）

// 解密手机号
1. 参数校验（AppID、OpenID、EncryptedData、IV）
2. 调用领域服务的认证器
3. 认证器执行：
   - 获取用户的 session_key
   - 使用 session_key 解密加密数据
   - 返回手机号
4. 返回手机号
```

## DTOs（数据传输对象）

### WechatApp DTOs

```go
// 创建微信应用 DTO
type CreateWechatAppDTO struct {
    AppID     string         // 微信应用 ID（必填）
    Name      string         // 应用名称（必填）
    Type      AppType        // 应用类型（必填：MiniProgram/MP）
    AppSecret string         // AppSecret（可选，创建时设置）
}

// 微信应用结果 DTO
type WechatAppResult struct {
    ID     string      // 内部 ID
    AppID  string      // 微信应用 ID
    Name   string      // 应用名称
    Type   AppType     // 应用类型
    Status Status      // 应用状态
}
```

### WechatSession DTOs

```go
// 微信登录 DTO
type LoginWithCodeDTO struct {
    AppID  string  // 微信应用 ID（必填）
    JSCode string  // 微信登录码（必填）
}

// 解密手机号 DTO
type DecryptPhoneDTO struct {
    AppID         string  // 微信应用 ID（必填）
    OpenID        string  // 用户 OpenID（必填）
    EncryptedData string  // 加密数据（必填）
    IV            string  // 加密算法的初始向量（必填）
}

// 登录结果 DTO
type LoginResult struct {
    Provider     string   // 身份提供商（wechat_miniprogram）
    AppID        string   // 微信应用 ID
    OpenID       string   // 用户 OpenID
    UnionID      *string  // 用户 UnionID（可选）
    DisplayName  *string  // 显示名称（可选）
    AvatarURL    *string  // 头像 URL（可选）
    Phone        *string  // 手机号（可选）
    Email        *string  // 邮箱（可选）
    ExpiresInSec int      // 过期时间（秒）
    SessionKey   string   // Session Key（加密后）
    Version      int      // 会话版本
}
```

## 依赖注入

应用服务层通过构造函数注入依赖的端口（Port），遵循依赖倒置原则（DIP）：

```go
// 应用服务依赖的端口集合
type ApplicationServicesDependencies struct {
    // WechatApp 依赖
    WechatAppRepo      wechatappport.WechatAppRepository
    WechatAppCreator   wechatappport.WechatAppCreator
    WechatAppQuerier   wechatappport.WechatAppQuerier
    CredentialRotater  wechatappport.CredentialRotater
    AccessTokenCacher  wechatappport.AccessTokenCacher
    AppTokenProvider   wechatappport.AppTokenProvider
    AccessTokenCache   wechatappport.AccessTokenCache

    // WechatSession 依赖
    WechatAuthenticator wechatsessionport.Authenticator
}

// 创建所有应用服务实例
services := NewApplicationServices(deps)
```

## 设计原则

### 1. 单一职责原则（SRP）

每个应用服务专注于特定的业务用例：
- `WechatAppApplicationService` - 应用管理
- `WechatAppCredentialApplicationService` - 凭据管理
- `WechatAppTokenApplicationService` - 令牌管理
- `WechatAuthApplicationService` - 认证服务

### 2. 接口隔离原则（ISP）

应用服务接口按职责细粒度划分，客户端只依赖需要的接口。

### 3. 依赖倒置原则（DIP）

应用服务依赖抽象的端口（Port），而不是具体的实现：
- 依赖领域层的 Driving Ports（领域服务接口）
- 依赖领域层的 Driven Ports（仓储、外部服务接口）
- 具体实现由基础设施层提供

### 4. 开闭原则（OCP）

应用服务对扩展开放、对修改关闭：
- 新增用例通过新增应用服务实现
- 不修改现有应用服务

### 5. 事务边界管理

应用服务是事务的自然边界：
- 一个应用服务方法对应一个用例
- 一个用例对应一个事务
- 事务由应用服务层管理（通过 UoW 或仓储）

## 与其他层的关系

### 应用服务层 vs 领域服务层

| 维度 | 应用服务层 | 领域服务层 |
|------|-----------|-----------|
| 职责 | 用例编排、事务管理、DTO 转换 | 领域逻辑、业务规则 |
| 粒度 | 粗粒度（面向用例） | 细粒度（面向领域概念） |
| 依赖 | 依赖领域服务、仓储 | 依赖领域对象、仓储抽象 |
| 复用 | 低复用（用例特定） | 高复用（领域通用） |
| 测试 | 集成测试 | 单元测试 |

### 应用服务层 vs 接口层

| 维度 | 应用服务层 | 接口层 |
|------|-----------|--------|
| 职责 | 用例编排、业务流程 | 协议适配、参数验证、错误处理 |
| 关注点 | 业务逻辑 | 技术细节（HTTP、gRPC） |
| DTO | 业务 DTO | 协议 DTO（Request/Response） |
| 复用 | 可被多种接口复用 | 特定协议 |

## 使用示例

### 创建微信应用

```go
// 1. 准备 DTO
dto := wechatapp.CreateWechatAppDTO{
    AppID:     "wx1234567890",
    Name:      "测试小程序",
    Type:      domain.MiniProgram,
    AppSecret: "abcdef1234567890abcdef1234567890",
}

// 2. 调用应用服务
result, err := appServices.WechatApp.CreateApp(ctx, dto)
if err != nil {
    return err
}

// 3. 使用结果
fmt.Printf("Created app: %s (ID: %s)\n", result.Name, result.ID)
```

### 获取访问令牌

```go
// 调用应用服务（自动处理缓存和刷新）
token, err := appServices.WechatAppToken.GetAccessToken(ctx, "wx1234567890")
if err != nil {
    return err
}

// 使用令牌调用微信 API
// ...
```

### 微信登录

```go
// 1. 准备 DTO
dto := wechatsession.LoginWithCodeDTO{
    AppID:  "wx1234567890",
    JSCode: "071AbcDef123456",
}

// 2. 调用应用服务
result, err := appServices.WechatAuth.LoginWithCode(ctx, dto)
if err != nil {
    return err
}

// 3. 使用结果
fmt.Printf("Login success: OpenID=%s, UnionID=%v\n", 
    result.OpenID, result.UnionID)
```

## 扩展点

### 添加新的应用服务

1. 在 `application/<subdomain>/` 目录创建新的应用服务
2. 定义应用服务接口和 DTOs（`services.go`）
3. 实现应用服务（`services_impl.go`）
4. 在 `application/services.go` 中注册新服务
5. 提供依赖注入配置

### 添加新的用例

在现有应用服务接口中添加新方法：
```go
type WechatAppApplicationService interface {
    // 现有方法
    CreateApp(ctx context.Context, dto CreateWechatAppDTO) (*WechatAppResult, error)
    GetApp(ctx context.Context, appID string) (*WechatAppResult, error)
    
    // 新增用例
    UpdateApp(ctx context.Context, appID string, dto UpdateWechatAppDTO) (*WechatAppResult, error)
}
```

## 最佳实践

1. **保持应用服务薄**：应用服务应该是编排者，不要在应用服务中编写业务逻辑
2. **使用 DTO 隔离**：应用服务的输入输出使用 DTO，不要暴露领域对象
3. **明确事务边界**：一个应用服务方法对应一个事务
4. **错误处理**：在应用服务层包装领域层错误，提供更友好的错误信息
5. **参数验证**：在应用服务层进行基本的参数验证
6. **幂等性设计**：对于状态变更操作，考虑幂等性设计
7. **异步处理**：对于长时间运行的操作，考虑使用异步模式

## 测试策略

### 单元测试

测试应用服务的编排逻辑，使用 Mock 模拟依赖：

```go
func TestCreateApp(t *testing.T) {
    // 1. 准备 Mock 依赖
    mockRepo := &mockWechatAppRepository{}
    mockCreator := &mockWechatAppCreator{}
    // ...

    // 2. 创建应用服务
    service := NewWechatAppApplicationService(
        mockRepo, mockCreator, mockQuerier, mockRotater,
    )

    // 3. 执行测试
    dto := CreateWechatAppDTO{...}
    result, err := service.CreateApp(context.Background(), dto)

    // 4. 验证结果
    assert.NoError(t, err)
    assert.NotNil(t, result)
    // ...
}
```

### 集成测试

测试应用服务与真实依赖的集成：

```go
func TestCreateAppIntegration(t *testing.T) {
    // 1. 准备测试环境（真实数据库、Redis 等）
    // 2. 创建真实的依赖实例
    // 3. 执行测试
    // 4. 验证数据库状态
    // 5. 清理测试数据
}
```

## 总结

应用服务层是 IDP 模块的关键协调层，它：
- 编排领域服务完成业务用例
- 管理事务边界
- 提供粗粒度的业务接口
- 隔离领域层和接口层
- 通过 DTO 进行数据转换

通过良好的应用服务层设计，我们可以：
- 保持领域层的纯粹性
- 提高代码的可测试性
- 实现接口的可替换性
- 支持业务的快速迭代
