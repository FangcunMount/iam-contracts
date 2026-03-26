# 认证、Token、JWKS

本文回答：`iam-contracts` 的认证域当前到底落地了哪些对象和主链路，Token/JWKS 如何工作，哪些能力已经可讲成现状，哪些还不能讲过头。

## 30 秒结论

- 当前认证域的真实主轴是：`Account / Credential / Authenticater / Principal / Token / JWKS`。
- 当前登录主链是：`REST /authn/login -> prepareAuthentication -> Authenticater -> Principal -> TokenIssuer -> AccessToken(JWT) + RefreshToken(Redis)`。
- 当前访问令牌是 `RS256` JWT，刷新令牌是 Redis 中保存的随机 UUID；刷新时会轮换旧 refresh token。
- 当前 gRPC 认证面主要是 `VerifyToken / RefreshToken / RevokeToken / RevokeRefreshToken / GetJWKS`；`IssueServiceToken` 仍未实现。
- 当前不能讲成现状的两点是：没有独立 `Session` 聚合；Token Claims 也没有稳定承载角色/权限等完整授权上下文。

## 重点速查

| 关注点 | 当前答案 | 真实落点 |
| ---- | ---- | ---- |
| 账户对象 | 登录账户，关联内部 `User` 与外部身份入口 | [../../internal/apiserver/domain/authn/account/account.go](../../internal/apiserver/domain/authn/account/account.go) |
| 凭据对象 | 独立存储密码、OTP、微信、企业微信认证材料 | [../../internal/apiserver/domain/authn/credential/credential.go](../../internal/apiserver/domain/authn/credential/credential.go) |
| 认证场景 | `password / phone_otp / oauth_wx_minip / oauth_wecom / jwt_token` | [../../internal/apiserver/domain/authn/authentication/types.go](../../internal/apiserver/domain/authn/authentication/types.go) |
| 认证器 | `Authenticater` 负责按场景挑策略并返回 `AuthDecision` | [../../internal/apiserver/domain/authn/authentication/authenticater.go](../../internal/apiserver/domain/authn/authentication/authenticater.go) |
| Token 颁发 | 访问令牌走 JWT+JWKS，刷新令牌落 Redis | [../../internal/apiserver/domain/authn/token/issuer.go](../../internal/apiserver/domain/authn/token/issuer.go)、[../../internal/apiserver/infra/redis/token-store.go](../../internal/apiserver/infra/redis/token-store.go) |
| Token 验证 | 验签 + 过期检查 + 黑名单检查 | [../../internal/apiserver/domain/authn/token/verifyer.go](../../internal/apiserver/domain/authn/token/verifyer.go) |
| JWT 生成 | `RS256`、Header 带 `kid` | [../../internal/apiserver/infra/jwt/generator.go](../../internal/apiserver/infra/jwt/generator.go) |
| JWKS 密钥 | `active / grace / retired` 三态 | [../../internal/apiserver/domain/authn/jwks/key.go](../../internal/apiserver/domain/authn/jwks/key.go) |
| gRPC 服务 | `AuthService + JWKSService` 的一部分实现已落地 | [../../internal/apiserver/interface/authn/grpc/service.go](../../internal/apiserver/interface/authn/grpc/service.go)、[../../api/grpc/iam/authn/v1/authn.proto](../../api/grpc/iam/authn/v1/authn.proto) |

## 1. 当前模型

### 1.1 `Account`

`Account` 当前表示“可登录账户”，不是“用户档案”，也不是“Session 根聚合”。它当前主要维护：

- `UserID`
- `Type`
- `AppID`
- `ExternalID`
- `UniqueID`
- `Profile / Meta`
- `Status`

关键事实：

- `Account` 不内嵌凭据列表
- `Account` 不内嵌会话列表
- `User` 仍属于用户域，不在认证域内部重建

### 1.2 `Credential`

`Credential` 是独立实体，主要存：

- `AccountID`
- `IDP / IDPIdentifier / AppID`
- `Material / Algo / ParamsJSON`
- `Status / FailedAttempts / LockedUntil`

当前支持的主要类型：

- `password`
- `phone_otp`
- `oauth_wx_minip`
- `oauth_wecom`

关键事实：

- 密码材料落库存储
- 手机 OTP 本身不落库
- OAuth 场景主要存外部身份标识和扩展参数

### 1.3 `Principal`

认证成功后的统一输出是 `Principal`，包含：

- `AccountID`
- `UserID`
- `TenantID`
- `AMR`
- `Claims`

当前 Token 颁发直接消费 `Principal`，而不是先创建一个独立 `Session` 聚合。

## 2. 当前认证主链

### 2.1 登录

当前 REST 登录入口在 [../../internal/apiserver/interface/authn/restful/router.go](../../internal/apiserver/interface/authn/restful/router.go)，核心接口包括：

- `POST /api/v1/authn/login`
- `POST /api/v1/authn/refresh_token`
- `POST /api/v1/authn/logout`
- `POST /api/v1/authn/verify`

