# 分层职责反思 - 应用服务层过重问题

## 问题分析

### 当前重构的问题

在刚完成的重构中，我犯了一个**严重的架构错误**：

```go
// ❌ 应用服务层承担了太多领域逻辑
func (s *userApplicationService) Register(ctx context.Context, dto RegisterUserDTO) (*UserResult, error) {
    return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        phone := meta.NewPhone(dto.Phone)
        
        // 这些都是领域逻辑！
        if err := userService.ValidatePhoneUnique(ctx, tx.Users, phone); err != nil {
            return err
        }
        
        user, err := userService.CreateUserEntity(dto.Name, phone)
        if err != nil {
            return err
        }
        
        if dto.Email != "" {
            email := meta.NewEmail(dto.Email)
            user.UpdateEmail(email)
        }
        
        if err := tx.Users.Create(ctx, user); err != nil {
            return perrors.WrapC(err, code.ErrDatabase, "failed to create user")
        }
        
        // 查询和转换...
    })
}
```

**问题**：
1. ❌ 应用服务直接调用多个领域函数，自己组织业务逻辑
2. ❌ 领域服务退化成"工具函数"，没有承载领域知识
3. ❌ 应用服务层过重，违背了"薄应用层"原则
4. ❌ 领域逻辑分散在应用层，不利于复用

---

## 正确的分层职责

###  领域服务层（Domain Service Layer）

**职责**：承载领域知识，实现业务逻辑

```go
// ✅ 领域服务 - 承载"注册用户"这个领域知识
type UserDomainService struct {
    repo port.UserRepository
}

// Register 注册新用户 - 完整的领域业务逻辑
func (s *UserDomainService) Register(ctx context.Context, name string, phone meta.Phone, email string) (*domain.User, error) {
    // 1. 验证手机号唯一性（领域规则）
    if err := s.ensurePhoneUnique(ctx, phone); err != nil {
        return nil, err
    }

    // 2. 创建用户实体（领域工厂）
    user, err := domain.NewUser(name, phone)
    if err != nil {
        return nil, err
    }

    // 3. 设置邮箱（领域逻辑）
    if email != "" {
        emailObj := meta.NewEmail(email)
        user.UpdateEmail(emailObj)
    }

    // 4. 持久化（使用仓储）
    if err := s.repo.Create(ctx, user); err != nil {
        return nil, perrors.WrapC(err, code.ErrDatabase, "create user failed")
    }

    // 5. 查询创建的用户并返回
    return s.repo.FindByPhone(ctx, phone)
}

// Rename 重命名用户 - 完整的领域业务逻辑
func (s *UserDomainService) Rename(ctx context.Context, userID domain.UserID, name string) error {
    // 1. 验证参数（领域规则）
    name = strings.TrimSpace(name)
    if name == "" {
        return perrors.WithCode(code.ErrUserBasicInfoInvalid, "name cannot be empty")
    }

    // 2. 查询用户
    user, err := s.FindByID(ctx, userID)
    if err != nil {
        return err
    }

    // 3. 修改实体
    user.Name = name

    // 4. 持久化
    return s.repo.Update(ctx, user)
}
```

**特点**：
- ✅ **有状态**：持有 Repository 依赖
- ✅ **领域知识载体**：每个方法是一个完整的业务操作
- ✅ **可复用**：可被多个应用服务调用
- ✅ **职责单一**：只关注领域逻辑，不关心事务、DTO转换

### 应用服务层（Application Service Layer）

**职责**：用例编排、事务管理、DTO转换

```go
// ✅ 应用服务 - 轻量级编排者
type userApplicationService struct {
    uow uow.UnitOfWork
}

// Register 注册用户用例 - 只做编排
func (s *userApplicationService) Register(ctx context.Context, dto RegisterUserDTO) (*UserResult, error) {
    var result *UserResult
    
    err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        // 1. 创建领域服务
        domainSvc := service.NewUserDomainService(tx.Users)
        
        // 2. 调用领域服务完成业务逻辑
        phone := meta.NewPhone(dto.Phone)
        user, err := domainSvc.Register(ctx, dto.Name, phone, dto.Email)
        if err != nil {
            return err
        }
        
        // 3. 转换为 DTO
        result = toUserResult(user)
        return nil
    })
    
    return result, err
}

// Rename 重命名用例 - 只做编排
func (s *userApplicationService) Rename(ctx context.Context, userID string, name string) error {
    return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        domainSvc := service.NewUserDomainService(tx.Users)
        uid := domain.NewUserID(parseUint64(userID))
        return domainSvc.Rename(ctx, uid, name)
    })
}
```

