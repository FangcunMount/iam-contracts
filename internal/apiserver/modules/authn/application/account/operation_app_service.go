package account

import (
	"context"
	"errors"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	perrors "github.com/FangcunMount/iam-contracts/pkg/errors"
	"gorm.io/gorm"
)

// operationAccountApplicationService 运营账号应用服务实现
type operationAccountApplicationService struct {
	uow uow.UnitOfWork
}

var _ OperationAccountApplicationService = (*operationAccountApplicationService)(nil)

// NewOperationAccountApplicationService 创建运营账号应用服务
func NewOperationAccountApplicationService(
	uow uow.UnitOfWork,
) OperationAccountApplicationService {
	return &operationAccountApplicationService{
		uow: uow,
	}
}

// UpdateCredential 更新运营账号凭据用例
// 流程：
// 1. 在事务中查找运营账号
// 2. 更新密码哈希（TODO: 应该使用密码适配器）
// 3. 自动重置失败次数和解锁
func (s *operationAccountApplicationService) UpdateCredential(
	ctx context.Context,
	dto UpdateOperationCredentialDTO,
) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 验证运营账号存在
		_, err := tx.Operation.FindByUsername(ctx, dto.Username)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrDatabase, "operation account not found")
			}
			return perrors.WrapC(err, code.ErrDatabase, "find operation account failed")
		}

		// TODO: 应该使用密码适配器进行哈希
		passwordHash := []byte(dto.Password)

		// 更新密码哈希
		if err := tx.Operation.UpdateHash(ctx, dto.Username, passwordHash, dto.HashAlgo, nil); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "update credential failed")
		}

		// 自动重置失败次数（使用username而不是accountID）
		if err := tx.Operation.ResetFailures(ctx, dto.Username); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "reset failures failed")
		}

		return nil
	})
}

// ChangeUsername 修改运营账号用户名用例
// 流程：
// 1. 验证新用户名不存在
// 2. 更新用户名
// 3. 自动解锁账号
func (s *operationAccountApplicationService) ChangeUsername(
	ctx context.Context,
	dto ChangeUsernameDTO,
) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 查找旧运营账号
		operation, err := tx.Operation.FindByUsername(ctx, dto.OldUsername)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrDatabase, "operation account not found")
			}
			return perrors.WrapC(err, code.ErrDatabase, "find operation account failed")
		}

		// 验证新用户名不存在
		_, err = tx.Operation.FindByUsername(ctx, dto.NewUsername)
		if err == nil {
			return perrors.WithCode(code.ErrInvalidArgument, "new username already exists")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return perrors.WrapC(err, code.ErrDatabase, "check new username failed")
		}

		// 更新用户名
		if err := tx.Operation.UpdateUsername(ctx, operation.AccountID, dto.NewUsername); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "update username failed")
		}

		// 自动解锁账号（使用新用户名）
		if err := tx.Operation.Unlock(ctx, dto.NewUsername); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "unlock account failed")
		}

		return nil
	})
}

// GetByUsername 根据用户名获取运营账号用例
func (s *operationAccountApplicationService) GetByUsername(
	ctx context.Context,
	username string,
) (*AccountResult, error) {
	var result *AccountResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 查找运营账号
		operation, err := tx.Operation.FindByUsername(ctx, username)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrDatabase, "operation account not found")
			}
			return perrors.WrapC(err, code.ErrDatabase, "find operation account failed")
		}

		// 查找关联的账号
		account, err := tx.Accounts.FindByID(ctx, operation.AccountID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrDatabase, "account not found")
			}
			return perrors.WrapC(err, code.ErrDatabase, "find account failed")
		}

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

// ResetFailures 重置失败次数用例
func (s *operationAccountApplicationService) ResetFailures(
	ctx context.Context,
	username string,
) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 验证运营账号存在
		_, err := tx.Operation.FindByUsername(ctx, username)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrDatabase, "operation account not found")
			}
			return perrors.WrapC(err, code.ErrDatabase, "find operation account failed")
		}

		// 重置失败次数
		if err := tx.Operation.ResetFailures(ctx, username); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "reset failures failed")
		}

		return nil
	})
}

// UnlockAccount 解锁运营账号用例
func (s *operationAccountApplicationService) UnlockAccount(
	ctx context.Context,
	username string,
) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 验证运营账号存在
		_, err := tx.Operation.FindByUsername(ctx, username)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrDatabase, "operation account not found")
			}
			return perrors.WrapC(err, code.ErrDatabase, "find operation account failed")
		}

		// 解锁账号
		if err := tx.Operation.Unlock(ctx, username); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "unlock account failed")
		}

		return nil
	})
}
