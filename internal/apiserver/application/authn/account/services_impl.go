package account

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/uow"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/account"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"gorm.io/gorm"
)

// ============= AccountApplicationService 实现 =============

// accountApplicationService 账户应用服务实现
type accountApplicationService struct {
	uow uow.UnitOfWork
}

// accountApplicationService 实现 AccountApplicationService 接口
var _ AccountApplicationService = (*accountApplicationService)(nil)

// NewAccountApplicationService 创建账户应用服务
func NewAccountApplicationService(uow uow.UnitOfWork) AccountApplicationService {
	return &accountApplicationService{uow: uow}
}

// GetAccountByID 根据ID获取账户
func (s *accountApplicationService) GetAccountByID(ctx context.Context, accountID meta.ID) (*AccountResult, error) {
	var result *AccountResult
	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err := tx.Accounts.GetByID(ctx, accountID)
		if err != nil {
			if perrors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrCredentialNotFound, "account not found")
			}
			return err
		}
		result = toAccountResult(account)
		return nil
	})
	return result, err
}

func (s *accountApplicationService) FindByExternalRef(
	ctx context.Context,
	accountType domain.AccountType,
	appID domain.AppId,
	externalID domain.ExternalID,
) (*AccountResult, error) {
	var result *AccountResult
	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err := tx.Accounts.GetByExternalIDAppId(ctx, externalID, appID)
		if err != nil {
			if perrors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrCredentialNotFound, "account not found")
			}
			return err
		}
		result = toAccountResult(account)
		return nil
	})
	return result, err
}

func (s *accountApplicationService) FindByUniqueID(ctx context.Context, uniqueID domain.UnionID) (*AccountResult, error) {
	var result *AccountResult
	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err := tx.Accounts.GetByUniqueID(ctx, uniqueID)
		if err != nil {
			if perrors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrCredentialNotFound, "account not found")
			}
			return err
		}
		result = toAccountResult(account)
		return nil
	})
	return result, err
}

func (s *accountApplicationService) SetUniqueID(ctx context.Context, accountID meta.ID, uniqueID domain.UnionID) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		editor := domain.NewEditor(tx.Accounts)
		_, err := editor.SetUniqueID(ctx, accountID, uniqueID)
		return err
	})
}

func (s *accountApplicationService) UpdateProfile(ctx context.Context, accountID meta.ID, profile map[string]string) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		editor := domain.NewEditor(tx.Accounts)
		_, err := editor.UpdateProfile(ctx, accountID, profile)
		return err
	})
}

func (s *accountApplicationService) UpdateMeta(ctx context.Context, accountID meta.ID, meta map[string]string) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		editor := domain.NewEditor(tx.Accounts)
		_, err := editor.UpdateMeta(ctx, accountID, meta)
		return err
	})
}

func (s *accountApplicationService) EnableAccount(ctx context.Context, accountID meta.ID) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 使用新的 StatusManager 接口
		statusManager := domain.NewStatusManager(tx.Accounts)
		account, err := statusManager.Activate(ctx, accountID)
		if err != nil {
			return err
		}

		// 持久化状态变更
		return tx.Accounts.UpdateStatus(ctx, account.ID, account.Status)
	})
}

func (s *accountApplicationService) DisableAccount(ctx context.Context, accountID meta.ID) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 使用新的 StatusManager 接口
		statusManager := domain.NewStatusManager(tx.Accounts)
		account, err := statusManager.Disable(ctx, accountID)
		if err != nil {
			return err
		}

		// 持久化状态变更
		return tx.Accounts.UpdateStatus(ctx, account.ID, account.Status)
	})
}

func (s *accountApplicationService) ArchiveAccount(ctx context.Context, accountID meta.ID) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 使用新的 StatusManager 接口
		statusManager := domain.NewStatusManager(tx.Accounts)
		account, err := statusManager.Archive(ctx, accountID)
		if err != nil {
			return err
		}

		// 持久化状态变更
		return tx.Accounts.UpdateStatus(ctx, account.ID, account.Status)
	})
}

func (s *accountApplicationService) DeleteAccount(ctx context.Context, accountID meta.ID) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 使用新的 StatusManager 接口
		statusManager := domain.NewStatusManager(tx.Accounts)
		account, err := statusManager.Delete(ctx, accountID)
		if err != nil {
			return err
		}

		// 持久化状态变更
		return tx.Accounts.UpdateStatus(ctx, account.ID, account.Status)
	})
}

// ============= Helper Functions =============

func toAccountResult(account *domain.Account) *AccountResult {
	return &AccountResult{
		AccountID:  account.ID,
		UserID:     account.UserID,
		Type:       account.Type,
		AppID:      account.AppID,
		ExternalID: account.ExternalID,
		UniqueID:   account.UniqueID,
		Profile:    account.Profile,
		Meta:       account.Meta,
		Status:     account.Status,
	}
}
