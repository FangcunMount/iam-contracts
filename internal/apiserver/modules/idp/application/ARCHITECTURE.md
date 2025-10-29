# IDP 模块应用服务层架构图

## 整体架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Interface Layer                             │
│                  (HTTP API / gRPC / Event Handlers)                 │
│                                                                      │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐ │
│  │  HTTP Handlers   │  │  gRPC Services   │  │  Event Listeners │ │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘ │
└────────────────────────────────┬────────────────────────────────────┘
                                 │
                                 │ DTOs (Request/Response)
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       Application Layer                             │
│                     (Use Case Orchestration)                        │
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  ApplicationServices (services.go)                            │ │
│  │  ┌─────────────────┐  ┌─────────────────────────────────┐    │ │
│  │  │ WechatApp       │  │ WechatSession                   │    │ │
│  │  │ - App Mgmt      │  │ - Auth                          │    │ │
│  │  │ - Credential    │  │                                 │    │ │
│  │  │ - Token         │  │                                 │    │ │
│  │  └─────────────────┘  └─────────────────────────────────┘    │ │
│  └───────────────────────────────────────────────────────────────┘ │
└────────────────────────────────┬────────────────────────────────────┘
                                 │
                                 │ Driving Ports (Interfaces)
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          Domain Layer                                │
│                    (Business Logic & Rules)                          │
│                                                                      │
│  ┌──────────────────────┐           ┌──────────────────────┐       │
│  │  WechatApp Domain    │           │  WechatSession       │       │
│  │  ┌────────────────┐  │           │  ┌────────────────┐ │       │
│  │  │ Entities       │  │           │  │ Entities       │ │       │
│  │  │ - WechatApp    │  │           │  │ - Session      │ │       │
│  │  │ - Credentials  │  │           │  │ - Claim        │ │       │
│  │  │ - AccessToken  │  │           │  └────────────────┘ │       │
│  │  └────────────────┘  │           │                      │       │
│  │  ┌────────────────┐  │           │  ┌────────────────┐ │       │
│  │  │ Domain Services│  │           │  │ Domain Services│ │       │
│  │  │ - Creator      │  │           │  │ - Authenticator│ │       │
│  │  │ - Querier      │  │           │  └────────────────┘ │       │
│  │  │ - Rotater      │  │           │                      │       │
│  │  │ - TokenCacher  │  │           │                      │       │
│  │  └────────────────┘  │           │                      │       │
│  │  ┌────────────────┐  │           │                      │       │
│  │  │ Ports (Driven) │  │           │                      │       │
│  │  │ - Repository   │  │           │                      │       │
│  │  │ - Cache        │  │           │                      │       │
│  │  │ - Vault        │  │           │                      │       │
│  │  │ - Provider     │  │           │                      │       │
│  │  └────────────────┘  │           │                      │       │
│  └──────────────────────┘           └──────────────────────┘       │
└────────────────────────────────┬────────────────────────────────────┘
                                 │
                                 │ Driven Ports (Interfaces)
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Infrastructure Layer                            │
│              (Database / Cache / External APIs)                     │
│                                                                      │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐   │
│  │   MySQL    │  │   Redis    │  │  Wechat    │  │    KMS     │   │
│  │ Repository │  │   Cache    │  │    API     │  │   Vault    │   │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

## 应用服务层详细结构

```
application/
│
├── services.go                      # 应用服务聚合根
│   └── ApplicationServices          # 所有应用服务的集合
│       ├── WechatApp                # 微信应用管理
│       ├── WechatAppCredential      # 微信应用凭据管理
│       ├── WechatAppToken           # 微信应用令牌管理
│       └── WechatAuth               # 微信认证
│
├── wechatapp/                       # 微信应用子模块
│   ├── services.go                  # 应用服务接口定义
│   │   ├── WechatAppApplicationService
│   │   ├── WechatAppCredentialApplicationService
│   │   ├── WechatAppTokenApplicationService
│   │   ├── CreateWechatAppDTO       # 输入 DTO
│   │   └── WechatAppResult          # 输出 DTO
│   │
│   └── services_impl.go             # 应用服务实现
│       ├── wechatAppApplicationService
│       ├── wechatAppCredentialApplicationService
│       └── wechatAppTokenApplicationService
│
└── wechatsession/                   # 微信会话子模块
    ├── services.go                  # 应用服务接口定义
    │   ├── WechatAuthApplicationService
    │   ├── LoginWithCodeDTO         # 输入 DTO
    │   ├── DecryptPhoneDTO          # 输入 DTO
    │   └── LoginResult              # 输出 DTO
    │
    └── services_impl.go             # 应用服务实现
        └── wechatAuthApplicationService
```

## 依赖关系图

