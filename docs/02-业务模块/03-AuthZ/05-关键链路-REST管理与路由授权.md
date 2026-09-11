# 关键链路：REST 管理与路由授权

> 状态：已实现 · 本文以当前 Gin router、AuthN/AuthZ middleware、permission catalog 与 OpenAPI 为依据。

## 结论

REST v3 只承担 AuthZ 管理；外部服务的授权判定由 gRPC v3 `Check` 提供。IAM 自身的管理路由先由 AuthN middleware 建立 Principal，
再由 AuthZ middleware 使用明确的 Resource/Action 施权；服务间写入使用可信服务身份与 Assignment 约束。

## REST v3：管理接口

REST 路由统一挂在 `/api/v4/authz`：

| 资源 | 主要路径 | 用途 |
| --- | --- | --- |
| Role | `/api/v4/authz/roles` | 创建、查询、更新、删除角色 |
| Assignment | `/api/v4/authz/assignments` | 增量授予、撤销与查询直接关系 |
| PermissionGrant | `/api/v4/authz/grants` | 管理角色能力 |
| RoleInheritance（已退役） | `/api/v4/authz/role-inheritances` | 旧路径固定返回 `410 Gone`，无数据库依赖 |
| Resource | `/api/v4/authz/resources` | 管理资源与动作；弃用 attribute_schema 仅接受合法空值 |

完整 method/path 以 `api/rest/authz.v4.yaml` 为准。REST 不提供 `/api/v4/authz/check`；需要判定的可信服务调用 gRPC。

REST 是控制面，不是请求期权限决策面。若业务服务为了判定而调用 Role/Grant 列表并在本地重新实现 matcher，就会绕过快照与 Decision 语义。
服务间正确路径见 [gRPC 服务间授权与 SDK](06-关键链路-gRPC服务间授权与SDK.md)。

## AuthZ REST 路由与 Permission 矩阵

| Method + Path | Resource | Action | 业务语义 |
| --- | --- | --- | --- |
| `POST /api/v4/authz/roles` | `iam:authz:collection:roles` | `create` | 创建 Role |
| `GET /api/v4/authz/roles` | 同上 | `list` | 列出 Role |
| `GET /api/v4/authz/roles/:id` | 同上 | `read` | 读取可见的 Role |
| `PUT /api/v4/authz/roles/:id` | 同上 | `update` | 更新 Role |
| `DELETE /api/v4/authz/roles/:id` | 同上 | `delete` | 删除未被引用 Role |
| `GET /api/v4/authz/roles/:id/assignments` | `iam:authz:collection:assignments` | `list` | 按 Role 列直接 Assignment |
| `POST /api/v4/authz/assignments/grant` | 同上 | `grant` | 增量授予 Assignment |
| `POST /api/v4/authz/assignments/revoke` | 同上 | `revoke` | 按 Subject+Role 撤销 |
| `DELETE /api/v4/authz/assignments/:id` | 同上 | `revoke` | 按 Assignment ID 撤销 |
| `GET /api/v4/authz/assignments/subject` | 同上 | `list` | 按 Subject 列直接 Assignment |
| `POST /api/v4/authz/grants` | `iam:authz:collection:permission_grants` | `create` | 创建 managed PermissionGrant |
| `DELETE /api/v4/authz/grants/:id` | 同上 | `revoke` | 撤销 Grant |
| `GET /api/v4/authz/roles/:id/grants` | 同上 | `list` | 列角色的 Grant |
| `POST /api/v4/authz/role-inheritances` | （无） | （无） | 已退役，返回 410 |
| `GET /api/v4/authz/role-inheritances` | （无） | （无） | 已退役，返回 410 |
| `DELETE /api/v4/authz/role-inheritances/:id` | （无） | （无） | 已退役，返回 410 |
| `POST /api/v4/authz/resources` | `iam:authz:collection:resources` | `create` | 注册 Resource catalog |
| `GET /api/v4/authz/resources` | 同上 | `list` | 列 Resource |
| `GET /api/v4/authz/resources/:id` | 同上 | `read` | 按 ID 读 Resource |
| `GET /api/v4/authz/resources/key/:key` | 同上 | `read` | 按 key 读 Resource |
| `PUT /api/v4/authz/resources/:id` | 同上 | `update` | 更新 action/schema |
| `DELETE /api/v4/authz/resources/:id` | 同上 | `delete` | 删除未被引用 Resource |
| `POST /api/v4/authz/resources/validate-action` | 同上 | `validate_action` | 验证 catalog 是否登记 Action |

`GET /api/v4/authz/health` 是例外：它在受保护路由组之前注册，只返回 `status=ok,module=authz`。它不证明 runtime snapshot、MySQL、
policy subscriber 或全局 readiness 正常。

## 路由注册的 fail-closed 边界