登录主链见 [../../internal/apiserver/application/authn/login/services_impl.go](../../internal/apiserver/application/authn/login/services_impl.go)：

1. 根据请求字段推断认证场景
2. 构造 `AuthInput`
3. 调用 `Authenticater.Authenticate`
4. 认证成功后拿到 `Principal`
5. 调用 `TokenIssuer.IssueToken`
6. 返回 `TokenPair`

### 2.2 刷新与撤销

当前刷新链见 [../../internal/apiserver/domain/authn/token/refresher.go](../../internal/apiserver/domain/authn/token/refresher.go)：

1. 从 Redis 读取 refresh token
2. 检查是否存在、是否过期
3. 恢复最小 `Principal`
4. 签发新的 TokenPair
5. 删除旧 refresh token

撤销链当前分两种：

- access token：加入 Redis 黑名单
- refresh token：直接从 Redis 删除

### 2.3 gRPC

当前 gRPC 服务实现见 [../../internal/apiserver/interface/authn/grpc/service.go](../../internal/apiserver/interface/authn/grpc/service.go)。

当前真正已注册并实现的主要 RPC：

- `VerifyToken`
- `RefreshToken`
- `RevokeToken`
- `RevokeRefreshToken`
- `GetJWKS`

当前不能讲成现状的点：

- `IssueServiceToken` 在 proto 里存在，但运行时返回 `Unimplemented`

## 3. 当前 Token 与 JWKS

### 3.1 Token

当前访问令牌由 [../../internal/apiserver/infra/jwt/generator.go](../../internal/apiserver/infra/jwt/generator.go) 生成：

- 算法：`RS256`
- Header：带 `kid`
- Claims 当前稳定可证明的内容：
  - `sub`
  - `iss`
  - `iat`
  - `exp`
  - `user_id`
  - `account_id`

当前刷新令牌由 [../../internal/apiserver/domain/authn/token/issuer.go](../../internal/apiserver/domain/authn/token/issuer.go) 创建并存入 Redis。

如果配置未显式给出 TTL，当前装配默认值在 [../../internal/apiserver/container/assembler/authn.go](../../internal/apiserver/container/assembler/authn.go)：

- access token：15 分钟
- refresh token：7 天

### 3.2 JWKS

当前 JWKS 密钥模型见 [../../internal/apiserver/domain/authn/jwks/key.go](../../internal/apiserver/domain/authn/jwks/key.go)：

| 状态 | 当前语义 |
| ---- | ---- |
| `active` | 可签名、可验签、可发布 |
| `grace` | 不再签名，但继续验签并发布 |
| `retired` | 退役，不再发布 |

当前还具备：

- 启动时自动初始化首个活跃密钥
- Cron 版密钥轮换调度器

对应落点：

- [../../internal/apiserver/container/assembler/authn.go](../../internal/apiserver/container/assembler/authn.go)
- [../../internal/apiserver/infra/scheduler/key_rotation_cron_scheduler.go](../../internal/apiserver/infra/scheduler/key_rotation_cron_scheduler.go)
- [../../configs/keys/README.md](../../configs/keys/README.md)

## 4. 当前安全边界

当前已能稳定讲成现状的安全能力：

- `Argon2id` 密码哈希  
  [../../internal/apiserver/infra/crypto/argon2_hasher.go](../../internal/apiserver/infra/crypto/argon2_hasher.go)
- JWT 验签
- 访问令牌黑名单
- 刷新令牌轮换
- JWKS 公钥发布与密钥轮换

当前不能讲过头的地方：

1. 没有独立 `Session` 聚合与会话仓储
2. 中间件里 `RequireRole / RequirePermission` 仍是 stub  
   [../../internal/pkg/middleware/authn/jwt_middleware.go](../../internal/pkg/middleware/authn/jwt_middleware.go)
3. 不能把 proto 里所有 Claims 预留字段都讲成当前运行时已稳定输出

## 5. 当前最准确的口径

如果只用一句话概括当前认证域，我会这样讲：

`iam-contracts` 当前已经形成了“多场景认证 + JWT/Refresh Token 生命周期 + JWKS 密钥管理”这条主线，但它还不是一个以 Session 为中心的完整认证平台。`

## 6. 继续往下读

1. [../03-接口与集成/01-REST契约与接入.md](../03-接口与集成/01-REST契约与接入.md)
2. [../03-接口与集成/02-gRPC契约与接入.md](../03-接口与集成/02-gRPC契约与接入.md)
3. [../05-专题分析/01-认证链路：从登录请求到 Token 与 JWKS.md](../05-专题分析/01-认证链路：从登录请求到 Token 与 JWKS.md)
4. [../../api/rest/authn.v1.yaml](../../api/rest/authn.v1.yaml)
5. [../../api/grpc/iam/authn/v1/authn.proto](../../api/grpc/iam/authn/v1/authn.proto)
