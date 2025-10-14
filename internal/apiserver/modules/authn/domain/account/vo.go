package account

import "github.com/fangcun-mount/iam-contracts/pkg/util/idutil"

// Provider 认证提供者
type Provider string

const (
	ProviderPassword Provider = "op:password"
	ProviderWeChat   Provider = "wx:minip"
)

// AccountID 账号ID
type AccountID idutil.ID

func NewAccountID(value uint64) AccountID {
	return AccountID(idutil.NewID(value))
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
