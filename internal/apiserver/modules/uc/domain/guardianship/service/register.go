package service

import (
	"context"
	"time"

	childDomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship"
	guardport "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship/port"
	userport "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// GuardianshipRegister 监护关系注册领域服务
type GuardianshipRegister struct {
	userRepo userport.UserRepository
}

// 确保实现
var _ guardport.GuardianshipRegister = (*GuardianshipRegister)(nil)

// NewRegisterService 创建注册服务
func NewRegisterService(ur userport.UserRepository) *GuardianshipRegister {
	return &GuardianshipRegister{
		userRepo: ur,
	}
}

// RegisterChildWithGuardian 同时注册儿童和监护关系
// 领域逻辑：验证监护人存在 + 创建儿童实体 + 创建监护关系实体
// 注意：不包括持久化，返回创建的两个实体供应用层在事务中持久化
// 注意：这是一个跨聚合的复杂用例，需要应用层协调事务
func (s *GuardianshipRegister) RegisterChildWithGuardian(ctx context.Context, params guardport.RegisterChildWithGuardianParams) (*domain.Guardianship, *childDomain.Child, error) {
	// 验证监护人（用户）存在
	userEntity, err := s.userRepo.FindByID(ctx, params.UserID)
	if err != nil {
		return nil, nil, perrors.WrapC(err, code.ErrDatabase, "find user(%d) failed", params.UserID.Uint64())
	}
	if userEntity == nil {
		return nil, nil, perrors.WithCode(code.ErrUserInvalid, "user(%d) not found", params.UserID.Uint64())
	}

	// 创建儿童实体
	child, err := childDomain.NewChild(
		params.Name,
		childDomain.WithGender(params.Gender),
		childDomain.WithBirthday(params.Birthday),
	)
	if err != nil {
		return nil, nil, err
	}

	// 设置可选的身份证信息
	if params.IDCard.String() != "" {
		child.UpdateIDCard(params.IDCard)
	}

	// 设置可选的身高体重信息
	if params.Height != nil || params.Weight != nil {
		height := child.Height
		if params.Height != nil {
			height = *params.Height
		}
		weight := child.Weight
		if params.Weight != nil {
			weight = *params.Weight
		}
		child.UpdateHeightWeight(height, weight)
	}

	// 创建监护关系实体
	// 注意：此时 child.ID 还是空的，需要在应用层持久化 child 后再设置
	guard := &domain.Guardianship{
		User:          params.UserID,
		Child:         child.ID, // 将由应用层在持久化 child 后更新
		Rel:           params.Relation,
		EstablishedAt: time.Now(),
	}

	// 返回两个实体，由应用层在事务中持久化
	// 应用层需要：1. 持久化 child 2. 更新 guard.Child = child.ID 3. 持久化 guard
	return guard, child, nil
}
