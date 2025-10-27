package user

import (
	"github.com/FangcunMount/component-base/pkg/util/idutil"
)

// UserID 用户唯一标识
type UserID = idutil.ID

// NewUserID 创建用户ID
func NewUserID(value uint64) UserID {
	return idutil.NewID(value)
}

// UserStatus 用户状态
type UserStatus uint8

func (s UserStatus) Uint64() interface{} {
	panic("unimplemented")
}

const (
	UserActive   UserStatus = 1 + iota // 1：活跃
	UserInactive                       // 2：非活跃
	UserBlocked                        // 3：被封禁
)

// Value 获取状态值
func (s UserStatus) Value() uint8 {
	return uint8(s)
}

// String 获取状态字符串
func (s UserStatus) String() string {
	switch s {
	case UserActive:
		return "active"
	case UserInactive:
		return "inactive"
	case UserBlocked:
		return "blocked"
	default:
		return "unknown"
	}
}
