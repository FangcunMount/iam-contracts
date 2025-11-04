package account

import (
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// Account 第三方登录账号实体
type Account struct {
	ID         meta.ID
	UserID     meta.ID           // 关联到用户中心
	Type       AccountType       // 平台类型：wc-minip, wc-offi, wc-com, opera
	AppID      AppId             // 应用 ID：公众号 appid | 小程序 appid | 企业微信 corpid | 运营后台可为空
	ExternalID ExternalID        // 外部平台用户标识：公众号/小程序 openid | 企业微信 userid | 运营后台 username
	UniqueID   UnionID           // 全局唯一标识：公众号/小程序/企业微信 unionid | 运营后台为空
	Profile    map[string]string // 用户资料：昵称、头像等
	Meta       map[string]string // 额外元数据
	Status     AccountStatus     // 账号状态：激活、禁用、存档、删除
}

// NewAccount 创建第三方登录账号实体
func NewAccount(uID meta.ID, t AccountType, eID ExternalID, opts ...AccountOption) *Account {
	// 创建账号实体
	acc := &Account{
		ID:         uID,
		Type:       t,
		ExternalID: eID,
		Status:     StatusActive, // 默认激活
		Profile:    make(map[string]string),
		Meta:       make(map[string]string),
	}
	// 应用选项
	for _, opt := range opts {
		opt(acc)
	}
	return acc
}

// AccountOption 第三方登录账号选项
type AccountOption func(*Account)

func WithID(id meta.ID) AccountOption               { return func(a *Account) { a.ID = id } }
func WithAppID(appid AppId) AccountOption           { return func(a *Account) { a.AppID = appid } }
func WithExternalID(eid ExternalID) AccountOption   { return func(a *Account) { a.ExternalID = eid } }
func WithUnionID(uid UnionID) AccountOption         { return func(a *Account) { a.UniqueID = uid } }
func WithStatus(status AccountStatus) AccountOption { return func(a *Account) { a.Status = status } }
func WithProfile(profile map[string]string) AccountOption {
	return func(a *Account) { a.Profile = profile }
}
func WithMeta(meta map[string]string) AccountOption {
	return func(a *Account) { a.Meta = meta }
}

// 状态检查方法
func (a *Account) IsActive() bool   { return a.Status == StatusActive }
func (a *Account) IsDisabled() bool { return a.Status == StatusDisabled }
func (a *Account) IsArchived() bool { return a.Status == StatusArchived }
func (a *Account) IsDeleted() bool  { return a.Status == StatusDeleted }
