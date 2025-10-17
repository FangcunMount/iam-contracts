package user

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/uow"
	domainservice "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user/service"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// ============= 查询应用服务 =============

// UserQueryApplicationService 用户查询应用服务（只读）
type UserQueryApplicationService interface {
	// GetByID 根据 ID 查询用户
	GetByID(ctx context.Context, userID string) (*UserResult, error)
	// GetByPhone 根据手机号查询用户
	GetByPhone(ctx context.Context, phone string) (*UserResult, error)
}

// userQueryApplicationService 用户查询应用服务实现
type userQueryApplicationService struct {
	uow uow.UnitOfWork
}

// NewUserQueryApplicationService 创建用户查询应用服务
func NewUserQueryApplicationService(uow uow.UnitOfWork) UserQueryApplicationService {
	return &userQueryApplicationService{uow: uow}
}

// GetByID 根据 ID 查询用户
func (s *userQueryApplicationService) GetByID(ctx context.Context, userID string) (*UserResult, error) {
	var result *UserResult

	// 查询操作也通过 UoW，但不需要写操作，可以直接使用只读事务
	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		queryService := domainservice.NewQueryService(tx.Users)

		userIDObj, err := parseUserID(userID)
		if err != nil {
			return err
		}

		user, err := queryService.FindByID(ctx, userIDObj)
		if err != nil {
			return err
		}

		result = toUserResult(user)
		return nil
	})

	return result, err
}

// GetByPhone 根据手机号查询用户
func (s *userQueryApplicationService) GetByPhone(ctx context.Context, phone string) (*UserResult, error) {
	var result *UserResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		queryService := domainservice.NewQueryService(tx.Users)

		phoneObj := meta.NewPhone(phone)

		user, err := queryService.FindByPhone(ctx, phoneObj)
		if err != nil {
			return err
		}

		result = toUserResult(user)
		return nil
	})

	return result, err
}
