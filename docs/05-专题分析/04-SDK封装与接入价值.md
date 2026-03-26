# SDK 封装与接入价值

本文回答：为什么 `pkg/sdk` 应该被视为 `iam-contracts` 的主阅读路径之一，它当前到底封装了什么复杂度，什么场景优先看 SDK，什么场景仍应直接回到 REST / gRPC 契约，以及当前还不能讲过头的边界在哪里。

**与业务域、契约的分工**：`02-业务域` 描述 **iam-apiserver 内**模块与能力；`api/rest`、`api/grpc` 为字段与接口真值；**本篇**说明 **`pkg/sdk` 如何收口接入复杂度**。须牢记：**SDK 暴露方法 ≠ 服务端已实现**，以下表与 [`interface/authn/grpc/service.go`](../../internal/apiserver/interface/authn/grpc/service.go) 为准。

## 30 秒结论

- `pkg/sdk` 不是一个薄薄的 client wrapper，而是 `iam-contracts` 把“如何安全、稳定、低成本地消费 IAM 能力”收口给业务方的统一接入层。
- 当前 SDK 已经把多类高频接入复杂度收进来了：统一 gRPC 客户端、JWT 本地验签与 JWKS 管理、服务间认证 Token 生命周期、身份 / 监护 / IDP 的便捷客户端、授权 PDP（`Authz().Check`）。
- 因此对业务系统来说，今天更推荐的顺序不是“先读全部 proto / OpenAPI”，而是“先看 SDK 能不能直接承载接入场景，再回到契约层确认字段与边界”。
- 但 SDK 也不是新的真值层：字段、服务名、错误码和兼容性仍以 `api/rest/*.yaml`、`api/grpc/**/*.proto` 为准。
- 授权 PDP：已有 `client.Authz()`，封装 `iam.authz.v1.AuthorizationService/Check`（Casbin Enforce）；批量判定、缓存策略仍以业务与契约为准。

## 重点速查

| 想回答的问题 | 先打开哪里 |
| ---- | ---- |
| SDK 今天到底提供了哪些统一入口？ | [../../pkg/sdk/sdk.go](../../pkg/sdk/sdk.go)、[../../pkg/sdk/docs/README.md](../../pkg/sdk/docs/README.md) |
| 用户 JWT 更推荐怎么接？ | [../../pkg/sdk/docs/03-jwt-verification.md](../../pkg/sdk/docs/03-jwt-verification.md)、[../../pkg/sdk/auth/verifier.go](../../pkg/sdk/auth/verifier.go) |
| 服务间认证应该先看哪里？ | [../../pkg/sdk/docs/04-service-auth.md](../../pkg/sdk/docs/04-service-auth.md)、[../../pkg/sdk/auth/service_auth.go](../../pkg/sdk/auth/service_auth.go) |
| 身份 / 监护关系查询更适合怎么接？ | [../../pkg/sdk/identity/client.go](../../pkg/sdk/identity/client.go)、[../../pkg/sdk/identity/guardianship.go](../../pkg/sdk/identity/guardianship.go) |
| SDK 和 REST / gRPC 契约到底怎么分工？ | [../03-接口与集成/01-REST契约与接入.md](../03-接口与集成/01-REST契约与接入.md)、[../03-接口与集成/02-gRPC契约与接入.md](../03-接口与集成/02-gRPC契约与接入.md) |
| 服务间做策略判定（PDP）怎么接？ | [../../pkg/sdk/authz/client.go](../../pkg/sdk/authz/client.go)、[../../api/grpc/iam/authz/v1/authz.proto](../../api/grpc/iam/authz/v1/authz.proto)、[../03-接口与集成/03-授权接入与边界.md](../03-接口与集成/03-授权接入与边界.md) |

## SDK 封装与 `iam-apiserver` gRPC 行为对照

接入前请对照：**客户端有方法**只表示 **stub 可调**；**服务端**是否在 [`internal/apiserver/interface`](../../internal/apiserver/interface) 中实现、是否返回 `Unimplemented`，需单独核对。

