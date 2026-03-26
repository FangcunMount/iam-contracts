# HTTP认证中间件与身份上下文

本文回答：`iam-contracts` 的 HTTP JWT 中间件今天是怎么工作的，哪些身份字段会写进上下文，哪些路由真正用了它，以及 `RequireRole / RequirePermission` 当前到底是什么状态。

## 30 秒结论

- 中央 HTTP 认证中间件是 [../../internal/pkg/middleware/authn/jwt_middleware.go](../../internal/pkg/middleware/authn/jwt_middleware.go) 里的 `JWTAuthMiddleware`，它依赖 `AuthnModule.TokenService.VerifyToken()` 做统一验 token。
- 当前 token 提取顺序是：`Authorization` Header -> query `token` -> cookie `access_token`。
- 验证成功后，中间件会把 `claims`、`user_id`、`account_id`、`token_id` 写进 `gin.Context`，并把 `user_id` 写进 `request.Context`。
- 当前真正消费这套中间件的是 `/api/v1/identity/*`、`/api/v1/suggest/*` 和条件式的 `/api/v1/admin/*`。
- `RequireRole` 和 `RequirePermission` 现在还是 stub：它们只检查“是否已认证”，不会真的查角色或权限。

## 重点速查

| 关注点 | 当前答案 | 真实落点 |
| ---- | ---- | ---- |
| 中间件实现 | `JWTAuthMiddleware` | [../../internal/pkg/middleware/authn/jwt_middleware.go](../../internal/pkg/middleware/authn/jwt_middleware.go) |
| 上下文字段常量 | `user_id / account_id / token_id / claims` | [../../internal/pkg/middleware/authn/context_keys.go](../../internal/pkg/middleware/authn/context_keys.go) |
| 中央创建位置 | Router 根据 `AuthnModule.TokenService` 创建 | [../../internal/apiserver/routers.go](../../internal/apiserver/routers.go) |
| 受保护的用户路由 | `/api/v1/identity/*` | [../../internal/apiserver/interface/uc/restful/router.go](../../internal/apiserver/interface/uc/restful/router.go) |
| 受保护的 suggest 路由 | `/api/v1/suggest/child` | [../../internal/apiserver/interface/suggest/restful/handler.go](../../internal/apiserver/interface/suggest/restful/handler.go) |
| 条件式 admin 路由 | `/api/v1/admin/*` | [../../internal/apiserver/routers.go](../../internal/apiserver/routers.go) |
| 当前未统一挂 JWT 的模块 | `authn / authz / idp` | [../../internal/apiserver/routers.go](../../internal/apiserver/routers.go) |

## 1. 中间件创建与注入

中央 router 当前逻辑很直接：

1. 如果 `r.container.AuthnModule` 和 `TokenService` 都存在，就创建 `JWTAuthMiddleware`
2. `user` 模块注入 `AuthRequired()`
3. `suggest` 模块注入 `AuthRequired()`
4. `/api/v1/admin` 这组路由在中间件存在时也挂 `AuthRequired()`

如果认证模块没初始化成功，当前 router 的行为不是“阻止这些路由注册”，而是：

- `user` 模块收到一个 no-op 中间件，仍会注册
- `suggest` 模块收到一个 no-op 中间件，仍会注册
- `/api/v1/admin` 不会挂认证中间件

这意味着今天更准确的口径是：

- “JWT 保护是条件式启用的”
- 不是“所有需要认证的 HTTP 路由都绝对被保护”

## 2. `AuthRequired()` 的当前行为

### 2.1 token 提取顺序

中间件按下面顺序找 token：

1. `Authorization` Header
2. query 参数 `token`
3. cookie `access_token`

对 Header，它同时接受：

- `Bearer <token>`
- 直接传 token 字符串

### 2.2 验证失败时

`AuthRequired()` 在这些情况会直接 `Abort()`：

