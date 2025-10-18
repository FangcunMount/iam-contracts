package idutil

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
)

type ID struct {
	value uint64
}

// NewID 创建一个新的 ID 实例
func NewID(value uint64) ID {
	return ID{value: value}
}

// Uint64 返回 ID 的 uint64 值
func (id ID) Uint64() uint64 {
	return id.value
}

// String 返回 ID 的字符串表示
func (id ID) String() string {
	return fmt.Sprintf("%d", id.value)
}

// MarshalJSON 实现 json.Marshaler 接口，输出数字
func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.value)
}

// UnmarshalJSON 实现 json.Unmarshaler 接口，接受数字
func (id *ID) UnmarshalJSON(b []byte) error {
	var v uint64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	id.value = v
	return nil
}

// Scan 实现 sql.Scanner 接口，从数据库整数读取
func (id *ID) Scan(src interface{}) error {
	if src == nil {
		id.value = 0
		return nil
	}
	switch v := src.(type) {
	case int64:
		id.value = uint64(v)
		return nil
	case uint64:
		id.value = v
		return nil
	case []byte:
		// 可能是字符串形式的数字
		s := string(v)
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		id.value = n
		return nil
	case string:
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return err
		}
		id.value = n
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into ID", src)
	}
}

// GormDBDataType 实现 schema.GormDBDataTypeInterface，告诉 GORM 数据类型
func (ID) GormDBDataType(db string) string {
	return "bigint"
}

// Value 实现 driver.Valuer 接口，将 ID 写入数据库
func (id ID) Value() (driver.Value, error) {
	return int64(id.value), nil
}

// IsZero 是否为零值
func (id ID) IsZero() bool {
	return id.value == 0
}

// Equal 比较两个 ID 是否相等
func (id ID) Equal(other ID) bool {
	return id.value == other.value
}
