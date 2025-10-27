package user

import (
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/FangcunMount/iam-contracts/pkg/errors"
)

// User 基础用户（身份锚点）
type User struct {
	ID     UserID
	Name   string
	Phone  meta.Phone
	Email  meta.Email
	IDCard meta.IDCard
	Status UserStatus
}

// NewUser 创建新用户（完整信息）
func NewUser(name string, phone meta.Phone, opts ...UserOption) (*User, error) {
	if name == "" {
		return nil, errors.WithCode(code.ErrUserBasicInfoInvalid, "name cannot be empty")
	}
	if phone.IsEmpty() {
		return nil, errors.WithCode(code.ErrUserBasicInfoInvalid, "phone cannot be empty")
	}

	user := &User{
		Name:   name,
		Phone:  phone,
		Status: UserActive, // 新用户默认为活跃状态
	}
	for _, opt := range opts {
		opt(user)
	}

	return user, nil
}

// UserOption 用户选项，用于创建用户时的可选参数
type UserOption func(*User)

// With*** 用户选项函数
func WithID(id UserID) UserOption              { return func(u *User) { u.ID = id } }
func WithEmail(email meta.Email) UserOption    { return func(u *User) { u.Email = email } }
func WithIDCard(idCard meta.IDCard) UserOption { return func(u *User) { u.IDCard = idCard } }
func WithStatus(status UserStatus) UserOption  { return func(u *User) { u.Status = status } }

// UserStatus 用户状态
func (u *User) Activate()   { u.Status = UserActive }
func (u *User) Deactivate() { u.Status = UserInactive }
func (u *User) Block()      { u.Status = UserBlocked }

// 检查用户状态
func (u *User) IsUsable() bool   { return u.Status == UserActive }
func (u *User) IsBlocked() bool  { return u.Status == UserBlocked }
func (u *User) IsInactive() bool { return u.Status == UserInactive }

// UpdatePhone 更新手机号
func (u *User) UpdatePhone(p meta.Phone) { u.Phone = p }

// UpdateEmail 更新邮箱
func (u *User) UpdateEmail(e meta.Email) { u.Email = e }

// UpdateIDCard 更新身份证
func (u *User) UpdateIDCard(idc meta.IDCard) { u.IDCard = idc }
