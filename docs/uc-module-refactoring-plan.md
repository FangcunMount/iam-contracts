# UC 模块重构设计方案

## 问题分析

### 当前架构问题

1. **应用层实现领域端口**
   ```go
   // application/user/register.go
   type UserRegister struct { ... }
   var _ port.UserRegister = (*UserRegister)(nil)  // ❌ 应用层实现领域端口
   ```

2. **领域端口职责不清**
   - `domain/user/port/service.go` 定义的是"driving ports"（主动端口），应该由应用层定义
   - 应用服务直接实现这些接口，导致应用层和领域层职责混淆

3. **缺少真正的领域服务**
   - 没有领域工厂方法
   - 没有领域验证函数
   - 业务规则分散在应用层

4. **Handler 跨层依赖**
   ```go
   // Handler 依赖领域端口（应该依赖应用服务）
   registerSrv port.UserRegister
   profileSrv  port.UserProfileEditor
   querySrv    port.UserQueryer
   ```

5. **缺少 UoW 事务管理**
   - 虽然有 UnitOfWork，但应用服务没有使用
   - 缺少跨聚合的事务控制

## 重构目标

### 正确的 DDD 分层架构

```
┌─────────────────────────────────────────┐
│  Interface Layer (Handler)              │
│  - 参数验证                              │
│  - 调用应用服务                          │
│  - 返回响应                              │
└──────────────┬──────────────────────────┘
               │ depends on
               ↓
┌──────────────┴──────────────────────────┐
│  Application Layer (Application Service)│
│  - 用例编排                              │
│  - 事务管理 (UnitOfWork)                │
│  - DTO 转换                              │
└──────────────┬──────────────────────────┘
               │ calls
               ↓
┌──────────────┴──────────────────────────┐
│  Domain Layer (Domain Service)          │
│  - 工厂方法 (CreateEntity)              │
│  - 验证函数 (ValidateXxx)               │
│  - 业务规则                              │
└──────────────┬──────────────────────────┘
               │ uses
               ↓
┌──────────────┴──────────────────────────┐
│  Infrastructure Layer (Repository)      │
│  - 数据持久化                            │
│  - 外部服务适配                          │
└─────────────────────────────────────────┘
```

### 端口分离原则

1. **Driving Ports**（主动端口）- 应用层接口
   - 定义在 `application/xxx/services.go`
   - 由应用服务实现
   - Handler 依赖这些接口

2. **Driven Ports**（被动端口）- 领域层接口
   - 定义在 `domain/xxx/port/repo.go`
   - 由基础设施层实现
   - 领域层和应用层使用这些接口

## 重构方案

### 1. User 聚合重构

#### 1.1 创建领域服务（工厂方法 + 验证）

**文件**: `domain/user/service/factory.go`

```go
package service

import (
    "context"
    domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
    "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user/port"
    "github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// CreateUserEntity 创建用户实体
func CreateUserEntity(name string, phone meta.Phone) (*domain.User, error) {
    return domain.NewUser(name, phone)
}

// ValidatePhoneUnique 验证手机号唯一性
func ValidatePhoneUnique(ctx context.Context, repo port.UserRepository, phone meta.Phone) error {
    existing, err := repo.FindByPhone(ctx, phone)
    if err != nil {
        // 处理错误
    }
    if existing != nil {
        return errors.New("phone already exists")
    }
    return nil
}

// ValidateUserExists 验证用户存在
func ValidateUserExists(ctx context.Context, repo port.UserRepository, userID domain.UserID) (*domain.User, error) {
    user, err := repo.FindByID(ctx, userID)
    if err != nil {
        return nil, err
    }
    if user == nil {
        return nil, errors.New("user not found")
    }
    return user, nil
}
```

#### 1.2 创建应用服务接口和 DTOs

**文件**: `application/user/services.go`

```go
package user

import (
    "context"
    domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
    "github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// ============= 应用服务接口（Driving Ports）=============

// UserApplicationService 用户应用服务 - 基本管理
type UserApplicationService interface {
    Register(ctx context.Context, dto RegisterUserDTO) (*UserResult, error)
    GetByID(ctx context.Context, userID string) (*UserResult, error)
    GetByPhone(ctx context.Context, phone string) (*UserResult, error)
}

// UserProfileApplicationService 用户资料应用服务
type UserProfileApplicationService interface {
    Rename(ctx context.Context, userID string, newName string) error
    UpdateContact(ctx context.Context, dto UpdateContactDTO) error
    UpdateIDCard(ctx context.Context, userID string, idCard string) error
}

// UserStatusApplicationService 用户状态应用服务
type UserStatusApplicationService interface {
    Activate(ctx context.Context, userID string) error
    Deactivate(ctx context.Context, userID string) error
    Block(ctx context.Context, userID string) error
}

// ============= DTOs =============

type RegisterUserDTO struct {
    Name  string
    Phone string
    Email string // optional
}

type UpdateContactDTO struct {
    UserID string
    Phone  string
    Email  string
}

type UserResult struct {
    ID       string
    Name     string
    Phone    string
    Email    string
    IDCard   string
    Status   string
    CreateAt string
    UpdateAt string
}
```

