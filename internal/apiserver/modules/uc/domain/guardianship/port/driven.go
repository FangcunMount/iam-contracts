package port

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	guardianship "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
)

// GuardianshipRepository 监护关系存储接口
type GuardianshipRepository interface {
	Create(ctx context.Context, guardianship *guardianship.Guardianship) error
	FindByID(ctx context.Context, id idutil.ID) (*guardianship.Guardianship, error)
	FindByChildID(ctx context.Context, id child.ChildID) (guardianships []*guardianship.Guardianship, err error)
	FindByUserID(ctx context.Context, id user.UserID) (guardianships []*guardianship.Guardianship, err error)
	Update(ctx context.Context, guardianship *guardianship.Guardianship) error
}