| SDK 入口（概括） | 服务端当前行为（概括） | 说明 |
| ---------------- | ---------------------- | ---- |
| `Auth().VerifyToken` / `RefreshToken` / `RevokeToken` / `RevokeRefreshToken` / `GetJWKS` | 模块装配完整则正常；**token/jwks 服务未注入**时返回 `Unimplemented` | [`interface/authn/grpc/service.go`](../../internal/apiserver/interface/authn/grpc/service.go) |
| `Auth().IssueServiceToken` | **恒返回 `Unimplemented`**（`issue service token not supported`） | [`IssueServiceToken`](../../internal/apiserver/interface/authn/grpc/service.go) 为桩；**`ServiceAuthHelper` 等依赖该 RPC 时，当前不可用**，需扩展服务端或调整接入设计 |
| `Authz().Check` | `Enforce`；`casbin == nil` 时 `Unavailable` | 与 [02-授权判定链路](./02-授权判定链路：角色、策略、资源、Assignment、Casbin.md) 一致 |
| `Identity()` / `Guardianship()` 等 | 以各 Service 注册与 `service_impl` 为准 | 部分 RPC 仍为占位，见 [03-监护关系链路](./03-监护关系链路：用户、儿童、Guardianship 的协作.md) |

## 1. 为什么这篇更适合放在专题分析层

`iam-contracts` 的价值不只在“定义了合同”，还在“业务系统能否低摩擦地把这些合同接起来”。

单看 `api/rest` 和 `api/grpc`，接入方仍要自己处理很多事情：

- 怎么建连
- TLS / mTLS 怎么配
- metadata / request-id 怎么带
- JWT 是本地验签还是远程校验
- JWKS 怎么刷新、怎么降级
- 服务 Token 什么时候刷新、失败后怎么退避
- 常用身份查询是否要每次都手写 gRPC request

SDK 的价值就在这里：它把这些“接 IAM 时反复出现、又不该在每个业务仓库里重写一遍”的事情先收成统一入口。

因此这篇更像“跨层判断专题”，而不是单纯的接口说明：

- 一头连着 `pkg/sdk` 这组真实代码和 SDK 文档
- 一头连着 REST / gRPC 契约与接入层
- 中间回答“为什么 SDK 不是薄 wrapper、它到底替业务方省了什么”

这类判断更适合放在 `05-专题分析/`。

## 2. 当前 SDK 实际封了什么

### 2.1 统一客户端入口

当前统一入口是 `sdk.Client`：

- `Auth()`
- `Authz()`
- `Identity()`
- `Guardianship()`
- `IDP()`

证据：

- [../../pkg/sdk/sdk.go](../../pkg/sdk/sdk.go)

这意味着业务方通常只需要先建一个 `sdk.NewClient(...)`，再从统一入口拿子客户端，而不是分别维护多套 gRPC 连接和初始化逻辑。

### 2.2 认证消费层

SDK 当前已经把认证消费面的高频能力收口为：

- `Auth().VerifyToken(...)`
- `Auth().RefreshToken(...)`
- `Auth().RevokeToken(...)`
- `Auth().IssueServiceToken(...)`
- `Auth().GetJWKS(...)`

证据：

- [../../pkg/sdk/auth/client.go](../../pkg/sdk/auth/client.go)

**注意**：`IssueServiceToken` 在 SDK 中存在，但 **服务端当前固定 `Unimplemented`**（见上表）；服务间认证若走该 RPC，须先确认部署的是否为已实现版本或改用其他凭证策略。

这部分让“调用认证服务”不必从业务仓库直接面对原始 gRPC stub。

### 2.3 JWT 本地验签与 JWKS 管理

SDK 当前不仅能“远程校验 token”，还额外封装了：

- `sdk.NewTokenVerifier(...)`
- `sdk.NewJWKSManager(...)`
- `sdk.NewJWKSManagerWithClient(...)`

证据：

- [../../pkg/sdk/sdk.go](../../pkg/sdk/sdk.go)
- [../../pkg/sdk/auth/verifier.go](../../pkg/sdk/auth/verifier.go)
- [../../pkg/sdk/docs/03-jwt-verification.md](../../pkg/sdk/docs/03-jwt-verification.md)

这部分真正屏蔽的复杂度包括：

- 本地 JWKS 验签
- 远程验证降级
- audience / issuer / claim 校验
- JWKS 获取与刷新链路

### 2.4 服务间认证生命周期

SDK 还封了 `ServiceAuthHelper`：

- 申请服务 Token
- 提前刷新
- 失败退避
- 熔断与降级
- 用认证上下文执行调用

证据：

- [../../pkg/sdk/sdk.go](../../pkg/sdk/sdk.go)
- [../../pkg/sdk/auth/service_auth.go](../../pkg/sdk/auth/service_auth.go)
- [../../pkg/sdk/docs/04-service-auth.md](../../pkg/sdk/docs/04-service-auth.md)

这部分不是简单“帮你调用一次 `IssueServiceToken`”，而是把整个服务 Token 生命周期管理抽出来了。

