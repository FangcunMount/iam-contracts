# 基础设施层文件清单

## 总览

基础设施层包含 **8 个 Go 源文件** 和 **3 个文档/配置文件**，分为 4 个子包。

## 文件结构

```
infra/
├── infrastructure.go              # 基础设施服务聚合器（主入口）
├── README.md                      # 架构说明和使用指南
├── REFACTORING_SUMMARY.md         # 重构总结
│
├── mysql/                         # MySQL 持久化
│   ├── schema.sql                 # 数据库表结构
│   ├── wechatapp_po.go           # 微信应用持久化对象（PO）
│   └── wechatapp_repository.go   # 微信应用仓储实现
│
├── redis/                         # Redis 缓存
│   ├── accesstoken_cache.go      # 访问令牌缓存
│   └── wechatsession_repository.go # 微信会话仓储
│
├── crypto/                        # 加密服务
│   └── secret_vault.go           # 密钥加密（AES-GCM）
│
└── wechatapi/                     # 微信 API 集成
    ├── code2session_client.go    # 微信认证服务
    └── token_provider.go         # 访问令牌提供器
```

## 文件详情

### 主入口

| 文件 | 行数 | 功能 | 编译状态 |
|------|------|------|----------|
| `infrastructure.go` | ~120 | 聚合所有基础设施服务，提供统一初始化入口 | ✅ 通过 |

### MySQL 持久化（3 个文件）

| 文件 | 行数 | 功能 | 编译状态 |
|------|------|------|----------|
| `mysql/schema.sql` | ~50 | 数据库表结构定义 | N/A |
| `mysql/wechatapp_po.go` | ~70 | 持久化对象（PO）及映射逻辑 | ✅ 通过 |
| `mysql/wechatapp_repository.go` | ~110 | 仓储实现（CRUD 操作） | ✅ 通过 |

**提供的接口：**
- `port.WechatAppRepository`: 微信应用仓储

**关键功能：**
- 领域对象 ↔ PO 映射
- 数据库 CRUD 操作
- 查询优化（索引支持）

### Redis 缓存（2 个文件）

| 文件 | 行数 | 功能 | 编译状态 |
|------|------|------|----------|
| `redis/accesstoken_cache.go` | ~114 | 访问令牌缓存，支持分布式锁 | ✅ 通过 |
| `redis/wechatsession_repository.go` | ~120 | 微信会话存储，支持 TTL | ✅ 通过 |

**提供的接口：**
- `port.AccessTokenCache`: 访问令牌缓存
- `port.WechatSessionRepository`: 微信会话仓储

**关键功能：**
- 分布式锁（防止并发刷新令牌）
- TTL 自动过期
- JSON 序列化存储

### 加密服务（1 个文件）

| 文件 | 行数 | 功能 | 编译状态 |
|------|------|------|----------|
| `crypto/secret_vault.go` | ~94 | AES-256-GCM 加密服务 | ✅ 通过 |

**提供的接口：**
- `port.SecretVault`: 密钥加密服务

**关键功能：**
- AES-256-GCM 认证加密
- 随机 nonce 生成
- 防篡改验证

### 微信 API 集成（2 个文件）

| 文件 | 行数 | 功能 | 编译状态 |
|------|------|------|----------|
| `wechatapi/code2session_client.go` | ~110 | 微信登录认证（code2session、手机号解密） | ✅ 通过 |
| `wechatapi/token_provider.go` | ~105 | 访问令牌获取（小程序、公众号） | ✅ 通过 |

**提供的服务：**
- `AuthService`: 微信认证服务
- `TokenProvider`: 访问令牌提供器

**关键功能：**
- 使用 silenceper/wechat/v2 SDK
- **不依赖领域对象**（使用原始类型）
- 支持小程序和公众号

## 架构特点

### 1. 六边形架构

```
┌─────────────────────────────────────┐
│        领域层 (Domain)               │
│     定义端口接口 (Port)               │
└─────────────────────────────────────┘
                 ▲
                 │ 实现接口
                 │
┌─────────────────────────────────────┐
│   基础设施层 (Infrastructure)         │
│                                     │
│  ┌──────────────────────────────┐  │
│  │  infrastructure.go           │  │ ← 入口
│  │  - NewInfrastructureServices │  │
│  └──────────────────────────────┘  │
│                                     │
│  ┌──────┐  ┌──────┐  ┌──────┐     │
│  │MySQL │  │Redis │  │Crypto│     │
│  └──────┘  └──────┘  └──────┘     │
│                                     │
│  ┌────────────────────────────┐   │
│  │    wechatapi               │   │
│  │  - AuthService             │   │
│  │  - TokenProvider           │   │
│  └────────────────────────────┘   │
└─────────────────────────────────────┘
```

### 2. 依赖注入

所有基础设施服务通过 `NewInfrastructureServices()` 统一初始化：

```go
deps := &InfrastructureDependencies{
    DB:             gormDB,
    RedisClient:    redisClient,
    EncryptionKey:  encryptionKey,
    WechatSDKCache: wechatCache,
}
infraServices, err := NewInfrastructureServices(deps)
```

### 3. 接口隔离

每个子系统实现特定的端口接口：