**特点**：
- ✅ **无业务逻辑**：只做编排
- ✅ **管理事务边界**：通过 UoW
- ✅ **DTO 转换**：Interface DTO ↔ Domain Entity
- ✅ **轻量级**：每个方法10行左右代码

---

## 对比分析

### 原来的代码（实际更合理）

```go
// application/user/register.go
type UserRegister struct {
    repo port.UserRepository  // 依赖仓储
}

func (s *UserRegister) Register(ctx, name, phone) (*domain.User, error) {
    // 完整的业务逻辑
    if err := ensurePhoneUnique(ctx, s.repo, phone); err != nil {
        return nil, err
    }
    
    u, err := domain.NewUser(name, phone)
    if err := s.repo.Create(ctx, u); err != nil {
        return nil, err
    }
    
    return s.repo.FindByPhone(ctx, phone)
}
```

**优点**：
- ✅ 应用服务承载领域逻辑（实际是领域服务）
- ✅ 业务逻辑集中，容易理解和维护
- ✅ 可以直接复用

**问题**：
- ❌ 命名不准确（叫 `application`，实际是 `domain service`）
- ❌ Handler 直接依赖这些服务（跨层依赖）
- ❌ 没有明确的事务边界

### 刚才的重构（过度设计）

```go
// domain/user/service/factory.go - 工具函数
func CreateUserEntity(name, phone) (*domain.User, error)
func ValidatePhoneUnique(ctx, repo, phone) error

// application/user/user_app_service.go - 过重
func (s *userApplicationService) Register(ctx, dto) (*UserResult, error) {
    // 应用服务承担了领域逻辑编排 ❌
    phone := meta.NewPhone(dto.Phone)
    if err := userService.ValidatePhoneUnique(ctx, tx.Users, phone); err != nil { }
    user, err := userService.CreateUserEntity(dto.Name, phone)
    // ...
}
```

**问题**：
- ❌ 领域服务退化成工具函数
- ❌ 应用服务承担领域逻辑编排
- ❌ 应用服务过重（50+ 行）
- ❌ 领域知识分散

---

## 正确的重构方案

### 方案一：重命名 + 简化（推荐）

**核心思想**：原来的代码实际已经很好了，只需要：

1. **重命名**：`application/user/*.go` → `domain/user/service/*.go`
2. **添加薄应用层**：真正的应用服务只做事务管理和DTO转换

#### 1. 领域服务（重命名原 application 层）

```go
// domain/user/service/user_service.go
package service

type UserDomainService struct {
    repo port.UserRepository
}

// NewUserDomainService 创建用户领域服务
func NewUserDomainService(repo port.UserRepository) *UserDomainService {
    return &UserDomainService{repo: repo}
}

// Register 注册新用户（领域业务逻辑）
func (s *UserDomainService) Register(ctx context.Context, name string, phone meta.Phone) (*domain.User, error) {
    // 原 application/user/register.go 的代码
    if err := s.ensurePhoneUnique(ctx, phone); err != nil {
        return nil, err
    }
    
    u, err := domain.NewUser(name, phone)
    if err != nil {
        return nil, err
    }
    
    if err := s.repo.Create(ctx, u); err != nil {
        return nil, perrors.WrapC(err, code.ErrDatabase, "create user failed")
    }
    
    return s.repo.FindByPhone(ctx, phone)
}

// Rename 重命名用户（领域业务逻辑）
func (s *UserDomainService) Rename(ctx context.Context, userID domain.UserID, name string) error {
    // 原 application/user/editor.go 的代码
    name = strings.TrimSpace(name)
    if name == "" {
        return perrors.WithCode(code.ErrUserBasicInfoInvalid, "name cannot be empty")
    }
    
    u, err := s.FindByID(ctx, userID)
    if err != nil {
        return err
    }
    
    u.Name = name
    return s.repo.Update(ctx, u)
}

// FindByID 查询用户
func (s *UserDomainService) FindByID(ctx context.Context, userID domain.UserID) (*domain.User, error) {
    // 原 application/user/query.go 的代码
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        return nil, perrors.WrapC(err, code.ErrDatabase, "find user failed")
    }
    if user == nil {
        return nil, perrors.WithCode(code.ErrUserNotExist, "user not found")
    }
    return user, nil
}

// 其他方法类似...
```

#### 2. 应用服务（新增薄层）