### 2.5 身份、监护、IDP 的便捷客户端

今天 SDK 已经给出便捷客户端的领域主要是：

| 子客户端 | 当前覆盖 |
| ---- | ---- |
| `Authz()` | PDP：`Check` / `CheckRequest`（`AuthorizationService`） |
| `Identity()` | 用户、孩子的读取与生命周期操作 |
| `Guardianship()` | 监护关系查询与命令操作 |
| `IDP()` | 微信应用查询 |

证据：

- [../../pkg/sdk/authz/client.go](../../pkg/sdk/authz/client.go)
- [../../pkg/sdk/identity/client.go](../../pkg/sdk/identity/client.go)
- [../../pkg/sdk/identity/guardianship.go](../../pkg/sdk/identity/guardianship.go)
- [../../pkg/sdk/idp/client.go](../../pkg/sdk/idp/client.go)

### 2.6 统一配置、传输与可观测性支撑

SDK 不只是一组业务方法，还收了：

- `config`
- `transport`
- `observability`
- `errors`

证据：

- [../../pkg/sdk/sdk.go](../../pkg/sdk/sdk.go)
- [../../pkg/sdk/docs/02-configuration.md](../../pkg/sdk/docs/02-configuration.md)
- [../../pkg/sdk/transport/](../../pkg/sdk/transport/)
- [../../pkg/sdk/observability/](../../pkg/sdk/observability/)

这就是为什么它比“自己生成 gRPC client 然后手写一层 wrapper”更有接入价值。

## 3. SDK 到底替业务方省掉了什么

可以把它理解成 3 层节省：

| 层 | 如果没有 SDK | 有 SDK 后 |
| ---- | ---- | ---- |
| 连接层 | 自己管 endpoint、TLS、dial option、metadata、interceptor | 先用 `sdk.Config` 和统一 `sdk.NewClient(...)` |
| 认证层 | 自己决定 JWT 本地验签 / 远程校验 / JWKS 刷新 | 直接看 `TokenVerifier` / `JWKSManager` |
| 业务调用层 | 自己维护 proto client 和 request 组装 | 先用 `Auth()` / `Authz()` / `Identity()` / `Guardianship()` / `IDP()` |

所以这篇文档的核心判断不是“SDK 有没有提供几个 API”，而是：

`pkg/sdk` 已经在把 IAM 的接入复杂度产品化。

## 4. 什么时候先看 SDK，什么时候回到契约层

### 4.1 优先看 SDK 的场景

这几类场景今天都应优先看 SDK：

| 场景 | 先看哪里 |
| ---- | ---- |
| 网关 / BFF 校验用户 JWT | [../../pkg/sdk/docs/03-jwt-verification.md](../../pkg/sdk/docs/03-jwt-verification.md) |
| 后端服务间认证 | [../../pkg/sdk/docs/04-service-auth.md](../../pkg/sdk/docs/04-service-auth.md) |
| 服务读取用户 / 孩子 / 监护关系 | [../../pkg/sdk/docs/01-quick-start.md](../../pkg/sdk/docs/01-quick-start.md)、[../../pkg/sdk/identity/](../../pkg/sdk/identity/) |
| 服务间策略判定（Enforce） | [../../pkg/sdk/authz/client.go](../../pkg/sdk/authz/client.go)、[../../api/grpc/iam/authz/v1/authz.proto](../../api/grpc/iam/authz/v1/authz.proto) |
| 需要统一处理 TLS、metadata、request-id | [../../pkg/sdk/docs/02-configuration.md](../../pkg/sdk/docs/02-configuration.md)、[../../pkg/sdk/transport/](../../pkg/sdk/transport/) |

### 4.2 必须回到契约层的场景

这些问题仍然应该回到 `api/`：

- 某个字段的精确定义
- proto service / method 的兼容性
- OpenAPI 的路径、状态码、schema
- 某个能力是不是正式对外暴露

对应入口：

- [../03-接口与集成/01-REST契约与接入.md](../03-接口与集成/01-REST契约与接入.md)
- [../03-接口与集成/02-gRPC契约与接入.md](../03-接口与集成/02-gRPC契约与接入.md)
- [../../api/rest/README.md](../../api/rest/README.md)
- [../../api/grpc/README.md](../../api/grpc/README.md)

因此更准确的关系是：

- SDK 负责“更好接”
- 契约层负责“到底是什么”

## 5. 当前不要讲过头的边界

### 5.1 不要把 SDK 讲成完整 IAM 的等价替身

