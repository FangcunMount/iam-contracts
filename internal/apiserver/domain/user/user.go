package user

import "github.com/fangcun-mount/iam-contracts/internal/pkg/meta"

// User 基础用户（身份锚点）
type User struct {
	ID     UserID
	Name   string
	Phone  meta.Phone
	Email  meta.Email
	IDCard meta.IDCard
	Status UserStatus
}

// UserStatus 用户状态
func (u *User) Activate()   { u.Status = UserActive }
func (u *User) Deactivate() { u.Status = UserInactive }
func (u *User) Block()      { u.Status = UserBlocked }

// 检查用户状态
func (u *User) IsUsable() bool   { return u.Status == UserActive }
func (u *User) IsBlocked() bool  { return u.Status == UserBlocked }
func (u *User) IsInactive() bool { return u.Status == UserInactive }

// Rename 重命名用户
func (u *User) Rename(n string) { u.Name = n }

// UpdatePhone 更新手机号
func (u *User) UpdatePhone(p meta.Phone) { u.Phone = p }

// UpdateEmail 更新邮箱
func (u *User) UpdateEmail(e meta.Email) { u.Email = e }

// UpdateIDCard 更新身份证
func (u *User) UpdateIDCard(idc meta.IDCard) { u.IDCard = idc }
