package port

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// ChildRepository 儿童档案存储接口
type ChildRepository interface {
	Create(ctx context.Context, child child.Child) error
	FindByID(ctx context.Context, id child.ChildID) (child.Child, error)
	FindByName(ctx context.Context, name string) (child.Child, error)
	FindSimilar(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) (children []child.Child, err error)
	Update(ctx context.Context, child child.Child) error
}
