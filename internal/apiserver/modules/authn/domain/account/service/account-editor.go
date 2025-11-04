package service

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/errors"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// AccountEditor 领域编辑服务
// 职责：封装账号编辑的业务规则
// 不包含：事务管理（由应用层负责）
type AccountEditor struct {
	repo port.AccountRepo
}

// AccountEditor 接口类实现，确保 AccountEditor 实现了所有的接口方法
var _ port.AccountEditor = (*AccountEditor)(nil)

// NewAccountEditor creates a new AccountEditor instance
func NewAccountEditor(repo port.AccountRepo) *AccountEditor {
	return &AccountEditor{
		repo: repo,
	}
}

// SetUniqueID 设置唯一标识
func (es *AccountEditor) SetUniqueID(ctx context.Context, accountID meta.ID, uniqueID domain.UnionID) (*domain.Account, error) {
	// 检查唯一标识是否为空
	if uniqueID == "" {
		return nil, errors.WithCode(code.ErrInvalidUniqueID, "uniqueID cannot be empty")
	}
	// 查询账号
	account, err := es.repo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.WithCode(code.ErrNotFoundAccount, "account not found")
	}

	// 使用领域对象的方法设置 uniqueID
	if !account.SetUniqueID(uniqueID) {
		return nil, errors.WithCode(code.ErrUniqueIDExists, "uniqueID already set")
	}

	return account, nil
}

// UpdateProfile 更新账号资料
func (es *AccountEditor) UpdateProfile(ctx context.Context, accountID meta.ID, profile map[string]string) (*domain.Account, error) {
	// 查询账号
	account, err := es.repo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.WithCode(code.ErrNotFoundAccount, "account not found")
	}

	// 检查是否可以更新
	if !account.CanUpdateProfile() {
		return nil, errors.WithCode(code.ErrInvalidArgument, "cannot update profile for deleted account")
	}

	// 使用领域对象的方法更新资料
	account.UpdateProfile(profile)

	return account, nil
}

// UpdateMeta 更新账号元数据
func (es *AccountEditor) UpdateMeta(ctx context.Context, accountID meta.ID, meta map[string]string) (*domain.Account, error) {
	// 查询账号
	account, err := es.repo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.WithCode(code.ErrNotFoundAccount, "account not found")
	}

	// 检查是否可以更新
	if !account.CanUpdateMeta() {
		return nil, errors.WithCode(code.ErrInvalidArgument, "cannot update meta for deleted account")
	}

	// 使用领域对象的方法更新元数据
	account.UpdateMeta(meta)

	return account, nil
}
