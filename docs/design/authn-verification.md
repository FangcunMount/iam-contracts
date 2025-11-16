# IAM AuthN Verification Architecture

本方案描述 IAM 作为统一认证中心时，其它业务系统如何以 **本地 JWT 验签 + IAM 托管能力** 的组合方式完成认证。目标是兼顾性能与安全：业务请求不必每次 RPC 到 IAM，但仍能复用 IAM 维护的密钥、策略以及吊销能力。

---

## 🎯 设计目标

1. **本地快速验签**：业务服务可以就近验证 JWT，降低延迟与对 IAM 可用性的依赖。
2. **密钥集中治理**：签名密钥/JWKS 仅由 IAM 生成、轮换与发布；外部系统不维护密钥。
3. **一致的策略判定**：与账号状态、黑名单、吊销等强业务耦合的逻辑仍由 IAM 决策，避免各系统重复实现。
4. **统一 SDK/中间件**：向各服务提供标准化的认证中间件/库，封装 JWKS 获取、缓存、fallback 以及 gRPC 调用细节。
5. **平滑演进**：与现有 REST login/token/JWKS 体系兼容，便于逐步引入 gRPC 接口。

---

## 🔧 角色与职责

| 组件 | 主要职责 |
|------|----------|
| IAM-AuthN | 发布 JWKS、发放 JWT/Refresh Token、管理吊销/黑名单、提供 gRPC `Verify/Refresh/Revoke` API、维持密钥周期。 |
| IAM SDK (Go) | 在业务服务中提供中间件：缓存 JWKS、本地验签 token、必要时调用 IAM gRPC 获取 Claim/黑名单状态。 |
| 业务服务 | 引入 SDK，在 ingress (HTTP/gRPC) 中调用统一中间件，不再直接处理密钥或策略。 |

---

## 🧱 架构概览

```text
┌───────────────┐            ┌─────────────────────┐
│  Biz Service  │            │      IAM AuthN      │
│  (HTTP/GPRC)  │            │  (REST + gRPC APIs) │
└─────┬─────────┘            └─────────┬───────────┘
      │                                 │
      │1. download/cached JWKS (HTTPS)  │
      │ <------------------------------ │
      │                                 │
      │2. inbound request               │
      │   ├─ extract JWT                │
      │   ├─ verify with cached JWKS    │
      │   ├─ optional: call Verify gRPC │
      │   └─ set context claims         │
      │                                 │
      │3. Refresh/Revoke/Login via gRPC │
      │-------------------------------->│
      │                                 │
      │4. Event subscription (可选)     │
      │<--------------------------------│
```

---

## 🔁 关键流程

### 1. 登录/颁发 (REST)

1. 客户端调用 `/api/v1/auth/login`。AuthHandler 组合 `login.LoginService` 和 `tokenIssuer` 颁发 Access/Refresh Token。
2. Refresh/Logout/Revoke 仍使用现有 REST 端点，后续可平移到 gRPC `AuthService`.

### 2. JWKS 发布与缓存

1. IAM 内部密钥调度器定期生成新 key，并通过 `/api/v1/admin/jwks/**` 管理。
2. `/.well-known/jwks.json` 暴露发布版本，附带 `ETag/Last-Modified`，供 SDK 拉取。
3. SDK 维护一个本地缓存（内存 + fallback 文件）：
   - 启动时拉取最新 JWKS；
   - 后续按 `Cache-Control` 或轮询间隔刷新；
   - 支持并发更新、ETag 对比。

### 3. 业务请求认证

1. 中间件从 Header/Cookie 解析 Bearer Token。
2. 使用本地缓存的 JWKS 验签：
   - 支持 `kid` 精确匹配；
   - 校验 `exp/nbf/aud/iss`。
3. 若验签失败或 `kid` 不存在，可触发一次 JWKS 刷新后重试。
4. 验签成功后得到基础 Claims。
5. 若需要额外安全判断（黑名单、状态变更、ban 等），中间件调用 IAM gRPC `VerifyToken`：
   - 请求包含 token、请求 ID、调用方服务名；
   - 响应返回 `Valid`、标准 Claims、黑名单/撤销标记；
   - IAM 内部会查询 Redis Blacklist、凭证状态等。
6. 中间件根据结果决定是否放行，并将 Claims 写入请求 context（供后续角色/权限使用）。

### 4. Token 刷新与撤销

