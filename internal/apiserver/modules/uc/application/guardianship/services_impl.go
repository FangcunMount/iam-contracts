package guardianship

import (
	"context"
	"fmt"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/uow"
	childdomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship"
	guardport "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship/port"
	domainservice "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship/service"
	userdomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// ============= 应用服务实现 =============

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
		managerService := domainservice.NewManagerService(tx.Guardianships, tx.Children, tx.Users)

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
		managerService := domainservice.NewManagerService(tx.Guardianships, tx.Children, tx.Users)

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

// RegisterChildWithGuardian 同时注册儿童和监护关系
func (s *guardianshipApplicationService) RegisterChildWithGuardian(ctx context.Context, dto RegisterChildWithGuardianDTO) (*GuardianshipResult, error) {
	var result *GuardianshipResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		registerService := domainservice.NewRegisterService(tx.Users)

		// 转换 ID 和值对象
		userID, err := parseUserID(dto.UserID)
		if err != nil {
			return err
		}

		gender := parseGender(dto.ChildGender)
		birthday := meta.NewBirthday(dto.ChildBirthday)
		relation := parseRelation(dto.Relation)

		// 构建参数
		params := guardport.RegisterChildWithGuardianParams{
			Name:     dto.ChildName,
			Gender:   gender,
			Birthday: birthday,
			UserID:   userID,
			Relation: relation,
		}

		// 设置可选参数
		if dto.ChildIDCard != "" {
			params.IDCard = meta.NewIDCard("", dto.ChildIDCard)
		}
		if dto.ChildHeight != nil {
			h, _ := meta.NewHeightFromFloat(float64(*dto.ChildHeight))
			params.Height = &h
		}
		if dto.ChildWeight != nil {
			w, _ := meta.NewWeightFromFloat(float64(*dto.ChildWeight) / 1000.0)
			params.Weight = &w
		}

		// 调用领域服务创建儿童和监护关系
		guardianship, child, err := registerService.RegisterChildWithGuardian(ctx, params)
		if err != nil {
			return err
		}

		// 持久化儿童（先持久化以生成ID）
		if err := tx.Children.Create(ctx, child); err != nil {
			return err
		}

		// 更新监护关系的儿童ID
		guardianship.Child = child.ID

		// 持久化监护关系
		if err := tx.Guardianships.Create(ctx, guardianship); err != nil {
			return err
		}

		// 转换为 DTO
		result = toGuardianshipResult(guardianship, child)
		return nil
	})

	return result, err
}

// IsGuardian 检查是否为监护人
func (s *guardianshipApplicationService) IsGuardian(ctx context.Context, userID string, childID string) (bool, error) {
	var isGuardian bool

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		queryService := domainservice.NewQueryService(tx.Guardianships, tx.Children)

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
		isGuardian, err = queryService.IsGuardian(ctx, uid, cid)
		return err
	})

	return isGuardian, err
}

// GetByUserIDAndChildID 查询监护关系
func (s *guardianshipApplicationService) GetByUserIDAndChildID(ctx context.Context, userID string, childID string) (*GuardianshipResult, error) {
	var result *GuardianshipResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		queryService := domainservice.NewQueryService(tx.Guardianships, tx.Children)

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
		guardianship, err := queryService.FindByUserIDAndChildID(ctx, uid, cid)
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
		queryService := domainservice.NewQueryService(tx.Guardianships, tx.Children)

		// 转换 ID
		uid, err := parseUserID(userID)
		if err != nil {
			return err
		}

		// 调用领域服务查询
		guardianships, err := queryService.FindListByUserID(ctx, uid)
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
		queryService := domainservice.NewQueryService(tx.Guardianships, tx.Children)

		// 转换 ID
		cid, err := parseChildID(childID)
		if err != nil {
			return err
		}

		// 调用领域服务查询
		guardianships, err := queryService.FindListByChildID(ctx, cid)
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

// ============= DTO 转换辅助函数 =============

// parseUserID 解析用户ID字符串
func parseUserID(userID string) (userdomain.UserID, error) {
	var id uint64
	_, err := fmt.Sscanf(userID, "%d", &id)
	if err != nil {
		return userdomain.UserID{}, err
	}
	return userdomain.NewUserID(id), nil
}

// parseChildID 解析儿童ID字符串
func parseChildID(childID string) (childdomain.ChildID, error) {
	var id uint64
	_, err := fmt.Sscanf(childID, "%d", &id)
	if err != nil {
		return childdomain.ChildID{}, err
	}
	return childdomain.NewChildID(id), nil
}

// parseGender 解析性别字符串
func parseGender(gender string) meta.Gender {
	switch gender {
	case "male", "男":
		return meta.GenderMale
	case "female", "女":
		return meta.GenderFemale
	default:
		return meta.GenderOther
	}
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
		ID:            int64(guardianship.ID),
		UserID:        guardianship.User.String(),
		ChildID:       guardianship.Child.String(),
		Relation:      string(guardianship.Rel), // Relation 是 string 类型
		EstablishedAt: guardianship.EstablishedAt.Format("2006-01-02 15:04:05"),
	}

	// 添加儿童信息
	if child != nil {
		result.ChildName = child.Name
		result.ChildGender = child.Gender.String()
		result.ChildBirthday = child.Birthday.String()
	}

	return result
}