| 子系统 | 实现的接口 | 定义位置 |
|--------|-----------|---------|
| MySQL | `WechatAppRepository` | `domain/wechatapp/port/driven.go` |
| Redis | `AccessTokenCache` | `domain/wechatapp/port/driven.go` |
| Redis | `WechatSessionRepository` | `domain/wechatsession/port/driven.go` |
| Crypto | `SecretVault` | `domain/wechatapp/port/driven.go` |

### 4. 无领域依赖

微信 API 集成（`wechatapi/`）**完全不依赖领域对象**：

```go
// ✅ 使用原始类型
func (s *AuthService) Code2Session(
    ctx context.Context,
    appID string,      // 而不是 app *domain.WechatApp
    appSecret string,
    jsCode string,
) (*Code2SessionResult, error)
```

## 技术栈

| 组件 | 版本 | 用途 |
|------|------|------|
| GORM | v2 | MySQL ORM |
| go-redis | v9 | Redis 客户端 |
| silenceper/wechat | v2 | 微信 SDK |
| crypto/aes | std | AES 加密 |

## 测试覆盖率

### 单元测试状态

| 文件 | 测试文件 | 覆盖率 | 状态 |
|------|---------|-------|------|
| `wechatapp_repository.go` | 待创建 | 0% | ⏳ 待实现 |
| `accesstoken_cache.go` | 待创建 | 0% | ⏳ 待实现 |
| `wechatsession_repository.go` | 待创建 | 0% | ⏳ 待实现 |
| `secret_vault.go` | 待创建 | 0% | ⏳ 待实现 |
| `code2session_client.go` | 待创建 | 0% | ⏳ 待实现 |
| `token_provider.go` | 待创建 | 0% | ⏳ 待实现 |

### 集成测试

- [ ] MySQL 持久化集成测试
- [ ] Redis 缓存集成测试
- [ ] 微信 API Mock 测试
- [ ] 端到端流程测试

## 使用示例

### 初始化

```go
package main

import (
    "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/infra"
)

func main() {
    // 准备依赖
    deps := &infra.InfrastructureDependencies{
        DB:             setupMySQL(),
        RedisClient:    setupRedis(),
        EncryptionKey:  loadEncryptionKey(),
        WechatSDKCache: nil, // SDK 使用内存缓存
    }
    
    // 创建基础设施服务
    infraServices, err := infra.NewInfrastructureServices(deps)
    if err != nil {
        log.Fatal(err)
    }
    
    // 传递给领域层或应用层
    domainServices := domain.NewDomainServices(infraServices)
}
```

### 使用仓储

```go
// 查询微信应用
app, err := infraServices.WechatAppRepository.GetByAppID(
    ctx,
    "wx1234567890abcdef",
)
```

### 使用缓存

```go
// 缓存访问令牌
token := &domain.AppAccessToken{
    Token:     "ACCESS_TOKEN",
    ExpiresAt: time.Now().Add(2 * time.Hour),
}
err := infraServices.AccessTokenCache.Set(ctx, "wx123", token)
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
```

## 编译验证

所有文件编译通过：

```bash
✅ infrastructure.go          - No errors
✅ mysql/wechatapp_po.go      - No errors
✅ mysql/wechatapp_repository.go - No errors
✅ redis/accesstoken_cache.go - No errors
✅ redis/wechatsession_repository.go - No errors
✅ crypto/secret_vault.go     - No errors
✅ wechatapi/code2session_client.go - No errors
✅ wechatapi/token_provider.go - No errors
```

## 后续工作

### 1. 完善文档

- [ ] 为每个子包创建 `README.md`
- [ ] 添加 godoc 注释示例
- [ ] 创建 UML 类图

### 2. 编写测试

- [ ] 单元测试（每个文件 80% 覆盖率）
- [ ] 集成测试（真实数据库/Redis）
- [ ] Mock 测试（微信 API）

### 3. 性能优化

- [ ] 数据库查询优化（索引分析）
- [ ] Redis 连接池配置
- [ ] 缓存预热策略

### 4. 监控和日志

- [ ] 添加 Prometheus 指标
- [ ] 结构化日志（zerolog/zap）
- [ ] 分布式追踪（OpenTelemetry）

## 相关文档

- [基础设施层 README](./README.md) - 架构说明和使用指南
- [重构总结](./REFACTORING_SUMMARY.md) - 重构过程和改进点
- [IDP 模块总览](../README.md) - 模块整体架构
- [领域层文档](../domain/README.md) - 领域模型和端口定义
- [应用层文档](../application/README.md) - 应用服务和 DTO

## 贡献指南

### 添加新的仓储

1. 在 `mysql/` 创建 PO 和仓储实现
2. 在领域层 `port/driven.go` 定义接口
3. 在 `infrastructure.go` 中注册服务

### 添加新的外部 API

1. 在 `infra/` 创建新的子包
2. **使用原始类型**作为输入输出
3. 创建 Result 结构体封装返回数据
4. 在 `infrastructure.go` 中聚合服务

---

**最后更新：** [当前日期]  
**状态：** ✅ 重构完成，所有文件编译通过  
**架构合规性：** ✅ 符合六边形架构原则
