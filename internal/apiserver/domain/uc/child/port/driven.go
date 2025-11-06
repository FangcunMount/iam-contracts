package port

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/child"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ChildRepository 儿童档案存储接口
type ChildRepository interface {
	Create(ctx context.Context, child *child.Child) error
	FindByID(ctx context.Context, id child.ChildID) (*child.Child, error)
	FindByName(ctx context.Context, name string) (*child.Child, error)
	FindByIDCard(ctx context.Context, idCard meta.IDCard) (*child.Child, error)
	FindListByName(ctx context.Context, name string) (children []*child.Child, err error)
	FindListByNameAndBirthday(ctx context.Context, name string, birthday meta.Birthday) (children []*child.Child, err error)
	FindSimilar(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) (children []*child.Child, err error)
	Update(ctx context.Context, child *child.Child) error
}
