# iam-contracts

IAM（Identity & Access Management），是 User Service + Auth Service + AuthZ Service 的“对外契约”（OpenAPI/Proto、资源-动作表、错误码、JWKS 规范等）

## 整体架构

### 全局上下文（C4-Context）

```mermaid
flowchart LR
  classDef svc fill:#eef,stroke:#446,stroke-width:1px;
  classDef biz fill:#fefee0,stroke:#b9a,stroke-width:1px;
  classDef infra fill:#f8f8f8,stroke:#999,stroke-width:1px;
  classDef c fill:#fff,stroke:#666,stroke-width:1px;

  subgraph Clients[Clients]
    A1["WeChat 小程序"]:::c
    A2["运营后台 Web"]:::c
    A3["第三方应用"]:::c
  end

  subgraph IAM["IAM 平台 / Monorepo 运行时边界"]
    US["User Service\nUser / Account / ActorLink / Profile"]:::svc
    AS["Auth Service\n登录 / JWT / Refresh / JWKS"]:::svc
    PDP["AuthZ Service\nRBAC + 关系授权判定"]:::svc
  end

  subgraph Biz["业务域服务"]
    B1["collection-server\n量表测评"]:::biz
    B2["hospital-server\n互联网医院"]:::biz
    B3["training-server\n训练中心"]:::biz
  end

  subgraph Infra["共享基础设施"]
    MQ["Event Bus\nRedis Stream / Kafka"]:::infra
    KMS["KMS\n密钥管理"]:::infra
    JWKS["JWKS 公钥集"]:::infra
    RDB["MySQL"]:::infra
    RED["Redis"]:::infra
    O11y["日志 / 指标 / 链路"]:::infra
  end

  %% 客户端登录到 AS
  A1 -->|WeChat code / OAuth2+PKCE| AS
  A2 -->|OAuth2+PKCE| AS
  A3 -->|OAuth2 授权码 / 客户端凭据| AS
  AS -->|Access / Refresh JWT| A1
  AS -->|JWKS 发布| JWKS

  %% 业务服务校验&鉴权
  B1 -->|Bearer JWT| AS
  B2 -->|Bearer JWT| AS
  B3 -->|Bearer JWT| AS

  B1 -->|鉴权请求| PDP
  B2 -->|鉴权请求| PDP
  B3 -->|鉴权请求| PDP
  PDP -->|关系授权查询| US

  %% 存储与密钥
  US --- RDB
  PDP --- RDB
  AS --- RDB
  AS --- RED
  PDP --- RED
  AS --- KMS
  AS --- JWKS

  %% 事件
  US == 发布事件 ==> MQ
  PDP == 订阅失效事件 ==> MQ
```

### 模型服务设计（核心数据/关系）

```mermaid

classDiagram
  class User {
    +id: UUID
    +status: int
    +nickname: string
    +avatar: string
    +created_at: datetime
    +updated_at: datetime
  }

  class Account {
    +id: bigint
    +user_id: UUID
    +provider: string  // wechat/qwechat/esign/local
    +external_id: string // unionid/openid/uid
    +meta_json: json
    +created_at: datetime
  }

  class ActorLink {
    +id: bigint
    +user_id: UUID
    +actor_type: string   // testee/patient/student/doctor/teacher
    +actor_id: string
    +scope_type: string   // system/org/project/questionnaire
    +scope_id: string
    +relation: string     // self/parent/guardian/doctor/...
    +can_read: bool
    +can_write: bool
    +valid_from: datetime
    +valid_to: datetime?
    +granted_by: UUID?
  }

  class PersonProfile {
    +id: UUID
    +legal_name: string
    +gender: tinyint
    +dob: date
    +id_type: string
    +id_no_enc: bytes     // 字段级加密
    +id_no_hash: bytes    // 检索/唯一约束
    +created_at: datetime
    +updated_at: datetime
  }

  class Role {
    +code: string  // writer/auditor/org_admin/...
  }

  class RolePermission {
    +id: bigint
    +role_code: string
    +resource: string  // answersheet/testee/report/...
    +action: string    // read/write/submit/lock/...
  }

  class UserRole {
    +id: bigint
    +user_id: UUID
    +role_code: string
    +scope_type: string
    +scope_id: string
    +granted_at: datetime
    +revoked_at: datetime?
  }

  %% 关系
  User "1" --> "*" Account : 绑定
  User "1" --> "*" ActorLink : 关系/代填
  PersonProfile <.. User : (可选绑定)
  Role "1" --> "*" RolePermission : 定义权限
  User "1" --> "*" UserRole : 被授予角色

```

### 运行时上下文（调用链/时序）

