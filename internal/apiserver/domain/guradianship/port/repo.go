package port

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child"
	guardianship "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/guradianship"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
)

// GuardianshipRepository 监护关系存储接口
type GuardianshipRepository interface {
	Create(ctx context.Context, guardianship *guardianship.Guardianship) error
	FindByID(ctx context.Context, id idutil.ID) (*guardianship.Guardianship, error)
	FindByChildID(ctx context.Context, id child.ChildID) (guardianships []*guardianship.Guardianship, err error)
	FindByUserID(ctx context.Context, id user.UserID) (guardianships []*guardianship.Guardianship, err error)
	Update(ctx context.Context, guardianship *guardianship.Guardianship) error
}
