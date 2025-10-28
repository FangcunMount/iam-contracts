package wechatapp

// AppType 微信应用类型
type AppType string

const (
	MiniProgram AppType = "MiniProgram" // 小程序
	MP          AppType = "MP"          // 公众号
)

// String 获取类型字符串
func (t AppType) String() string {
	return string(t)
}

// Status 微信应用状态
type Status string

const (
	StatusEnabled  Status = "Enabled"  // 已启用
	StatusDisabled Status = "Disabled" // 已禁用
	StatusArchived Status = "Archived" // 已归档
)

// CryptoAlg 加密算法
type CryptoAlg string

const (
	AlgAES256 CryptoAlg = "AES256" // 对称
	AlgSM4    CryptoAlg = "SM4"

	AlgRSA CryptoAlg = "RSA" // 非对称
	AlgSM2 CryptoAlg = "SM2"
)
