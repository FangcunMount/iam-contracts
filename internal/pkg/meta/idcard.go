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
