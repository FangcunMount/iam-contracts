# UC 模块 Port 架构设计

## 六边形架构中的 Port 定义

根据六边形架构（Hexagonal Architecture）原则，Port 分为两类：

### 1. **Driving Ports（驱动端口）** - 由领域层实现

这些是领域层对外提供的接口，由**领域服务（Domain Service）**实现，供**应用层（Application Layer）**调用。

**命名规范**：使用 `***er` 后缀，符合 Go 惯用法
- `UserRegister` - 用户注册器
- `UserProfileEditor` - 用户资料编辑器  
- `ChildQueryer` - 儿童查询器
- `GuardianshipManager` - 监护关系管理器

**特点**：
- 定义在 `domain/*/port/driving.go`
- 按功能职责拆分，遵循接口隔离原则
- 承载领域知识和业务规则
- 由领域服务实现

### 2. **Driven Ports（被驱动端口）** - 由基础设施层实现

这些是领域层依赖的外部接口，由**基础设施层（Infrastructure Layer）**实现，供**领域服务**使用。

**命名规范**：使用 `***Repository` 后缀
- `UserRepository` - 用户仓储
- `ChildRepository` - 儿童仓储
- `GuardianshipRepository` - 监护关系仓储

**特点**：
- 定义在 `domain/*/port/driven.go`  
- 通常是数据访问接口
- 遵循仓储模式
- 由基础设施层实现（如 MySQL、Redis）

---

## User 模块 Port 定义

### Driving Ports (`domain/user/port/driving.go`)

```go
// UserRegister 用户注册领域服务接口
type UserRegister interface {
    Register(ctx context.Context, name string, phone meta.Phone) (*user.User, error)
}

// UserProfileEditor 用户资料管理领域服务接口
type UserProfileEditor interface {
    Rename(ctx context.Context, userID user.UserID, name string) error
    UpdateContact(ctx context.Context, userID user.UserID, phone meta.Phone, email meta.Email) error
    UpdateIDCard(ctx context.Context, userID user.UserID, idCard meta.IDCard) error
}

// UserStatusChanger 用户状态管理领域服务接口
type UserStatusChanger interface {
    Activate(ctx context.Context, userID user.UserID) error
    Deactivate(ctx context.Context, userID user.UserID) error
    Block(ctx context.Context, userID user.UserID) error
}

// UserQueryer 用户查询领域服务接口
type UserQueryer interface {
    FindByID(ctx context.Context, userID user.UserID) (*user.User, error)
    FindByPhone(ctx context.Context, phone meta.Phone) (*user.User, error)
}
```

### Driven Ports (`domain/user/port/repo.go`)

```go
// UserRepository 用户存储接口 - 被驱动端口
type UserRepository interface {
    Create(ctx context.Context, user *user.User) error
    FindByID(ctx context.Context, id user.UserID) (*user.User, error)
    FindByPhone(ctx context.Context, phone meta.Phone) (*user.User, error)
    Update(ctx context.Context, user *user.User) error
}
```

---

## Child 模块 Port 定义

### Driving Ports (`domain/child/port/driving.go`)

```go
// ChildRegister 儿童注册领域服务接口
type ChildRegister interface {
    Register(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) (*child.Child, error)
    RegisterWithIDCard(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday, idCard meta.IDCard) (*child.Child, error)
}

// ChildProfileEditor 儿童资料管理领域服务接口
type ChildProfileEditor interface {
    Rename(ctx context.Context, childID child.ChildID, name string) error
    UpdateIDCard(ctx context.Context, childID child.ChildID, idCard meta.IDCard) error
    UpdateProfile(ctx context.Context, childID child.ChildID, gender meta.Gender, birthday meta.Birthday) error
    UpdateHeightWeight(ctx context.Context, childID child.ChildID, height meta.Height, weight meta.Weight) error
}

// ChildQueryer 儿童查询领域服务接口
type ChildQueryer interface {
    FindByID(ctx context.Context, childID child.ChildID) (*child.Child, error)
    FindByIDCard(ctx context.Context, idCard meta.IDCard) (*child.Child, error)
    FindListByName(ctx context.Context, name string) ([]*child.Child, error)
    FindListByNameAndBirthday(ctx context.Context, name string, birthday meta.Birthday) ([]*child.Child, error)
    FindSimilar(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) ([]*child.Child, error)
}
```

### Driven Ports (`domain/child/port/driven.go`)

```go
// ChildRepository 儿童档案存储接口
type ChildRepository interface {
    Create(ctx context.Context, child *child.Child) error
    FindByID(ctx context.Context, id child.ChildID) (*child.Child, error)
    FindByName(ctx context.Context, name string) (*child.Child, error)
    FindByIDCard(ctx context.Context, idCard meta.IDCard) (*child.Child, error)
    FindListByName(ctx context.Context, name string) ([]*child.Child, error)
    FindListByNameAndBirthday(ctx context.Context, name string, birthday meta.Birthday) ([]*child.Child, error)
    FindSimilar(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) ([]*child.Child, error)
    Update(ctx context.Context, child *child.Child) error
}
```

