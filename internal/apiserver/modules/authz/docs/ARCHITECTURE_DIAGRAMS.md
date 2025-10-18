# AuthZ 模块架构图

## 系统架构图 (Mermaid)

```mermaid
graph TB
    subgraph "Client & Business Services"
        FE[前端/调用方]
        SVC[业务服务<br/>UseCase + PEP DomainGuard]
    end

    subgraph "Authn Module"
        JWKS[JWT 验签 JWKS]
    end

    subgraph "AuthZ Module - PAP/PRP"
        PAP[PAP 管理 API<br/>角色/赋权/策略/资源]
        PRP[(PRP MySQL<br/>casbin_rule<br/>authz_roles<br/>authz_assignments<br/>authz_resources<br/>authz_policy_versions)]
        VERSION[Version Manager<br/>version++ & 广播]
    end

    subgraph "Runtime - PDP"
        ENF[Casbin CachedEnforcer<br/>PDP 决策点]
        LRU[(本地 LRU Cache)]
    end

    subgraph "Infrastructure"
        REDIS[(Redis Pub/Sub<br/>policy_changed)]
    end

    FE --> SVC
    SVC --> JWKS
    SVC --> ENF
    ENF --> LRU
    ENF --> PRP
    PAP --> PRP
    PAP --> VERSION
    VERSION --> REDIS
    REDIS --> ENF

    style PAP fill:#e1f5ff
    style ENF fill:#fff3e0
    style PRP fill:#f3e5f5
    style VERSION fill:#e8f5e9
```

## 分层架构图

```mermaid
graph TB
    subgraph "接口层 Interface"
        REST[REST API<br/>PAP 管理接口]
        SDK[Go SDK<br/>PEP DomainGuard]
    end

    subgraph "应用层 Application"
        ROLE_SVC[RoleService]
        ASSIGN_SVC[AssignmentService]
        POLICY_SVC[PolicyService]
        RES_SVC[ResourceService]
        VER_SVC[VersionService]
    end

    subgraph "领域层 Domain"
        ROLE_DOM[Role 聚合]
        ASSIGN_DOM[Assignment 聚合]
        RES_DOM[Resource 聚合]
        POLICY_DOM[Policy 聚合]
        PORTS[Port 接口<br/>RoleRepo<br/>AssignmentRepo<br/>ResourceRepo<br/>PolicyVersionRepo<br/>CasbinPort]
    end

    subgraph "基础设施层 Infrastructure"
        MYSQL[MySQL Repositories<br/>PO + Mapper + Repo]
        CASBIN[Casbin Adapter<br/>Enforcer 封装]
        REDIS_INFRA[Redis PubSub<br/>版本通知]
    end

    REST --> ROLE_SVC
    REST --> ASSIGN_SVC
    REST --> POLICY_SVC
    REST --> RES_SVC
    SDK --> CASBIN

    ROLE_SVC --> PORTS
    ASSIGN_SVC --> PORTS
    POLICY_SVC --> PORTS
    RES_SVC --> PORTS
    VER_SVC --> PORTS

    PORTS -.实现.-> MYSQL
    PORTS -.实现.-> CASBIN
    PORTS -.实现.-> REDIS_INFRA

    ROLE_DOM -.依赖.-> PORTS
    ASSIGN_DOM -.依赖.-> PORTS
    RES_DOM -.依赖.-> PORTS
    POLICY_DOM -.依赖.-> PORTS

    style REST fill:#e1f5ff
    style SDK fill:#e1f5ff
    style ROLE_SVC fill:#fff3e0
    style ASSIGN_SVC fill:#fff3e0
    style POLICY_SVC fill:#fff3e0
    style RES_SVC fill:#fff3e0
    style VER_SVC fill:#fff3e0
    style ROLE_DOM fill:#e8f5e9
    style ASSIGN_DOM fill:#e8f5e9
    style RES_DOM fill:#e8f5e9
    style POLICY_DOM fill:#e8f5e9
    style PORTS fill:#f3e5f5
    style MYSQL fill:#fce4ec
    style CASBIN fill:#fce4ec
    style REDIS_INFRA fill:#fce4ec
```

## 权限判定流程图

```mermaid
sequenceDiagram
    participant Client as 前端客户端
    participant UC as 业务 UseCase
    participant PEP as DomainGuard<br/>(PEP)
    participant PDP as CachedEnforcer<br/>(PDP)
    participant Cache as 本地缓存
    participant PRP as MySQL<br/>(PRP)

    Client->>UC: GetForm(id)
    UC->>PEP: Can().Read("scale:form:*").All()
    PEP->>PDP: Enforce(user, tenant, obj, act)
    
    alt 缓存命中
        PDP->>Cache: 查询缓存
        Cache-->>PDP: 返回决策结果
    else 缓存未命中
        PDP->>PRP: 查询 casbin_rule
        PRP-->>PDP: 返回策略规则
        PDP->>Cache: 更新缓存
    end
    
    PDP-->>PEP: true/false
    
    alt 拥有全局权限
        PEP-->>UC: Allow
        UC->>UC: repo.FindByID(id)
        UC-->>Client: 返回表单
    else 无全局权限
        UC->>PEP: Can().Read("scale:form:*").Own(userID)
        PEP->>PDP: Enforce(user, tenant, obj, "read_own")
        PDP-->>PEP: true/false
        
        alt 拥有所有者权限
            PEP-->>UC: Allow
            UC->>UC: repo.FindByID(id)
            UC->>UC: 校验 form.OwnerID == userID
            
            alt 是所有者
                UC-->>Client: 返回表单
            else 不是所有者
                UC-->>Client: 403 Forbidden
            end
        else 无所有者权限
            PEP-->>UC: Deny
            UC-->>Client: 403 Forbidden
        end
    end
```

