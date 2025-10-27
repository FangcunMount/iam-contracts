package uow

import (
	"context"

	"gorm.io/gorm"

	childport "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/child/port"
	guardport "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship/port"
	userport "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user/port"
	childrepo "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/infra/mysql/child"
	guardrepo "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/infra/mysql/guardianship"
	userrepo "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/infra/mysql/user"
	dbmysql "github.com/FangcunMount/iam-contracts/internal/pkg/database/mysql"
	txpkg "github.com/FangcunMount/iam-contracts/internal/pkg/database/tx"
)

// TxRepositories 聚合事务中可使用的仓储集合。
type TxRepositories struct {
	Guardianships guardport.GuardianshipRepository
	Children      childport.ChildRepository
	Users         userport.UserRepository
}

// UnitOfWork 提供业务事务边界。
type UnitOfWork interface {
	WithinTx(ctx context.Context, fn func(tx TxRepositories) error) error
}

var _ txpkg.UnitOfWork[TxRepositories] = (*gormUnitOfWork)(nil)

// NewUnitOfWork 创建基于 GORM 的 UnitOfWork。
func NewUnitOfWork(db *gorm.DB) UnitOfWork {
	return &gormUnitOfWork{
		base: dbmysql.NewUnitOfWork(db),
	}
}

type gormUnitOfWork struct {
	base *dbmysql.UnitOfWork
}

func (u *gormUnitOfWork) WithinTx(ctx context.Context, fn func(tx TxRepositories) error) error {
	if u == nil || u.base == nil {
		return fn(TxRepositories{})
	}

	return u.base.WithinTransaction(ctx, func(tx *gorm.DB) error {
		repos := TxRepositories{
			Guardianships: guardrepo.NewRepository(tx),
			Children:      childrepo.NewRepository(tx),
			Users:         userrepo.NewRepository(tx),
		}
		return fn(repos)
	})
}
