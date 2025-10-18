# 用户中心 - 领域模型设计

> [返回用户中心文档](./README.md)

本文档详细介绍用户中心的领域模型设计，包括聚合根、实体和值对象。

---

## 3. 领域模型

### 3.1 聚合根设计

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

### 3.2 实体（Entities）

#### 3.2.1 User 聚合

```go
// internal/apiserver/modules/uc/domain/user/user.go
package user

type User struct {
    ID     UserID          // 唯一标识
    Name   string          // 用户名
    Phone  meta.Phone      // 手机号（值对象）
    Email  meta.Email      // 邮箱（值对象）
    IDCard meta.IDCard     // 身份证（值对象）
    Status UserStatus      // 状态（枚举）
}

// 工厂方法
func NewUser(name string, phone meta.Phone, opts ...UserOption) (*User, error)

// 领域方法
func (u *User) Activate()                     // 激活
func (u *User) Deactivate()                   // 停用
func (u *User) Block()                        // 封禁
func (u *User) UpdatePhone(p meta.Phone)      // 更新手机
func (u *User) UpdateEmail(e meta.Email)      // 更新邮箱
func (u *User) UpdateIDCard(idc meta.IDCard)  // 更新身份证
```

**业务规则**:

- ✅ 用户名不能为空
- ✅ 手机号必填且唯一
- ✅ 只有激活状态的用户才能登录
- ✅ 封禁用户无法进行任何操作

#### 3.2.2 Child 聚合

```go
// internal/apiserver/modules/uc/domain/child/child.go
package child

type Child struct {
    ID       ChildID
    Name     string
    IDCard   meta.IDCard
    Gender   meta.Gender
    Birthday meta.Birthday
    Height   meta.Height
    Weight   meta.Weight
}

// 工厂方法
func NewChild(name string, opts ...ChildOption) (*Child, error)

// 领域方法
func (c *Child) Rename(name string)
func (c *Child) UpdateIDCard(idc meta.IDCard)
func (c *Child) UpdateProfile(g meta.Gender, d meta.Birthday)
func (c *Child) UpdateHeightWeight(h meta.Height, w meta.Weight)
```

**业务规则**:

- ✅ 儿童姓名不能为空
- ✅ 性别、生日可选但一旦设置不建议修改
- ✅ 身份证号唯一（如果提供）
- ✅ 身高体重为监测数据，可多次更新

#### 3.2.3 Guardianship 聚合

```go
// internal/apiserver/modules/uc/domain/guardianship/guardianship.go
package guardianship

type Guardianship struct {
    ID        GuardianshipID
    UserID    user.UserID
    ChildID   child.ChildID
    Relation  Relation      // 监护关系类型
    GrantedAt time.Time
}

// 工厂方法
func NewGuardianship(
    userID user.UserID, 
    childID child.ChildID, 
    relation Relation,
) (*Guardianship, error)

// 领域方法
func (g *Guardianship) IsActive() bool
```

**业务规则**:

- ✅ 同一用户和儿童只能有一条监护关系
- ✅ 监护关系一旦建立不可修改，只能撤销后重新授予
- ✅ 必须同时提供有效的用户 ID 和儿童 ID

### 3.3 值对象（Value Objects）

```go
// internal/pkg/meta/phone.go
type Phone struct {
    CountryCode string  // 国家代码，如 +86
    Number      string  // 号码
}

// internal/pkg/meta/birthday.go
type Birthday struct {
    Year  int
    Month int
    Day   int
}

// internal/pkg/meta/gender.go
type Gender int
const (
    GenderUnknown Gender = 0
    GenderMale    Gender = 1
    GenderFemale  Gender = 2
)

// internal/pkg/meta/idcard.go
type IDCard struct {
    Name   string  // 姓名
    Number string  // 身份证号
}

// internal/pkg/meta/height.go
type Height struct {
    Centimeters float64
}

// internal/pkg/meta/weight.go
type Weight struct {
    Kilograms float64
}
```

**特性**:

- ✅ 不可变（Immutable）
- ✅ 值相等性
- ✅ 自包含验证逻辑
- ✅ 无副作用方法

---
