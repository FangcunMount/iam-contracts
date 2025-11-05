package uow

import (
	"context"

	"gorm.io/gorm"

	drivenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	acctrepo "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/mysql/account"
	"github.com/FangcunMount/iam-contracts/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam-contracts/internal/pkg/database/tx"
)

// TxRepositories 聚合事务中可使用的仓储集合。
type TxRepositories struct {
	Accounts    drivenPort.AccountRepo
	Credentials drivenPort.CredentialRepo
}

// UnitOfWork 提供业务事务边界。
type UnitOfWork interface {
	WithinTx(ctx context.Context, fn func(tx TxRepositories) error) error
}

var _ tx.UnitOfWork[TxRepositories] = (*gormUnitOfWork)(nil)

// NewUnitOfWork 创建基于 GORM 的 UnitOfWork。
func NewUnitOfWork(db *gorm.DB) UnitOfWork {
	return &gormUnitOfWork{
		base: mysql.NewUnitOfWork(db),
	}
}

type gormUnitOfWork struct {
	base *mysql.UnitOfWork
}

func (u *gormUnitOfWork) WithinTx(ctx context.Context, fn func(tx TxRepositories) error) error {
	if u == nil || u.base == nil {
		return fn(TxRepositories{})
	}

	return u.base.WithinTransaction(ctx, func(tx *gorm.DB) error {
		repos := TxRepositories{
			Accounts:    acctrepo.NewAccountRepository(tx),
			Credentials: acctrepo.NewCredentialRepository(tx),
		}
		return fn(repos)
	})
}
