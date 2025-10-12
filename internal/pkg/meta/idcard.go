package meta

// IDCard 身份证
type IDCard struct {
	name   string
	number string
}

// NewIDCard 创建一个新的 IDCard 实例
func NewIDCard(name, number string) IDCard {
	return IDCard{name: name, number: number}
}

// Name 返回身份证姓名
func (idc IDCard) Name() string {
	return idc.name
}

// Number 返回身份证号码
func (idc IDCard) Number() string {
	return idc.number
}

// String 返回身份证的字符串表示
func (idc IDCard) String() string {
	return idc.number
}

// Equal 比较两个 IDCard 是否相等
func (idc IDCard) Equal(other IDCard) bool {
	return idc.name == other.name && idc.number == other.number
}
