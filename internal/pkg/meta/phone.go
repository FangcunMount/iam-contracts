package meta

import "database/sql/driver"

type Phone struct {
	number string
}

// NewPhone 创建一个新的 Phone 实例
func NewPhone(number string) Phone {
	return Phone{number: number}
}

// Number 返回电话号码
func (p Phone) Number() string {
	return p.number
}

// String 返回电话号码字符串
func (p Phone) String() string {
	return p.number
}

// Equal 比较两个 Phone 是否相等
func (p Phone) Equal(other Phone) bool {
	return p.number == other.number
}

// IsEmpty 判断电话号码是否为空
func (p Phone) IsEmpty() bool {
	return p.number == ""
}

// Value 实现 driver.Valuer 接口，返回数据库存储值
func (p Phone) Value() (driver.Value, error) {
	if p.IsEmpty() {
		return nil, nil
	}
	return p.number, nil
}

// Scan 实现 sql.Scanner 接口，从数据库读取值
func (p *Phone) Scan(src interface{}) error {
	if src == nil {
		*p = Phone{}
		return nil
	}
	switch v := src.(type) {
	case string:
		*p = Phone{number: v}
		return nil
	case []byte:
		*p = Phone{number: string(v)}
		return nil
	default:
		return nil
	}
}
