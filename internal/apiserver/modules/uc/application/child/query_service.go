package child

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/uow"
	domainservice "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child/service"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// ============= 查询应用服务 =============

// ChildQueryApplicationService 儿童查询应用服务（只读）
type ChildQueryApplicationService interface {
	// GetByID 根据 ID 查询儿童
	GetByID(ctx context.Context, childID string) (*ChildResult, error)
	// GetByIDCard 根据身份证查询儿童
	GetByIDCard(ctx context.Context, idCard string) (*ChildResult, error)
	// FindSimilar 查找相似儿童（姓名、性别、生日）
	FindSimilar(ctx context.Context, name string, gender string, birthday string) ([]*ChildResult, error)
}

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
		queryService := domainservice.NewQueryService(tx.Children)

		childIDObj, err := parseChildID(childID)
		if err != nil {
			return err
		}

		child, err := queryService.FindByID(ctx, childIDObj)
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
		queryService := domainservice.NewQueryService(tx.Children)

		idCardObj := meta.NewIDCard("", idCard)

		child, err := queryService.FindByIDCard(ctx, idCardObj)
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
		queryService := domainservice.NewQueryService(tx.Children)

		genderObj := parseGender(gender)
		birthdayObj := meta.NewBirthday(birthday)

		children, err := queryService.FindSimilar(ctx, name, genderObj, birthdayObj)
		if err != nil {
			return err
		}

		results = toChildResults(children)
		return nil
	})

	return results, err
}
