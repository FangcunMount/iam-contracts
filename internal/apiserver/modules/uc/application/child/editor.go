package child

import (
	"context"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	port "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// ChildProfileEditor 儿童档案资料编辑应用服务
type ChildProfileEditor struct {
	repo port.ChildRepository
}

// 确保 ChildProfileEditor 实现 port.ChildProfileEditor
var _ port.ChildProfileEditor = (*ChildProfileEditor)(nil)

// NewProfileService 创建儿童档案资料服务
func NewProfileService(repo port.ChildRepository) *ChildProfileEditor {
	return &ChildProfileEditor{repo: repo}
}

// Rename 重命名儿童档案
func (s *ChildProfileEditor) Rename(ctx context.Context, childID domain.ChildID, name string) error {
	child, err := NewQueryService(s.repo).FindByID(ctx, childID)
	if err != nil {
		return err
	}

	child.Rename(name)

	if err := s.repo.Update(ctx, child); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "rename child(%s) failed", childID.String())
	}

	return nil
}

// UpdateIDCard 更新儿童身份证信息
func (s *ChildProfileEditor) UpdateIDCard(ctx context.Context, childID domain.ChildID, idCard meta.IDCard) error {
	child, err := NewQueryService(s.repo).FindByID(ctx, childID)
	if err != nil {
		return err
	}

	child.UpdateIDCard(idCard)

	if err := s.repo.Update(ctx, child); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "update id card for child(%s) failed", childID.String())
	}

	return nil
}

// UpdateProfile 更新儿童基础信息
func (s *ChildProfileEditor) UpdateProfile(ctx context.Context, childID domain.ChildID, gender meta.Gender, birthday meta.Birthday) error {
	child, err := NewQueryService(s.repo).FindByID(ctx, childID)
	if err != nil {
		return err
	}

	child.UpdateProfile(gender, birthday)

	if err := s.repo.Update(ctx, child); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "update profile for child(%s) failed", childID.String())
	}

	return nil
}

// UpdateHeightWeight 更新儿童身高体重信息
func (s *ChildProfileEditor) UpdateHeightWeight(ctx context.Context, childID domain.ChildID, height meta.Height, weight meta.Weight) error {
	child, err := NewQueryService(s.repo).FindByID(ctx, childID)
	if err != nil {
		return err
	}

	child.UpdateHeightWeight(height, weight)

	if err := s.repo.Update(ctx, child); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "update height/weight for child(%s) failed", childID.String())
	}

	return nil
}
