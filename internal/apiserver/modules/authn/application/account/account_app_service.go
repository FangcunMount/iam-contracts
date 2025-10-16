package account

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/adapter"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	domainService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/service"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// accountApplicationService 账号应用服务
// 职责：编排账号管理的业务用例，管理事务边界
type accountApplicationService struct {
	uow         uow.UnitOfWork
	userAdapter adapter.UserAdapter
}

// NewAccountApplicationService 创建账号应用服务
func NewAccountApplicationService(
	uow uow.UnitOfWork,
	userAdapter adapter.UserAdapter,
) AccountApplicationService {
	return &accountApplicationService{
		uow:         uow,
		userAdapter: userAdapter,
	}
}

// CreateOperationAccount 创建运营账号（使用用例）
// 业务流程：
// 1. 验证用户存在
// 2. 在事务中创建账号实体和运营账号实体
// 3. 持久化到数据库
func (s *accountApplicationService) CreateOperationAccount(
	ctx context.Context,
	dto CreateOperationAccountDTO,
) (*AccountResult, error) {
	// 1. 验证用户存在
	exists, err := s.userAdapter.ExistsUser(ctx, dto.UserID)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "check user existence failed")
	}
	if !exists {
		return nil, perrors.WithCode(code.ErrUserNotFound, "user(%s) not found", dto.UserID.String())
	}

	var result *AccountResult

	// 2. 在事务中创建账号和运营账号
	err = s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 使用领域服务验证账号不存在
		if err := domainService.ValidateAccountNotExists(
			ctx, tx.Accounts, domain.ProviderPassword, dto.Username, nil,
		); err != nil {
			return err
		}

		// 使用工厂方法创建账号实体
		account, err := domainService.CreateAccountEntity(
			dto.UserID, domain.ProviderPassword, dto.Username, nil,
		)
		if err != nil {
			return err
		}

		// 持久化账号
		if err := tx.Accounts.Create(ctx, account); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "create account failed")
		}

		// 验证运营账号不存在
		if err := domainService.ValidateOperationNotExists(
			ctx, tx.Operation, dto.Username,
		); err != nil {
			return err
		}

		// 创建运营账号实体
		operation, err := domainService.CreateOperationAccountEntity(
			account.ID, dto.Username,
		)
		if err != nil {
			return err
		}

		// 持久化运营账号
		if err := tx.Operation.Create(ctx, operation); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "create operation account failed")
		}

		// 构建返回结果
		result = &AccountResult{
			Account:       account,
			OperationData: operation,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetAccountByID 根据ID查询账号
func (s *accountApplicationService) GetAccountByID(
	ctx context.Context,
	accountID domain.AccountID,
) (*AccountResult, error) {
	// 查询操作不需要事务
	var account *domain.Account
	var err error

	err = s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err = tx.Accounts.FindByID(ctx, accountID)
		return err
	})

	if err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "account not found")
	}

	return &AccountResult{
		Account: account,
	}, nil
}

// ListAccountsByUserID 根据用户ID查询账号列表
func (s *accountApplicationService) ListAccountsByUserID(
	ctx context.Context,
	userID domain.UserID,
) ([]*domain.Account, error) {
	// 查询操作不需要事务，但需要使用类型断言检查是否支持ListByUserID
	var accounts []*domain.Account
	var err error

	err = s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 使用类型断言检查仓储是否支持ListByUserID方法
		type accountLister interface {
			ListByUserID(ctx context.Context, userID domain.UserID) ([]*domain.Account, error)
		}

		if lister, ok := tx.Accounts.(accountLister); ok {
			accounts, err = lister.ListByUserID(ctx, userID)
			if err != nil {
				return perrors.WrapC(err, code.ErrDatabase, "list accounts by user failed")
			}
			return nil
		}

		return perrors.WithCode(code.ErrInternalServerError, "account repository does not support list by user id")
	})

	if err != nil {
		return nil, err
	}

	return accounts, nil
}

// EnableAccount 启用账号
func (s *accountApplicationService) EnableAccount(
	ctx context.Context,
	accountID domain.AccountID,
) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err := tx.Accounts.FindByID(ctx, accountID)
		if err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "account not found")
		}

		// 调用领域方法修改状态
		account.Activate()

		// 持久化修改
		if err := tx.Accounts.UpdateStatus(ctx, account.ID, account.Status); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "enable account failed")
		}

		return nil
	})
}

// DisableAccount 禁用账号
func (s *accountApplicationService) DisableAccount(
	ctx context.Context,
	accountID domain.AccountID,
) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err := tx.Accounts.FindByID(ctx, accountID)
		if err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "account not found")
		}

		// 调用领域方法修改状态
		account.Disable()

		// 持久化修改
		if err := tx.Accounts.UpdateStatus(ctx, account.ID, account.Status); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "disable account failed")
		}

		return nil
	})
}
