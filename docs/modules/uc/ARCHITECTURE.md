# 用户中心 - 架构设计

> [返回用户中心文档](./README.md)

本文档详细介绍用户中心的分层架构和 CQRS 实现。

---

## 4. 分层架构

### 4.1 完整分层图

```text
┌──────────────────────────────────────────────────────────────────┐
│                       Interface Layer (接口层)                    │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  RESTful API                        gRPC API                     │
│  ┌────────────────────┐            ┌────────────────────┐       │
│  │  UserHandler       │            │  IdentityService   │       │
│  │  - CreateUser()    │            │  - GetUser()       │       │
│  │  - GetUser()       │            │  - GetChild()      │       │
│  │  - PatchUser()     │            │  - IsGuardian()    │       │
│  │  - GetProfile()    │            │  - ListChildren()  │       │
│  └────────────────────┘            └────────────────────┘       │
│                                                                   │
│  ┌────────────────────┐            ┌────────────────────┐       │
│  │  ChildHandler      │            │  Request/Response  │       │
│  │  - RegisterChild() │            │  DTOs              │       │
│  │  - GetChild()      │            └────────────────────┘       │
│  │  - PatchChild()    │                                         │
│  │  - ListMyChildren()│                                         │
│  │  - SearchChildren()│                                         │
│  └────────────────────┘                                         │
│                                                                   │
│  ┌────────────────────┐                                         │
│  │ GuardianshipHandler│                                         │
│  │  - Grant()         │                                         │
│  │  - Revoke()        │                                         │
│  │  - List()          │                                         │
│  └────────────────────┘                                         │
│                                                                   │
└───────────────────────┬──────────────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────────────────┐
│                   Application Layer (应用层)                      │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  命令服务 (Command Services)                                      │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  UserApplicationService                                     │ │
│  │  - Register(dto) -> UserResult                             │ │
│  │                                                             │ │
│  │  UserProfileApplicationService                              │ │
│  │  - Rename(userID, name) -> error                           │ │
│  │  - UpdateContact(dto) -> error                             │ │
│  │  - UpdateIDCard(userID, idCard) -> error                   │ │
│  │                                                             │ │
│  │  UserStatusApplicationService                               │ │
│  │  - Activate(userID) -> error                               │ │
│  │  - Deactivate(userID) -> error                             │ │
│  │  - Block(userID) -> error                                  │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  查询服务 (Query Services - CQRS)                                 │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  UserQueryApplicationService                                │ │
│  │  - GetByID(userID) -> UserResult                           │ │
│  │  - GetByPhone(phone) -> UserResult                         │ │
│  │                                                             │ │
│  │  ChildQueryApplicationService                               │ │
│  │  - GetByID(childID) -> ChildResult                         │ │
│  │  - GetByIDCard(idCard) -> ChildResult                      │ │
│  │  - FindSimilar(name, gender, birthday) -> []ChildResult    │ │
│  │                                                             │ │
│  │  GuardianshipQueryApplicationService                        │ │
│  │  - IsGuardian(userID, childID) -> bool                     │ │
│  │  - ListChildrenByUserID(userID) -> []GuardianshipResult    │ │
│  │  - ListGuardiansByChildID(childID) -> []GuardianshipResult │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  事务边界 (Unit of Work)                                          │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  UnitOfWork                                                 │ │
│  │  - WithinTx(ctx, fn) -> error                              │ │
│  │  - TxRepositories {Users, Children, Guardianships}         │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
└───────────────────────┬──────────────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────────────────┐
│                     Domain Layer (领域层)                         │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  聚合根 (Aggregates)                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────────┐        │
│  │    User     │  │   Child     │  │  Guardianship    │        │
│  │  (实体+方法) │  │  (实体+方法) │  │   (实体+方法)     │        │
│  └─────────────┘  └─────────────┘  └──────────────────┘        │
│                                                                   │
│  值对象 (Value Objects)                                           │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Phone, Email, IDCard, Gender, Birthday, Height, Weight    │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  领域服务 (Domain Services)                                       │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  UserRegister              ChildRegister                    │ │
│  │  - Register()              - Register()                     │ │
│  │  - RegisterWithIDCard()    - RegisterWithIDCard()           │ │
│  │                                                             │ │
│  │  UserProfileEditor         ChildProfileEditor               │ │
│  │  - Rename()                - Rename()                       │ │
│  │  - UpdateContact()         - UpdateProfile()                │ │
│  │  - UpdateIDCard()          - UpdateHeightWeight()           │ │
│  │                                                             │ │
│  │  UserStatusChanger         GuardianshipManager              │ │
│  │  - Activate()              - Grant()                        │ │
│  │  - Deactivate()            - Revoke()                       │ │
│  │  - Block()                                                  │ │
│  │                                                             │ │
│  │  UserQueryer               ChildQueryer                     │ │
│  │  - FindByID()              - FindByID()                     │ │
│  │  - FindByPhone()           - FindByIDCard()                 │ │
│  │                            - FindSimilar()                  │ │
│  │                                                             │ │
│  │                            GuardianshipQueryer              │ │
│  │                            - IsGuardian()                   │ │
│  │                            - ListByUserID()                 │ │
│  │                            - ListByChildID()                │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  领域端口 (Ports)                                                 │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Driving Ports (主动端口 - 领域服务对外提供)                  │ │
│  │  - UserRegister, UserProfileEditor, UserStatusChanger      │ │
│  │  - ChildRegister, ChildProfileEditor                        │ │
│  │  - GuardianshipManager, GuardianshipRegister               │ │
│  │                                                             │ │
│  │  Driven Ports (被动端口 - 领域依赖的外部能力)                  │ │
│  │  - UserRepository, ChildRepository, GuardianshipRepository │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
└───────────────────────┬──────────────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────────────────┐
│                 Infrastructure Layer (基础设施层)                  │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  MySQL 仓储实现 (Repository Implementations)                      │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  UserRepository (实现 port.UserRepository)                  │ │
│  │  - Create(user) -> error                                    │ │
│  │  - Update(user) -> error                                    │ │
│  │  - FindByID(id) -> User                                     │ │
│  │  - FindByPhone(phone) -> User                               │ │
│  │                                                             │ │
│  │  ChildRepository (实现 port.ChildRepository)                │ │
│  │  - Create(child) -> error                                   │ │
│  │  - Update(child) -> error                                   │ │
│  │  - FindByID(id) -> Child                                    │ │
│  │  - FindByIDCard(idCard) -> Child                            │ │
│  │  - FindSimilar(name, gender, birthday) -> []Child           │ │
│  │                                                             │ │
│  │  GuardianshipRepository (实现 port.GuardianshipRepository)  │ │
│  │  - Create(guardianship) -> error                            │ │
│  │  - Delete(id) -> error                                      │ │
│  │  - FindByUserIDAndChildID(userID, childID) -> Guardianship │ │
│  │  - ListByUserID(userID) -> []Guardianship                   │ │
│  │  - ListByChildID(childID) -> []Guardianship                 │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  持久化对象 (PO - Persistence Objects)                            │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  UserPO, ChildPO, GuardianshipPO (GORM Models)             │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  数据库 (Database)                                                │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  MySQL 8.0                                                  │ │
│  │  - users 表                                                 │ │
│  │  - children 表                                              │ │
│  │  - guardianships 表                                         │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

### 4.2 依赖方向

```text
Interface Layer (依赖↓)
    ↓
