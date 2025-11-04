package meta

import "github.com/FangcunMount/component-base/pkg/util/idutil"

// ID 是基础实体标识类型
type ID idutil.ID

// NewID 从 uint64 创建 ID
func NewID(value uint64) ID {
	return ID(idutil.NewID(value))
}

// ToUint64 将 ID 转换为 uint64
func (id ID) ToUint64() uint64 {
	return idutil.ID(id).Uint64()
}

// String 返回 ID 的字符串表示
func (id ID) String() string {
	return idutil.ID(id).String()
}

// MarshalJSON 实现 json.Marshaler 接口，输出数字
func (id ID) MarshalJSON() ([]byte, error) {
	return idutil.ID(id).MarshalJSON()
}

// UnmarshalJSON 实现 json.Unmarshaler 接口，接受数字
func (id *ID) UnmarshalJSON(b []byte) error {
	var uid idutil.ID
	if err := uid.UnmarshalJSON(b); err != nil {
		return err
	}
	*id = ID(uid)
	return nil
}

// Scan 实现 sql.Scanner 接口，从数据库整数读取
func (id *ID) Scan(src interface{}) error {
	var uid idutil.ID
	if err := uid.Scan(src); err != nil {
		return err
	}
	*id = ID(uid)
	return nil
}
