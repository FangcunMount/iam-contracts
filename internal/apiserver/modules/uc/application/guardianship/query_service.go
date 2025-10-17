package guardianship

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/uow"
	domainservice "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship/service"
)

// ============= 查询应用服务 =============

// GuardianshipQueryApplicationService 监护关系查询应用服务（只读）
type GuardianshipQueryApplicationService interface {
	// IsGuardian 检查是否为监护人
	IsGuardian(ctx context.Context, userID string, childID string) (bool, error)
	// GetByUserIDAndChildID 查询监护关系
	GetByUserIDAndChildID(ctx context.Context, userID string, childID string) (*GuardianshipResult, error)
	// ListChildrenByUserID 列出用户监护的所有儿童
	ListChildrenByUserID(ctx context.Context, userID string) ([]*GuardianshipResult, error)
	// ListGuardiansByChildID 列出儿童的所有监护人
	ListGuardiansByChildID(ctx context.Context, childID string) ([]*GuardianshipResult, error)
}

// guardianshipQueryApplicationService 监护关系查询应用服务实现
type guardianshipQueryApplicationService struct {
	uow uow.UnitOfWork
}

// NewGuardianshipQueryApplicationService 创建监护关系查询应用服务
func NewGuardianshipQueryApplicationService(uow uow.UnitOfWork) GuardianshipQueryApplicationService {
	return &guardianshipQueryApplicationService{uow: uow}
}

// IsGuardian 检查是否为监护人
func (s *guardianshipQueryApplicationService) IsGuardian(ctx context.Context, userID string, childID string) (bool, error) {
	var isGuardian bool

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		queryService := domainservice.NewQueryService(tx.Guardianships, tx.Children)

		userIDObj, err := parseUserID(userID)
		if err != nil {
			return err
		}

		childIDObj, err := parseChildID(childID)
		if err != nil {
			return err
		}

		isGuardian, err = queryService.IsGuardian(ctx, userIDObj, childIDObj)
		return err
	})

	return isGuardian, err
}

// GetByUserIDAndChildID 查询监护关系
func (s *guardianshipQueryApplicationService) GetByUserIDAndChildID(ctx context.Context, userID string, childID string) (*GuardianshipResult, error) {
	var result *GuardianshipResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		queryService := domainservice.NewQueryService(tx.Guardianships, tx.Children)

		userIDObj, err := parseUserID(userID)
		if err != nil {
			return err
		}

		childIDObj, err := parseChildID(childID)
		if err != nil {
			return err
		}

		guardianship, err := queryService.FindByUserIDAndChildID(ctx, userIDObj, childIDObj)
		if err != nil {
			return err
		}

		// 查询儿童信息
		child, err := tx.Children.FindByID(ctx, guardianship.Child)
		if err != nil {
			return err
		}

		result = toGuardianshipResult(guardianship, child)
		return nil
	})

	return result, err
}

// ListChildrenByUserID 列出用户监护的所有儿童
func (s *guardianshipQueryApplicationService) ListChildrenByUserID(ctx context.Context, userID string) ([]*GuardianshipResult, error) {
	var results []*GuardianshipResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		queryService := domainservice.NewQueryService(tx.Guardianships, tx.Children)

		userIDObj, err := parseUserID(userID)
		if err != nil {
			return err
		}

		guardianships, err := queryService.FindListByUserID(ctx, userIDObj)
		if err != nil {
			return err
		}

		// 遍历查询每个儿童信息
		results = make([]*GuardianshipResult, 0, len(guardianships))
		for _, g := range guardianships {
			if g == nil {
				continue
			}
			child, err := tx.Children.FindByID(ctx, g.Child)
			if err != nil {
				return err
			}
			results = append(results, toGuardianshipResult(g, child))
		}

		return nil
	})

	return results, err
}

// ListGuardiansByChildID 列出儿童的所有监护人
func (s *guardianshipQueryApplicationService) ListGuardiansByChildID(ctx context.Context, childID string) ([]*GuardianshipResult, error) {
	var results []*GuardianshipResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		queryService := domainservice.NewQueryService(tx.Guardianships, tx.Children)

		childIDObj, err := parseChildID(childID)
		if err != nil {
			return err
		}

		guardianships, err := queryService.FindListByChildID(ctx, childIDObj)
		if err != nil {
			return err
		}

		// 查询儿童信息（所有监护关系共享同一个儿童）
		child, err := tx.Children.FindByID(ctx, childIDObj)
		if err != nil {
			return err
		}

		results = make([]*GuardianshipResult, 0, len(guardianships))
		for _, g := range guardianships {
			if g == nil {
				continue
			}
			results = append(results, toGuardianshipResult(g, child))
		}

		return nil
	})

	return results, err
}