---

## Guardianship 模块 Port 定义

### Driving Ports (`domain/guardianship/port/driving.go`)

```go
// GuardianshipManager 监护关系管理领域服务接口
type GuardianshipManager interface {
    AddGuardian(ctx context.Context, userID user.UserID, childID child.ChildID, relation guardianship.Relation) error
    RemoveGuardian(ctx context.Context, userID user.UserID, childID child.ChildID) error
}

// GuardianshipQueryer 监护关系查询领域服务接口
type GuardianshipQueryer interface {
    FindByUserIDAndChildID(ctx context.Context, userID user.UserID, childID child.ChildID) (*guardianship.Guardianship, error)
    FindByUserIDAndChildName(ctx context.Context, userID user.UserID, childName string) ([]*guardianship.Guardianship, error)
    FindListByChildID(ctx context.Context, childID child.ChildID) ([]*guardianship.Guardianship, error)
    FindListByUserID(ctx context.Context, userID user.UserID) ([]*guardianship.Guardianship, error)
    IsGuardian(ctx context.Context, userID user.UserID, childID child.ChildID) (bool, error)
}

// GuardianshipRegister 监护关系注册领域服务接口
// 负责同时注册儿童和监护关系的复杂用例
type GuardianshipRegister interface {
    RegisterChildWithGuardian(ctx context.Context, params RegisterChildWithGuardianParams) (*guardianship.Guardianship, *child.Child, error)
}
```

### Driven Ports (`domain/guardianship/port/driven.go`)

```go
// GuardianshipRepository 监护关系存储接口
type GuardianshipRepository interface {
    Create(ctx context.Context, guardianship *guardianship.Guardianship) error
    FindByID(ctx context.Context, id idutil.ID) (*guardianship.Guardianship, error)
    FindByChildID(ctx context.Context, id child.ChildID) ([]*guardianship.Guardianship, error)
    FindByUserID(ctx context.Context, id user.UserID) ([]*guardianship.Guardianship, error)
    Update(ctx context.Context, guardianship *guardianship.Guardianship) error
}
```

---

## 分层调用关系

```
┌─────────────────────────────────────────────────────────────┐
│                    Interface Layer                           │
│                     (Handler)                                │
└──────────────────────┬──────────────────────────────────────┘
                       │ 调用
                       ↓
┌─────────────────────────────────────────────────────────────┐
│                 Application Layer                            │
│                (Application Service)                         │
│  - 事务管理（UnitOfWork）                                     │
│  - DTO 转换                                                   │
│  - 编排领域服务                                               │
└──────────────────────┬──────────────────────────────────────┘
                       │ 调用 Driving Ports
                       ↓
┌─────────────────────────────────────────────────────────────┐
│                   Domain Layer                               │
│                 (Domain Service)                             │
│  - 实现 Driving Ports 接口                                    │
│  - 承载领域知识和业务规则                                      │
│  - 使用 Driven Ports (Repository)                            │
└──────────────────────┬──────────────────────────────────────┘
                       │ 调用 Driven Ports
                       ↓
┌─────────────────────────────────────────────────────────────┐
│               Infrastructure Layer                           │
│                (Repository 实现)                             │
│  - 实现 Driven Ports 接口                                     │
│  - 访问数据库、缓存等外部资源                                  │
└─────────────────────────────────────────────────────────────┘
```

---

## 设计原则

### 1. 接口隔离原则（ISP）
- Driving Ports 按功能职责拆分
- 每个接口职责单一，易于理解和实现
- 避免"胖接口"

### 2. 依赖倒置原则（DIP）
- 高层模块（应用层）依赖抽象（Driving Ports）
- 低层模块（领域层）也依赖抽象（Driven Ports）
- 抽象不依赖具体，具体依赖抽象

### 3. 单一职责原则（SRP）
- 每个 Port 接口有明确的职责边界
- `UserRegister` 只负责注册
- `UserProfileEditor` 只负责资料编辑
- `UserQueryer` 只负责查询

### 4. Go 惯用法
- Driving Ports 使用 `***er` 命名
- Driven Ports 使用 `***Repository` 命名
- 接口小而专注

---

## 下一步工作

1. ✅ **定义 Driving Ports** - 已完成
   - User: UserRegister, UserProfileEditor, UserStatusChanger, UserQueryer
   - Child: ChildRegister, ChildProfileEditor, ChildQueryer
   - Guardianship: GuardianshipManager, GuardianshipQueryer, GuardianshipRegister

2. ⏳ **实现领域服务** - 进行中
   - ✅ UserDomainService 已实现所有 User Driving Ports
   - ⏳ ChildDomainService 待实现
   - ⏳ GuardianshipDomainService 待实现

3. ⏳ **简化应用服务**
   - 重写应用服务，只做事务管理和 DTO 转换
   - 调用领域服务实现业务逻辑

---

**创建日期**: 2025-01-16  
**架构模式**: 六边形架构（Hexagonal Architecture）  
**设计原则**: SOLID、接口隔离、依赖倒置
