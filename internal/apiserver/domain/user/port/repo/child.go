package repo

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// ChildRepository 儿童档案存储接口
type ChildRepository interface {
	Create(ctx context.Context, child user.Child) error
	FindByID(ctx context.Context, id user.ChildID) (user.Child, error)
	FindByName(ctx context.Context, name string) (user.Child, error)
	FindSimilar(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) (children []user.Child, err error)
	Update(ctx context.Context, child user.Child) error
}
