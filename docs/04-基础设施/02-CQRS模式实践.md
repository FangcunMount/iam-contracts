# CQRS 模式实践

> 🎯 **核心结论**: 通过命令查询分离，优化读写性能，提升代码可维护性

---

## 1. 模式概述

### 1.1 CQRS 架构图

```text
┌─────────────────────────────────────────────────────────────────┐
│                           客户端                                 │
│                     ┌─────────┬─────────┐                       │
│                     │  写请求  │  读请求  │                       │
│                     └────┬────┴────┬────┘                       │
│                          │         │                             │
└──────────────────────────┼─────────┼─────────────────────────────┘
                           │         │
                           ▼         ▼
┌──────────────────────────────────────────────────────────────────┐
│                        Application Layer                          │
├─────────────────────────────┬────────────────────────────────────┤
│      Command Service        │        Query Service               │
│  ┌─────────────────────┐   │   ┌─────────────────────┐          │
│  │ UserAppService      │   │   │ UserQueryService    │          │
│  │ - Register()        │   │   │ - GetByID()         │          │
│  │ - UpdateProfile()   │   │   │ - GetByPhone()      │          │
│  │ - BindChild()       │   │   │ - ListChildren()    │          │
│  └─────────────────────┘   │   └─────────────────────┘          │
└─────────────────────────────┴────────────────────────────────────┘
                           │         │
                           ▼         ▼
┌──────────────────────────────────────────────────────────────────┐
│                         Domain Layer                              │
├─────────────────────────────┬────────────────────────────────────┤
│    Domain Services          │    Query直接访问数据               │
│    Aggregate Roots          │    (跳过领域层)                    │
│    Domain Events            │                                    │
└─────────────────────────────┴────────────────────────────────────┘
                           │         │
                           ▼         ▼
┌──────────────────────────────────────────────────────────────────┐
│                      Infrastructure Layer                         │
├─────────────────────────────┬────────────────────────────────────┤
│      Write Repository       │        Read Repository             │
│  ┌─────────────────────┐   │   ┌─────────────────────┐          │
│  │ 通过聚合根操作       │   │   │ 直接 SQL 查询       │          │
│  │ 保证业务规则         │   │   │ 优化查询性能        │          │
│  └─────────────────────┘   │   └─────────────────────┘          │
└─────────────────────────────┴────────────────────────────────────┘
                           │         │
                           ▼         ▼
┌──────────────────────────────────────────────────────────────────┐
│                         Database                                  │
│                    ┌───────────────────┐                         │
│                    │      MySQL        │                         │
│                    └───────────────────┘                         │
└──────────────────────────────────────────────────────────────────┘
```

### 1.2 为什么使用 CQRS

| 问题 | CQRS 解决方案 |
|------|-------------|
| 读写模型不一致 | 分离命令和查询，各自优化 |
| 复杂查询影响写性能 | 查询直接访问数据库，跳过领域层 |
| 聚合根加载过重 | 查询只加载需要的字段 |
| 代码职责混乱 | 命令和查询分离，职责清晰 |

---

## 2. 命令端设计

### 2.1 命令服务

```go
// 伪代码: 命令应用服务
// 源码: internal/apiserver/application/uc/user_app_service.go

type UserAppService struct {
    userRepo   UserRepository
    registerSvc *RegisterService
    eventBus   EventBus
}

// 命令: 注册用户 (写操作)
func (s *UserAppService) Register(ctx context.Context, cmd RegisterCommand) (*User, error) {
    // 1. 使用领域服务处理业务逻辑
    user, err := s.registerSvc.Register(ctx, RegisterRequest{
        Profile: cmd.Profile,
        Contact: cmd.Contact,
    })
    if err != nil {
        return nil, err
    }
    
    // 2. 发布领域事件
    s.eventBus.Publish(UserRegisteredEvent{
        UserID:    user.ID,
        Timestamp: time.Now(),
    })
    
    return user, nil
}

// 命令: 更新用户档案 (写操作)
func (s *UserAppService) UpdateProfile(ctx context.Context, cmd UpdateProfileCommand) error {
    // 1. 加载聚合根
    user, err := s.userRepo.FindByID(ctx, cmd.UserID)
    if err != nil {
        return err
    }
    
    // 2. 执行领域逻辑
    if err := user.UpdateProfile(cmd.Profile); err != nil {
        return err
    }
    
    // 3. 持久化
    return s.userRepo.Save(ctx, user)
}

// 命令: 绑定儿童 (写操作)
func (s *UserAppService) BindChild(ctx context.Context, cmd BindChildCommand) error {
    // ... 通过领域服务处理跨聚合操作
}
```

