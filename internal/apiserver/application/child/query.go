package child

import (
	"context"
	"errors"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child"
	port "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"gorm.io/gorm"
)

// ChildQueryer 儿童档案查询应用服务
type ChildQueryer struct {
	repo port.ChildRepository
}

// 确保 ChildQueryer 实现 port.ChildQueryer
var _ port.ChildQueryer = (*ChildQueryer)(nil)

// NewQueryService 创建儿童档案查询服务
func NewQueryService(repo port.ChildRepository) *ChildQueryer {
	return &ChildQueryer{repo: repo}
}

// FindByID 根据ID查询儿童档案
func (s *ChildQueryer) FindByID(ctx context.Context, childID domain.ChildID) (*domain.Child, error) {
	c, err := s.repo.FindByID(ctx, childID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, perrors.WithCode(code.ErrUserNotFound, "child(%s) not found", childID.String())
		}
		return nil, perrors.WrapC(err, code.ErrDatabase, "find child by id(%s) failed", childID.String())
	}
	return c, nil
}

// FindByIDCard 根据身份证号查询儿童档案
func (s *ChildQueryer) FindByIDCard(ctx context.Context, idCard meta.IDCard) (*domain.Child, error) {
	c, err := s.repo.FindByIDCard(ctx, idCard)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, perrors.WithCode(code.ErrUserNotFound, "child with idcard(%s) not found", idCard.String())
		}
		return nil, perrors.WrapC(err, code.ErrDatabase, "find child by idcard(%s) failed", idCard.String())
	}
	return c, nil
}

// FindListByName 根据姓名查询儿童档案列表
func (s *ChildQueryer) FindListByName(ctx context.Context, name string) ([]*domain.Child, error) {
	children, err := s.repo.FindListByName(ctx, name)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "find children by name(%s) failed", name)
	}
	return toChildPointers(children), nil
}

// FindListByNameAndBirthday 根据姓名和生日查询儿童档案列表
func (s *ChildQueryer) FindListByNameAndBirthday(ctx context.Context, name string, birthday meta.Birthday) ([]*domain.Child, error) {
	children, err := s.repo.FindListByNameAndBirthday(ctx, name, birthday)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "find children by name(%s) and birthday(%s) failed", name, birthday.String())
	}
	return toChildPointers(children), nil
}
