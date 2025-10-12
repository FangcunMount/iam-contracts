package child

import (
	"context"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child"
	port "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// ChildFinder 相似儿童档案查询应用服务
type SimilarChildFinder struct {
	repo port.ChildRepository
}

// 确保 SimilarChildFinder 实现 port.SimilarChildFinder
var _ port.SimilarChildFinder = (*SimilarChildFinder)(nil)

// NewFinderService 创建儿童档案查询服务
func NewFinderService(repo port.ChildRepository) *SimilarChildFinder {
	return &SimilarChildFinder{repo: repo}
}

// FindChilds 查找相似儿童档案
func (s *SimilarChildFinder) FindChilds(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) ([]*domain.Child, error) {
	children, err := s.repo.FindSimilar(ctx, name, gender, birthday)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "find similar children(%s) failed", name)
	}

	// 如果没有找到相似的儿童档案，返回空列表
	if len(children) == 0 {
		return []*domain.Child{}, nil
	}

	// 找到相似儿童档案，返回结果
	return toChildPointers(children), nil
}