## 策略管理流程图

```mermaid
sequenceDiagram
    participant Admin as 管理员
    participant API as PAP REST API
    participant App as Application Service
    participant Domain as Domain Service
    participant MySQL as MySQL (PRP)
    participant Casbin as Casbin Adapter
    participant Version as Version Service
    participant Redis as Redis Pub/Sub
    participant Worker as 业务服务<br/>(Subscriber)

    Admin->>API: POST /authz/policies
    API->>App: PolicyService.AddPolicy()
    App->>Domain: 校验资源和动作
    Domain-->>App: 校验通过
    
    App->>Casbin: AddPolicy(p规则)
    Casbin->>MySQL: INSERT casbin_rule
    MySQL-->>Casbin: Success
    
    App->>Version: IncrementVersion(tenant)
    Version->>MySQL: UPDATE policy_version<br/>SET version = version + 1
    Version->>Redis: PUBLISH authz:policy_changed<br/>{tenant, version}
    
    Redis-->>Worker: 接收通知
    Worker->>Worker: Enforcer.InvalidateCache()
    
    Version-->>App: 新版本号
    App-->>API: Success
    API-->>Admin: 200 OK
```

## XACML 架构映射

```mermaid
graph LR
    subgraph "XACML 标准"
        PEP_X[PEP<br/>执行点]
        PDP_X[PDP<br/>决策点]
        PRP_X[PRP<br/>存储点]
        PAP_X[PAP<br/>管理点]
    end

    subgraph "AuthZ 实现"
        PEP_I[interface/sdk/pep/<br/>DomainGuard]
        PDP_I[infra/casbin/<br/>CachedEnforcer]
        PRP_I[infra/mysql/<br/>casbin_rule + 领域表]
        PAP_I[application/ +<br/>interface/restful/]
    end

    PEP_X -.对应.-> PEP_I
    PDP_X -.对应.-> PDP_I
    PRP_X -.对应.-> PRP_I
    PAP_X -.对应.-> PAP_I

    style PEP_X fill:#e1f5ff
    style PDP_X fill:#fff3e0
    style PRP_X fill:#f3e5f5
    style PAP_X fill:#e8f5e9
    style PEP_I fill:#e1f5ff
    style PDP_I fill:#fff3e0
    style PRP_I fill:#f3e5f5
    style PAP_I fill:#e8f5e9
```

## 依赖关系图

```mermaid
graph TD
    REST[interface/restful]
    SDK[interface/sdk/pep]
    APP[application/*]
    DOMAIN[domain/*]
    PORT[domain/*/port/driven]
    MYSQL[infra/mysql]
    CASBIN[infra/casbin]
    REDIS[infra/redis]

    REST --> APP
    REST --> MYSQL
    SDK --> CASBIN
    APP --> PORT
    MYSQL --> PORT
    CASBIN --> PORT
    REDIS --> PORT
    PORT --> DOMAIN

    style DOMAIN fill:#e8f5e9
    style PORT fill:#f3e5f5
    style APP fill:#fff3e0
    style REST fill:#e1f5ff
    style SDK fill:#e1f5ff
    style MYSQL fill:#fce4ec
    style CASBIN fill:#fce4ec
    style REDIS fill:#fce4ec
```

## 图例说明

- 🔵 **蓝色**: 接口层（REST API / SDK）
- 🟡 **橙色**: 应用层（Application Services）
- 🟢 **绿色**: 领域层（Domain Models & Services）
- 🟣 **紫色**: 端口层（Port 接口定义）
- 🔴 **红色**: 基础设施层（MySQL / Casbin / Redis）

## 使用建议

1. **架构图**: 理解整体组件交互关系
2. **分层架构图**: 理解分层依赖关系和六边形架构
3. **权限判定流程图**: 理解 PEP → PDP → PRP 的判定流程
4. **策略管理流程图**: 理解 PAP 管理策略和版本广播机制
5. **XACML 映射**: 理解标准架构与实现的对应关系
6. **依赖关系图**: 理解各层之间的依赖方向（依赖倒置原则）

---

**提示**: 可使用支持 Mermaid 的工具查看图表，如 VS Code 插件、GitHub、Typora 等。
