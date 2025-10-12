package user

import (
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// UserID 用户唯一标识
type UserID = meta.ID

// NewUserID 创建用户ID
func NewUserID(value uint64) UserID {
	return meta.NewID(value)
}

// ChildID 儿童唯一标识
type ChildID = meta.ID

// NewChildID 创建儿童ID
func NewChildID(value uint64) ChildID {
	return meta.NewID(value)
}

// UserStatus 用户状态
type UserStatus uint8

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

type Relation string // 监护关系
const (
	RelSelf         Relation = "self"         // 自己
	RelParent       Relation = "parent"       // 父母
	RelGrandparents Relation = "grandparents" // 祖父母
	RelOther        Relation = "other"        // 其他
)
