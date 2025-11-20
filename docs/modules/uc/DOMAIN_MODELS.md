# 用户中心 - 领域模型设计

> [返回用户中心文档](./README.md)

本文档详细介绍用户中心的领域模型设计，包括聚合根、实体、值对象和领域服务，深入阐述每个模型的职责和实现的领域知识。

---

## 目录

1. [领域概述](#1-领域概述)
2. [领域模型总览](#2-领域模型总览)
3. [User 聚合根](#3-user-聚合根)
4. [Child 聚合根](#4-child-聚合根)
5. [Guardianship 聚合根](#5-guardianship-聚合根)
6. [值对象](#6-值对象)
7. [领域服务](#7-领域服务)
8. [仓储接口](#8-仓储接口)

---

## 1. 领域概述

用户中心（UC）领域负责管理用户身份信息、儿童档案以及监护关系，是整个 IAM 系统的核心身份数据来源。

### 1.1 核心领域概念

- **用户（User）**：系统中的身份主体，拥有基本信息和状态
- **儿童（Child）**：需要被监护的未成年人档案
- **监护关系（Guardianship）**：连接用户和儿童的授权关系
- **身份锚点**：用户作为认证和授权的基础身份标识

### 1.2 领域边界

**本领域负责**：

- ✅ 用户基本信息管理（姓名、联系方式、身份证）
- ✅ 用户状态生命周期（激活、停用、封禁）
- ✅ 儿童档案创建和维护
- ✅ 监护关系的建立和撤销
- ✅ 用户和儿童的唯一性约束

**本领域不负责**：

- ❌ 用户认证（由 Authn 模块负责）
- ❌ 用户授权（由 Authz 模块负责）
- ❌ 儿童健康数据分析（业务领域）
- ❌ 数据持久化细节（由基础设施层负责）

---

## 2. 领域模型总览

### 2.1 聚合根设计

```text
┌─────────────────────────────────────────────────────────────┐
│                     UC Domain Model                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────┐         ┌─────────────────┐           │
│  │  User (聚合根)  │         │ Child (聚合根)  │           │
│  ├─────────────────┤         ├─────────────────┤           │
│  │ + ID            │         │ + ID            │           │
│  │ + Name          │         │ + Name          │           │
│  │ + Phone  (VO)   │◄───────►│ + Gender  (VO)  │           │
│  │ + Email  (VO)   │         │ + Birthday (VO) │           │
│  │ + IDCard (VO)   │   监护   │ + IDCard  (VO)  │           │
│  │ + Status (Enum) │   关系   │ + Height  (VO)  │           │
│  └─────────────────┘         │ + Weight  (VO)  │           │
│                               └─────────────────┘           │
│          │                            │                     │
│          │                            │                     │
│          │    ┌──────────────────┐    │                     │
│          └───►│  Guardianship    │◄───┘                     │
│               │   (聚合根)        │                          │
│               ├──────────────────┤                          │
│               │ + ID             │                          │
│               │ + UserID         │                          │
│               │ + ChildID        │                          │
│               │ + Relation (VO)  │                          │
│               │ + GrantedAt      │                          │
│               └──────────────────┘                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 聚合根职责划分

| 聚合根 | 核心职责 | 不变性维护 |
|--------|---------|-----------|
| **User** | 用户身份信息管理、状态生命周期 | 手机号唯一、状态转换规则 |
| **Child** | 儿童档案信息管理、成长数据记录 | 姓名必填、身份证唯一 |
| **Guardianship** | 监护关系建立和撤销 | 用户-儿童关系唯一性 |

---

## 3. User 聚合根

### 3.1 领域概念

**User（用户）** 是系统中的身份主体和锚点，代表一个真实用户在系统中的数字身份。用户聚合负责维护用户的基本信息、联系方式和生命周期状态。

### 3.2 聚合根定义

```go
// internal/apiserver/domain/uc/user/user.go
package user

type User struct {
    ID     meta.ID         // 唯一标识（统一 ID 类型）
    Name   string          // 用户名
    Phone  meta.Phone      // 手机号（值对象）
    Email  meta.Email      // 邮箱（值对象）
    IDCard meta.IDCard     // 身份证（值对象）
    Status UserStatus      // 状态（枚举）
}

// 工厂方法
func NewUser(name string, phone meta.Phone, opts ...UserOption) (*User, error)

// 领域方法 - 状态管理
func (u *User) Activate()                     // 激活用户
func (u *User) Deactivate()                   // 停用用户
func (u *User) Block()                        // 封禁用户

// 领域方法 - 信息更新
func (u *User) Rename(name string)            // 更新用户名
func (u *User) UpdatePhone(p meta.Phone)      // 更新手机号
func (u *User) UpdateEmail(e meta.Email)      // 更新邮箱
func (u *User) UpdateIDCard(idc meta.IDCard)  // 更新身份证

// 领域方法 - 状态查询
func (u *User) IsUsable() bool                // 是否可用（激活状态）
func (u *User) IsBlocked() bool               // 是否封禁
func (u *User) IsInactive() bool              // 是否停用
```

### 3.3 领域职责

| 职责类别 | 具体职责 | 实现的领域知识 |
|---------|---------|---------------|
| **身份管理** | 唯一身份标识、基本信息维护 | 用户是系统的身份锚点，手机号全局唯一 |
| **状态生命周期** | 激活、停用、封禁状态转换 | 封禁用户无法操作，停用用户暂时不可用 |
| **联系方式管理** | 手机号、邮箱的更新和验证 | 手机号变更需验证唯一性 |
| **个人信息保护** | 身份证号码的更新 | 身份证号敏感信息，需加密存储 |

### 3.4 业务不变性

User 聚合根维护以下业务不变性（Invariants）：

1. **唯一性约束**
   - ✅ 用户名不能为空
   - ✅ 手机号必填且全局唯一
   - ✅ 用户 ID 全局唯一

2. **状态约束**
   - ✅ 用户状态必须是有效枚举值（Active/Inactive/Blocked）
   - ✅ 只有激活（Active）状态的用户才能正常使用系统
   - ✅ 封禁（Blocked）用户无法进行任何操作

3. **数据完整性**
   - ✅ 手机号格式必须有效（E.164 格式）
   - ✅ 邮箱格式必须有效（如果提供）
   - ✅ 身份证号格式必须有效（如果提供）

### 3.5 状态转换图

```text
     创建
      │
      ▼
  [Active]  ←──── Activate() ────┐
      │                          │
      │ Deactivate()             │
      ▼                          │
  [Inactive] ───────────────────┘
      │
      │ Block()
      ▼
  [Blocked]
   (终态)
```

**状态转换规则**：

- Active → Inactive：临时停用，可重新激活
- Inactive → Active：重新激活
- Active/Inactive → Blocked：永久封禁，不可逆

---

## 4. Child 聚合根

### 4.1 领域概念

**Child（儿童）** 代表需要被监护的未成年人档案，包含儿童的基本信息和成长数据。儿童聚合独立于用户存在，通过监护关系与用户关联。

### 4.2 聚合根定义

```go
// internal/apiserver/domain/uc/child/child.go
package child

type Child struct {
    ID       meta.ID         // 唯一标识（统一 ID 类型）
    Name     string          // 儿童姓名
    IDCard   meta.IDCard     // 身份证（值对象）
    Gender   meta.Gender     // 性别（值对象）
    Birthday meta.Birthday   // 出生日期（值对象）
    Height   meta.Height     // 身高（值对象）
    Weight   meta.Weight     // 体重（值对象）
}

// 工厂方法
func NewChild(name string, opts ...ChildOption) (*Child, error)

// 领域方法 - 基本信息管理
func (c *Child) Rename(name string)                                 // 修改姓名
func (c *Child) UpdateIDCard(idc meta.IDCard)                       // 更新身份证
func (c *Child) UpdateProfile(g meta.Gender, d meta.Birthday)       // 更新档案信息
func (c *Child) UpdateHeightWeight(h meta.Height, w meta.Weight)    // 更新身高体重
```

### 4.3 领域职责

| 职责类别 | 具体职责 | 实现的领域知识 |
|---------|---------|---------------|
| **档案管理** | 儿童基本信息维护 | 儿童档案独立存在，可被多个监护人关联 |
| **身份识别** | 姓名、性别、生日、身份证 | 身份证号唯一，可用于查重和身份验证 |
| **成长数据** | 身高体重记录 | 身高体重随时间变化，支持多次更新 |
| **查重检测** | 基于姓名+生日+性别查找相似儿童 | 防止重复建档，保护儿童隐私 |

### 4.4 业务不变性

Child 聚合根维护以下业务不变性：

1. **唯一性约束**
   - ✅ 儿童姓名不能为空
   - ✅ 身份证号唯一（如果提供）
   - ✅ 儿童 ID 全局唯一

2. **数据完整性**
   - ✅ 性别必须是有效枚举值（Unknown/Male/Female）
   - ✅ 生日不能是未来日期
   - ✅ 身高体重必须是合理的正数值

3. **信息稳定性**
   - ✅ 性别、生日属于核心档案信息，一旦设置不建议频繁修改
   - ✅ 身高体重属于监测数据，可随时更新

### 4.5 查重策略

儿童档案的查重基于以下维度：

```text
查重匹配度 = f(姓名相似度, 生日匹配, 性别匹配)

高风险重复：
- 姓名完全相同 + 生日相同 + 性别相同
- 建议提示用户确认

中风险重复：
- 姓名相似 + 生日相同
- 提供候选列表供用户选择

低风险：
- 仅姓名相似
- 提示但允许继续创建
```

---

## 5. Guardianship 聚合根

### 5.1 领域概念

**Guardianship（监护关系）** 表示用户对儿童的监护权，连接用户和儿童两个聚合根。监护关系独立存在，具有自己的生命周期。

### 5.2 聚合根定义

```go
// internal/apiserver/domain/uc/guardianship/guardianship.go
package guardianship

type Guardianship struct {
    ID            meta.ID         // 唯一标识（统一 ID 类型）
    User          meta.ID         // 用户 ID（监护人）
    Child         meta.ID         // 儿童 ID（被监护人）
    Rel           Relation        // 监护关系类型
    EstablishedAt time.Time       // 建立时间
    RevokedAt     *time.Time      // 撤销时间（nil 表示未撤销）
}

// 工厂方法
func NewGuardianship(
    userID meta.ID, 
    childID meta.ID, 
    relation Relation,
) (*Guardianship, error)

// 领域方法
func (g *Guardianship) IsActive() bool     // 监护关系是否有效
func (g *Guardianship) Revoke()            // 撤销监护关系
```

### 5.3 领域职责

| 职责类别 | 具体职责 | 实现的领域知识 |
|---------|---------|---------------|
| **关系建立** | 创建用户与儿童的监护关系 | 监护关系是授权的基础，必须先建立才能操作儿童数据 |
| **关系管理** | 监护关系的撤销和查询 | 支持监护权的转移（撤销旧关系+建立新关系） |
| **关系类型** | 区分父母、监护人等角色 | 不同关系类型可能有不同的权限（扩展点） |
| **唯一性保证** | 同一用户-儿童对只能有一个有效关系 | 防止重复授权，保证关系清晰 |

### 5.4 业务不变性

Guardianship 聚合根维护以下业务不变性：

1. **关系唯一性**
   - ✅ 同一用户和儿童只能有一条**有效**的监护关系
   - ✅ 已撤销的关系不影响新关系的建立
   - ✅ 监护关系 ID 全局唯一

2. **引用完整性**
   - ✅ User ID 必须指向有效的用户
   - ✅ Child ID 必须指向有效的儿童
   - ✅ 关系类型必须是有效的枚举值

3. **时间约束**
   - ✅ 建立时间不能是未来时间
   - ✅ 撤销时间必须晚于建立时间
   - ✅ 已撤销的关系不可再次撤销

### 5.5 监护关系类型

```go
type Relation string

const (
    RelationParent   Relation = "parent"    // 父母
    RelationGuardian Relation = "guardian"  // 监护人
)
```

**关系类型说明**：

- **parent（父母）**：生物学父母或法定监护人
- **guardian（监护人）**：经授权的其他监护人（如祖父母、亲属等）

### 5.6 监护关系生命周期

```text
     创建
      │
      ▼
  [Active]  ────────────────┐
   有效状态                  │
      │                     │ 查询监护人
      │                     │ 查询儿童
      │ Revoke()            │ 操作儿童数据
      ▼                     │
  [Revoked] ◄───────────────┘
   已撤销
   (终态)
```

**生命周期说明**：

- 监护关系一旦建立即进入 Active 状态
- Active 状态可以执行所有监护相关操作
- 撤销后进入 Revoked 终态，不可恢复
- 需要重新授权时，创建新的监护关系

---

## 6. 值对象

### 6.1 值对象定义

值对象是 DDD 中描述领域概念的重要组成部分，它们通过值相等性而非身份来识别。

```go
// internal/pkg/meta/phone.go
type Phone struct {
    CountryCode string  // 国家代码，如 +86
    Number      string  // 电话号码
}
func (p Phone) String() string          // E.164 格式化
func (p Phone) IsEmpty() bool           // 是否为空
func (p Phone) Equal(other Phone) bool  // 值相等性比较

// internal/pkg/meta/email.go
type Email struct {
    Address string  // 邮箱地址
}
func (e Email) String() string          // 获取邮箱地址
func (e Email) IsEmpty() bool           // 是否为空
func (e Email) Equal(other Email) bool  // 值相等性比较

// internal/pkg/meta/birthday.go
type Birthday struct {
    Year  int
    Month int
    Day   int
}
func (b Birthday) String() string       // YYYY-MM-DD 格式
func (b Birthday) IsZero() bool         // 是否为零值
func (b Birthday) Age() int             // 计算年龄

// internal/pkg/meta/gender.go
type Gender int
const (
    GenderUnknown Gender = 0  // 未知
    GenderMale    Gender = 1  // 男
    GenderFemale  Gender = 2  // 女
)
func (g Gender) String() string         // 性别描述

// internal/pkg/meta/idcard.go
type IDCard struct {
    Number string  // 身份证号
}
func (i IDCard) String() string         // 脱敏显示
func (i IDCard) IsEmpty() bool          // 是否为空
func (i IDCard) Validate() error        // 验证身份证格式

// internal/pkg/meta/height.go
type Height struct {
    Centimeters float64  // 厘米
}
func (h Height) String() string         // 格式化输出
func (h Height) IsZero() bool           // 是否为零值

// internal/pkg/meta/weight.go
type Weight struct {
    Kilograms float64  // 千克
}
func (w Weight) String() string         // 格式化输出
func (w Weight) IsZero() bool           // 是否为零值
```

### 6.2 值对象特性

| 特性 | 说明 | 实现的领域知识 |
|-----|------|---------------|
| **不可变性** | 一旦创建不可修改 | 保证值对象的线程安全和语义清晰 |
| **值相等性** | 通过值而非引用比较 | 两个电话号码相同即视为相等 |
| **自包含验证** | 验证逻辑封装在值对象内 | 无效的值对象无法被创建 |
| **领域语义** | 表达领域概念而非原始类型 | `meta.Phone` 比 `string` 更具语义 |
| **无副作用** | 所有方法都是纯函数 | 调用方法不会改变对象状态 |

### 6.3 值对象的领域意义

#### Phone（电话号码）

**领域知识**：

- 电话号码是用户的唯一身份标识
- 支持国际化（E.164 格式）
- 手机号全局唯一，是重要的业务主键

**验证规则**：

- 必须符合 E.164 国际标准
- 中国大陆手机号：11 位数字，1开头

#### IDCard（身份证）

**领域知识**：

- 身份证号是敏感信息，需要加密存储
- 显示时需要脱敏（如：110***********1234）
- 可用于实名验证和查重

**验证规则**：

- 18 位数字或末位 X
- 校验码验证
- 出生日期合法性验证

#### Birthday（生日）

**领域知识**：

- 儿童的生日用于年龄计算和查重
- 生日不可是未来日期
- 可计算当前年龄

#### Height / Weight（身高体重）

**领域知识**：

- 儿童成长监测数据
- 支持多次更新记录成长曲线
- 需要合理范围验证

---

## 7. 领域服务

领域服务封装了跨聚合根的业务逻辑或复杂的业务规则，它们是无状态的，通过依赖仓储来访问聚合根。

### 7.1 User 领域服务

#### 7.1.1 Validator（用户验证器）

**职责**：封装用户相关的验证规则和业务检查

```go
// internal/apiserver/domain/uc/user/validator.go
type Validator interface {
    // ValidateRegister 验证注册参数
    // 领域知识：手机号必须全局唯一
    ValidateRegister(ctx context.Context, name string, phone meta.Phone) error
    
    // ValidateRename 验证改名参数
    // 领域知识：用户名不能为空
    ValidateRename(name string) error
    
    // ValidateUpdateContact 验证更新联系方式参数
    // 领域知识：手机号变更需要验证唯一性
    ValidateUpdateContact(ctx context.Context, user *User, 
        phone meta.Phone, email meta.Email) error
    
    // CheckPhoneUnique 检查手机号唯一性
    CheckPhoneUnique(ctx context.Context, phone meta.Phone) error
}
```

**实现的领域知识**：

1. **手机号唯一性约束**
   - 注册时检查手机号是否已存在
   - 更新手机号时检查新手机号的唯一性
   - 只有手机号变更时才需要检查

2. **基本信息验证**
   - 用户名不能为空或纯空格
   - 手机号格式必须合法
   - 邮箱格式必须合法（如果提供）

#### 7.1.2 ProfileEditor（资料编辑器）

**职责**：负责用户资料的修改操作，协调验证和状态更新

```go
// internal/apiserver/domain/uc/user/profile_editor.go
type ProfileEditor interface {
    // Rename 修改用户名称
    // 领域知识：验证新名称 + 更新用户实体 + 返回修改后的用户
    Rename(ctx context.Context, id meta.ID, newName string) (*User, error)
    
    // UpdateContact 更新联系方式
    // 领域知识：验证唯一性 + 更新用户实体 + 返回修改后的用户
    UpdateContact(ctx context.Context, id meta.ID, 
        phone meta.Phone, email meta.Email) (*User, error)
    
    // UpdateIDCard 更新身份证
    // 领域知识：验证身份证格式 + 更新用户实体
    UpdateIDCard(ctx context.Context, id meta.ID, idCard meta.IDCard) (*User, error)
}
```

**实现的领域知识**：

1. **资料修改流程**
   - 加载用户聚合根
   - 调用 Validator 验证新数据
   - 调用聚合根方法修改状态
   - 返回修改后的聚合根（由应用层持久化）

2. **关注点分离**
   - 验证逻辑由 Validator 负责
   - 状态更新由聚合根方法负责
   - 持久化由应用层负责

#### 7.1.3 Lifecycler（生命周期管理器）

**职责**：负责用户状态的变更操作

```go
// internal/apiserver/domain/uc/user/lifecycler.go
type Lifecycler interface {
    // Activate 激活用户
    // 领域知识：只有非封禁状态的用户可以激活
    Activate(ctx context.Context, id meta.ID) (*User, error)
    
    // Deactivate 停用用户
    // 领域知识：临时停用，可重新激活
    Deactivate(ctx context.Context, id meta.ID) (*User, error)
    
    // Block 封禁用户
    // 领域知识：永久封禁，不可逆操作
    Block(ctx context.Context, id meta.ID) (*User, error)
}
```

**实现的领域知识**：

1. **状态转换规则**
   - Active ⇄ Inactive：双向转换，可多次切换
   - Active/Inactive → Blocked：单向转换，不可逆
   - Blocked 状态不能转换到其他状态

2. **业务语义**
   - Deactivate：临时不可用（如用户主动停用）
   - Block：永久封禁（如违规行为）

### 7.2 Child 领域服务

#### 7.2.1 Validator（儿童验证器）

**职责**：封装儿童档案相关的验证规则

```go
// internal/apiserver/domain/uc/child/validator.go
type Validator interface {
    // ValidateRegister 验证注册参数
    // 领域知识：姓名不能为空，生日不能是未来
    ValidateRegister(ctx context.Context, name string, 
        gender meta.Gender, birthday meta.Birthday) error
    
    // ValidateRename 验证改名参数
    ValidateRename(name string) error
    
    // ValidateUpdateProfile 验证资料更新参数
    ValidateUpdateProfile(gender meta.Gender, birthday meta.Birthday) error
}
```

**实现的领域知识**：

1. **档案完整性**
   - 儿童姓名不能为空
   - 生日必须是过去的日期
   - 性别必须是有效的枚举值

2. **查重检测**
   - 基于姓名+生日+性别查找相似儿童
   - 提供候选列表供用户确认
   - 防止重复建档

#### 7.2.2 ProfileEditor（档案编辑器）

**职责**：负责儿童档案的修改操作

```go
// internal/apiserver/domain/uc/child/editor.go
type ProfileEditor interface {
    // Rename 修改儿童姓名
    Rename(ctx context.Context, childID meta.ID, name string) (*Child, error)
    
    // UpdateIDCard 更新身份证
    UpdateIDCard(ctx context.Context, childID meta.ID, 
        idCard meta.IDCard) (*Child, error)
    
    // UpdateProfile 更新档案信息（性别、生日）
    UpdateProfile(ctx context.Context, childID meta.ID, 
        gender meta.Gender, birthday meta.Birthday) (*Child, error)
    
    // UpdateHeightWeight 更新身高体重
    UpdateHeightWeight(ctx context.Context, childID meta.ID, 
        height meta.Height, weight meta.Weight) (*Child, error)
}
```

**实现的领域知识**：

1. **信息稳定性分级**
   - **核心档案**（姓名、性别、生日）：不建议频繁修改
   - **监测数据**（身高、体重）：支持多次更新

2. **编辑流程**
   - 验证 → 加载聚合根 → 更新状态 → 返回

### 7.3 Guardianship 领域服务

#### 7.3.1 Manager（监护关系管理器）

**职责**：负责监护关系的建立和撤销

```go
// internal/apiserver/domain/uc/guardianship/manager.go
type Manager interface {
    // AddGuardian 添加监护人
    // 领域知识：
    // 1. 验证用户和儿童存在性
    // 2. 验证监护关系不重复
    // 3. 创建监护关系实体
    AddGuardian(ctx context.Context, userID meta.ID, 
        childID meta.ID, relation Relation) (*Guardianship, error)
    
    // RemoveGuardian 撤销监护
    // 领域知识：
    // 1. 查询监护关系
    // 2. 验证关系存在且有效
    // 3. 标记为已撤销
    RemoveGuardian(ctx context.Context, userID meta.ID, 
        childID meta.ID) (*Guardianship, error)
}
```

**实现的领域知识**：

1. **监护关系唯一性**
   - 同一用户-儿童对只能有一个**有效**的监护关系
   - 检查所有现有监护关系，确保无重复

2. **引用完整性验证**
   - User ID 必须指向有效的用户
   - Child ID 必须指向有效的儿童
   - 验证失败时返回明确的错误信息

3. **撤销语义**
   - 撤销不是删除，而是标记为失效
   - 保留历史记录，支持审计
   - 撤销后可以重新授权（创建新关系）

### 7.4 领域服务设计原则

#### 7.4.1 无状态性

所有领域服务都是无状态的，不存储任何领域对象：

```go
// ✅ 正确：领域服务无状态
type validator struct {
    repo Repository  // 仅依赖仓储接口
}

// ❌ 错误：领域服务不应该缓存聚合根
type validator struct {
    users map[string]*User  // 不要这样做
}
```

#### 7.4.2 职责单一

每个领域服务只负责一类职责：

- **Validator**：验证规则
- **ProfileEditor**：资料编辑
- **Lifecycler**：状态管理
- **Manager**：关系管理

#### 7.4.3 依赖倒置

领域服务依赖仓储接口（Driven Port），而非具体实现：

```go
type Validator interface {
    ValidateRegister(ctx context.Context, name string, phone meta.Phone) error
}

type validator struct {
    repo Repository  // 依赖接口，不是具体实现
}
```

---

## 8. 仓储接口

仓储接口定义了聚合根的持久化操作，是领域层定义、基础设施层实现的 Driven Port。

### 8.1 User 仓储

```go
// internal/apiserver/domain/uc/user/repository.go
type Repository interface {
    Create(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id meta.ID) (*User, error)
    FindByPhone(ctx context.Context, phone meta.Phone) (*User, error)
    Update(ctx context.Context, user *User) error
}
```

**职责**：

- Create：持久化新用户
- FindByID：根据 ID 查询用户
- FindByPhone：根据手机号查询（支持唯一性检查）
- Update：更新用户信息（使用乐观锁）

### 8.2 Child 仓储

```go
// internal/apiserver/domain/uc/child/repository.go
type Repository interface {
    Create(ctx context.Context, child *Child) error
    FindByID(ctx context.Context, id meta.ID) (*Child, error)
    FindByIDCard(ctx context.Context, idCard meta.IDCard) (*Child, error)
    FindSimilar(ctx context.Context, name string, gender meta.Gender, 
        birthday meta.Birthday) ([]*Child, error)
    Update(ctx context.Context, child *Child) error
}
```

**职责**：

- Create：持久化新儿童档案
- FindByID：根据 ID 查询儿童
- FindByIDCard：根据身份证查询
- FindSimilar：查找相似儿童（查重）
- Update：更新儿童档案

### 8.3 Guardianship 仓储

```go
// internal/apiserver/domain/uc/guardianship/repository.go
type Repository interface {
    Create(ctx context.Context, guardianship *Guardianship) error
    FindByID(ctx context.Context, id meta.ID) (*Guardianship, error)
    FindByUserID(ctx context.Context, userID meta.ID) ([]*Guardianship, error)
    FindByChildID(ctx context.Context, childID meta.ID) ([]*Guardianship, error)
    FindByUserAndChild(ctx context.Context, userID, childID meta.ID) (*Guardianship, error)
    Update(ctx context.Context, guardianship *Guardianship) error
}
```

**职责**：

- Create：持久化新监护关系
- FindByID：根据 ID 查询
- FindByUserID：查询用户的所有监护关系
- FindByChildID：查询儿童的所有监护人
- FindByUserAndChild：查询特定用户-儿童的监护关系
- Update：更新监护关系（如撤销）

---

## 9. 总结

### 9.1 聚合根职责总结

| 聚合根 | 核心职责 | 关键领域知识 |
|-------|---------|------------|
| **User** | 用户身份管理、状态生命周期 | 手机号唯一、状态转换规则、身份锚点 |
| **Child** | 儿童档案管理、成长数据记录 | 查重检测、信息稳定性、监测数据 |
| **Guardianship** | 监护关系建立和撤销 | 关系唯一性、引用完整性、撤销语义 |

### 9.2 领域服务职责总结

| 领域服务 | 所属聚合 | 核心职责 | 关键领域知识 |
|---------|---------|---------|------------|
| **Validator** | User | 用户验证 | 手机号唯一性、基本信息验证 |
| **ProfileEditor** | User | 资料编辑 | 验证协调、状态更新 |
| **Lifecycler** | User | 状态管理 | 状态转换规则、业务语义 |
| **Validator** | Child | 儿童验证 | 档案完整性、查重检测 |
| **ProfileEditor** | Child | 档案编辑 | 信息稳定性分级 |
| **Manager** | Guardianship | 关系管理 | 唯一性约束、引用完整性 |

### 9.3 设计亮点

1. **清晰的领域边界**：三个聚合根职责明确，互不干扰
2. **丰富的领域知识**：每个模型都体现了真实业务规则
3. **高内聚低耦合**：领域服务协调聚合根，但不破坏封装
4. **可测试性**：领域逻辑独立于基础设施，易于单元测试
5. **扩展性**：通过值对象和领域服务，易于扩展新功能

---

**最后更新**: 2025-11-20  
**维护团队**: IAM Team