```mermaid

sequenceDiagram
  autonumber
  participant MP as 微信小程序
  participant AS as Auth Service
  participant US as User Service
  participant JW as JWKS/KMS
  participant B as collection-server
  participant PDP as AuthZ Service

  Note over MP,AS: ① 登录（Account→User→JWT）
  MP->>AS: POST /auth/wechat:login (js_code)
  AS->>AS: adapters.wechat.code2session → openid/unionid
  AS->>US: FindUserByAccount(provider=wechat, external_id=unionid)
  US-->>AS: user_id 或 空
  AS->>US: 若无则 CreateUser + BindAccount
  US-->>AS: user_id
  AS->>JW: 用私钥签 JWT（sub=user_id, kid=K-2025-10）
  AS-->>MP: {access_token, refresh_token}

  Note over MP,B: ② 业务请求（RBAC + 关系授权）
  MP->>B: POST /v1/answer-sheets/{id}:submit\nAuthorization: Bearer JWT
  B->>B: 验签(JWKS) & 解析 sub=user_id
  B->>PDP: AllowOnActor(user_id, "answersheet","submit", scope=questionnaire:PHQ9, actor=testee:123, needWrite=true)
  PDP->>US: HasDelegation(user_id, actor=testee:123, scope=PHQ9)
  US-->>PDP: true/false
  PDP-->>B: allow=true/false, reason
  B-->>MP: 200 / 403

```

### Monorepo 内部组件（服务与适配器）

```mermaid
flowchart TB
  classDef cmp fill:#fff,stroke:#333,stroke-width:1px;

  subgraph repo["iam-platform\nMonorepo 根"]
    subgraph AuthService["services/auth-service"]
      A1["adapters.wechat\ncode2session / MP OA / webhook"]:::cmp
      A2["adapters.qwechat / esign / localpwd"]:::cmp
      A3["issuer\nJWT / Refresh 发行·旋转·黑名单"]:::cmp
      A4["jwks provider\nkid 轮换 / 公钥发布"]:::cmp
      A5["http / grpc api"]:::cmp
    end

    subgraph UserService["services/user-service"]
      U1["user 聚合"]:::cmp
      U2["account 绑定"]:::cmp
      U3["user↔actorLink\nscope / 有效期 / 读写"]:::cmp
      U4["personProfile\n可选"]:::cmp
      U5["http / grpc api"]:::cmp
    end

    subgraph AuthZService["services/authz-service"]
      Z1["rbac\nroles / role_permissions 热加载"]:::cmp
      Z2["delegation\n调用 user-service HasDelegation"]:::cmp
      Z3["decision\nAllow / AllowOnActor"]:::cmp
      Z4["cache\nLRU + Redis，事件驱动失效"]:::cmp
      Z5["http / grpc api"]:::cmp
    end

    subgraph Libs["libs\nMonorepo 内共享库"]
      L1["authn-middleware\nJWT 验签 / JWKS 缓存"]:::cmp
      L2["authz-client\nPDP SDK + 本地缓存"]:::cmp
      L3["common\nerrors / dto / tracing / config"]:::cmp
    end

    subgraph Infra["infra/"]
      I1["helm / helmfile / compose"]:::cmp
      I2["migrations / seeds"]:::cmp
      I3["observability\notel / metrics / log"]:::cmp
    end
  end

  %% 连接
  A1 -->|账户映射| U2
  A2 -->|账户映射| U2
  Z2 -->|关系查询| U5
  L1 -->|三大服务共用| A5
  L1 -->|三大服务共用| U5
  L1 -->|三大服务共用| Z5
  L2 -->|业务服务复用| Z5
```

## 代码结构

以下以代码块形式展示项目的目录树，便于快速浏览仓库布局：

```text
iam-contracts/
├─ cmd/                # 可执行程序入口 (例如 cmd/apiserver)
├─ configs/            # 配置文件与证书 (yaml, env, mysql/redis 等)
├─ build/              # 构建/打包/infra 相关脚本与说明
├─ internal/           # 应用内部实现（不可被外部模块导入）
│  └─ apiserver/       # API server：路由、组装、domain、infra 适配器
├─ pkg/                # 可复用库（log/errors/database/util 等）
├─ proto/              # Protobuf / gRPC 定义与生成脚本（如存在）
├─ docs/               # 文档、设计说明、操作手册
├─ scripts/            # 开发与维护脚本
├─ Makefile
└─ go.mod / go.sum
```

简单约定：

- `internal/` 用于包含服务实现与业务逻辑（通常按六边形架构组织：ports/adapters/domain）。
- `pkg/` 提供可复用的库，尽量保持无全局状态，便于在 monorepo 内多服务复用。
- 配置中的敏感信息请使用 Vault / CI secrets 或环境变量注入，不要把明文凭证提交到仓库。

如需更细粒度的目录说明（比如每个包的职责与常见入口函数），我可以把此内容拆成 `docs/code-structure.md` 并在 README 中链接过去。
