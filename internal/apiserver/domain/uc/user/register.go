package user

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// register 用户注册应用服务
type register struct {
	repo Repository
}

// 确保 register 实现了 Register 接口
var _ Register = (*register)(nil)

// NewRegisterService 创建用户注册服务
func NewRegisterService(repo Repository) *register {
	return &register{repo: repo}
}

// Register 注册新用户
// 领域逻辑：验证手机号唯一性 + 创建用户实体
// 注意：不包括持久化，返回创建的实体供应用层持久化
func (s *register) Register(ctx context.Context, name string, phone meta.Phone) (*User, error) {
	// 领域规则：确保手机号唯一
	if err := ensurePhoneUnique(ctx, s.repo, phone); err != nil {
		return nil, err
	}

	// 领域工厂：创建用户实体
	user, err := NewUser(name, phone)
	if err != nil {
		return nil, err
	}

	// 返回实体，由应用层负责持久化
	return user, nil
}
