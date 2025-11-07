package child

import (
	"context"
	"fmt"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/uow"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/child"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ============= 应用服务实现 =============

// ======================================
// ==== ChildApplicationService 实现 =====
// ======================================

// childApplicationService 儿童应用服务实现
type childApplicationService struct {
	uow uow.UnitOfWork
}

// NewChildApplicationService 创建儿童应用服务
func NewChildApplicationService(uow uow.UnitOfWork) ChildApplicationService {
	return &childApplicationService{uow: uow}
}

// Register 注册新儿童档案
func (s *childApplicationService) Register(ctx context.Context, dto RegisterChildDTO) (*ChildResult, error) {
	var result *ChildResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建验证器
		validator := domain.NewValidator(tx.Children)

		// 转换 DTO 为值对象
		gender := parseGender(dto.Gender)
		birthday := meta.NewBirthday(dto.Birthday)

		// 验证注册参数
		if err := validator.ValidateRegister(ctx, dto.Name, gender, birthday); err != nil {
			return err
		}

		// 创建儿童实体
		var newChild *domain.Child
		var err error
		var idCard meta.IDCard

		if dto.IDCard != "" {
			// 带身份证注册
			idCard, err = meta.NewIDCard(dto.Name, dto.IDCard)
			if err != nil {
				return err
			}
			newChild, err = domain.NewChild(
				dto.Name,
				domain.WithGender(gender),
				domain.WithBirthday(birthday),
				domain.WithIDCard(idCard),
			)
		} else {
			// 普通注册
			newChild, err = domain.NewChild(
				dto.Name,
				domain.WithGender(gender),
				domain.WithBirthday(birthday),
			)
		}

		if err != nil {
			return err
		}

		// 设置可选的身高体重
		if dto.Height != nil || dto.Weight != nil {
			height := newChild.Height
			if dto.Height != nil {
				h, _ := meta.NewHeightFromFloat(float64(*dto.Height))
				height = h
			}
			weight := newChild.Weight
			if dto.Weight != nil {
				// DTO中的Weight是克，需要转换为千克
				w, _ := meta.NewWeightFromFloat(float64(*dto.Weight) / 1000.0)
				weight = w
			}
			newChild.UpdateHeightWeight(height, weight)
		}

		// 持久化儿童
		if err := tx.Children.Create(ctx, newChild); err != nil {
			return err
		}

		// 转换为 DTO
		result = toChildResult(newChild)
		return nil
	})

	return result, err
}

// ==============================================
// ==== ChildProfileApplicationService 实现 =====
// ==============================================

// childProfileApplicationService 儿童资料应用服务实现
type childProfileApplicationService struct {
	uow uow.UnitOfWork
}

// NewChildProfileApplicationService 创建儿童资料应用服务
func NewChildProfileApplicationService(uow uow.UnitOfWork) ChildProfileApplicationService {
	return &childProfileApplicationService{uow: uow}
}

// Rename 修改儿童姓名
func (s *childProfileApplicationService) Rename(ctx context.Context, childID string, newName string) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		validator := domain.NewValidator(tx.Children)
		profileService := domain.NewProfileService(tx.Children, validator)

		// 转换 ID
		id, err := parseChildID(childID)
		if err != nil {
			return err
		}

		// 调用领域服务修改姓名
		modifiedChild, err := profileService.Rename(ctx, id, newName)
		if err != nil {
			return err
		}

		// 持久化修改
		return tx.Children.Update(ctx, modifiedChild)
	})
}

// UpdateIDCard 更新身份证
func (s *childProfileApplicationService) UpdateIDCard(ctx context.Context, childID string, name string, idCard string) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		validator := domain.NewValidator(tx.Children)
		profileService := domain.NewProfileService(tx.Children, validator)

		// 转换 ID
		id, err := parseChildID(childID)
		if err != nil {
			return err
		}

		// 转换身份证
		idCardVO, err := meta.NewIDCard(name, idCard)
		if err != nil {
			return err
		}

		// 调用领域服务更新身份证
		modifiedChild, err := profileService.UpdateIDCard(ctx, id, idCardVO)
		if err != nil {
			return err
		}

		// 持久化修改
		return tx.Children.Update(ctx, modifiedChild)
	})
}

// UpdateProfile 更新基本信息（性别、生日）
func (s *childProfileApplicationService) UpdateProfile(ctx context.Context, dto UpdateChildProfileDTO) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		validator := domain.NewValidator(tx.Children)
		profileService := domain.NewProfileService(tx.Children, validator)

		// 转换 ID
		id, err := parseChildID(dto.ChildID)
		if err != nil {
			return err
		}

		// 转换值对象
		gender := parseGender(dto.Gender)
		birthday := meta.NewBirthday(dto.Birthday)

		// 调用领域服务更新资料
		modifiedChild, err := profileService.UpdateProfile(ctx, id, gender, birthday)
		if err != nil {
			return err
		}

		// 持久化修改
		return tx.Children.Update(ctx, modifiedChild)
	})
}

