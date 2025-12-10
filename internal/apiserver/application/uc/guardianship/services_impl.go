package guardianship

import (
	"context"
	"fmt"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/uow"
	childdomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/child"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/guardianship"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/guardianship"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ============= 应用服务实现 =============

// =============================================
// ==== GuardianshipApplicationService 实现 =====
// =============================================
// guardianshipApplicationService 监护关系应用服务实现
type guardianshipApplicationService struct {
	uow uow.UnitOfWork
}

// NewGuardianshipApplicationService 创建监护关系应用服务
func NewGuardianshipApplicationService(uow uow.UnitOfWork) GuardianshipApplicationService {
	return &guardianshipApplicationService{uow: uow}
}

// AddGuardian 添加监护人
func (s *guardianshipApplicationService) AddGuardian(ctx context.Context, dto AddGuardianDTO) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		managerService := guardianship.NewManagerService(tx.Guardianships, tx.Children, tx.Users)

		// 转换 ID
		userID, err := parseUserID(dto.UserID)
		if err != nil {
			return err
		}
		childID, err := parseChildID(dto.ChildID)
		if err != nil {
			return err
		}

		// 转换关系
		relation := parseRelation(dto.Relation)

		// 调用领域服务添加监护人
		guardianship, err := managerService.AddGuardian(ctx, userID, childID, relation)
		if err != nil {
			return err
		}

		// 持久化监护关系
		return tx.Guardianships.Create(ctx, guardianship)
	})
}

// RemoveGuardian 移除监护人
func (s *guardianshipApplicationService) RemoveGuardian(ctx context.Context, dto RemoveGuardianDTO) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		managerService := guardianship.NewManagerService(tx.Guardianships, tx.Children, tx.Users)

		// 转换 ID
		userID, err := parseUserID(dto.UserID)
		if err != nil {
			return err
		}
		childID, err := parseChildID(dto.ChildID)
		if err != nil {
			return err
		}

		// 调用领域服务移除监护人
		guardianship, err := managerService.RemoveGuardian(ctx, userID, childID)
		if err != nil {
			return err
		}

		// 持久化修改
		return tx.Guardianships.Update(ctx, guardianship)
	})
}

// ===========================================
// === GuardianshipQueryApplicationService 实现 ===
// ===========================================

// GetByUserIDAndChildID 查询监护关系
func (s *guardianshipApplicationService) GetByUserIDAndChildID(ctx context.Context, userID string, childID string) (*GuardianshipResult, error) {
	var result *GuardianshipResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务

		// 转换 ID
		uid, err := parseUserID(userID)
		if err != nil {
			return err
		}
		cid, err := parseChildID(childID)
		if err != nil {
			return err
		}

		// 调用领域服务查询
		guardianship, err := tx.Guardianships.FindByUserIDAndChildID(ctx, uid, cid)
		if err != nil {
			return err
		}

		// 查询儿童信息
		child, err := tx.Children.FindByID(ctx, guardianship.Child)
		if err != nil {
			return err
		}

		// 转换为 DTO
		result = toGuardianshipResult(guardianship, child)
		return nil
	})

	return result, err
}

// ListChildrenByUserID 列出用户监护的所有儿童
func (s *guardianshipApplicationService) ListChildrenByUserID(ctx context.Context, userID string) ([]*GuardianshipResult, error) {
	var results []*GuardianshipResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务

		// 转换 ID
		uid, err := parseUserID(userID)
		if err != nil {
			return err
		}

		// 调用领域服务查询
		guardianships, err := tx.Guardianships.FindByUserID(ctx, uid)
		if err != nil {
			return err
		}

		// 查询儿童信息并转换为 DTO
		for _, g := range guardianships {
			child, err := tx.Children.FindByID(ctx, g.Child)
			if err != nil {
				continue
			}
			results = append(results, toGuardianshipResult(g, child))
		}

		return nil
	})

	return results, err
}

// ListGuardiansByChildID 列出儿童的所有监护人
func (s *guardianshipApplicationService) ListGuardiansByChildID(ctx context.Context, childID string) ([]*GuardianshipResult, error) {
	var results []*GuardianshipResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务

		// 转换 ID
		cid, err := parseChildID(childID)
		if err != nil {
			return err
		}

		// 调用领域服务查询
		guardianships, err := tx.Guardianships.FindByChildID(ctx, cid)
		if err != nil {
			return err
		}

		// 查询儿童信息并转换为 DTO
		child, err := tx.Children.FindByID(ctx, cid)
		if err != nil {
			return err
		}

		for _, g := range guardianships {
			results = append(results, toGuardianshipResult(g, child))
		}

		return nil
	})

	return results, err
}

// ==================================================
// ==== GuardianshipQueryApplicationService 实现 =====
// ==================================================

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

		userIDObj, err := parseUserID(userID)
		if err != nil {
			return err
		}

		childIDObj, err := parseChildID(childID)
		if err != nil {
			return err
		}

		isGuardian, err = tx.Guardianships.IsGuardian(ctx, userIDObj, childIDObj)
		return err
	})

	return isGuardian, err
}

// GetByUserIDAndChildID 查询监护关系
func (s *guardianshipQueryApplicationService) GetByUserIDAndChildID(ctx context.Context, userID string, childID string) (*GuardianshipResult, error) {
	var result *GuardianshipResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {

		userIDObj, err := parseUserID(userID)
		if err != nil {
			return err
		}

		childIDObj, err := parseChildID(childID)
		if err != nil {
			return err
		}

		guardianship, err := tx.Guardianships.FindByUserIDAndChildID(ctx, userIDObj, childIDObj)
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

		userIDObj, err := parseUserID(userID)
		if err != nil {
			return err
		}

		guardianships, err := tx.Guardianships.FindByUserID(ctx, userIDObj)
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

		childIDObj, err := parseChildID(childID)
		if err != nil {
			return err
		}

		guardianships, err := tx.Guardianships.FindByChildID(ctx, childIDObj)
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

// ============= DTO 转换辅助函数 =============

// parseUserID 解析用户ID字符串
func parseUserID(userID string) (meta.ID, error) {
	var id uint64
	_, err := fmt.Sscanf(userID, "%d", &id)
	if err != nil {
		return meta.FromUint64(0), err
	}

	return meta.FromUint64(id), nil
}

// parseChildID 解析儿童ID字符串
func parseChildID(childID string) (meta.ID, error) {
	var id uint64
	_, err := fmt.Sscanf(childID, "%d", &id)
	if err != nil {
		return meta.FromUint64(0), err
	}
	return meta.FromUint64(id), nil
}

// parseRelation 解析关系字符串
func parseRelation(relation string) domain.Relation {
	switch relation {
	case "parent", "父母":
		return domain.RelParent
	case "grandparents", "祖父母":
		return domain.RelGrandparents
	default:
		return domain.RelOther
	}
}

// toGuardianshipResult 将领域实体转换为 DTO
func toGuardianshipResult(guardianship *domain.Guardianship, child *childdomain.Child) *GuardianshipResult {
	if guardianship == nil {
		return nil
	}

	result := &GuardianshipResult{
		ID:            guardianship.ID.Uint64(),
		UserID:        guardianship.User.String(),
		ChildID:       guardianship.Child.String(),
		Relation:      string(guardianship.Rel), // Relation 是 string 类型
		EstablishedAt: guardianship.EstablishedAt.Format("2006-01-02 15:04:05"),
	}

	// 添加儿童信息
	if child != nil {
		result.ChildName = child.Name
		result.ChildGender = child.Gender.Value()
		result.ChildBirthday = child.Birthday.String()
	}

	return result
}
