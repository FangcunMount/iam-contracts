# 角色、策略、资源、Assignment

本文回答：`iam-contracts` 授权域当前到底建模了哪些对象，Casbin 在当前实现里扮演什么角色，哪些链路已落地，哪些还不能讲成完整在线判定服务。

## 30 秒结论

- 当前授权域最完整的不是“在线判定中心”，而是“授权管理面 + Casbin 规则落地链”。
- 当前核心对象是：`Role / Resource / Assignment / PolicyRule / PolicyVersion`。
- 当前策略写链是：`REST -> PolicyCommandService -> role/resource repo -> PolicyRule -> CasbinAdapter -> PolicyVersion.Increment -> 可选 VersionNotifier`。
- 当前赋权写链是：`REST -> AssignmentCommandService -> Assignment repo -> Casbin g 规则双写`。
- 当前没有独立 `authz gRPC`，也没有公共 `Enforce / Allow` 接口；`RequirePermission` 仍是 stub。

## 重点速查

| 关注点 | 当前答案 | 真实落点 |
| ---- | ---- | ---- |
| 角色对象 | 基础元数据 + `role:<name>` 键 | [../../internal/apiserver/domain/authz/role/role.go](../../internal/apiserver/domain/authz/role/role.go) |
| 资源对象 | `key / app / domain / type / actions` | [../../internal/apiserver/domain/authz/resource/resource.go](../../internal/apiserver/domain/authz/resource/resource.go) |
| 赋权对象 | `subject -> role`，带 `tenant_id` 与 `granted_by` | [../../internal/apiserver/domain/authz/assignment/assignment.go](../../internal/apiserver/domain/authz/assignment/assignment.go) |
| 策略值对象 | `PolicyRule(Sub, Dom, Obj, Act)` | [../../internal/apiserver/domain/authz/policy/rule.go](../../internal/apiserver/domain/authz/policy/rule.go) |
| 版本对象 | 租户级策略版本 | [../../internal/apiserver/domain/authz/policy/policy_version.go](../../internal/apiserver/domain/authz/policy/policy_version.go) |
| Casbin 模型 | 显式带 `dom` 的租户隔离模型 | [../../internal/apiserver/infra/casbin/model.conf](../../internal/apiserver/infra/casbin/model.conf) |
| 写链 | Policy / Assignment 双写 Casbin | [../../internal/apiserver/application/authz/policy/command_service.go](../../internal/apiserver/application/authz/policy/command_service.go)、[../../internal/apiserver/application/authz/assignment/command_service.go](../../internal/apiserver/application/authz/assignment/command_service.go) |
| 当前缺口 | 无独立判定 API，中间件权限校验未闭环 | [../../internal/pkg/middleware/authn/jwt_middleware.go](../../internal/pkg/middleware/authn/jwt_middleware.go) |

## 1. 当前模型

### 1.1 `Role`

`Role` 当前字段非常克制：

- `ID`
- `Name`
- `DisplayName`
- `TenantID`
- `Description`

它最关键的行为不是维护权限树，而是产出 Casbin 角色键：

- `role:<name>`

所以今天不能把 `Role` 讲成“内嵌 Permission 列表的聚合根”。

### 1.2 `Resource`

`Resource` 当前更像授权资源目录，核心字段包括：

- `Key`
- `DisplayName`
- `AppName`
- `Domain`
- `Type`
- `Actions`
- `Description`

当前资源键采用结构化形式，例如：

- `<app>:<domain>:<type>:*`

这说明它更像“资源目录 + 动作集合”，不是“HTTP 路由匹配规则”。

### 1.3 `Assignment`

`Assignment` 当前描述“谁在什么租户下被赋予哪个角色”，主要字段是：

- `SubjectType`
- `SubjectID`
- `RoleID`
- `TenantID`
- `GrantedBy`

它最终会映射成 Casbin `g` 规则。

### 1.4 `PolicyRule / PolicyVersion`

当前策略对象最关键的是：

