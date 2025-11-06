package user

import (
	"context"
	"fmt"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/uow"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ============= 应用服务实现 =============

// ======================================
// ==== UserApplicationService 实现 =====
// ======================================

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
		registerService := user.NewRegisterService(tx.Users)

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

// =============================================
// ==== UserProfileApplicationService 实现 =====
// =============================================

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
		profileService := user.NewProfileService(tx.Users)

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
		profileService := user.NewProfileService(tx.Users)

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
		profileService := user.NewProfileService(tx.Users)

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

// ===========================================
// ==== UserStatusApplicationService 实现 =====
// ===========================================

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
		statusService := user.NewStatusService(tx.Users)

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
		statusService := user.NewStatusService(tx.Users)

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
		statusService := user.NewStatusService(tx.Users)

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

// ===========================================
// ==== UserQueryApplicationService 实现 =====
// ===========================================

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

		userIDObj, err := parseUserID(userID)
		if err != nil {
			return err
		}

		user, err := tx.Users.FindByID(ctx, userIDObj)
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

		phoneObj := meta.NewPhone(phone)

		user, err := tx.Users.FindByPhone(ctx, phoneObj)
		if err != nil {
			return err
		}

		result = toUserResult(user)
		return nil
	})

	return result, err
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