- 刷新：业务服务直接调用 IAM REST/gRPC `RefreshToken`，获取新的 TokenPair。
- 撤销：当检测到异常时，调用 `RevokeToken` 或 `RevokeRefreshToken`，IAM 更新 Redis Blacklist，后续 `Verify` 会判定无效。

### 5. 事件监听（可选）

- IAM 后续可通过 `IdentityStream.SubscribeUserEvents/SubscribeGuardianshipEvents` 扩展到 AuthN：推送 token 吊销、用户禁用等事件。业务服务订阅后可主动清理本地缓存。

---

## 🧩 gRPC 接口建议

在 `api/grpc/iam/authn/v1/authn.proto`（待新增）中提供以下 service：

| Service | Method | 描述 |
|---------|--------|------|
| `AuthService` | `VerifyToken(VerifyTokenRequest)` | 在本地验签之后调用，返回黑名单/状态信息及标准化 Claims。 |
|             | `RefreshToken(RefreshTokenRequest)` | 使用 refresh token 生成新的 TokenPair。 |
|             | `RevokeToken` / `RevokeRefreshToken` | 主动吊销 token。 |
|             | `IssueServiceToken` (可选) | 给内部服务签发短期 token，供服务间调用。 |
| `JWKSService` (可选) | `GetJWKS` | 提供 gRPC 版本的 JWKS，便于服务无 HTTP client 时使用。 |

请求中需附带 `caller` 信息（服务名、实例 ID、trace ID），方便 IAM 审计与限流。

---

## 🛠️ SDK / 中间件设计

- **语言**：先实现 Go 版本，其它语言按需扩展。
- **模块职责**：
  1. `JWKSManager`：负责 `.well-known/jwks.json` 拉取、缓存、刷新、metrics。
  2. `Verifier`：调用 `JWKSManager` 做本地验证；可配置是否强制调用 `AuthService.VerifyToken`。
  3. `Middleware`：集成在 HTTP/gRPC Server，完成 token 提取、验证、Context 注入、错误处理。
  4. `Client`：封装 `AuthService` 的 gRPC Stub，支持重试/熔断。
- **配置项**：
  - JWKS endpoint、刷新间隔、最大缓存 key 数；
  - 是否强制 gRPC Verify（`always` / `on-miss` / `never`）；
  - 验签参数（允许的 issuer、audience、clock skew）。
- **熔断策略**：
  - IAM gRPC 不可用时仍允许仅凭本地验签放行（可配置），并记录告警；
  - JWKS 拉取失败时使用旧缓存，并加速重试。

---

## 🔐 安全对策

1. **Key rotation**：IAM 通过 Scheduler 定期生成新密钥，旧密钥进入 grace → retired → cleanup；JWKS 中同时包含多把 active/grace key，确保业务有时间刷新。
2. **Blacklist**：Revoke 操作将 token ID 放入 Redis Blacklist，`VerifyToken` 或 `TokenVerifyer` 会检查；业务若仅本地验签无法得知撤销，因此强烈建议关键操作前调用 `VerifyToken`。
3. **Metadata**：所有 gRPC 调用必须带 `authorization` Service Token，以及 `x-request-id`；IAM 会记录日志方便审计。
4. **Metrics/Alerts**：SDK 暴露指标如 JWKS 拉取失败次数、gRPC Verify 错误率、token 验签耗时；IAM 侧对 Verify/Revoke/Refresh 做 QPS 限制。

---

## 📚 文档/落地建议

1. **Proto & README**：在 `api/grpc/iam/authn/v1` 下定义新的 proto，并在 `api/grpc/README.md` 增补 AuthN Service 章节。
2. **SDK 文档**：在 `docs/sdk/authn.md` 编写示例，指导业务如何引入中间件、配置 JWKS、处理错误。
3. **接入 checklist**：
   - 获取服务专用 token；
   - 配置 JWKS endpoint + 刷新策略；
   - 在 ingress middleware 使用 SDK；
   - 关键操作前调用 `VerifyToken`；
   - 需要时订阅 AuthN/Identity 事件。

---

## ✅ 总结

通过“本地验签 + IAM 托管策略”的混合模式，可以：

- 保持各业务服务的低延迟；
- 让密钥、黑名单、策略仍由 IAM 集中治理；
- 暴露 gRPC 接口供刷新/吊销/验证等受控操作；
- 通过官方 SDK 统一中间件逻辑，避免每个系统重复实现认证细节。

后续即可据此补齐 proto、SDK 与运维文档，逐步替换现有 HTTP-only 集成方式。***
