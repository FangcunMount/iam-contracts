# 用户、儿童、Guardianship

本文回答：用户域在 `iam-contracts` 里到底承载哪些身份对象、当前 REST / gRPC 暴露面如何分工、`User / Child / Guardianship` 在代码里怎样落地，以及这一组现状版正文应该从哪里读起。

## 30 秒结论

- 用户域当前负责三类核心事实：`User` 这个身份锚点、`Child` 这个儿童档案、`Guardianship` 这个用户与儿童之间的关系记录。
- 当前对外暴露面分成两种风格：REST 更偏“当前登录用户上下文”，gRPC 更偏“服务间显式传 ID”；两者不完全对称，不应视为两套完全等价接口。
- 当前最容易讲过头的地方有 4 个：`GET /identity/guardians` 合同存在但 router 没注册、`children/register` 不是原子事务、`revoked_at` 没有在查询链里统一过滤、旧设计稿里的“主/次监护人、邀请码、最多 2 个监护人”不是当前代码事实。
- 用户域现状版正文已经迁入 `docs/02-业务域/`；旧 `docs/03-用户域/` 只保留为历史资料，不再属于主阅读路径。
- 接入边界看 [../03-接口与集成/04-身份接入与监护关系边界.md](../03-接口与集成/04-身份接入与监护关系边界.md)，内部主链路看 [../05-专题分析/03-监护关系链路：用户、儿童、Guardianship 的协作.md](../05-专题分析/03-监护关系链路：用户、儿童、Guardianship 的协作.md)。

## 重点速查

| 关注点 | 当前答案 | 真实落点 |
| ---- | ---- | ---- |
| 核心模型 | `User / Child / Guardianship` 三类事实 | [../../internal/apiserver/domain/uc/](../../internal/apiserver/domain/uc/) |
| REST 契约 | `/api/v1/identity/*`，偏当前用户上下文 | [../../api/rest/identity.v1.yaml](../../api/rest/identity.v1.yaml)、[../../internal/apiserver/interface/uc/restful/router.go](../../internal/apiserver/interface/uc/restful/router.go) |
| gRPC 契约 | `IdentityRead / GuardianshipQuery / GuardianshipCommand / IdentityLifecycle` | [../../api/grpc/iam/identity/v1/identity.proto](../../api/grpc/iam/identity/v1/identity.proto)、[../../internal/apiserver/interface/uc/grpc/identity/service.go](../../internal/apiserver/interface/uc/grpc/identity/service.go) |
| 应用层拆分 | `user / child / guardianship / uow` | [../../internal/apiserver/application/uc/](../../internal/apiserver/application/uc/) |
| 持久化落点 | `infra/mysql/{user,child,guardianship}` | [../../internal/apiserver/infra/mysql/](../../internal/apiserver/infra/mysql/) |
| 运行时装配 | `UserModule` 同时装配 REST 与 gRPC | [../../internal/apiserver/container/assembler/user.go](../../internal/apiserver/container/assembler/user.go) |
| 接入边界 | 当前该怎么接、哪些不能默认 | [../03-接口与集成/04-身份接入与监护关系边界.md](../03-接口与集成/04-身份接入与监护关系边界.md) |
| 主链路专题 | `children/register`、授监护、查询与判定链 | [../05-专题分析/03-监护关系链路：用户、儿童、Guardianship 的协作.md](../05-专题分析/03-监护关系链路：用户、儿童、Guardianship 的协作.md) |

## 1. 模块边界

### 1.1 负责什么

- 用户基础资料与身份对象管理
- 儿童档案管理
- 监护关系建模、查询与撤销
- 对认证域、业务系统和内部服务提供身份与关系数据

### 1.2 不负责什么

- 登录、Token、JWKS、会话：这属于 [认证、Token、JWKS](./01-authn-认证、Token、JWKS.md)
- 角色、策略、资源和权限判定：这属于 [角色、策略、资源、Assignment](./02-authz-角色、策略、资源、Assignment.md)
- gRPC 服务器装配、mTLS、ACL：这属于 [运行时层](../01-运行时/README.md)

## 2. 当前模型快照

| 模型 | 当前职责 | 当前不应讲过头的地方 |
| ---- | ---- | ---- |
| `User` | 身份锚点，维护姓名、昵称、手机号、邮箱、身份证、状态 | 当前代码里没有把 guardianship、roles 直接挂在 `User` 聚合上 |
| `Child` | 儿童档案，维护姓名、身份证、性别、生日、身高、体重 | 当前验证只要求名字和生日等基本信息，不应讲成复杂档案规则中心 |
| `Guardianship` | 用户与儿童之间的关系记录，维护 relation、建立时间、撤销时间 | 当前没有主/次监护人、邀请码、最多 2 个监护人等机制 |

