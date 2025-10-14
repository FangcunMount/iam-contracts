# apiserver 代码结构说明

本文档描述了项目中 `internal/apiserver` 下主要模块的代码结构（重点关注领域层、应用层、以及 MySQL 仓储层），并给出职责说明与接口—实现映射，便于快速理解与后续维护。

## 概览（顶层）

```text
internal/apiserver/
├─ app.go
├─ auth.go
├─ config/
├─ container/
├─ database.go
├─ domain/             # 领域层（实体、值对象、领域接口/端口）
├─ infra/              # 基础设施适配器（这里以 mysql 为主）
├─ interface/          # 外部接口（rest/grpc 等）
├─ options/
├─ routers.go
├─ run.go
└─ server.go
```

## 1) 领域层 — `internal/apiserver/domain`

结构（重点包）：

```text
internal/apiserver/domain/
├─ child/
│  ├─ child.go        # Child 实体
│  ├─ vo.go           # Child 的值对象（身份证、生日等）
│  └─ port/           # Child 相关的端口（仓储/服务接口）
├─ guardianship/      # 域名已修正为 guardianship
│  ├─ guardianship.go # Guardianship 实体（UserID, ChildID, Rel, EstablishedAt, RevokedAt）
│  └─ port/           # Guardianship 的仓储与服务接口（repo.go / service.go）
└─ user/
   ├─ user.go         # User 实体
   ├─ vo.go           # User 的 VO（Phone, Email 等）
   └─ port/           # User 的仓储/服务接口
```

说明：

- 每个领域对象配套实体 `*.go` 和值对象 `vo.go`。
- 通过 `port` 包（接口）将基础设施依赖抽象，方便测试与替换实现。
- 领域层定义了核心接口，例如：
  - `domain/guardianship/port/repo.go` 定义 `GuardianshipRepository`（Create/Find/Update）
  - `domain/guardianship/port/service.go` 定义应用可调用的服务接口（Manager/Examiner/Queryer）并以 `context.Context` 为首参。

## 2) 应用层 — `internal/apiserver/application`

结构（与 domain 对应）：

```text
internal/apiserver/application/
├─ child/            # child 的注册/查询/编辑服务
   ├─ register.go  # ChildRegisterer（RegisterChild）
   ├─ finder.go    # ChildFinder（FindByID, FindListByName）
   ├─ editor.go    # ChildEditor（UpdateChild）
   └─ query.go     # ChildQueryer（FindByUserID, FindListByName）
├─ user/             # user 的注册/查询/编辑/状态变更服务
   ├─ register.go   # UserRegisterer（RegisterUser）    
   ├─ finder.go     # UserFinder（FindByID, FindListByName）
   ├─ editor.go     # UserEditor（UpdateUser）
   └─ status.go     # UserStatusUpdater（UpdateStatus）
└─ guardianship/     # guardianship 的应用实现（已拆分为 manager/examiner/query）
   ├─ manager.go     # GuardianshipManager（AddGuardian, RemoveGuardian）
   ├─ examiner.go    # PaternityExaminer（IsGuardian）
   └─ query.go       # GuardianshipQueryer（FindBy..., FindListBy...）
```

说明：

- 应用层负责把多个领域操作组合成业务用例。
- 约定与 `application/user` 保持一致：所有方法接受 `ctx context.Context`，通过构造函数注入仓储接口，错误使用项目的 `perrors` + `code` 包包装。
- `guardianship` 应用实现拆成三部分：
  - Manager：添加/撤销监护关系（处理校验、去重、创建/更新）
  - Examiner：判定某 user 是否为 child 的有效监护人
  - Queryer：执行查询（按 childID、userID、childName 等）

## 3) 仓储层（MySQL 实现）— `internal/pkg/database/mysql`

结构摘录：

```text

internal/pkg/database/mysql/
├─ audit.go           # 审计字段与 Syncable 相关方法（SetID 等）
└─ base.go            # BaseRepository 泛型封装（通用 CRUD）
```

要点：

- PO（持久化对象）与 Mapper 的分离便于 DB 与 domain 转换。
- `BaseRepository[*PO]` 提供通用 Create/Find/Update 等，并暴露 CreateAndSync/UpdateAndSync 回调，用来把数据库生成的 ID/时间回写到 domain 对象。
- `audit.go` 的 `SetID` 签名需与 `database/mysql.Syncable` 约定一致（仓库中已做相应调整）。

## 4) 关键接口 → 实现 → 应用 的映射

- Child
  - 接口：`domain/child/port` 中的 `ChildRepository`
  - 实现：`infra/mysql/child/repo.go` + `mapper.go`, `child.go (PO)`
  - 应用：`application/child/*`

- User
  - 接口：`domain/user/port` 中的 `UserRepository`
  - 实现：`infra/mysql/user/*`
  - 应用：`application/user/*`

- Guardianship
  - 接口：`domain/guardianship/port`（包括 `GuardianshipRepository` 与 `service.go` 中的 Manager/Examiner/Queryer）
  - 实现：`infra/mysql/guardianship/*`（PO/Mapper/Repo）
  - 应用：`application/guardianship/manager.go`, `examiner.go`, `query.go`

## 5) 辅助组件

- `pkg/errors`（perrors）+ `internal/pkg/code`：错误包装与错误码规范
- `internal/pkg/meta`：项目通用的值对象（Phone、IDCard、Birthday 等）
- `infra/mysql/base.go`：通用仓储基类
