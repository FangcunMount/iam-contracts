package account

import (
	"database/sql/driver"
	"fmt"
	"strconv"
)

// UserID 代表认证模块中的用户标识
// 注意: 这是 authn 模块自己的 UserID 类型,与 uc/domain/user.UserID 独立
// 通过防腐层(UserAdapter)与用户中心的 UserID 进行转换
type UserID uint64

// NewUserID 从 uint64 创建 UserID
func NewUserID(id uint64) UserID {
	return UserID(id)
}

// ParseUserID 从字符串解析 UserID
func ParseUserID(s string) (UserID, error) {
	if s == "" {
		return 0, fmt.Errorf("user id cannot be empty")
	}
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user id: %w", err)
	}
	return UserID(id), nil
}

// Value 返回 uint64 值
func (u UserID) Value() uint64 {
	return uint64(u)
}

// String 返回字符串表示
func (u UserID) String() string {
	return strconv.FormatUint(uint64(u), 10)
}

// IsZero 检查是否为零值
func (u UserID) IsZero() bool {
	return u == 0
}

// Scan 实现 sql.Scanner 接口
func (u *UserID) Scan(value interface{}) error {
	if value == nil {
		*u = 0
		return nil
	}
	switch v := value.(type) {
	case int64:
		*u = UserID(v)
	case uint64:
		*u = UserID(v)
	case []byte:
		id, err := strconv.ParseUint(string(v), 10, 64)
		if err != nil {
			return err
		}
		*u = UserID(id)
	case string:
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return err
		}
		*u = UserID(id)
	default:
		return fmt.Errorf("cannot scan %T into UserID", value)
	}
	return nil
}

// DriverValue 实现 driver.Valuer 接口
func (u UserID) DriverValue() (driver.Value, error) {
	return int64(u), nil
}