// UpdateHeightWeight 更新身高体重
func (s *childProfileApplicationService) UpdateHeightWeight(ctx context.Context, dto UpdateHeightWeightDTO) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		validator := domain.NewValidator(tx.Children)
		profileService := domain.NewProfileService(tx.Children, validator)

		// 转换 ID
		id, err := parseChildID(dto.ChildID)
		if err != nil {
			return err
		}

		// 转换值对象
		height, _ := meta.NewHeightFromFloat(float64(dto.Height))
		// DTO中的Weight是克，需要转换为千克
		weight, _ := meta.NewWeightFromFloat(float64(dto.Weight) / 1000.0)

		// 调用领域服务更新身高体重
		modifiedChild, err := profileService.UpdateHeightWeight(ctx, id, height, weight)
		if err != nil {
			return err
		}

		// 持久化修改
		return tx.Children.Update(ctx, modifiedChild)
	})
}

// ============================================
// ==== ChildQueryApplicationService 实现 =====
// ============================================

// childQueryApplicationService 儿童查询应用服务实现
type childQueryApplicationService struct {
	uow uow.UnitOfWork
}

// NewChildQueryApplicationService 创建儿童查询应用服务
func NewChildQueryApplicationService(uow uow.UnitOfWork) ChildQueryApplicationService {
	return &childQueryApplicationService{uow: uow}
}

// GetByID 根据 ID 查询儿童
func (s *childQueryApplicationService) GetByID(ctx context.Context, childID string) (*ChildResult, error) {
	var result *ChildResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {

		childIDObj, err := parseChildID(childID)
		if err != nil {
			return err
		}

		child, err := tx.Children.FindByID(ctx, childIDObj)
		if err != nil {
			return err
		}

		result = toChildResult(child)
		return nil
	})

	return result, err
}

// GetByIDCard 根据身份证查询儿童
func (s *childQueryApplicationService) GetByIDCard(ctx context.Context, idCard string) (*ChildResult, error) {
	var result *ChildResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {

		idCardObj, err := meta.NewIDCard("", idCard)
		if err != nil {
			return err
		}

		child, err := tx.Children.FindByIDCard(ctx, idCardObj)
		if err != nil {
			return err
		}

		result = toChildResult(child)
		return nil
	})

	return result, err
}

// FindSimilar 查找相似儿童（姓名、性别、生日）
func (s *childQueryApplicationService) FindSimilar(ctx context.Context, name string, gender string, birthday string) ([]*ChildResult, error) {
	var results []*ChildResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {

		genderObj := parseGender(gender)
		birthdayObj := meta.NewBirthday(birthday)

		children, err := tx.Children.FindSimilar(ctx, name, genderObj, birthdayObj)
		if err != nil {
			return err
		}

		results = toChildResults(children)
		return nil
	})

	return results, err
}

// ============= DTO 转换辅助函数 =============

// parseChildID 解析儿童ID字符串
func parseChildID(childID string) (meta.ID, error) {
	var id uint64
	_, err := fmt.Sscanf(childID, "%d", &id)
	if err != nil {
		return meta.FromUint64(0), err
	}
	parsedChildID := meta.FromUint64(id)
	return parsedChildID, nil
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

// toChildResult 将领域实体转换为 DTO
func toChildResult(child *domain.Child) *ChildResult {
	if child == nil {
		return nil
	}

	// Height和Weight使用Tenths()方法获取内部值
	// Height: tenths of cm (170.5cm -> 1705) -> 需要返回cm
	// Weight: tenths of kg (70.5kg -> 705) -> 需要返回克
	heightTenths := child.Height.Tenths()
	weightTenths := child.Weight.Tenths()

	return &ChildResult{
		ID:       child.ID.String(),
		Name:     child.Name,
		IDCard:   child.IDCard.String(),
		Gender:   child.Gender.String(),
		Birthday: child.Birthday.String(),
		Height:   uint32(heightTenths / 10),  // tenths of cm -> cm
		Weight:   uint32(weightTenths * 100), // tenths of kg -> grams (1kg=1000g, 0.1kg=100g)
	}
}

// toChildResults 将领域实体列表转换为 DTO 列表
func toChildResults(children []*domain.Child) []*ChildResult {
	results := make([]*ChildResult, 0, len(children))
	for _, child := range children {
		if child != nil {
			results = append(results, toChildResult(child))
		}
	}
	return results
}