## 3. 合同与运行时分工

### 3.1 REST

REST 当前更像“带用户态语义的业务接口”，例如：

- `GET /identity/me`
- `GET /identity/me/children`
- `POST /identity/children/register`
- `POST /identity/guardians/grant`

它更适合前端、BFF 或“当前登录用户”视角的业务接入。  
边界和已知漂移请直接看 [../03-接口与集成/04-身份接入与监护关系边界.md](../03-接口与集成/04-身份接入与监护关系边界.md)。

### 3.2 gRPC

gRPC 当前更像“服务间显式按 ID 查询或下命令”的接口，例如：

- `GetUser / GetChild`
- `IsGuardian / ListChildren / ListGuardians`
- `AddGuardian / RevokeGuardian`

它更适合内部服务和明确传 `user_id / child_id` 的调用场景。

## 4. 真实代码结构

```text
internal/apiserver/
├── interface/uc/
│   ├── grpc/identity/
│   └── restful/
├── application/uc/
│   ├── user/
│   ├── child/
│   ├── guardianship/
│   └── uow/
├── domain/uc/
│   ├── user/
│   ├── child/
│   └── guardianship/
└── infra/mysql/
    ├── user/
    ├── child/
    └── guardianship/
```

| 层 | 当前主要回答什么 | 先看哪里 |
| ---- | ---- | ---- |
| `interface` | 今天到底暴露了哪些 REST / gRPC 能力 | [../../internal/apiserver/interface/uc/restful/router.go](../../internal/apiserver/interface/uc/restful/router.go)、[../../internal/apiserver/interface/uc/grpc/identity/service.go](../../internal/apiserver/interface/uc/grpc/identity/service.go) |
| `application` | `user / child / guardianship` 用例怎么编排 | [../../internal/apiserver/application/uc/](../../internal/apiserver/application/uc/) |
| `domain` | 核心对象、关系和领域规则是什么 | [../../internal/apiserver/domain/uc/](../../internal/apiserver/domain/uc/) |
| `infra/mysql` | 最终怎么落库 | [../../internal/apiserver/infra/mysql/](../../internal/apiserver/infra/mysql/) |

## 5. 当前最重要的风险边界

这一组文档里，用户域自己应先承认这 4 个事实：

1. `children/register` 当前是“先建 child、再建 guardianship”的两段事务。
2. `GET /identity/guardians` 当前还停留在合同和 handler 注解层，router 没注册。
3. `revoked_at` 还没有在查询判定链里统一过滤。
4. 旧设计稿中的“邀请、主/次监护人、最多两个监护人”不是当前代码事实。

## 6. 领域模型展开

### 6.1 当前模型划分

| 模型 | 更准确的定位 |
| ---- | ---- |
| `User` | 身份锚点，不直接承载监护关系列表 |
| `Child` | 儿童档案，不直接承载监护人集合 |
| `Guardianship` | 连接 `User` 和 `Child` 的关系记录 |

这和旧设计稿里那种“User 聚合内嵌 guardianships、roles、primary/secondary 监护状态”的讲法不同。当前代码没有把这些东西组织成一个大聚合。

### 6.2 `User` 当前到底是什么

[../../internal/apiserver/domain/uc/user/user.go](../../internal/apiserver/domain/uc/user/user.go) 里的 `User` 当前字段只有：

- `ID`
- `Name`
- `Nickname`
- `Phone`
- `Email`
- `IDCard`
- `Status`

当前已明确落地的行为包括：

- `Rename`
- `UpdateNickname`
- `UpdatePhone`
- `UpdateEmail`
- `UpdateIDCard`
- `Activate`
- `Deactivate`
- `Block`

用户状态枚举在 [../../internal/apiserver/domain/uc/user/types.go](../../internal/apiserver/domain/uc/user/types.go) 里定义为：

- `active`
- `inactive`
- `blocked`

[../../internal/apiserver/domain/uc/user/validator.go](../../internal/apiserver/domain/uc/user/validator.go) 当前真正能证明的验证规则主要有：

- 名称不能为空
- 注册时如果传了手机号，要检查唯一性
- 更新联系方式时，如果手机号发生变化，也要检查唯一性

所以今天不能把 `User` 讲成“拥有复杂 profile 校验和多维角色约束”的对象。

### 6.3 `Child` 当前到底是什么

[../../internal/apiserver/domain/uc/child/child.go](../../internal/apiserver/domain/uc/child/child.go) 里的 `Child` 当前字段是：

- `ID`
- `Name`
- `IDCard`
- `Gender`
- `Birthday`
- `Height`
- `Weight`