今天它是高价值接入层，但不是新的真值层。

字段、服务和错误语义还是要回到：

- `api/rest/*.yaml`
- `api/grpc/**/*.proto`

### 5.2 不要把 SDK 的 authz 讲成「完整授权平台」

当前 `sdk.Client` 已提供 `Authz()`，封装单次 `Check`（gRPC），与 Casbin 模型一致。

仍未默认覆盖、需按业务补的部分包括：批量判定、本地策略缓存与版本同步、与 HTTP 路由 guard 的等价封装——这些仍以契约为准，见：

- [../03-接口与集成/03-授权接入与边界.md](../03-接口与集成/03-授权接入与边界.md)
- [../02-业务域/02-authz-角色、策略、资源、Assignment.md](../02-业务域/02-authz-角色、策略、资源、Assignment.md)

### 5.3 不要把 `IssueServiceToken` 讲成已交付的服务间发牌能力

服务端未实现时，SDK 调用只会得到 gRPC 错误；须与上文 **SDK 与 iam-apiserver 对照表** 一并阅读。

### 5.4 不要把框架中间件塞进 SDK 的概念里

SDK 更适合承载：

- 统一 client
- 统一认证消费能力
- 统一身份 / 监护 / IDP 消费能力

而像 `gin` 路由 guard、HTTP 错误响应、从上下文提 `user_id` / `tenant_id` 这类逻辑，更像业务服务或框架适配层的职责。

## 6. 后续可增强的 authz SDK 能力

已在仓库中提供 `pkg/sdk/authz` 与 `client.Authz()`（`Check` / `CheckRequest`）。

仍可演进的方向（非承诺）：

- `BatchCheck` 等与 proto 对齐的批量接口
- 与策略版本通知联动的客户端侧缓存策略
- 更细的错误语义封装

框架级 `gin` guard 仍建议放在业务或 `iam-apiserver` 中间件，而非 SDK。

## 7. 推荐阅读路径

如果你想理解“为什么 SDK 是 IAM 集成主轴的一部分”，建议按这个顺序读：

1. [../../pkg/sdk/docs/README.md](../../pkg/sdk/docs/README.md)
2. [../../pkg/sdk/docs/01-quick-start.md](../../pkg/sdk/docs/01-quick-start.md)
3. [../../pkg/sdk/docs/03-jwt-verification.md](../../pkg/sdk/docs/03-jwt-verification.md)
4. [../../pkg/sdk/docs/04-service-auth.md](../../pkg/sdk/docs/04-service-auth.md)
5. [../03-接口与集成/05-QS接入IAM.md](../03-接口与集成/05-QS接入IAM.md)
6. [../03-接口与集成/02-gRPC契约与接入.md](../03-接口与集成/02-gRPC契约与接入.md)

## 8. 一句话总结

`pkg/sdk` 对 `iam-contracts` 的意义，不是“多了一套调用封装”，而是“把 IAM 的接入复杂度统一收口成一个可以直接被业务系统消费的产品化入口”。

## 9. 如何验证本文结论（本地）

在**仓库根目录**执行（需 `rg`）。**SDK 侧**与 **服务端**应对照阅读：客户端存在方法不代表 RPC 已落地。

**`pkg/sdk`（封装是否存在、入口名是否仍一致）**

```bash
rg -n "func NewClient|func \(.*\) Auth\(\)|Authz\(\)|Identity\(\)" pkg/sdk/sdk.go
rg -n "IssueServiceToken" pkg/sdk/auth/client.go
rg -n "IssueServiceToken|authClient" pkg/sdk/auth/service_auth.go
rg -n "func \(.*\) Check\(" pkg/sdk/authz/client.go
rg -n "NewTokenVerifier|NewJWKSManager" pkg/sdk/sdk.go
```

**`iam-apiserver`（与 IssueServiceToken / PDP 相关的服务端事实）**

```bash
rg -n "IssueServiceToken" internal/apiserver/interface/authn/grpc/service.go
rg -n "AuthorizationService|Check\(" internal/apiserver/interface/authz/grpc/service.go
```

**读结果提示**：`auth/client.go` 中 **`IssueServiceToken`** 若仍存在，而 `interface/authn/grpc/service.go` 中同名 RPC **仍返回 `Unimplemented`**，则服务间发牌链 **在默认服务端不可用**；`authz/client` 的 **`Check`** 应与 [`02-授权判定链路](./02-授权判定链路：角色、策略、资源、Assignment、Casbin.md) 中 PDP 描述一致。