```go
// application/user/user_app_service.go
package user

type userApplicationService struct {
    uow uow.UnitOfWork
}

// Register 注册用户用例 - 只做编排
func (s *userApplicationService) Register(ctx context.Context, dto RegisterUserDTO) (*UserResult, error) {
    var result *UserResult
    
    err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        // 创建领域服务
        domainSvc := service.NewUserDomainService(tx.Users)
        
        // 调用领域服务
        phone := meta.NewPhone(dto.Phone)
        user, err := domainSvc.Register(ctx, dto.Name, phone)
        if err != nil {
            return err
        }
        
        // 设置可选字段（领域操作）
        if dto.Email != "" {
            email := meta.NewEmail(dto.Email)
            user.UpdateEmail(email)
            if err := tx.Users.Update(ctx, user); err != nil {
                return err
            }
        }
        
        // 转换为 DTO
        result = toUserResult(user)
        return nil
    })
    
    return result, err
}

// GetByID 查询用户用例
func (s *userApplicationService) GetByID(ctx context.Context, userID string) (*UserResult, error) {
    var result *UserResult
    
    err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        domainSvc := service.NewUserDomainService(tx.Users)
        uid := domain.NewUserID(parseUint64(userID))
        
        user, err := domainSvc.FindByID(ctx, uid)
        if err != nil {
            return err
        }
        
        result = toUserResult(user)
        return nil
    })
    
    return result, err
}
```

**特点**：
- ✅ 应用服务只有 10-15 行代码
- ✅ 领域逻辑在领域服务中
- ✅ 清晰的事务边界
- ✅ DTO 转换在应用层

---

## 重构步骤

### Step 1: 重构 User 领域服务

1. **删除当前的 domain/user/service/factory.go**（工具函数）

2. **创建完整的领域服务**：

```go
// domain/user/service/user_service.go
type UserDomainService struct {
    repo port.UserRepository
}

func NewUserDomainService(repo port.UserRepository) *UserDomainService

// 完整的业务方法（从原 application/ 层迁移）
func (s *UserDomainService) Register(...)
func (s *UserDomainService) Rename(...)
func (s *UserDomainService) UpdateContact(...)
func (s *UserDomainService) UpdateIDCard(...)
func (s *UserDomainService) Activate(...)
func (s *UserDomainService) Deactivate(...)
func (s *UserDomainService) Block(...)
func (s *UserDomainService) FindByID(...)
func (s *UserDomainService) FindByPhone(...)
```

3. **简化应用服务**：

```go
// application/user/user_app_service.go
// 每个方法只做：
// 1. 在事务中创建领域服务
// 2. 调用领域服务
// 3. DTO 转换
// 4. 返回结果
```

### Step 2: 重构 Child 和 Guardianship 领域服务

类似 User 的重构方式。

### Step 3: 更新 Handler 和 DI 容器

Handler 依赖应用服务（不变），DI 容器注册应用服务（不变）。

---

## 架构对比

### 重构前（原代码）
```
Handler 
  ↓ (依赖)
Application Service (实际是 Domain Service)
  ↓ (使用)
Repository
```

**问题**：
- ❌ Handler 跨层依赖（没有真正的应用层）
- ❌ 缺少事务管理
- ❌ 命名混乱

### 当前重构（过度设计）
```
Handler
  ↓
Application Service (过重，50+ 行)
  ↓
Domain Service (工具函数，无状态)
  ↓
Repository
```

**问题**：
- ❌ 应用服务过重
- ❌ 领域服务退化
- ❌ 领域逻辑分散

### 正确架构（推荐）
```
Handler (参数验证、响应格式化)
  ↓
Application Service (事务管理、DTO转换，10-15行)
  ↓
Domain Service (领域逻辑，30-50行)
  ↓
Repository
```

**优势**：
- ✅ 分层清晰
- ✅ 职责明确
- ✅ 领域服务可复用
- ✅ 应用服务轻量

---

## 总结

您的批评非常准确！当前重构确实存在**应用服务层过重**的问题。

### 核心错误
1. ❌ 领域服务设计成无状态工具函数
2. ❌ 应用服务承担领域逻辑编排
3. ❌ 违背"薄应用层"原则

### 正确做法
1. ✅ 领域服务 = 有状态 + 持有 Repository + 承载领域知识
2. ✅ 应用服务 = 轻量级 + 事务管理 + DTO 转换 + 调用领域服务
3. ✅ Handler = 参数验证 + 调用应用服务 + 响应格式化

### 下一步
是否需要我按照正确的架构重新重构 User 聚合？
