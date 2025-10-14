package guardianship

import (
	"context"
	"time"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/uow"
	childDomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	childport "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child/port"
	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship"
	guardport "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship/port"
	userDomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	userport "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	txpkg "github.com/fangcun-mount/iam-contracts/internal/pkg/database/tx"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// GuardianshipManager 应用层实现
type GuardianshipManager struct {
	repo      guardport.GuardianshipRepository
	childRepo childport.ChildRepository
	userRepo  userport.UserRepository
	txRunner  txpkg.Runner[uow.TxRepositories]
}

// 确保实现
var _ guardport.GuardianshipManager = (*GuardianshipManager)(nil)

// NewManagerService 创建管理服务
func NewManagerService(r guardport.GuardianshipRepository, cr childport.ChildRepository, ur userport.UserRepository, tx uow.UnitOfWork) *GuardianshipManager {
	return &GuardianshipManager{
		repo:      r,
		childRepo: cr,
		userRepo:  ur,
		txRunner:  txpkg.Runner[uow.TxRepositories]{UoW: tx},
	}
}

// AddGuardian 添加监护人
func (s *GuardianshipManager) AddGuardian(ctx context.Context, childID childDomain.ChildID, userID userDomain.UserID, relation domain.Relation) error {
	return s.txRunner.WithinTx(ctx, func(txRepos uow.TxRepositories) error {
		childRepo := s.resolveChildRepo(txRepos)
		guardRepo := s.resolveGuardRepo(txRepos)
		userRepo := s.resolveUserRepo(txRepos)

		c, err := childRepo.FindByID(ctx, childID)
		if err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "find child failed")
		}
		if c == nil {
			return perrors.WithCode(code.ErrUserInvalid, "child not found")
		}

		u, err := userRepo.FindByID(ctx, userID)
		if err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "find user failed")
		}
		if u == nil {
			return perrors.WithCode(code.ErrUserInvalid, "user not found")
		}

		guardians, err := guardRepo.FindByChildID(ctx, childID)
		if err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "find guardians failed")
		}
		for _, g := range guardians {
			if g == nil {
				continue
			}
			if g.User == userID && g.IsActive() {
				return perrors.WithCode(code.ErrUserInvalid, "guardian already exists")
			}
		}

		newGuard := &domain.Guardianship{
			User:          userID,
			Child:         childID,
			Rel:           relation,
			EstablishedAt: time.Now(),
		}

		if err := guardRepo.Create(ctx, newGuard); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "create guardianship failed")
		}

		return nil
	})
}

// RemoveGuardian 撤销监护
func (s *GuardianshipManager) RemoveGuardian(ctx context.Context, childID childDomain.ChildID, userID userDomain.UserID) error {
	return s.txRunner.WithinTx(ctx, func(txRepos uow.TxRepositories) error {
		guardRepo := s.resolveGuardRepo(txRepos)

		guardians, err := guardRepo.FindByChildID(ctx, childID)
		if err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "find guardians failed")
		}

		var target *domain.Guardianship
		for _, g := range guardians {
			if g == nil {
				continue
			}
			if g.User == userID && g.IsActive() {
				target = g
				break
			}
		}

		if target == nil {
			return perrors.WithCode(code.ErrUserInvalid, "active guardian not found")
		}

		target.Revoke(time.Now())
		if err := guardRepo.Update(ctx, target); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "revoke guardianship failed")
		}

		return nil
	})
}

// RegisterChildWithGuardian 在一个事务中创建儿童并授予监护权。
func (s *GuardianshipManager) RegisterChildWithGuardian(ctx context.Context, params guardport.RegisterChildParams) (*domain.Guardianship, *childDomain.Child, error) {
	var (
		newGuard *domain.Guardianship
		newChild *childDomain.Child
	)

	err := s.txRunner.WithinTx(ctx, func(txRepos uow.TxRepositories) error {
		childRepo := s.resolveChildRepo(txRepos)
		guardRepo := s.resolveGuardRepo(txRepos)
		userRepo := s.resolveUserRepo(txRepos)

		userEntity, err := userRepo.FindByID(ctx, params.UserID)
		if err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "find user(%d) failed", params.UserID.Value())
		}
		if userEntity == nil {
			return perrors.WithCode(code.ErrUserInvalid, "user(%d) not found", params.UserID.Value())
		}

		child, err := childDomain.NewChild(
			params.Name,
			childDomain.WithGender(params.Gender),
			childDomain.WithBirthday(params.Birthday),
		)
		if err != nil {
			return err
		}

		child.UpdateIDCard(params.IDCard)

		height := child.Height
		if params.Height != nil {
			height = *params.Height
		}
		weight := child.Weight
		if params.Weight != nil {
			weight = *params.Weight
		}
		child.UpdateHeightWeight(height, weight)

		if err := childRepo.Create(ctx, child); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "create child(%s) failed", params.Name)
		}

		guard := &domain.Guardianship{
			User:          params.UserID,
			Child:         child.ID,
			Rel:           params.Relation,
			EstablishedAt: time.Now(),
		}
		if err := guardRepo.Create(ctx, guard); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "create guardianship failed")
		}

		newChild = child
		newGuard = guard

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return newGuard, newChild, nil
}

func (s *GuardianshipManager) resolveChildRepo(tx uow.TxRepositories) childport.ChildRepository {
	if tx.Children != nil {
		return tx.Children
	}
	return s.childRepo
}

func (s *GuardianshipManager) resolveGuardRepo(tx uow.TxRepositories) guardport.GuardianshipRepository {
	if tx.Guardianships != nil {
		return tx.Guardianships
	}
	return s.repo
}

func (s *GuardianshipManager) resolveUserRepo(tx uow.TxRepositories) userport.UserRepository {
	if tx.Users != nil {
		return tx.Users
	}
	return s.userRepo
}
