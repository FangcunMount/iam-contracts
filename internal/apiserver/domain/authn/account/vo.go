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
