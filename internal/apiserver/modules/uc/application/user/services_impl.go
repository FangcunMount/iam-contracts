package user

import (
	"context"
	"fmt"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/uow"
	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	domainservice "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user/service"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// userApplicationService 用户应用服务实现
type userApplicationService struct {
	uow uow.UnitOfWork
}

// NewUserApplicationService 创建用户应用服务
func NewUserApplicationService(uow uow.UnitOfWork) UserApplicationService {
	return &userApplicationService{uow: uow}
}

// Register 注册新用户
func (s *userApplicationService) Register(ctx context.Context, dto RegisterUserDTO) (*UserResult, error) {
	var result *UserResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		registerService := domainservice.NewRegisterService(tx.Users)

		// 转换 DTO 为值对象
		phone := meta.NewPhone(dto.Phone)

		// 调用领域服务创建用户实体
		user, err := registerService.Register(ctx, dto.Name, phone)
		if err != nil {
			return err
		}

		// 设置可选的邮箱
		if dto.Email != "" {
			email := meta.NewEmail(dto.Email)
			user.UpdateEmail(email)
		}

		// 持久化用户
		if err := tx.Users.Create(ctx, user); err != nil {
			return err
		}

		// 转换为 DTO
		result = toUserResult(user)
		return nil
	})

	return result, err
}

// GetByID 根据 ID 查询用户
func (s *userApplicationService) GetByID(ctx context.Context, userID string) (*UserResult, error) {
	var result *UserResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		queryService := domainservice.NewQueryService(tx.Users)

		// 转换 ID
		id, err := parseUserID(userID)
		if err != nil {
			return err
		}

		// 调用领域服务查询
		user, err := queryService.FindByID(ctx, id)
		if err != nil {
			return err
		}

		// 转换为 DTO
		result = toUserResult(user)
		return nil
	})

	return result, err
}

// GetByPhone 根据手机号查询用户
func (s *userApplicationService) GetByPhone(ctx context.Context, phone string) (*UserResult, error) {
	var result *UserResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		queryService := domainservice.NewQueryService(tx.Users)

		// 转换手机号
		phoneVO := meta.NewPhone(phone)

		// 调用领域服务查询
		user, err := queryService.FindByPhone(ctx, phoneVO)
		if err != nil {
			return err
		}

		// 转换为 DTO
		result = toUserResult(user)
		return nil
	})

	return result, err
}

// userProfileApplicationService 用户资料应用服务实现
type userProfileApplicationService struct {
	uow uow.UnitOfWork
}

// NewUserProfileApplicationService 创建用户资料应用服务
func NewUserProfileApplicationService(uow uow.UnitOfWork) UserProfileApplicationService {
	return &userProfileApplicationService{uow: uow}
}

// Rename 修改用户名称
func (s *userProfileApplicationService) Rename(ctx context.Context, userID string, newName string) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		profileService := domainservice.NewProfileService(tx.Users)

		// 转换 ID
		id, err := parseUserID(userID)
		if err != nil {
			return err
		}

		// 调用领域服务修改名称
		user, err := profileService.Rename(ctx, id, newName)
		if err != nil {
			return err
		}

		// 持久化修改
		return tx.Users.Update(ctx, user)
	})
}

// UpdateContact 更新联系方式
func (s *userProfileApplicationService) UpdateContact(ctx context.Context, dto UpdateContactDTO) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		profileService := domainservice.NewProfileService(tx.Users)

		// 转换 ID
		id, err := parseUserID(dto.UserID)
		if err != nil {
			return err
		}

		// 转换值对象
		phone := meta.NewPhone(dto.Phone)
		email := meta.NewEmail(dto.Email)

		// 调用领域服务更新联系方式
		user, err := profileService.UpdateContact(ctx, id, phone, email)
		if err != nil {
			return err
		}

		// 持久化修改
		return tx.Users.Update(ctx, user)
	})
}

// UpdateIDCard 更新身份证
func (s *userProfileApplicationService) UpdateIDCard(ctx context.Context, userID string, idCard string) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		profileService := domainservice.NewProfileService(tx.Users)

		// 转换 ID
		id, err := parseUserID(userID)
		if err != nil {
			return err
		}

		// 转换身份证 (NewIDCard 需要name和number两个参数，这里我们只传number，name留空)
		idCardVO := meta.NewIDCard("", idCard)

		// 调用领域服务更新身份证
		user, err := profileService.UpdateIDCard(ctx, id, idCardVO)
		if err != nil {
			return err
		}

		// 持久化修改
		return tx.Users.Update(ctx, user)
	})
}

// userStatusApplicationService 用户状态应用服务实现
type userStatusApplicationService struct {
	uow uow.UnitOfWork
}

// NewUserStatusApplicationService 创建用户状态应用服务
func NewUserStatusApplicationService(uow uow.UnitOfWork) UserStatusApplicationService {
	return &userStatusApplicationService{uow: uow}
}

// Activate 激活用户
func (s *userStatusApplicationService) Activate(ctx context.Context, userID string) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		statusService := domainservice.NewStatusService(tx.Users)

		// 转换 ID
		id, err := parseUserID(userID)
		if err != nil {
			return err
		}

		// 调用领域服务激活用户
		user, err := statusService.Activate(ctx, id)
		if err != nil {
			return err
		}

		// 持久化修改
		return tx.Users.Update(ctx, user)
	})
}

// Deactivate 停用用户
func (s *userStatusApplicationService) Deactivate(ctx context.Context, userID string) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		statusService := domainservice.NewStatusService(tx.Users)

		// 转换 ID
		id, err := parseUserID(userID)
		if err != nil {
			return err
		}

		// 调用领域服务停用用户
		user, err := statusService.Deactivate(ctx, id)
		if err != nil {
			return err
		}

		// 持久化修改
		return tx.Users.Update(ctx, user)
	})
}

// Block 封禁用户
func (s *userStatusApplicationService) Block(ctx context.Context, userID string) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 创建领域服务
		statusService := domainservice.NewStatusService(tx.Users)

		// 转换 ID
		id, err := parseUserID(userID)
		if err != nil {
			return err
		}

		// 调用领域服务封禁用户
		user, err := statusService.Block(ctx, id)
		if err != nil {
			return err
		}

		// 持久化修改
		return tx.Users.Update(ctx, user)
	})
}

// ============= DTO 转换辅助函数 =============

// parseUserID 解析用户ID字符串
func parseUserID(userID string) (domain.UserID, error) {
	// 将字符串转换为uint64
	var id uint64
	_, err := fmt.Sscanf(userID, "%d", &id)
	if err != nil {
		return domain.UserID{}, err
	}
	return domain.NewUserID(id), nil
}

// toUserResult 将领域实体转换为 DTO
func toUserResult(user *domain.User) *UserResult {
	if user == nil {
		return nil
	}

	return &UserResult{
		ID:     user.ID.String(),
		Name:   user.Name,
		Phone:  user.Phone.String(),
		Email:  user.Email.String(),
		IDCard: user.IDCard.String(),
		Status: user.Status,
	}
}