### 2.2 命令对象

```go
// 伪代码: 命令定义
// 源码: internal/apiserver/application/uc/command.go

// 注册命令
type RegisterCommand struct {
    Profile Profile
    Contact Contact
}

// 更新档案命令
type UpdateProfileCommand struct {
    UserID  UserID
    Profile Profile
}

// 绑定儿童命令
type BindChildCommand struct {
    GuardianID   UserID
    ChildID      ChildID
    GuardianType GuardianType
}

// 命令验证
func (c *RegisterCommand) Validate() error {
    if err := c.Profile.Validate(); err != nil {
        return err
    }
    return c.Contact.Validate()
}
```

---

## 3. 查询端设计

### 3.1 查询服务

```go
// 伪代码: 查询应用服务
// 源码: internal/apiserver/application/uc/user_query_service.go

type UserQueryService struct {
    db *gorm.DB  // 直接使用 DB，跳过领域层
}

// 查询: 获取用户 (读操作)
func (s *UserQueryService) GetByID(ctx context.Context, userID string) (*UserDTO, error) {
    var dto UserDTO
    
    // 直接 SQL 查询，只获取需要的字段
    err := s.db.WithContext(ctx).
        Table("users").
        Select("id, nickname, avatar, phone, created_at").
        Where("id = ?", userID).
        Scan(&dto).Error
        
    if err != nil {
        return nil, err
    }
    
    return &dto, nil
}

// 查询: 分页列表 (读操作)
func (s *UserQueryService) List(ctx context.Context, query ListUsersQuery) (*PagedResult[UserDTO], error) {
    var users []UserDTO
    var total int64
    
    db := s.db.WithContext(ctx).Table("users")
    
    // 应用过滤条件
    if query.Status != "" {
        db = db.Where("status = ?", query.Status)
    }
    
    // 统计总数
    db.Count(&total)
    
    // 分页查询
    err := db.
        Select("id, nickname, avatar, phone, created_at").
        Offset(query.Offset()).
        Limit(query.PageSize).
        Order("created_at DESC").
        Scan(&users).Error
        
    if err != nil {
        return nil, err
    }
    
    return &PagedResult[UserDTO]{
        Items: users,
        Total: total,
        Page:  query.Page,
        Size:  query.PageSize,
    }, nil
}

// 查询: 复杂聚合查询
func (s *UserQueryService) GetUserWithChildren(ctx context.Context, userID string) (*UserWithChildrenDTO, error) {
    var dto UserWithChildrenDTO
    
    // 使用 JOIN 一次获取用户和儿童信息
    err := s.db.WithContext(ctx).Raw(`
        SELECT 
            u.id, u.nickname, u.avatar,
            c.id as child_id, c.name as child_name, c.birthday as child_birthday
        FROM users u
        LEFT JOIN guardianships g ON u.id = g.guardian_id AND g.status = 'active'
        LEFT JOIN children c ON g.child_id = c.id
        WHERE u.id = ?
    `, userID).Scan(&dto).Error
    
    return &dto, err
}
```

### 3.2 查询对象与 DTO