当前已明确落地的行为包括：

- `Rename`
- `UpdateIDCard`
- `UpdateProfile`
- `UpdateHeightWeight`

[../../internal/apiserver/domain/uc/child/validator.go](../../internal/apiserver/domain/uc/child/validator.go) 当前真正能证明的规则主要有：

- 注册时 `name` 不能为空
- 注册和更新时 `birthday` 不能为空

因此今天不能把 `Child` 讲成“已经内置未成年人年龄判断、学校年级规则、复杂档案校验”的聚合。

### 6.4 `Guardianship` 当前到底是什么

[../../internal/apiserver/domain/uc/guardianship/guardianship.go](../../internal/apiserver/domain/uc/guardianship/guardianship.go) 里的 `Guardianship` 当前字段是：

- `ID`
- `User`
- `Child`
- `Rel`
- `EstablishedAt`
- `RevokedAt`

relation 当前只有：

- `self`
- `parent`
- `grandparents`
- `other`

当前关系状态不是单独的 `status` 枚举，而是：

- `RevokedAt == nil` 表示活跃
- `RevokedAt != nil` 表示已撤销

对应行为只有两个：

- `IsActive()`
- `Revoke(at time.Time)`

这和旧文档里的 `pending / active / inactive / expired` 状态机完全不是一回事。

### 6.5 当前哪些规则真的在领域层成立

最关键的跨对象规则不在 `User` 自己身上，而在 [../../internal/apiserver/domain/uc/guardianship/manager.go](../../internal/apiserver/domain/uc/guardianship/manager.go)：

1. 校验 child 存在
2. 校验 user 存在
3. 校验同一 `user + child` 组合不存在重复活跃关系

撤销监护关系时，则是按 `child_id` 找全部 guardians，找到目标 `user_id` 的活跃关系后调用 `Revoke(...)`。

当前代码里没有查到这些规则：

- 每个儿童最多 2 个监护人
- 必须有 1 个主监护人
- 只有主监护人才能管理其他监护关系
- 邀请码 / 待接受状态
- 解除后自动触发复杂状态迁移

## 7. 监护关系与查询判定

### 7.1 当前主链路

当前监护关系最核心的写链和读链有 3 组：

1. `POST /api/v1/identity/children/register`
2. `POST /api/v1/identity/guardians/grant` / `GuardianshipCommand.AddGuardian`
3. `GET /identity/me/children`、`IsGuardian`、`ListChildren`、`ListGuardians`

当前 `children/register` 的真实顺序是：

1. REST handler 从上下文拿当前 `user_id`
2. 调 `childApp.Register(...)` 创建 `Child`
3. 调 `guardApp.AddGuardian(...)` 创建 `Guardianship`
4. 再查一次 guardianship 组回包

也就是说，今天的真实模型不是“创建 child 时自动带一个内嵌 guardianship”，而是两个相邻但独立的写步骤。

### 7.2 当前 REST 与 gRPC 的覆盖范围并不对称

| 能力 | REST 当前状态 | gRPC 当前状态 |
| ---- | ---- | ---- |
| 当前用户资料 | `已实现` | 不主打这个语义 |
| 我的孩子列表 | `已实现` | 可用 `ListChildren`，但语义是显式传 `user_id` |
| 注册儿童并授监护 | `已实现` | `待补证据`：没有对应 child-create 命令 |
| 显式授予监护 | `已实现` | `已实现` |
| 撤销监护 | `待补证据`：REST 未暴露 | `已实现` |
| 列出监护关系 | `待补证据`：YAML 有、router 无 | `已实现` |
| 判定是否为监护人 | 没有稳定公开的 REST 等价面 | `已实现` |

这也是为什么集成方文档会建议：

- 当前用户业务流优先看 REST
- 服务间显式按 ID 查询/判定优先看 gRPC

### 7.3 查询与访问控制链

REST `ListMyChildren()` 当前逻辑是：

1. 从上下文取当前 `user_id`
2. 调 `guardQuery.ListChildrenByUserID(...)`
3. 对每条关系再调 `childQuery.GetByID(...)`
4. 组装 `ChildResponse`

`GetChild / PatchChild` 当前都会先调用：

- `guardQuery.GetByUserIDAndChildID(rawUserID, childID)`

只要返回非空，就继续读取或修改 child。

gRPC `IsGuardian` 当前也是先调：

- `guardianshipQuerySvc.IsGuardian(...)`

若为真，再回查一条 guardianship 详情填到响应里。

### 7.4 当前最关键的风险边界

#### `children/register` 不是原子闭环

当前 `children/register` 是“先建 child，再建 guardianship”的两段事务。  
如果第二步失败，child 已经落库，不会自动回滚。

