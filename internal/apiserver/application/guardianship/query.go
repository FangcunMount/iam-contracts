package guardianship

import (
	"context"
	"errors"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child"
	childport "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child/port"
	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/guardianship"
	guardport "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/guardianship/port"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"gorm.io/gorm"
)

// GuardianshipQueryer 应用层查询实现
type GuardianshipQueryer struct {
	repo      guardport.GuardianshipRepository
	childRepo childport.ChildRepository
}

var _ guardport.GuardianshipQueryer = (*GuardianshipQueryer)(nil)

// NewQueryService 创建查询服务
func NewQueryService(r guardport.GuardianshipRepository, cr childport.ChildRepository) *GuardianshipQueryer {
	return &GuardianshipQueryer{repo: r, childRepo: cr}
}

// FindByUserIDAndChildID 实现
func (s *GuardianshipQueryer) FindByUserIDAndChildID(ctx context.Context, userID user.UserID, childID child.ChildID) (*domain.Guardianship, error) {
	// Note: keep signature compatible with interface (types should be user.UserID, child.ChildID)
	guardians, err := s.repo.FindByChildID(ctx, childID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, perrors.WithCode(code.ErrUserNotFound, "guardianship not found")
		}
		return nil, perrors.WrapC(err, code.ErrDatabase, "find guardians by child failed")
	}

	for _, g := range guardians {
		if g == nil {
			continue
		}
		if g.User == userID {
			return g, nil
		}
	}
	return nil, perrors.WithCode(code.ErrUserNotFound, "guardianship not found")
}

// FindByUserIDAndChildName 根据用户ID和儿童姓名查询监护关系
func (s *GuardianshipQueryer) FindByUserIDAndChildName(ctx context.Context, userID user.UserID, childName string) ([]*domain.Guardianship, error) {
	// 查询匹配姓名的儿童档案
	children, err := s.childRepo.FindListByName(ctx, childName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []*domain.Guardianship{}, nil
		}
		return nil, perrors.WrapC(err, code.ErrDatabase, "find children by name failed")
	}

	var res []*domain.Guardianship
	for _, c := range children {
		if c == nil {
			continue
		}
		guardians, err := s.repo.FindByChildID(ctx, c.ID)
		if err != nil {
			return nil, perrors.WrapC(err, code.ErrDatabase, "find guardians by child failed")
		}
		for _, g := range guardians {
			if g == nil {
				continue
			}
			if g.User == userID {
				res = append(res, g)
			}
		}
	}
	return res, nil
}

// FindListByChildID 列出某儿童的监护人
func (s *GuardianshipQueryer) FindListByChildID(ctx context.Context, childID child.ChildID) ([]*domain.Guardianship, error) {
	return s.repo.FindByChildID(ctx, childID)
}

// FindListByUserID 列出用户监护的所有儿童
func (s *GuardianshipQueryer) FindListByUserID(ctx context.Context, userID user.UserID) ([]*domain.Guardianship, error) {
	return s.repo.FindByUserID(ctx, userID)
}