```go
// 伪代码: 查询对象和 DTO
// 源码: internal/apiserver/application/uc/dto.go

// 查询参数
type ListUsersQuery struct {
    Page     int
    PageSize int
    Status   string
    Keyword  string
}

func (q *ListUsersQuery) Offset() int {
    return (q.Page - 1) * q.PageSize
}

// 用户 DTO (只包含需要展示的字段)
type UserDTO struct {
    ID        string    `json:"id"`
    Nickname  string    `json:"nickname"`
    Avatar    string    `json:"avatar"`
    Phone     string    `json:"phone"`
    CreatedAt time.Time `json:"created_at"`
}

// 用户带儿童 DTO
type UserWithChildrenDTO struct {
    UserDTO
    Children []ChildDTO `json:"children"`
}

// 分页结果
type PagedResult[T any] struct {
    Items []T   `json:"items"`
    Total int64 `json:"total"`
    Page  int   `json:"page"`
    Size  int   `json:"size"`
}
```

---

## 4. Handler 层实现

### 4.1 统一 Handler

```go
// 伪代码: REST Handler
// 源码: internal/apiserver/interface/rest/user_handler.go

type UserHandler struct {
    commandService *UserAppService
    queryService   *UserQueryService
}

func NewUserHandler(cmd *UserAppService, query *UserQueryService) *UserHandler {
    return &UserHandler{
        commandService: cmd,
        queryService:   query,
    }
}

// 写操作: 调用 Command Service
func (h *UserHandler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "invalid request"})
        return
    }
    
    user, err := h.commandService.Register(c.Request.Context(), RegisterCommand{
        Profile: req.ToProfile(),
        Contact: req.ToContact(),
    })
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, user)
}

// 读操作: 调用 Query Service
func (h *UserHandler) GetUser(c *gin.Context) {
    userID := c.Param("id")
    
    user, err := h.queryService.GetByID(c.Request.Context(), userID)
    if err != nil {
        c.JSON(404, gin.H{"error": "user not found"})
        return
    }
    
    c.JSON(200, user)
}

// 读操作: 分页列表
func (h *UserHandler) ListUsers(c *gin.Context) {
    query := ListUsersQuery{
        Page:     c.GetInt("page"),
        PageSize: c.GetInt("page_size"),
        Status:   c.Query("status"),
    }
    
    result, err := h.queryService.List(c.Request.Context(), query)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, result)
}
```

---

## 5. 最佳实践

### 5.1 何时使用 CQRS

```text
┌─────────────────────────────────────────────────────────────┐
│                    CQRS 适用场景                             │
├─────────────────────────────────────────────────────────────┤
│  ✅ 适用:                                                    │
│  - 读写比例失衡 (读多写少)                                   │
│  - 读写模型差异大                                            │
│  - 需要复杂查询聚合                                          │
│  - 需要读写分离数据库                                        │
│                                                              │
│  ❌ 不适用:                                                  │
│  - 简单 CRUD 应用                                            │
│  - 读写模型一致                                              │
│  - 团队经验不足                                              │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 注意事项

| 注意点 | 说明 |
|--------|------|
| **写操作必须通过聚合根** | 保证业务规则一致性 |
| **查询可以跳过领域层** | 性能优先，直接访问数据库 |
| **DTO 不暴露领域细节** | 查询返回专用 DTO |
| **命令应有幂等性** | 便于重试和错误恢复 |

### 5.3 代码组织

```text
application/uc/
├── user_app_service.go       # 命令服务 (写)
├── user_query_service.go     # 查询服务 (读)
├── command.go                # 命令定义
├── dto.go                    # DTO 定义
└── assembler.go              # 转换器
```

---

## 6. 源码索引

| 组件 | 路径 | 说明 |
|------|------|------|
| **命令服务** | | |
| UserAppService | `application/uc/user_app_service.go` | 用户命令服务 |
| ChildAppService | `application/uc/child_app_service.go` | 儿童命令服务 |
| **查询服务** | | |
| UserQueryService | `application/uc/user_query_service.go` | 用户查询服务 |
| ChildQueryService | `application/uc/child_query_service.go` | 儿童查询服务 |
| **DTO** | | |
| UserDTO | `application/uc/dto.go` | 用户数据传输对象 |
| PagedResult | `application/common/pagination.go` | 分页结果 |
