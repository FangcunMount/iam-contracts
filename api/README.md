# IAM Contracts - API 文档

> **IAM Contracts** 是一个统一身份认证与访问控制系统，提供 REST 和 gRPC 两类 API

## 📚 文档概述

本目录包含 IAM 系统的完整 API 规范：

```text
api/
├── README.md                    # 本文档：API 选型指南与概述
├── rest/                        # RESTful API 规范 (OpenAPI 3.1)
│   ├── authn.v1.yaml           # 认证 API：登录、令牌、账户管理
│   └── identity.v1.yaml        # 身份 API：用户、儿童、监护关系
└── grpc/                        # gRPC API 规范 (Protocol Buffers)
    ├── iam.authz.v1.proto      # 授权服务：权限判定、批量校验
    └── iam.identity.v1.proto   # 身份查询：用户、儿童、监护关系
```

---

## 🎯 核心能力

### 1. 认证中心 (AuthN)

- **账户管理**: 运营账户、微信账户的创建、绑定、状态管理
- **认证服务**: 用户名密码登录、微信登录、OAuth 集成
- **令牌服务**: JWT 颁发、刷新、验证、撤销
- **公钥服务**: JWKS 端点（符合 RFC 7517）

### 2. 用户中心 (Identity)

- **用户管理**: 用户档案的创建、查询、更新
- **儿童档案**: 儿童信息的注册、建档、查询
- **监护关系**: 监护人授权、撤销、查询

### 3. 授权中心 (AuthZ)

- **权限判定**: 基于 RBAC + ABAC 的访问控制
- **批量校验**: 支持批量权限判定（优化性能）
- **策略解释**: 提供权限决策的详细解释

---

## 🔀 协议选型指南

### 何时使用 REST API？

**适用场景**:

- ✅ 前端应用（Web、H5、小程序）
- ✅ 运营后台管理
- ✅ 外部系统集成
- ✅ 命令行工具/脚本
- ✅ 需要可读性和调试友好

**核心能力**:

| 功能域 | API 端点 | 说明 |
|--------|----------|------|
| **认证** | `POST /api/v1/auth/login` | 用户登录 |
| | `POST /api/v1/auth/refresh` | 刷新令牌 |
| | `POST /api/v1/auth/logout` | 退出登录 |
| **账户** | `POST /api/v1/accounts/operation` | 创建运营账号 |
| | `POST /api/v1/accounts/wechat/bind` | 绑定微信 |
| **用户** | `POST /api/v1/users` | 创建用户 |
| | `GET /api/v1/users/{userId}` | 查询用户 |
| | `PATCH /api/v1/users/{userId}` | 更新用户 |
| **儿童** | `POST /api/v1/children/register` | 注册儿童（含监护授权） |
| | `GET /api/v1/me/children` | 我的孩子列表 |
| | `GET /api/v1/children/{childId}` | 查询儿童档案 |
| **监护** | `POST /api/v1/guardians/grant` | 授予监护关系 |
| | `POST /api/v1/guardians/revoke` | 撤销监护关系 |
| | `GET /api/v1/guardians` | 查询监护关系 |

---

### 何时使用 gRPC API？

**适用场景**:

- ✅ 微服务间调用（高性能、低延迟）
- ✅ 高频权限判定（PDP 策略决策点）
- ✅ 监护关系查询（读侧优化）
- ✅ 批量操作（减少网络往返）

**核心能力**:

| 服务 | RPC 方法 | 说明 |
|------|----------|------|
| **AuthZ** | `Allow(AllowReq) → AllowResp` | 单次权限判定 |
| | `AllowOnActor(AllowOnActorReq)` | 基于 Actor 的权限判定 |
| | `BatchAllow(BatchAllowReq)` | 批量权限判定 |
| | `Explain(ExplainReq)` | 权限决策解释 |
| **IdentityRead** | `GetUser(GetUserReq)` | 查询用户信息 |
| | `GetChild(GetChildReq)` | 查询儿童档案 |
| **GuardianshipQuery** | `IsGuardian(IsGuardianReq)` | 判定监护关系 |
| | `ListChildren(ListChildrenReq)` | 列出监护儿童 |

---

## 🔐 安全与通用约定

### 认证机制

| 协议 | 认证方式 | 传输方式 |
|------|---------|---------|
| **REST** | `Authorization: Bearer <JWT>` | HTTPS (TLS 1.2+) |
| **gRPC** | `authorization` metadata | mTLS (双向认证) |

### 幂等性保证

**REST API**:

- 所有 `POST` 请求支持 `X-Idempotency-Key` header
- 使用 UUID v4 作为幂等键
- 服务端保证 24 小时内相同幂等键返回相同结果

