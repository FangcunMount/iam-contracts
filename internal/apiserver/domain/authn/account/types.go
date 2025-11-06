package account

// Provider 认证提供者
type Provider string

const (
	ProviderPassword Provider = "op:password"
	ProviderWeChat   Provider = "wx:minip"
)

// AppId 应用 ID
type AppId string

// Len returns the length of the AppId string
func (a AppId) Len() int {
	return len(a)
}

// ExternalID 外部平台用户标识
type ExternalID string

// Len returns the length of the ExternalID string
func (e ExternalID) Len() int {
	return len(e)
}

// UnionID 全局唯一标识
type UnionID string

// Len returns the length of the UnionID string
func (u UnionID) Len() int {
	return len(u)
}

// AccountType 账号类型
type AccountType string

const (
	TypeWcMinip AccountType = "wc-minip" // 微信小程序
	TypeWcOffi  AccountType = "wc-offi"  // 微信公众号
	TypeWcCom   AccountType = "wc-com"   // 企业微信
	TypeOpera   AccountType = "opera"    // 运营后台
)

// String 转字符串
func (a AccountType) String() string {
	return string(a)
}

// Validate 校验账号类型是否合法
func (a AccountType) Validate() bool {
	tList := []AccountType{TypeWcMinip, TypeWcOffi, TypeWcCom, TypeOpera}
	for _, t := range tList {
		if a == t {
			return true
		}
	}
	return false
}

// Status 账号状态
type AccountStatus int8

const (
	StatusDisabled AccountStatus = 0 // 禁用
	StatusActive   AccountStatus = 1 // 激活
	StatusArchived AccountStatus = 2 // 已归档
	StatusDeleted  AccountStatus = 3 // 已删除
)

func (s AccountStatus) String() string {
	switch s {
	case StatusDisabled:
		return "disabled"
	case StatusActive:
		return "active"
	case StatusArchived:
		return "archived"
	case StatusDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// AccessClaims 访问令牌载荷，JWT 声明
type AccessClaims struct {
	Sub string `json:"sub"`
	Aid string `json:"aid"`
	Aud string `json:"aud"`
	Iss string `json:"iss"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
	Jti string `json:"jti"`
	Kid string `json:"kid"`
	Sid string `json:"sid,omitempty"`
}