Application Layer (依赖↓)
    ↓
Domain Layer (核心，不依赖任何层)
    ↑ (实现)
Infrastructure Layer (实现领域端口)
```

**关键点**:

- ✅ 依赖倒置：基础设施层实现领域层定义的端口
- ✅ 领域独立：领域层不依赖任何外部框架
- ✅ 测试友好：可以轻松 Mock 端口进行单元测试

---

## 5. CQRS 实现

### 5.1 命令与查询分离

```text
┌─────────────────────────────────────────────────────────┐
│                   Handler Layer                          │
├─────────────────────────────────────────────────────────┤
│  UserHandler                                            │
│  - userApp: UserApplicationService         (命令 - 写)  │
│  - profileApp: UserProfileApplicationService (命令)     │
│  - userQuery: UserQueryApplicationService   (查询 - 读)  │
└──────────────┬────────────────────────┬─────────────────┘
               │                        │
               ▼                        ▼
    ┌──────────────────┐    ┌──────────────────┐
    │  Command Service │    │  Query Service   │
    ├──────────────────┤    ├──────────────────┤
    │ Register()       │    │ GetByID()        │
    │ Rename()         │    │ GetByPhone()     │
    │ UpdateContact()  │    │                  │
    │ Activate()       │    │                  │
    └──────────────────┘    └──────────────────┘
          │                        │
          ▼                        ▼
    ┌──────────────────┐    ┌──────────────────┐
    │  Domain Service  │    │  Domain Query    │
    │  (写操作+验证)    │    │  Service (只读)   │
    └──────────────────┘    └──────────────────┘
          │                        │
          └────────┬───────────────┘
                   ▼
          ┌──────────────────┐
          │   Repository     │
          └──────────────────┘
```

### 5.2 命令服务示例

```go
// internal/apiserver/application/uc/user/services_impl.go
type userApplicationService struct {
    uow uow.UnitOfWork
}

func (s *userApplicationService) Register(
    ctx context.Context, 
    dto RegisterUserDTO,
) (*UserResult, error) {
    var result *UserResult
    
    err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        // 1. 创建领域服务
        registerService := domainservice.NewRegisterService(tx.Users)
        
        // 2. 转换 DTO 为值对象
        phone := meta.NewPhone(dto.Phone)
        
        // 3. 调用领域服务创建实体
        user, err := registerService.Register(ctx, dto.Name, phone)
        if err != nil {
            return err
        }
        
        // 4. 设置可选字段
        if dto.Email != "" {
            email := meta.NewEmail(dto.Email)
            user.UpdateEmail(email)
        }
        
        // 5. 持久化
        if err := tx.Users.Create(ctx, user); err != nil {
            return err
        }
        
        // 6. 转换为结果 DTO
        result = toUserResult(user)
        return nil
    })
    
    return result, err
}
```

### 5.3 查询服务示例

```go
// internal/apiserver/application/uc/user/query_service.go
type userQueryApplicationService struct {
    uow uow.UnitOfWork
}

func (s *userQueryApplicationService) GetByID(
    ctx context.Context, 
    userID string,
) (*UserResult, error) {
    var result *UserResult
    
    err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        // 1. 创建查询服务
        queryService := domainservice.NewQueryService(tx.Users)
        
        // 2. 转换 ID
        id, err := parseUserID(userID)
        if err != nil {
            return err
        }
        
        // 3. 调用查询
        user, err := queryService.FindByID(ctx, id)
        if err != nil {
            return err
        }
        
        // 4. 转换为结果
        result = toUserResult(user)
        return nil
    })
    
    return result, err
}
```

### 5.4 优势分析

| 维度 | 命令（写） | 查询（读） |
|------|-----------|-----------|
| **事务** | 必须在事务中 | 可选只读事务 |
| **验证** | 完整的业务规则验证 | 最小验证 |
| **缓存** | 不缓存 | 可添加缓存 |
| **返回值** | 操作结果 DTO | 查询结果 DTO |
| **副作用** | 修改数据库状态 | 无副作用 |
| **优化** | 关注一致性 | 关注性能 |

---