AuthZ router 先注册模块局部 health，然后要求 Role handler、JWT `AuthRequired` 和 `PermissionOrGlobal` 都存在才继续注册受保护组。可选 handler 不存在时，
对应子路由不注册，而不是以无中间件方式暴露。

组合根还会检查 AuthZ module status、路由依赖和 JWT middleware。如果模块可用但授权中间件不可用，受保护路由应该整组不注册，不应退化为只验 JWT。

## 身份与信任边界

| 调用面 | 可信身份 | 额外限制 |
| --- | --- | --- |
| IAM REST 管理路由 | AuthN 用户 JWT | `RequirePermission(Resource, Action)` |
| 调试/运维路由 | AuthN 用户 JWT | 明确的运维 Resource/Action |

请求体中的 Subject、角色名或 actor 字符串不能替代传输层认证结果。AuthN middleware 只负责认证并写入可信请求上下文，不持有 Resource/Action，也不执行授权判定。

REST 路由上的 Principal 来自 AuthN token verifier 返回的已验证 claims。JWT middleware 将 UserID、LoginIdentityID、OrgID 和 TokenID 写入 request context。AuthZ `RouteDecisionService` 只使用其中 UserID 构造 `subject.Ref`，
把路由能力转换为领域 `Request`。

这意味着：

- URL/query/body 中的 `user_id` 是被管理对象，不是当前操作者身份。
- 操作者只能来自认证上下文，不能从请求参数推导。
- handler 内使用的 changed-by 应从 request context 中的已验证 UserID 派生，而不是接受任意 body actor。
- 路由授权在 handler 之前完成，handler 的领域校验仍然必须保留，两者分别保护“能否做”与“事实是否合法”。

## `RequirePermission`

所有管理路由统一检查 Resource/Action。允许则进入 handler；拒绝返回 403；运行时不可用返回 503，其他内部错误返回 500。每次请求只进行一个授权空间内的判断。

角色、Grant 和 Assignment 的应用服务还校验原始操作权限与管理保护。受保护角色需要额外的 `roles/manage_protected`，普通角色也不能承载敏感能力。用户权限通过当前策略判断；服务身份只能来自可信传输上下文，受管 Assignment 还按部署配置重新检查管理集合。

Resource 目录写入只接受具备对应操作权限的 actor。角色名和 `IsSystem` 不构成放行依据。普通管理保留 read/list/validate_action。

## AuthZ 管理路由

Role、Assignment、Grant 和 Resource 路由分别绑定各自 Resource/Action；继承路由已退役为 410。新增 handler 时必须同时更新：

- route registry；
- permission catalog；
- bootstrap/迁移所需 Grant；
- OpenAPI；
- route-contract 和 docs-facts 门禁。

不能用“已经登录”替代管理权限，也不能通过角色名称直接绕过 Grant。

REST handler 主要做四件事：绑定 DTO，从 URL/query/context 获取 ID 与 Tenant，构造 application command/query，将领域错误映射为 HTTP 响应。
它不应在 handler 内手工复制 Grant schema 或事务版本校验。

用户端 REST 使用数据库 ID 定位 Role/Resource/Grant/Inheritance；服务间 Assignment gRPC 为降低对 IAM 内部 ID 的耦合使用 stable role name。
两条传输路径最终仍必须进入同一 application/domain/UoW 不变量。

## 跨模块路由如何复用 AuthZ

AuthZ route authorizer 不只保护 `/api/v4/authz` 路由：

| 模块/路由类型 | Resource | Action 特征 | 授权方式 |
| --- | --- | --- | --- |
| AuthN JWKS 管理 | `iam:authn:collection:jwks` | rotate/retire 等明确动作 | `RequirePermission` |
| AuthN Session 撤销 | `iam:authn:collection:sessions` | `revoke`、`revoke_by_login_identity`、`revoke_by_user` | `RequirePermission` |
| IDP WeChat App 管理 | `iam:idp:collection:wechat_apps` | CRUD/list | `RequirePermission` |
| Suggest 搜索入口 | `iam:identity:collection:profiles` | `search` | 当前 Tenant `RequirePermission` |
| Cache governance debug | `iam:ops:collection:cache_governance` | `read` | 生产必须 `RequirePermission` |

这意味着 permission catalog 已是跨模块的路由合同。改 AuthN 管理 URL 时，不能只更新 AuthN 文档；还必须确认 Resource/Action、
bootstrap Grant 与 route contract 仍对齐。

## AuthN 管理路由

AuthN 的公开 JWKS 与受保护管理接口要区分：

- 公共 JWKS 只用于验签公钥发布；
- 管理 JWKS 与 Session 撤销路由使用用户 JWT；
- 路由分别检查 `jwks` 或 `sessions` Resource 下的明确 Action；
- 统一检查所需 Resource/Action 权限。

