package child

import (
	"context"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	port "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// ChildRegister 儿童档案注册应用服务
type ChildRegister struct {
	repo port.ChildRepository
}

// 确保 ChildRegister 实现 port.ChildRegister
var _ port.ChildRegister = (*ChildRegister)(nil)

// NewRegisterService 创建儿童档案注册服务
func NewRegisterService(repo port.ChildRepository) *ChildRegister {
	return &ChildRegister{repo: repo}
}

// Register 注册新的儿童档案
func (s *ChildRegister) Register(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) (*domain.Child, error) {
	child, err := domain.NewChild(
		name,
		domain.WithGender(gender),
		domain.WithBirthday(birthday),
	)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, child); err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "create child(%s) failed", name)
	}

	children, err := s.repo.FindSimilar(ctx, name, gender, birthday)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "find similar child(%s) failed", name)
	}

	if len(children) == 0 {
		return nil, perrors.WithCode(code.ErrUserInvalid, "child(%s) not persisted", name)
	}

	return latestChild(children), nil
}
