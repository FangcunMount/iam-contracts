package account

import (
	"context"
	"errors"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	perrors "github.com/FangcunMount/iam-contracts/pkg/errors"
	"gorm.io/gorm"
)

// accountLookupApplicationService 账号查找应用服务实现
// 职责：提供账号查询用例
type accountLookupApplicationService struct {
	uow uow.UnitOfWork
}

var _ AccountLookupApplicationService = (*accountLookupApplicationService)(nil)

// NewAccountLookupApplicationService 创建账号查找应用服务
func NewAccountLookupApplicationService(
	uow uow.UnitOfWork,
) AccountLookupApplicationService {
	return &accountLookupApplicationService{
		uow: uow,
	}
}

// FindByProvider 根据提供商查找账号用例
// 这是一个简单的查询操作，不需要复杂的业务逻辑
func (s *accountLookupApplicationService) FindByProvider(
	ctx context.Context,
	provider domain.Provider,
	externalID string,
	appID *string,
) (*domain.Account, error) {
	var account *domain.Account

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		var err error
		account, err = tx.Accounts.FindByRef(ctx, provider, externalID, appID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrDatabase, "account not found")
			}
			return perrors.WrapC(err, code.ErrDatabase, "find account failed")
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return account, nil
}