**gRPC API**:

- 调用方负责实现重试逻辑
- 使用唯一请求 ID（`x-request-id` metadata）
- 服务端保证操作语义幂等

### 请求追踪

| Header/Metadata | 格式 | 说明 |
|-----------------|------|------|
| `X-Request-Id` (REST) | UUID v4 | 链路追踪 ID |
| `x-request-id` (gRPC) | UUID v4 | 同上 |
| `X-Forwarded-For` | IP 地址 | 真实客户端 IP |

### 错误处理

**REST API** - 符合 RFC 7807 (Problem Details):

```json
{
  "code": "IAM-1001",
  "message": "用户名或密码错误",
  "reference": "https://api.example.com/docs/errors/IAM-1001"
}
```

**gRPC API** - 符合 Google API 设计指南:

```protobuf
status {
  code: 3  // INVALID_ARGUMENT
  message: "user_id is required"
  details {
    type_url: "type.googleapis.com/google.rpc.BadRequest"
    value: ...
  }
}
```

---

## 📖 详细文档

### REST API 文档

- [**认证 API (authn.v1.yaml)**](./rest/authn.v1.yaml)
  - 登录流程（用户名密码、微信）
  - 令牌管理（颁发、刷新、验证、撤销）
  - 账户管理（创建、绑定、状态控制）
  - JWKS 公钥集（符合 RFC 7517）

- [**身份 API (identity.v1.yaml)**](./rest/identity.v1.yaml)
  - 用户管理（CRUD）
  - 儿童档案（注册、建档、查询）
  - 监护关系（授予、撤销、查询）

### gRPC API 文档

- [**授权服务 (iam.authz.v1.proto)**](./grpc/iam.authz.v1.proto)
  - 权限判定（Allow、AllowOnActor）
  - 批量判定（BatchAllow）
  - 策略解释（Explain）

- [**身份查询 (iam.identity.v1.proto)**](./grpc/iam.identity.v1.proto)
  - 用户查询（GetUser）
  - 儿童查询（GetChild）
  - 监护判定（IsGuardian、ListChildren）

---

## 🚀 快速开始

### REST API 示例

```bash
# 1. 登录获取令牌
curl -X POST https://api.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "account_type": "operation",
    "username": "admin",
    "password": "your_password"
  }'

# 响应
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 86400,
  "refresh_token": "def50200..."
}

# 2. 使用令牌查询用户
curl -X GET https://api.example.com/api/v1/users/usr_123456 \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

# 3. 查询我的孩子
curl -X GET https://api.example.com/api/v1/me/children \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### gRPC API 示例

```go
// 1. 连接 gRPC 服务
conn, err := grpc.Dial("api.example.com:9090",
    grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
)
defer conn.Close()

// 2. 权限判定
authzClient := authzv1.NewAuthZClient(conn)
resp, err := authzClient.Allow(ctx, &authzv1.AllowReq{
    UserId:   "usr_123456",
    Resource: "answersheet",
    Action:   "submit",
    Scope: &authzv1.Scope{
        Type: "questionnaire",
        Id:   "PHQ9",
    },
})

// 3. 监护关系判定
identityClient := identityv1.NewGuardianshipQueryClient(conn)
isGuardian, err := identityClient.IsGuardian(ctx, &identityv1.IsGuardianReq{
    UserId:  "usr_123456",
    ChildId: "chd_789",
})
```

---

## 🔧 开发工具

### API 文档查看

**REST API**:

- 在线查看: [Swagger UI](https://api.example.com/swagger)
- 本地查看: `make swagger-ui`
- VSCode 插件: [OpenAPI (Swagger) Editor](https://marketplace.visualstudio.com/items?itemName=42Crunch.vscode-openapi)

**gRPC API**:

- 生成文档: `make proto-doc`
- 交互测试: [BloomRPC](https://github.com/bloomrpc/bloomrpc) / [Postman](https://www.postman.com/)
- VSCode 插件: [vscode-proto3](https://marketplace.visualstudio.com/items?itemName=zxh404.vscode-proto3)

### 代码生成

```bash
# REST API Mock Server
make rest-mock

# gRPC 客户端/服务端代码
make proto-gen

# TypeScript/JavaScript SDK
make sdk-ts

# Go SDK
make sdk-go
```

---

## 📞 支持与反馈

- **文档问题**: [GitHub Issues](https://github.com/FangcunMount/iam-contracts/issues)
- **API 变更**: 查看 [CHANGELOG.md](../CHANGELOG.md)
- **技术支持**: <api-support@example.com>

---

## 📄 许可证

本项目遵循 [MIT License](../LICENSE)