```
┌──────────────────────────────────────────────────────────────┐
│                  wechatAppApplicationService                  │
│                                                               │
│  Dependencies:                                                │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ • WechatAppRepository     (基础设施层提供)            │   │
│  │ • WechatAppCreator        (领域服务层)               │   │
│  │ • WechatAppQuerier        (领域服务层)               │   │
│  │ • CredentialRotater       (领域服务层)               │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  Methods:                                                     │
│  • CreateApp(dto) -> WechatAppResult                         │
│  • GetApp(appID) -> WechatAppResult                          │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│             wechatAppCredentialApplicationService             │
│                                                               │
│  Dependencies:                                                │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ • WechatAppRepository     (基础设施层提供)            │   │
│  │ • WechatAppQuerier        (领域服务层)               │   │
│  │ • CredentialRotater       (领域服务层)               │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  Methods:                                                     │
│  • RotateAuthSecret(appID, newSecret) -> error               │
│  • RotateMsgSecret(appID, token, key) -> error               │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│              wechatAppTokenApplicationService                 │
│                                                               │
│  Dependencies:                                                │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ • WechatAppQuerier        (领域服务层)               │   │
│  │ • AccessTokenCacher       (领域服务层)               │   │
│  │ • AppTokenProvider        (基础设施层提供)            │   │
│  │ • AccessTokenCache        (基础设施层提供)            │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  Methods:                                                     │
│  • GetAccessToken(appID) -> string                           │
│  • RefreshAccessToken(appID) -> string                       │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│               wechatAuthApplicationService                    │
│                                                               │
│  Dependencies:                                                │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ • Authenticator           (领域服务层)               │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  Methods:                                                     │
│  • LoginWithCode(dto) -> LoginResult                         │
│  • DecryptUserPhone(dto) -> string                           │
└──────────────────────────────────────────────────────────────┘
```

## 数据流图

### 用例 1: 创建微信应用

```
HTTP Request
    │
    ▼
┌────────────────────────┐
│  HTTP Handler          │
│  (Interface Layer)     │
└────────────────────────┘
    │ CreateWechatAppDTO
    ▼
┌────────────────────────────────────────────────────────┐
│  wechatAppApplicationService.CreateApp()               │
│  (Application Layer)                                   │
│                                                         │
│  1. creator.Create()           ──────────┐            │
│                                           │            │
│  2. rotater.RotateAuthSecret() ──────────┤            │
│                                           │            │
│  3. repo.Create()              ──────────┤            │
│                                           │            │
│  4. toWechatAppResult()                  │            │
└──────────────────────────────────────────┼────────────┘
                                           │
                                           ▼
                            ┌──────────────────────────┐
                            │  Domain Services         │
                            │  (Domain Layer)          │
                            │                          │
                            │  • WechatAppCreator      │
                            │  • CredentialRotater     │
                            └──────────────────────────┘
                                           │
                                           ▼
                            ┌──────────────────────────┐
                            │  WechatAppRepository     │
                            │  (Infrastructure Layer)  │
                            │                          │
                            │  MySQL Database          │
                            └──────────────────────────┘
```

### 用例 2: 获取访问令牌（带缓存）

```
HTTP Request
    │
    ▼
┌────────────────────────┐
│  HTTP Handler          │
│  (Interface Layer)     │
└────────────────────────┘
    │ appID
    ▼
┌─────────────────────────────────────────────────────────────┐
│  wechatAppTokenApplicationService.GetAccessToken()          │
│  (Application Layer)                                        │
│                                                              │
│  1. querier.QueryByAppID()     ─────────┐                  │
│                                          │                  │
│  2. tokenCacher.EnsureToken()  ─────────┤                  │
│                                          │                  │
└──────────────────────────────────────────┼──────────────────┘
                                           │
                                           ▼
                            ┌──────────────────────────────┐
                            │  AccessTokenCacher           │
                            │  (Domain Service)            │
                            │                              │
                            │  1. cache.Get()              │
                            │     └─> 缓存命中? 返回       │
                            │                              │
                            │  2. cache.TryLockRefresh()   │
                            │     └─> 单飞机制             │
                            │                              │
                            │  3. provider.Fetch()         │
                            │     └─> 调用微信 API         │
                            │                              │
                            │  4. cache.Set()              │
                            │     └─> 更新缓存             │
                            └──────────────────────────────┘
                                           │
                            ┌──────────────┴──────────────┐
                            │                             │
                            ▼                             ▼
                ┌────────────────────┐      ┌────────────────────┐
                │  Redis Cache       │      │  Wechat API Client │
                │  (Infrastructure)  │      │  (Infrastructure)  │
                └────────────────────┘      └────────────────────┘
```

### 用例 3: 微信登录

