package child

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ChildRegister 儿童档案注册应用服务
type ChildRegister struct {
	repo Repository
}

// 确保 ChildRegister 实现 Register 接口
var _ Register = (*ChildRegister)(nil)

// NewRegisterService 创建儿童档案注册服务
func NewRegisterService(repo Repository) *ChildRegister {
	return &ChildRegister{repo: repo}
}

// Register 注册新的儿童档案
// 领域逻辑：创建儿童实体
// 注意：不包括持久化，返回创建的实体供应用层持久化
func (s *ChildRegister) Register(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) (*Child, error) {
	child, err := NewChild(
		name,
		WithGender(gender),
		WithBirthday(birthday),
	)
	if err != nil {
		return nil, err
	}

	// 返回创建的实体，由应用层持久化
	return child, nil
}

// RegisterWithIDCard 注册新的儿童档案（带身份证）
// 领域逻辑：创建儿童实体并设置身份证信息
// 注意：不包括持久化，返回创建的实体供应用层持久化
func (s *ChildRegister) RegisterWithIDCard(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday, idCard meta.IDCard) (*Child, error) {
	child, err := NewChild(
		name,
		WithGender(gender),
		WithBirthday(birthday),
		WithIDCard(idCard),
	)
	if err != nil {
		return nil, err
	}

	// 返回创建的实体，由应用层持久化
	return child, nil
}
