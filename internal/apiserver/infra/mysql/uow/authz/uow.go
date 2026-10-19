package authz

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	appuow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	assignmentDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	assignmentrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/assignment"
	permissiongrantrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/permissiongrant"
	policyrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/policy"
	resourcerepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/resource"
	rolerepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/role"
	dbmysql "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/pkg/event"
)

var _ appuow.UnitOfWork = (*unitOfWork)(nil)

// NewUnitOfWork 创建基于 MySQL/GORM 的授权事务边界。
func NewUnitOfWork(db *gorm.DB, subjectResolver assignmentDomain.SubjectResolver, stagers ...event.Stager) appuow.UnitOfWork {
	var stager event.Stager
	if len(stagers) > 0 {
		stager = stagers[0]
	}
	return &unitOfWork{base: dbmysql.NewUnitOfWork(db), events: stager, subjectResolver: subjectResolver}
}

type unitOfWork struct {
	base            *dbmysql.UnitOfWork
	events          event.Stager
	subjectResolver assignmentDomain.SubjectResolver
}

func (u *unitOfWork) WithinTx(ctx context.Context, fn func(txCtx context.Context, tx appuow.TxRepositories) error) error {
	if fn == nil {
		return nil
	}
	if u == nil || u.base == nil {
		return dbmysql.ErrUnitOfWorkUnavailable
	}
	return u.within(ctx, fn)
}

func (u *unitOfWork) within(ctx context.Context, fn func(context.Context, appuow.TxRepositories) error, opts ...dbmysql.TxOptions) error {
	return u.base.WithinTransaction(ctx, func(txCtx context.Context) error {
		tx, err := dbmysql.RequireTx(txCtx)
		if err != nil {
			return err
		}
		repos := appuow.TxRepositories{
			Assignments:      assignmentrepo.NewRepository(tx),
			Roles:            rolerepo.NewRoleRepository(tx),
			Resources:        resourcerepo.NewResourceRepository(tx),
			PolicyVersions:   policyrepo.NewPolicyVersionRepository(tx),
			SubjectResolver:  u.subjectResolver,
			PermissionGrants: permissiongrantrepo.NewRepository(tx),
			Events:           u.events,
		}
		return fn(txCtx, repos)
	}, opts...)
}

// ReadPlan always starts a fresh host-owned read-only root. It never borrows a
// caller's existing snapshot or permits planning to stage an event.
func (u *unitOfWork) ReadPlan(ctx context.Context, fn func(context.Context, appuow.TxRepositories) error) error {
	if ctx == nil || fn == nil {
		return appuow.ErrWritePlanRequired
	}
	if u == nil || u.base == nil {
		return dbmysql.ErrUnitOfWorkUnavailable
	}
	if _, active := dbmysql.TxFromContext(ctx); active {
		return dbmysql.ErrActiveTransactionRequired
	}
	return u.within(ctx, func(readCtx context.Context, repos appuow.TxRepositories) error {
		repos.Events = nil
		return fn(readCtx, repos)
	}, dbmysql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
}