```
HTTP Request (JSCode)
    │
    ▼
┌────────────────────────┐
│  HTTP Handler          │
│  (Interface Layer)     │
└────────────────────────┘
    │ LoginWithCodeDTO
    ▼
┌─────────────────────────────────────────────────────────────┐
│  wechatAuthApplicationService.LoginWithCode()               │
│  (Application Layer)                                        │
│                                                              │
│  1. authenticator.LoginWithCode()  ─────────┐              │
│                                              │              │
│  2. toLoginResult()                          │              │
│                                              │              │
└──────────────────────────────────────────────┼──────────────┘
                                               │
                                               ▼
                                ┌──────────────────────────────┐
                                │  Authenticator               │
                                │  (Domain Service)            │
                                │                              │
                                │  1. 调用微信 code2Session    │
                                │                              │
                                │  2. 创建 WechatSession       │
                                │                              │
                                │  3. 加密 session_key         │
                                │                              │
                                │  4. 生成 ExternalClaim       │
                                │                              │
                                │  5. 持久化 Session           │
                                └──────────────────────────────┘
                                               │
                                ┌──────────────┴──────────────┐
                                │                             │
                                ▼                             ▼
                    ┌────────────────────┐      ┌────────────────────┐
                    │  Redis Session     │      │  Wechat API Client │
                    │  (Infrastructure)  │      │  (Infrastructure)  │
                    └────────────────────┘      └────────────────────┘
```

## 时序图：创建微信应用

```
┌─────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐     ┌─────────┐
│ Client  │     │ Handler │     │ AppService  │     │  Domain  │     │   DB    │
└────┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘     └────┬────┘
     │               │                 │                 │                │
     │  POST /apps   │                 │                 │                │
     ├──────────────>│                 │                 │                │
     │               │  CreateApp(dto) │                 │                │
     │               ├────────────────>│                 │                │
     │               │                 │  Create()       │                │
     │               │                 ├────────────────>│                │
     │               │                 │                 │  Query         │
     │               │                 │                 ├───────────────>│
     │               │                 │                 │<───────────────┤
     │               │                 │<────────────────┤                │
     │               │                 │  RotateSecret() │                │
     │               │                 ├────────────────>│                │
     │               │                 │<────────────────┤                │
     │               │                 │  repo.Create()  │                │
     │               │                 ├─────────────────┼───────────────>│
     │               │                 │                 │<───────────────┤
     │               │<────────────────┤                 │                │
     │               │  result         │                 │                │
     │<──────────────┤                 │                 │                │
     │  200 OK       │                 │                 │                │
     │               │                 │                 │                │
```

## 时序图：获取访问令牌（缓存命中）

```
┌─────────┐   ┌─────────┐   ┌────────────┐   ┌───────────┐   ┌──────┐
│ Client  │   │ Handler │   │ AppService │   │  Cacher   │   │Redis │
└────┬────┘   └────┬────┘   └─────┬──────┘   └─────┬─────┘   └──┬───┘
     │             │               │                │            │
     │  GET /token │               │                │            │
     ├────────────>│               │                │            │
     │             │ GetToken()    │                │            │
     │             ├──────────────>│                │            │
     │             │               │ EnsureToken()  │            │
     │             │               ├───────────────>│            │
     │             │               │                │ Get()      │
     │             │               │                ├───────────>│
     │             │               │                │<───────────┤
     │             │               │                │ Cache Hit  │
     │             │               │<───────────────┤            │
     │             │<──────────────┤ token          │            │
     │<────────────┤               │                │            │
     │  200 OK     │               │                │            │
```

## 时序图：获取访问令牌（缓存未命中，单飞刷新）

```
┌────────┐ ┌────────┐ ┌──────────┐ ┌────────┐ ┌──────┐ ┌────────┐
│Client1 │ │Client2 │ │AppService│ │ Cacher │ │Redis │ │Wechat  │
└───┬────┘ └───┬────┘ └────┬─────┘ └───┬────┘ └──┬───┘ └───┬────┘
    │          │            │           │         │         │
    │ GetToken │            │           │         │         │
    ├─────────────────────>│           │         │         │
    │          │ GetToken   │           │         │         │
    │          ├───────────>│           │         │         │
    │          │            │EnsureToken│         │         │
    │          │            ├──────────>│         │         │
    │          │            │           │ Get()   │         │
    │          │            │           ├────────>│         │
    │          │            │           │<────────┤         │
    │          │            │           │Miss     │         │
    │          │            │           │TryLock()│         │
    │          │            │           ├────────>│         │
    │          │            │           │<────────┤         │
    │          │            │           │OK       │         │
    │          │            │           │Fetch()  │         │
    │          │            │           ├─────────┼────────>│
    │          │            │           │<────────┼─────────┤
    │          │            │           │Set()    │         │
    │          │            │           ├────────>│         │
    │          │            │<──────────┤         │         │
    │<─────────────────────-┤token      │         │         │
    │          │            │           │TryLock()│         │
    │          │            │           ├────────>│         │
    │          │            │           │<────────┤         │
    │          │            │           │FAIL     │         │
    │          │            │           │Get()    │         │
    │          │            │           ├────────>│         │
    │          │            │           │<────────┤         │
    │          │<───────────┤token      │Hit      │         │
    │          │            │           │         │         │
```

## 说明

- **实线箭头**: 同步调用
- **虚线箭头**: 返回
- **框**: 各层组件
- **TryLock**: 分布式锁（单飞机制）
- **Cache Hit/Miss**: 缓存命中/未命中