具体路径与动作见 [AuthN：JWKS 与本地验签](../02-AuthN/06-关键链路-JWKS与本地验签.md)和
[Session、Token 与 JWKS](../02-AuthN/03-Session-Token与JWKS.md)。

## Suggest 接入

Suggest 不再根据旧的超级管理员布尔标志或角色名决定搜索范围。当前规则是：

- 平台域命中 `iam:identity:collection:profiles/list`，得到 AllProfile capability；
- 手机号搜索还需要 `iam:identity:collection:profiles/search_by_mobile`；
- 业务组织范围与最终查询仍由 Suggest/Identity 的业务链路处理。

授权请求不携带分区字段。

Suggest 外层路由要求 `profiles/search`；provider 根据 `profiles/list_all` 决定是否产生全量范围，全量手机号检索另行检查 `profiles/search_by_mobile_all`。普通范围保留 OrgID、操作人和关联档案约束。

## OpenAPI、Router 与 README 的责任

| 事实 | 首要真相源 |
| --- | --- |
| 运行时是否注册 method/path | Gin router |
| 对外 request/response schema | `api/rest/authz.v4.yaml` |
| 路由需要的 Resource/Action | router middleware 绑定 + permission catalog |
| 读者导航与边界 | `api/rest/README.md` 与本文 |

理想状态下四者一致，但它们不是同一层证据。OpenAPI 有路径不能证明当前组合根已注册；router 有路径也不能证明 README 中的 curl URL 没有遗留旧前缀。

docs-facts 现在会抽取 README 中带 HTTP method 的 URL，并与 OpenAPI 及少量明确的 runtime-only 路由对齐。这是反漂移检查，不是 OpenAPI 完整性或生产可达性验收。

## 失败语义

| 失败 | 预期类型 | 不应做的降级 |
| --- | --- | --- |
| token 缺失/无效 | 401/认证错误 | 进入授权或 handler |
| 已认证但权限检查 deny | 403 | 根据 role name 放行 |
| routeAuth 未配置 | 500 | 只做 JWT 后放行 |
| authorization runtime 错误 | 500 | 转成 403 隐藏故障 |
| handler DTO/领域输入错误 | 4xx | 跳过 command constructor |
| 引用冲突/已存在 | 冲突类业务错误 | 伪装成成功并静默忽略 |
| DB/UoW 失败 | 5xx | 留下部分版本/事件 |

## 测试与门禁

- `router_permissions_test.go` 锁定 AuthZ 子路由的 Resource/Action 绑定。
- `router_matrix_test.go` 锁定路由注册矩阵与模块局部 health。
- AuthN middleware 测试锁定 token 验证与 Principal 上下文；AuthZ middleware 测试锁定 单次权限判断、allow/deny/error 组合。
- `check-route-contracts.py` 比对实际路由与 permission catalog/contract。
- `check-openapi-contracts.py` 比对 OpenAPI 关键契约。
- `check-docs-facts.py` 锁定 REST 管理与 gRPC Check 分工，并校验 README 请求 URL。

一项门禁通过只能证明它编码的事实。例如 router matrix 通过不证明生产 ingress 已暴露路由，docs URL 通过也不证明 request schema 每个字段都已验收。

## 接入变更清单

新增 Resource/Action 或调用方时，至少检查：

1. Resource 注册和 attribute schema；
2. PermissionGrant 数据与角色管理保护边界；
3. route registry 与中间件；
4. gRPC 服务 ACL 和 Assignment constraints；
5. OpenAPI/proto/SDK；
6. bootstrap、维护校验与多实例 reload；
7. 拒绝路径、非空条件兼容拒绝与多角色并集测试；旧继承路径返回 410。

8. README 中带 HTTP method 的请求 URL，以及退役的 v2 AuthZ 前缀或 REST `check` 引用。

## 路由设计评审问题

1. 这个端点是管理授权事实，还是判定业务对象？后者应优先 gRPC Check。
2. Resource/Action 是真正的业务能力，还是为了迎合 HTTP verb 随意命名？
3. 这个动作是否涉及受保护角色或敏感能力？
4. 路由缺少 AuthZ 依赖时是不注册/返错，还是会意外放行？
5. OpenAPI、router、permission catalog、bootstrap 和 README 的 method/path 是否一致？

## 角色详情与不可用错误

Handler 从认证上下文提取操作者，应用查询按 RoleID 加载角色并检查可见性。无 manage_protected 时受保护角色返回 404；关联 Assignment 与 Grant 同样过滤。

任一首次检查返回 `ErrAuthorizationPolicyUnavailable` 时，中间件立即保留错误并返回 503。它不作为普通 DENY，也不继续寻找平台授权旁路。新鲜度合同见 [多实例策略收敛](04-关键链路-多实例策略收敛.md)。