- `PolicyRule(Sub, Dom, Obj, Act)`
- `GroupingRule(Sub, Dom, Role)`
- `PolicyVersion(TenantID, Version, ChangedBy, Reason)`

这说明授权域当前最稳定的抽象其实是：

- 规则
- 分配
- 版本

而不是一整套自定义 Permission DSL。

## 2. 当前 Casbin 模型

[../../internal/apiserver/infra/casbin/model.conf](../../internal/apiserver/infra/casbin/model.conf) 当前定义：

```text
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
```

当前最关键的事实：

- 请求、策略、分组都显式带 `dom`
- 当前是租户隔离模型
- 当前对象和动作是精确匹配，不是旧设计稿里的 `keyMatch2` 路径通配主流程

## 3. 当前写链

### 3.1 策略写链

[../../internal/apiserver/application/authz/policy/command_service.go](../../internal/apiserver/application/authz/policy/command_service.go) 当前 `AddPolicyRule / RemovePolicyRule` 流程是：

1. 查角色
2. 查资源
3. 构造 `PolicyRule`
4. 更新 Casbin `p` 规则
5. 递增 `PolicyVersion`
6. 如果配置了 `VersionNotifier`，再发版本变更通知

当前更准确的说法是：

- 策略变更会更新 Casbin 和版本表
- 版本通知是可选增强，不是每种部署形态的强依赖闭环

### 3.2 赋权写链

[../../internal/apiserver/application/authz/assignment/command_service.go](../../internal/apiserver/application/authz/assignment/command_service.go) 当前 `Grant` 流程是：

1. 校验命令
2. 校验角色存在
3. 取角色信息
4. 创建 `Assignment`
5. 写数据库
6. 写 Casbin `g` 规则
7. Casbin 写失败时回滚数据库记录

`Revoke / RevokeByID` 则反向执行，并在必要时做有限回滚。

这说明当前赋权链强调的是：

- 数据库 + Casbin 双写尽量一致

而不是：

- 事件驱动的最终一致性授权体系

## 4. 当前读链与判定边界

当前公开可证明的读能力主要是：

- 按角色查看策略
- 查看当前租户的策略版本
- 查看某个主体有哪些 Assignment

当前还不能讲成现状的能力：

- 公共 `Enforce(subject, object, action)` API
- 独立 `authz gRPC`
- 完整路由级权限中间件闭环

尤其是 [../../internal/pkg/middleware/authn/jwt_middleware.go](../../internal/pkg/middleware/authn/jwt_middleware.go) 里：

- `RequireRole` 仍是 stub
- `RequirePermission` 仍是 stub

所以今天更准确的口径是：

- `authz` 已经形成管理面
- 但在线判定面还没闭环

## 5. 当前版本同步能力

[../../internal/apiserver/infra/messaging/version_notifier.go](../../internal/apiserver/infra/messaging/version_notifier.go) 当前实现了版本通知器，主题与通道为：

- `iam.authz.policy_version`
- `iam-policy-sync`

这说明系统已经给“多实例策略版本同步”留出了真实实现位置。  
但今天仍应克制表达为：

- “已实现可选的版本通知能力”

而不是：

- “所有实例默认都会自动完成本地授权缓存刷新”

## 6. 当前最准确的口径

如果只用一句话概括当前授权域，我会这样讲：

`iam-contracts` 当前已经完成了角色、资源、策略、Assignment 到 Casbin 规则的管理链，但它还不是一个对外提供统一在线判定接口的完整授权中心。`

## 7. 继续往下读

1. [../03-接口与集成/03-授权接入与边界.md](../03-接口与集成/03-授权接入与边界.md)
2. [../05-专题分析/02-授权判定链路：角色、策略、资源、Assignment、Casbin.md](../05-专题分析/02-授权判定链路：角色、策略、资源、Assignment、Casbin.md)
3. [../../api/rest/authz.v1.yaml](../../api/rest/authz.v1.yaml)