- 没拿到 token
- `VerifyToken()` 返回错误
- `VerifyToken()` 返回 `resp == nil` 或 `resp.Valid == false`

当前失败响应会走统一的 `core.WriteResponse(...)`，并记录必要日志。

### 2.3 验证成功时

如果 `resp.Claims != nil`，当前中间件会写入：

| 位置 | 当前写入内容 |
| ---- | ---- |
| `request.Context` | `user_id` |
| `gin.Context` | `claims` |
| `gin.Context` | `user_id` |
| `gin.Context` | `account_id` |
| `gin.Context` | `token_id` |

这里有一个小边界需要说明：

- 当前 helper 里有 `GetCurrentSessionID()`
- 但 `AuthRequired()` 并没有写入 `session_id`

所以今天不能把“session id 已稳定进入 HTTP 身份上下文”讲成现状。

## 3. `AuthOptional()` 的当前行为

`AuthOptional()` 与 `AuthRequired()` 共用同一套 token 提取和验证逻辑，但区别是：

- 没 token，直接放行
- token 无效，也直接放行
- 只有在 token 验证成功时，才把 claims 和身份字段写进上下文

当前中央 router 没有把 `AuthOptional()` 用在现行主路由上，但这个能力已经存在。

## 4. 当前真正受保护的 HTTP 面

### 4.1 已明确挂上 `AuthRequired()`

| 路由组 | 当前状态 |
| ---- | ---- |
| `/api/v1/identity/*` | 已在 `api := engine.Group(\"/api/v1/identity\")` 后统一 `Use(deps.AuthMiddleware)` |
| `/api/v1/suggest/*` | 已在 `group := engine.Group(\"/api/v1/suggest\")` 后统一 `Use(deps.AuthMiddleware)` |
| `/api/v1/admin/*` | 仅在中央已创建 `authMiddleware` 时统一挂上 |

### 4.2 当前没有统一挂上的

| 路由组 | 当前状态 |
| ---- | ---- |
| `/api/v1/authn/*` | 公开登录与账号/JWKS 面，没有统一挂中央 JWT 中间件 |
| `/api/v1/authz/*` | router 层未统一挂中央 JWT 中间件 |
| `/api/v1/idp/*` | router 层未统一挂中央 JWT 中间件 |

这并不自动等于“这些接口都不需要认证设计”，但它是 router 层的真实现状。

## 5. 角色与权限中间件的当前状态

`RequireRole()` 和 `RequirePermission()` 当前都只做了最小动作：

1. 从 `gin.Context` 取 `account_id`
2. 如果没有认证信息，则返回未认证错误
3. 否则直接 `Next()`

它们当前没有：

- 查角色
- 查权限
- 调 Casbin
- 调 `authz` 服务

所以今天更准确的表达只能是：

- “预留了角色/权限中间件位置”
- 不能写成“HTTP 权限保护链已经闭环”

## 6. 当前最值得先记住的边界

### 已实现

- JWT token 的统一提取与校验
- `claims / user_id / account_id / token_id` 上下文注入
- `AuthRequired()` 与 `AuthOptional()` 两种中间件风格

### 待补证据

- 更完整的上下文消费点，还应继续沿具体 handler 和 application 服务核对

### 规划改造

- 如果未来要让 `RequireRole / RequirePermission` 真正接 `authz`，应在本层明确区分“当前 stub”和“未来闭环方案”

## 7. 继续往下读

1. [01-服务入口、HTTP 与模块装配.md](./01-服务入口、HTTP 与模块装配.md)
2. [../03-接口与集成/03-授权接入与边界.md](../03-接口与集成/03-授权接入与边界.md)
3. [../03-接口与集成/04-身份接入与监护关系边界.md](../03-接口与集成/04-身份接入与监护关系边界.md)
4. [../02-业务域/01-authn-认证、Token、JWKS.md](../02-业务域/01-authn-认证、Token、JWKS.md)
