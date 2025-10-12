package guardianship

import (
	"context"
	"time"

	childDomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child"
	childport "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child/port"
	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/guradianship"
	guardport "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/guradianship/port"
	userDomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	userport "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// GuardianshipManager 应用层实现
type GuardianshipManager struct {
	repo      guardport.GuardianshipRepository
	childRepo childport.ChildRepository
	userRepo  userport.UserRepository
}

// 确保实现
var _ guardport.GuardianshipManager = (*GuardianshipManager)(nil)

// NewManagerService 创建管理服务
func NewManagerService(r guardport.GuardianshipRepository, cr childport.ChildRepository, ur userport.UserRepository) *GuardianshipManager {
	return &GuardianshipManager{repo: r, childRepo: cr, userRepo: ur}
}

// AddGuardian 添加监护人
func (s *GuardianshipManager) AddGuardian(ctx context.Context, childID childDomain.ChildID, userID userDomain.UserID, relation domain.Relation) error {
	// 校验 child
	c, err := s.childRepo.FindByID(ctx, childID)
	if err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "find child failed")
	}
	if c == nil {
		return perrors.WithCode(code.ErrUserInvalid, "child not found")
	}

	// 校验 user
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "find user failed")
	}
	if u == nil {
		return perrors.WithCode(code.ErrUserInvalid, "user not found")
	}

	// 检查重复
	guardians, err := s.repo.FindListByChildID(ctx, childID)
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

	ng := &domain.Guardianship{
		User:          userID,
		Child:         childID,
		Rel:           relation,
		EstablishedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, ng); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "create guardianship failed")
	}

	return nil
}

// RemoveGuardian 撤销监护
func (s *GuardianshipManager) RemoveGuardian(ctx context.Context, childID childDomain.ChildID, userID userDomain.UserID) error {
	guardians, err := s.repo.FindListByChildID(ctx, childID)
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
	if err := s.repo.Update(ctx, target); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "revoke guardianship failed")
	}

	return nil
}
