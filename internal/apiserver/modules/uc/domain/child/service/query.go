package service

import (
	"context"
	"errors"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	port "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/child/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
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

// FindSimilar 根据姓名、性别和生日查询相似的儿童档案列表
func (s *ChildQueryer) FindSimilar(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) ([]*domain.Child, error) {
	children, err := s.repo.FindSimilar(ctx, name, gender, birthday)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "find similar children by name(%s), gender(%s), birthday(%s) failed", name, gender.String(), birthday.String())
	}
	return toChildPointers(children), nil
}
