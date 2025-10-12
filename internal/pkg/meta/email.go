package meta

import "database/sql/driver"

type Email struct {
	address string
}

// NewEmail 创建一个新的 Email 实例
func NewEmail(address string) Email {
	return Email{address: address}
}

// Address 返回邮箱地址
func (e Email) Address() string {
	return e.address
}

// String 返回邮箱地址字符串
func (e Email) String() string {
	return e.address
}

// Equal 比较两个 Email 是否相等
func (e Email) Equal(other Email) bool {
	return e.address == other.address
}

// IsEmpty 判断邮箱地址是否为空
func (e Email) IsEmpty() bool {
	return e.address == ""
}

// Value 实现 driver.Valuer 接口，返回数据库存储值
func (e Email) Value() (driver.Value, error) {
	if e.IsEmpty() {
		return nil, nil
	}
	return e.address, nil
}

// Scan 实现 sql.Scanner 接口，从数据库读取值
func (e *Email) Scan(src interface{}) error {
	if src == nil {
		*e = Email{}
		return nil
	}
	switch v := src.(type) {
	case string:
		*e = Email{address: v}
		return nil
	case []byte:
		*e = Email{address: string(v)}
		return nil
	default:
		return nil
	}
}
