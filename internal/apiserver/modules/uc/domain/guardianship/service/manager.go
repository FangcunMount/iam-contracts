package service

import (
	"context"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	childDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	childport "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/child/port"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship"
	guardport "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship/port"
	userDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	userport "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// GuardianshipManager 监护关系管理领域服务
type GuardianshipManager struct {
	repo      guardport.GuardianshipRepository
	childRepo childport.ChildRepository
	userRepo  userport.UserRepository
}

// 确保实现
var _ guardport.GuardianshipManager = (*GuardianshipManager)(nil)

// NewManagerService 创建管理服务
func NewManagerService(r guardport.GuardianshipRepository, cr childport.ChildRepository, ur userport.UserRepository) *GuardianshipManager {
	return &GuardianshipManager{
		repo:      r,
		childRepo: cr,
		userRepo:  ur,
	}
}

// AddGuardian 添加监护人
// 领域逻辑：验证用户和儿童存在性 + 验证监护关系不重复 + 创建监护实体
// 注意：不包括持久化，返回创建的监护关系实体供应用层持久化
func (s *GuardianshipManager) AddGuardian(ctx context.Context, userID userDomain.UserID, childID childDomain.ChildID, relation domain.Relation) (*domain.Guardianship, error) {
	// 验证儿童存在
	c, err := s.childRepo.FindByID(ctx, childID)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "find child failed")
	}
	if c == nil {
		return nil, perrors.WithCode(code.ErrUserInvalid, "child not found")
	}

	// 验证用户存在
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "find user failed")
	}
	if u == nil {
		return nil, perrors.WithCode(code.ErrUserInvalid, "user not found")
	}

	// 验证监护关系不重复
	guardians, err := s.repo.FindByChildID(ctx, childID)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "find guardians failed")
	}
	for _, g := range guardians {
		if g == nil {
			continue
		}
		if g.User == userID && g.IsActive() {
			return nil, perrors.WithCode(code.ErrUserInvalid, "guardian already exists")
		}
	}

	// 创建监护关系实体
	newGuard := &domain.Guardianship{
		User:          userID,
		Child:         childID,
		Rel:           relation,
		EstablishedAt: time.Now(),
	}

	// 返回创建的监护关系，由应用层持久化
	return newGuard, nil
}

// RemoveGuardian 撤销监护
// 领域逻辑：查询监护关系 + 撤销监护
// 注意：不包括持久化，返回修改后的监护关系实体供应用层持久化
func (s *GuardianshipManager) RemoveGuardian(ctx context.Context, userID userDomain.UserID, childID childDomain.ChildID) (*domain.Guardianship, error) {
	// 查询监护关系
	guardians, err := s.repo.FindByChildID(ctx, childID)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "find guardians failed")
	}

	// 查找目标监护关系
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
		return nil, perrors.WithCode(code.ErrUserInvalid, "active guardian not found")
	}

	// 撤销监护关系
	target.Revoke(time.Now())

	// 返回修改后的监护关系，由应用层持久化
	return target, nil
}
