package account

import (
	"time"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	userdomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
)

// Mapper 负责领域模型与持久化对象之间的转换。
type Mapper struct{}

// NewMapper 创建新的映射器实例。
func NewMapper() *Mapper {
	return &Mapper{}
}

// ToAccountPO 将领域模型转换为账号持久化对象。
func (m *Mapper) ToAccountPO(account *domain.Account) *AccountPO {
	if account == nil {
		return nil
	}

	po := &AccountPO{
		UserID:     idutil.ID(account.UserID),
		Provider:   string(account.Provider),
		ExternalID: account.ExternalID,
		AppID:      account.AppID,
		Status:     int8(account.Status),
	}

	if id := idutil.ID(account.ID); !id.IsZero() {
		po.ID = id
	}

	return po
}

// ToAccountBO 将账号持久化对象转换为领域模型。
func (m *Mapper) ToAccountBO(po *AccountPO) *domain.Account {
	if po == nil {
		return nil
	}

	opts := []domain.AccountOption{
		domain.WithID(domain.AccountID(po.ID)),
		domain.WithExternalID(po.ExternalID),
		domain.WithStatus(domain.AccountStatus(po.Status)),
	}

	if po.AppID != nil {
		opts = append(opts, domain.WithAppID(*po.AppID))
	}

	account := domain.NewAccount(userdomain.UserID(po.UserID), domain.Provider(po.Provider), opts...)
	return &account
}

// ToWeChatPO 将领域模型转换为微信账号持久化对象。
func (m *Mapper) ToWeChatPO(wx *domain.WeChatAccount) *WeChatAccountPO {
	if wx == nil {
		return nil
	}

	return &WeChatAccountPO{
		AccountID: idutil.ID(wx.AccountID),
		AppID:     wx.AppID,
		OpenID:    wx.OpenID,
		UnionID:   wx.UnionID,
		Nickname:  wx.Nickname,
		AvatarURL: wx.AvatarURL,
		Meta:      cloneBytes(wx.Meta),
	}
}

// ToWeChatBO 将微信账号持久化对象转换为领域模型。
func (m *Mapper) ToWeChatBO(po *WeChatAccountPO) *domain.WeChatAccount {
	if po == nil {
		return nil
	}

	opts := []domain.WeChatAccountOption{
		domain.WithWeChatAccountID(domain.AccountID(po.AccountID)),
		domain.WithWeChatMeta(cloneBytes(po.Meta)),
	}
	if po.UnionID != nil {
		opts = append(opts, domain.WithWeChatUnionID(*po.UnionID))
	}
	if po.Nickname != nil {
		opts = append(opts, domain.WithWeChatNickname(*po.Nickname))
	}
	if po.AvatarURL != nil {
		opts = append(opts, domain.WithWeChatAvatarURL(*po.AvatarURL))
	}

	wx := domain.NewWeChatAccount(domain.AccountID(po.AccountID), po.AppID, po.OpenID, opts...)
	return &wx
}

// ToOperationPO 将领域模型转换为运营账号持久化对象。
func (m *Mapper) ToOperationPO(oa *domain.OperationAccount) *OperationAccountPO {
	if oa == nil {
		return nil
	}

	return &OperationAccountPO{
		AccountID:      idutil.ID(oa.AccountID),
		Username:       oa.Username,
		PasswordHash:   cloneBytes(oa.PasswordHash),
		Algo:           oa.Algo,
		Params:         cloneBytes(oa.Params),
		FailedAttempts: oa.FailedAttempts,
		LockedUntil:    copyTimePtr(oa.LockedUntil),
		LastChangedAt:  oa.LastChangedAt,
	}
}

// ToOperationBO 将持久化对象转换为运营账号领域模型。
func (m *Mapper) ToOperationBO(po *OperationAccountPO) *domain.OperationAccount {
	if po == nil {
		return nil
	}

	opts := []domain.OperationAccountOption{
		domain.WithPasswordHash(cloneBytes(po.PasswordHash)),
		domain.WithParams(cloneBytes(po.Params)),
		domain.WithFailedAttempts(po.FailedAttempts),
		domain.WithLastChangedAt(po.LastChangedAt),
	}

	if po.LockedUntil != nil {
		opts = append(opts, domain.WithLockedUntil(copyTimePtr(po.LockedUntil)))
	}

	oa := domain.NewOperationAccount(
		domain.AccountID(po.AccountID),
		po.Username,
		po.Algo,
		opts...,
	)
	return &oa
}

func cloneBytes(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func copyTimePtr(src *time.Time) *time.Time {
	if src == nil {
		return nil
	}
	t := *src
	return &t
}