#### `revoked_at` 还没有在查询链里统一过滤

当前仓储方法：

- `FindByUserIDAndChildID`
- `FindByUserID`
- `FindByChildID`
- `IsGuardian`

都没有显式按 `revoked_at` 过滤。  
同时：

- `GuardianshipResult` 不带 `RevokedAt`
- REST `active` 过滤当前还是 stub

所以今天不能把“撤销关系已经完全不参与查询和判定”讲成现状。

#### relation 在三层之间还有漂移

当前至少有两处可证明的漂移：

- REST 的 `guardian` 在应用层会落成 `other`
- gRPC 的 `GRANDPARENT` 会先变成 `"grandparent"`，再因为应用层只识别 `"grandparents"` 而落成 `other`

#### 合同仍有缺口

[../../api/rest/identity.v1.yaml](../../api/rest/identity.v1.yaml) 里有：

- `GET /identity/guardians`

但 router 还没注册。  
同时 `children/register` 合同写的是 `201`，运行时当前实际返回 `200`。

[../../api/grpc/iam/identity/v1/identity.proto](../../api/grpc/iam/identity/v1/identity.proto) 里有：

- `UpdateGuardianRelation`
- `IdentityStream`

但当前实现是：

- `UpdateGuardianRelation` 返回 `Unimplemented`
- `IdentityStream` 没发现运行时注册

## 8. 事件合同与当前边界

### 8.1 当前事件状态先说结论

用户域今天的主协作方式仍然是同步调用，不是事件驱动。当前代码里没有查到用户域自己的业务事件发布链、事件总线接入、 Outbox 或运行中的 `IdentityStream` 服务注册。

当前更准确的口径是：

- 用户域“有事件合同设计”
- 但用户域“没有可证明正在运行的领域事件体系”

### 8.2 当前已经存在什么

[../../api/grpc/iam/identity/v1/identity.proto](../../api/grpc/iam/identity/v1/identity.proto) 当前已经定义：

- `UserEventType`
- `GuardianshipEventType`
- `UserEvent`
- `GuardianshipEvent`
- `SubscribeUserEventsRequest`
- `SubscribeGuardianshipEventsRequest`
- `service IdentityStream`

因此可以讲成现状的是：

- 事件流接口模型已经被设计进 proto

不能讲成现状的是：

- 这些流式接口已经被服务端实际提供

### 8.3 当前没有什么

[../../internal/apiserver/interface/uc/grpc/identity/service.go](../../internal/apiserver/interface/uc/grpc/identity/service.go) 当前只注册：

- `IdentityRead`
- `GuardianshipQuery`
- `GuardianshipCommand`
- `IdentityLifecycle`

没有：

- `RegisterIdentityStreamServer(...)`

[../../internal/apiserver/container/container.go](../../internal/apiserver/container/container.go) 虽然整体容器支持 `eventBus`，但 [../../internal/apiserver/container/assembler/user.go](../../internal/apiserver/container/assembler/user.go) 当前的 `UserModule.Initialize(...)` 只接收：

- `mysqlDB`

没有接收：

- `eventBus`
- `publisher`
- `notifier`

这和 `authz` 模块形成鲜明对比。`authz` 至少有策略版本通知器，而用户域当前连这层接线都还没有。

这轮核对下来，当前没有查到用户域里有这些已落地部件：

- 用户/儿童/监护关系事件发布器
- 事件 topic 常量
- NSQ/消息队列发布代码
- 用户域事件订阅器
- Outbox 表或投递任务

### 8.4 当前应如何理解用户域协作

用户域今天的真实协作方式，主要还是：

- REST：面向当前用户上下文
- gRPC：面向显式 ID 查询与命令
- 仓储/UoW：本地事务内完成写入与查询

更具体地说：

- 注册儿童并授监护：REST handler 串两个应用服务
- 监护判定：gRPC / REST 最终都查 guardianship repo
- 用户资料读取：`GetUser` / `GetUserProfile` 直接走查询服务

也就是说，当前跨模块协作的主轴不是“事件通知别人来更新”，而是“别的模块同步来查你”。

## 9. 继续往下读

1. [../03-接口与集成/04-身份接入与监护关系边界.md](../03-接口与集成/04-身份接入与监护关系边界.md)
2. [../05-专题分析/03-监护关系链路：用户、儿童、Guardianship 的协作.md](../05-专题分析/03-监护关系链路：用户、儿童、Guardianship 的协作.md)
3. [../../api/rest/identity.v1.yaml](../../api/rest/identity.v1.yaml)
4. [../../api/grpc/iam/identity/v1/identity.proto](../../api/grpc/iam/identity/v1/identity.proto)