#### 1.3 实现应用服务

**文件**: `application/user/user_app_service.go`

```go
package user

import (
    "context"
    "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/uow"
    domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
    domainService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user/service"
    "github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

type userApplicationService struct {
    uow uow.UnitOfWork
}

func NewUserApplicationService(uow uow.UnitOfWork) UserApplicationService {
    return &userApplicationService{uow: uow}
}

func (s *userApplicationService) Register(ctx context.Context, dto RegisterUserDTO) (*UserResult, error) {
    var result *UserResult
    
    err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        phone := meta.NewPhone(dto.Phone)
        
        // 1. 验证手机号唯一性（调用领域服务）
        if err := domainService.ValidatePhoneUnique(ctx, tx.Users, phone); err != nil {
            return err
        }
        
        // 2. 创建实体（调用领域工厂方法）
        user, err := domainService.CreateUserEntity(dto.Name, phone)
        if err != nil {
            return err
        }
        
        // 3. 持久化
        if err := tx.Users.Create(ctx, user); err != nil {
            return err
        }
        
        // 4. 可选：更新邮箱
        if dto.Email != "" {
            email := meta.NewEmail(dto.Email)
            var emptyPhone meta.Phone
            if err := user.UpdateContact(emptyPhone, email); err != nil {
                return err
            }
            if err := tx.Users.Update(ctx, user); err != nil {
                return err
            }
        }
        
        // 5. 查询并转换为 DTO
        created, err := tx.Users.FindByPhone(ctx, phone)
        if err != nil {
            return err
        }
        result = toUserResult(created)
        
        return nil
    })
    
    return result, err
}

func (s *userApplicationService) GetByID(ctx context.Context, userID string) (*UserResult, error) {
    var result *UserResult
    
    err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        id := domain.UserID(userID)
        user, err := tx.Users.FindByID(ctx, id)
        if err != nil {
            return err
        }
        result = toUserResult(user)
        return nil
    })
    
    return result, err
}

// 辅助函数：转换为 DTO
func toUserResult(user *domain.User) *UserResult {
    if user == nil {
        return nil
    }
    return &UserResult{
        ID:     string(user.ID),
        Name:   user.Name,
        Phone:  user.Phone.String(),
        Email:  user.Email.String(),
        IDCard: user.IDCard.String(),
        Status: string(user.Status),
        // ... 其他字段
    }
}
```

### 2. Child 聚合重构

类似 User，需要：
- 创建 `domain/child/service/factory.go` - 工厂方法和验证
- 创建 `application/child/services.go` - 应用服务接口和 DTOs
- 实现 `application/child/child_app_service.go` - 儿童管理应用服务
- 实现 `application/child/child_profile_app_service.go` - 儿童资料应用服务

### 3. Guardianship 聚合重构

这是一个特殊的聚合，涉及 User 和 Child 两个聚合根的关联：

- 创建 `domain/guardianship/service/factory.go`
- 创建 `application/guardianship/services.go`
- 实现 `application/guardianship/guardianship_app_service.go`

### 4. Handler 层更新

更新所有 Handler 的依赖，从领域端口改为应用服务：

```go
// 更新前
type UserHandler struct {
    registerSrv port.UserRegister       // ❌ 依赖领域端口
    profileSrv  port.UserProfileEditor
    querySrv    port.UserQueryer
}

// 更新后
type UserHandler struct {
    userService        user.UserApplicationService         // ✅ 依赖应用服务
    profileService     user.UserProfileApplicationService
    statusService      user.UserStatusApplicationService
}
```

### 5. DI 容器更新

更新 `container/assembler/user.go`，注册新的应用服务。

### 6. 清理工作

删除旧的应用层文件：
- `application/user/register.go`
- `application/user/editor.go`
- `application/user/query.go`
- `application/user/status-changer.go`
- `application/child/register.go`
- `application/child/editor.go`
- `application/child/query.go`
- `application/child/finder.go`
- `application/guardianship/manager.go`
- `application/guardianship/examiner.go`
- `application/guardianship/query.go`

删除或重命名领域端口：
- 将 `domain/xxx/port/service.go` 重命名为 `service_deprecated.go` 或直接删除

## 重构顺序

1. ✅ 分析问题，制定方案（本文档）
2. User 聚合重构
   - 创建领域服务（工厂方法）
   - 创建应用服务接口
   - 实现应用服务
3. Child 聚合重构
4. Guardianship 聚合重构
5. Handler 层更新
6. DI 容器更新
7. 清理旧代码
8. 编译验证
9. 文档更新

## 预期收益

1. **清晰的分层架构**: 每一层职责明确
2. **正确的事务管理**: 通过 UoW 管理跨仓储的事务
3. **可维护性提升**: 用例导向的应用服务更易理解
4. **可测试性提升**: 各层解耦，便于单元测试
5. **符合 DDD 最佳实践**: 端口分离，领域纯粹

## 注意事项

1. 保持领域层纯粹，不依赖应用层或基础设施层
2. 应用服务只依赖 UnitOfWork，在事务中获取仓储
3. Handler 层极简，只做参数验证、服务调用、响应返回
4. DTOs 独立于领域实体，避免泄漏领域模型到接口层
