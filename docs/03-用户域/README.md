# 用户域 (UC) 架构设计

> 👥 管理用户、儿童档案、监护关系的核心业务域

---

## 📊 架构全景图

```mermaid
graph TB
    subgraph "接口层"
        REST[REST API]
        GRPC[gRPC API]
    end
    
    subgraph "应用层"
        UAS[UserAppService]
        CAS[ChildAppService]
        GAS[GuardianAppService]
    end
    
    subgraph "领域层"
        subgraph "聚合根"
            USER[User 用户]
            CHILD[Child 儿童]
        end
        subgraph "实体"
            GUARDIAN[Guardianship 监护关系]
        end
        subgraph "领域服务"
            RS[RegisterService]
            GS[GuardianshipService]
        end
    end
    
    subgraph "基础设施"
        REPO[(MySQL)]
        EVENT[事件总线]
    end
    
    REST --> UAS
    REST --> CAS
    GRPC --> UAS
    UAS --> RS
    CAS --> CHILD
    GAS --> GS
    GS --> GUARDIAN
    USER --> REPO
    CHILD --> REPO
    GS --> EVENT
```

---

## 🎯 核心职责

| 职责 | 说明 | 详细文档 |
|------|------|---------|
| **用户管理** | 用户注册、档案管理 | [领域模型设计](./01-领域模型设计.md) |
| **儿童档案** | 儿童信息管理 | [领域模型设计](./01-领域模型设计.md) |
| **监护关系** | 监护人-儿童绑定 | [监护关系设计](./02-监护关系设计.md) |
| **领域事件** | 跨域协作事件 | [领域事件设计](./03-领域事件设计.md) |

---

## 🏗️ 设计思想

### 聚合边界设计

```text
┌─────────────────────────────────────────────────────────────┐
│                    聚合边界设计原则                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  User 聚合          Child 聚合           Guardianship       │
│  ┌─────────┐        ┌─────────┐         ┌─────────────┐    │
│  │  User   │        │  Child  │         │ Guardianship│    │
│  │ (聚合根)│        │ (聚合根)│         │   (实体)    │    │
│  │         │        │         │         └──────┬──────┘    │
│  │ Profile │        │ Profile │                │           │
│  │ Contact │        │ Health  │         属于 User 聚合?     │
│  └─────────┘        └─────────┘         还是独立聚合?       │
│                                                              │
│  设计决策:                                                   │
│  - User 和 Child 是独立聚合 (有独立生命周期)                │
│  - Guardianship 属于 User 聚合 (监护人视角管理)             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 监护关系建模

```mermaid
graph TB
    subgraph "监护关系"
        U1[用户A<br/>监护人]
        U2[用户B<br/>监护人]
        C1[儿童1]
        C2[儿童2]
        
        U1 -->|主监护人| C1
        U1 -->|监护人| C2
        U2 -->|监护人| C1
    end
```

---

## 📁 代码结构

```text
internal/apiserver/domain/uc/
├── entity/
│   ├── user.go              # 用户聚合根
│   ├── child.go             # 儿童聚合根
│   └── guardianship.go      # 监护关系实体
├── valueobject/
│   ├── profile.go           # 用户档案
│   ├── contact.go           # 联系方式
│   ├── child_profile.go     # 儿童档案
│   └── guardian_type.go     # 监护人类型
├── service/
│   ├── register_service.go  # 注册服务
│   └── guardianship_service.go  # 监护关系服务
├── port/
│   ├── repository.go        # 仓储端口
│   └── event_publisher.go   # 事件发布端口
└── event/
    ├── user_registered.go   # 用户注册事件
    └── child_bound.go       # 儿童绑定事件
```

---

## 🔗 上下游关系

```mermaid
graph LR
    subgraph "UC 用户域"
        A[用户服务]
    end
    
    subgraph "下游消费者"
        AUTHN[Authn 认证域]
        AUTHZ[Authz 授权域]
        QS[QS 测评系统]
    end
    
    A -->|用户信息| AUTHN
    A -->|用户角色| AUTHZ
    A -->|监护关系| QS
```

| 关系 | 服务 | 说明 |
|------|------|------|
| **被依赖** | Authn 域 | 提供用户信息用于 Token Claims |
| **被依赖** | Authz 域 | 提供用户角色信息 |
| **被依赖** | QS 系统 | 提供监护关系验证 |

---

## 📚 详细设计文档

| 文档 | 内容 | 阅读时间 |
|------|------|---------|
| [领域模型设计](./01-领域模型设计.md) | User、Child 聚合设计 | 10 min |
| [监护关系设计](./02-监护关系设计.md) | 监护人-儿童业务流程 | 12 min |
| [领域事件设计](./03-领域事件设计.md) | 事件发布、跨域协作 | 8 min |

---

## 🔑 关键决策

| 决策 | 选择 | 理由 |
|------|------|------|
| User/Child 关系 | 独立聚合 | 各有独立生命周期 |
| 监护关系归属 | 属于 User | 从监护人视角管理 |
| 儿童创建 | 由监护人创建 | 业务场景需要 |
| 跨域通信 | 领域事件 | 解耦，最终一致性 |
