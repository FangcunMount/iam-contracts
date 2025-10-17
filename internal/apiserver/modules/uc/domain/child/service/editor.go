package service

import (
	"context"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	port "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
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
// 领域逻辑：查询 + 修改实体
// 注意：不包括持久化，返回修改后的实体供应用层持久化
func (s *ChildProfileEditor) Rename(ctx context.Context, childID domain.ChildID, name string) (*domain.Child, error) {
	child, err := NewQueryService(s.repo).FindByID(ctx, childID)
	if err != nil {
		return nil, err
	}

	// 领域逻辑：重命名
	child.Rename(name)

	// 返回修改后的实体，由应用层持久化
	return child, nil
}

// UpdateIDCard 更新儿童身份证信息
// 领域逻辑：查询 + 修改实体
// 注意：不包括持久化，返回修改后的实体供应用层持久化
func (s *ChildProfileEditor) UpdateIDCard(ctx context.Context, childID domain.ChildID, idCard meta.IDCard) (*domain.Child, error) {
	child, err := NewQueryService(s.repo).FindByID(ctx, childID)
	if err != nil {
		return nil, err
	}

	// 领域逻辑：更新身份证
	child.UpdateIDCard(idCard)

	// 返回修改后的实体，由应用层持久化
	return child, nil
}

// UpdateProfile 更新儿童基础信息
// 领域逻辑：查询 + 修改实体
// 注意：不包括持久化，返回修改后的实体供应用层持久化
func (s *ChildProfileEditor) UpdateProfile(ctx context.Context, childID domain.ChildID, gender meta.Gender, birthday meta.Birthday) (*domain.Child, error) {
	child, err := NewQueryService(s.repo).FindByID(ctx, childID)
	if err != nil {
		return nil, err
	}

	// 领域逻辑：更新基础信息
	child.UpdateProfile(gender, birthday)

	// 返回修改后的实体，由应用层持久化
	return child, nil
}

// UpdateHeightWeight 更新儿童身高体重信息
// 领域逻辑：查询 + 修改实体
// 注意：不包括持久化，返回修改后的实体供应用层持久化
func (s *ChildProfileEditor) UpdateHeightWeight(ctx context.Context, childID domain.ChildID, height meta.Height, weight meta.Weight) (*domain.Child, error) {
	child, err := NewQueryService(s.repo).FindByID(ctx, childID)
	if err != nil {
		return nil, err
	}

	// 领域逻辑：更新身高体重
	child.UpdateHeightWeight(height, weight)

	// 返回修改后的实体，由应用层持久化
	return child, nil
}
